package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"devo/internal/config"
	"devo/internal/core/agentloop"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/core/skills"
	"devo/internal/interfaces/rest"
	"devo/internal/interfaces/tui"
	"devo/internal/pkg/logging"
	"devo/internal/storage/sqlite"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/tools"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type App struct {
	cfg           *config.GlobalConfig
	store         session.SessionStore
	loop          *agentloop.Loop
	handler       *rest.Handler
	skillsMgr     *skills.Manager
	mcpMgr        *mcp.Manager
	memoryMgr     *memory.Manager
	toolRegistry  *tools.Registry
	llm           llmclient.Client
	bgProcManager *tools.BackgroundProcessManager
	addr          string
	baseURL       string
	port          int
	devoDir       string
	tuiMode       bool
	webMode       bool
}

func NewApp(tuiMode, webMode bool, portFlag int) (*App, error) {
	app := &App{
		tuiMode: tuiMode,
		webMode: webMode,
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("[devo] Get working directory: %v", err)
	}
	cfg, err := config.LoadFullConfig(wd)
	if err != nil {
		logging.Warn(context.Background(), "config warning",
			"error", err,
		)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	app.cfg = cfg

	port := portFlag
	if port == 0 {
		port, err = findFreePort()
		if err != nil {
			return nil, fmt.Errorf("[devo] Find free port: %v", err)
		}
	}
	app.port = port
	app.addr = fmt.Sprintf("127.0.0.1:%d", port)
	app.baseURL = fmt.Sprintf("http://%s", app.addr)

	if err := app.initDB(); err != nil {
		return nil, err
	}

	app.initRegistry()
	app.initLLM()
	if err := app.initMemory(); err != nil {
		return nil, err
	}
	app.initSkills()
	app.initMCP()
	app.initTools()
	app.initHandler()
	return app, nil
}

func (a *App) initDB() error {
	dbPath := a.cfg.DBPath
	if dbPath == "" {
		dbPath = config.DBPath()
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,        // Don't include params in the SQL log
			Colorful:                  false,        // Disable color
		},
	)
	db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("[devo] Failed to open database: %v", err)
	}

	store, err := sqlite.NewGormStore(db)
	if err != nil {
		return fmt.Errorf("[devo] Failed to create store: %v", err)
	}
	a.store = store

	var _ session.SessionStore = store
	return nil
}

func (a *App) initRegistry() {
	a.toolRegistry = tools.NewRegistry()
}

func (a *App) initTools() {
	bgProcManager := tools.NewBackgroundProcessManager()
	a.bgProcManager = bgProcManager
	// Wire bidirectional: loop needs the manager to stop processes on
	// cancel/complete; manager needs the loop to forward background output
	// to the session EventBus via ForwardBackgroundOutput.
	a.loop.SetBackgroundProcessManager(bgProcManager)
	bgProcManager.SetOutputForwarder(a.loop)
	a.toolRegistry.Register(&tools.GlobTool{})
	a.toolRegistry.Register(&tools.ReadFileTool{})
	a.toolRegistry.Register(&tools.ListFilesTool{})
	a.toolRegistry.Register(&tools.SearchCodebaseTool{})
	a.toolRegistry.Register(&tools.WriteFileTool{})
	a.toolRegistry.Register(&tools.EditFileTool{})
	a.toolRegistry.Register(tools.NewExecPythonTool(a.bgProcManager))
	a.toolRegistry.Register(tools.NewListBackgroundProcessesTool(a.bgProcManager))
	a.toolRegistry.Register(tools.NewStopBackgroundProcessTool(a.bgProcManager))
	a.toolRegistry.Register(tools.NewUseSkillTool(a.skillsMgr))
	a.mcpMgr.RegisterTools(a.toolRegistry)
}

func (a *App) initLLM() {
	a.llm = providers.NewClient(a.cfg, a.toolRegistry)
	a.loop = agentloop.NewWithTools(a.store, a.llm, a.toolRegistry)
}

func (a *App) initMemory() error {
	pathLockManager := concurrency.NewPathLockManager()
	memoryFileStore, err := memory.DefaultFileStore()
	if err != nil {
		logging.Error(context.Background(), "failed to create memory file store",
			"error", err,
		)
		return fmt.Errorf("failed to create memory file store: %w", err)
	}
	a.memoryMgr = memory.NewManager(memoryFileStore, pathLockManager, a.loop.GetApprovalManager())
	a.loop.SetMemoryManager(a.memoryMgr)
	return nil
}

func (a *App) initSkills() {
	a.skillsMgr = skills.NewManager(skills.DefaultGlobalSkillsDir())

	wd, _ := os.Getwd()
	if err := a.skillsMgr.SetProjectDir(wd); err != nil {
		logging.Warn(context.Background(), "skills scan warning",
			"error", err,
		)
	}

	a.loop.SetSkillsManager(a.skillsMgr)

	solidifier := skills.NewSolidifier(a.llm, a.skillsMgr, a.store)
	a.loop.SetSolidifier(solidifier)
}

func (a *App) initMCP() {
	wd, _ := os.Getwd()
	a.mcpMgr = mcp.NewManager(wd)
	if err := a.mcpMgr.ConnectAll(context.Background()); err != nil {
		logging.Warn(context.Background(), "mcp connect partial failure",
			"error", err,
		)
	}
}

func (a *App) initHandler() {
	a.handler = rest.NewHandler(a.store, a.loop, a.memoryMgr, Version)
	a.handler.SetSkillsManager(a.skillsMgr)
	a.handler.SetLLMConfigured(a.cfg.LLM.APIKey != "")
	a.handler.SetAppConfig(a.cfg)
	a.handler.SetBgProcManager(a.bgProcManager)

	wd, _ := os.Getwd()
	a.handler.SetMcpManager(a.mcpMgr)

	if err := ensureProjectConfig(wd, a.skillsMgr, a.mcpMgr); err != nil {
		logging.Warn(context.Background(), "project config init warning",
			"error", err,
		)
	}

	a.devoDir = config.DevoDir()
	a.handler.SetProjectDir(wd)
	a.handler.SetUserConfigDir(a.devoDir)

	a.loadUserApprovalPolicy()
}

func (a *App) Run() {
	ctx := context.Background()

	mux := http.NewServeMux()
	a.handler.RegisterRoutes(mux)
	serveWebUI(mux)

	tracedMux := logging.TracingMiddleware(mux)

	logging.Info(ctx, "running crash recovery check")
	if err := a.loop.RecoverCrashedSessions(); err != nil {
		logging.Warn(ctx, "crash recovery scan failed (non-fatal)",
			"error", err,
		)
	}

	server := &http.Server{
		Addr:    a.addr,
		Handler: tracedMux,
	}

	go func() {
		logging.Info(ctx, "web server starting",
			"addr", a.addr,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Error(ctx, "server error",
				"error", err,
			)
		}
	}()

	waitForReady(a.baseURL, 10*time.Second)

	if a.webMode {
		openBrowser(a.baseURL)
	}

	if a.tuiMode {
		logging.Info(ctx, "launching TUI")
		tui.RedirectStdLog()
		tui.Launch(a.baseURL, Version)
		logging.Info(ctx, "TUI exited, shutting down server")
		server.Close()
		return
	}

	logging.Info(ctx, "server running",
		"base_url", a.baseURL,
	)
	select {}
}

func (a *App) Shutdown() {
	if a.bgProcManager != nil {
		// Kill every background process spawned by exec_python so devo exit
		// doesn't leave orphan children. Each Stop() kills the process group
		// and waits for the pipe-reader goroutine to exit.
		a.bgProcManager.Shutdown()
	}
	if a.store != nil {
		a.store.Close()
	}
}

func (a *App) loadUserApprovalPolicy() {
	a.handler.LoadUserApprovalPolicy()
}

func ensureProjectConfig(workingDir string, sm *skills.Manager, mcpMgr *mcp.Manager) error {
	cfg, err := config.LoadProjectConfig(workingDir)
	if err != nil {
		return err
	}
	if cfg != nil {
		return nil
	}

	allSkills := sm.GetAllSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	allMcpConfigs := mcpMgr.GetAllServerConfigs()
	mcpIDs := make([]string, 0, len(allMcpConfigs))
	for _, cfg := range allMcpConfigs {
		mcpIDs = append(mcpIDs, cfg.ServerID)
	}

	_, err = config.CreateDefaultProjectConfig(workingDir, skillNames, mcpIDs)
	if err != nil {
		return err
	}

	logging.Info(context.Background(), "created default project config",
		"path", workingDir+"/.devo/config.json",
	)
	return nil
}
