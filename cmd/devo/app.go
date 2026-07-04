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
	projectconfig "devo/internal/core/config"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/core/skills"
	"devo/internal/interfaces/rest"
	"devo/internal/interfaces/tui"
	"devo/internal/storage/sqlite"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/tools"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type App struct {
	cfg          *config.Config
	store        session.SessionStore
	loop         *agentloop.Loop
	handler      *rest.Handler
	skillsMgr    *skills.Manager
	mcpMgr       *mcp.Manager
	memoryMgr    *memory.Manager
	toolRegistry *tools.Registry
	llm          llmclient.Client
	addr         string
	baseURL      string
	port         int
	devoDir      string
	tuiMode      bool
	webMode      bool
}

func NewApp(tuiMode, webMode bool, portFlag int) (*App, error) {
	app := &App{
		tuiMode: tuiMode,
		webMode: webMode,
	}

	cfg, err := config.Load()
	if err != nil {
		log.Printf("[devo] Config warning: %v", err)
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
	app.initMemory()
	app.initSkills()
	app.initMCP()
	app.initTools()
	app.initHandler()

	return app, nil
}

func (a *App) initDB() error {
	dbPath := a.cfg.DBPath
	if dbPath == "" {
		dbPath = defaultDBPath()
	}

	db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
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
	a.toolRegistry.Register(&tools.ReadFileTool{})
	a.toolRegistry.Register(&tools.ListFilesTool{})
	a.toolRegistry.Register(&tools.SearchCodebaseTool{})
	a.toolRegistry.Register(&tools.WriteFileTool{})
	a.toolRegistry.Register(&tools.EditFileTool{})
	a.toolRegistry.Register(&tools.ExecPythonTool{})
	a.toolRegistry.Register(tools.NewExecuteCommandTool())
	a.toolRegistry.Register(tools.NewUseSkillTool(a.skillsMgr))
	a.mcpMgr.RegisterTools(a.toolRegistry)
}

func (a *App) initLLM() {
	a.llm = providers.NewClient(a.cfg, a.toolRegistry)
	a.loop = agentloop.NewWithTools(a.store, a.llm, a.toolRegistry)
}

func (a *App) initMemory() {
	pathLockManager := concurrency.NewPathLockManager()
	memoryFileStore, err := memory.DefaultFileStore()
	if err != nil {
		log.Fatalf("[devo] Failed to create memory file store: %v", err)
	}
	a.memoryMgr = memory.NewManager(memoryFileStore, pathLockManager, a.loop.GetApprovalManager())
	a.loop.SetMemoryManager(a.memoryMgr)
}

func (a *App) initSkills() {
	a.skillsMgr = skills.NewManager(skills.DefaultGlobalSkillsDir())

	wd, _ := os.Getwd()
	if err := a.skillsMgr.SetProjectDir(wd); err != nil {
		log.Printf("[devo] Skills scan warning: %v", err)
	}

	a.loop.SetSkillsManager(a.skillsMgr)

	solidifier := skills.NewSolidifier(a.llm, a.skillsMgr, a.store)
	a.loop.SetSolidifier(solidifier)
}

func (a *App) initMCP() {
	wd, _ := os.Getwd()
	a.mcpMgr = mcp.NewManager(wd)
	if err := a.mcpMgr.ConnectAll(context.Background()); err != nil {
		log.Printf("[devo] MCP manager: some servers failed to connect: %v", err)
	}
}

func (a *App) initHandler() {
	a.handler = rest.NewHandler(a.store, a.loop, a.memoryMgr, Version)
	a.handler.SetSkillsManager(a.skillsMgr)
	a.handler.SetLLMConfigured(a.cfg.LLM.APIKey != "")

	wd, _ := os.Getwd()
	a.handler.SetMcpManager(a.mcpMgr)

	if err := ensureProjectConfig(wd, a.skillsMgr, a.mcpMgr); err != nil {
		log.Printf("[devo] Project config init warning: %v", err)
	}

	a.devoDir = defaultDevoDir()
	a.handler.SetProjectDir(wd)
	a.handler.SetUserConfigDir(a.devoDir)

	a.loadUserApprovalPolicy()
}

func (a *App) Run() {
	mux := http.NewServeMux()
	a.handler.RegisterRoutes(mux)
	serveWebUI(mux)

	log.Printf("[devo] Running crash recovery check...")
	if err := a.loop.RecoverCrashedSessions(); err != nil {
		log.Printf("[devo] Crash recovery scan failed (non-fatal): %v", err)
	}

	server := &http.Server{
		Addr:    a.addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[devo] Web server starting on %s", a.addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[devo] Server error: %v", err)
		}
	}()

	waitForReady(a.baseURL, 10*time.Second)
	log.Printf("[devo] Server ready: %s", a.baseURL)

	if a.webMode {
		openBrowser(a.baseURL)
	}

	if a.tuiMode {
		log.Printf("[devo] Launching TUI...")
		tui.RedirectStdLog()
		tui.Launch(a.baseURL, Version)
		log.Printf("[devo] TUI exited, shutting down server...")
		server.Close()
		return
	}

	log.Printf("[devo] Server running on %s (press Ctrl+C to stop)", a.baseURL)
	select {}
}

func (a *App) Shutdown() {
	if a.store != nil {
		a.store.Close()
	}
}

func (a *App) loadUserApprovalPolicy() {
	a.handler.LoadUserApprovalPolicy()
}

func ensureProjectConfig(workingDir string, sm *skills.Manager, mcpMgr *mcp.Manager) error {
	cfg, err := projectconfig.Load(workingDir)
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

	_, err = projectconfig.CreateDefault(workingDir, skillNames, mcpIDs)
	if err != nil {
		return err
	}

	log.Printf("[devo] Created default project config at %s/.devo/config.json", workingDir)
	return nil
}
