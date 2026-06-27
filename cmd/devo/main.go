package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/pathsec"
	"devo/internal/taskexec/tools"
	webembed "devo/web"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var Version = "dev"

func main() {
	tuiMode := flag.Bool("tui", false, "Launch TUI mode")
	webMode := flag.Bool("web", false, "Auto-open browser on startup")
	portFlag := flag.Int("port", 0, "Port for web server (0 = auto-assign)")
	workspaceFlag := flag.String("workspace", "", "Default working directory (uses current directory if not set)")
	flag.Parse()

	if *workspaceFlag != "" {
		normalized := pathsec.NormalizePath(*workspaceFlag)
		if err := os.Chdir(normalized); err != nil {
			log.Fatalf("[devo] Failed to change working directory to %s: %v", *workspaceFlag, err)
		}
		log.Printf("[devo] Working directory set to: %s", normalized)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[devo] Config error: %v", err)
	}

	port := *portFlag
	if port == 0 {
		port, err = findFreePort()
		if err != nil {
			log.Fatalf("[devo] Find free port: %v", err)
		}
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	baseURL := fmt.Sprintf("http://%s", addr)

	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = defaultDBPath()
	}

	db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("[devo] Failed to open database: %v", err)
	}

	store, err := sqlite.NewGormStore(db)
	if err != nil {
		log.Fatalf("[devo] Failed to create store: %v", err)
	}
	defer store.Close()

	var _ session.SessionStore = store

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.SearchCodebaseTool{})
	toolRegistry.Register(&tools.WriteFileTool{})
	toolRegistry.Register(&tools.EditFileTool{})
	toolRegistry.Register(&tools.ExecPythonTool{})
	toolRegistry.Register(tools.NewExecuteCommandTool())

	llm := providers.NewClient(cfg, toolRegistry)
	loop := agentloop.NewWithTools(store, llm, toolRegistry)

	pathLockManager := concurrency.NewPathLockManager()
	memoryFileStore, err := memory.DefaultFileStore()
	if err != nil {
		log.Fatalf("[devo] Failed to create memory file store: %v", err)
	}
	memoryManager := memory.NewManager(memoryFileStore, pathLockManager, loop.GetApprovalManager())
	loop.SetMemoryManager(memoryManager)

	homeDir, _ := os.UserHomeDir()
	globalSkillsDir := filepath.Join(homeDir, ".devo", "skills")
	skillsManager := skills.NewManager(globalSkillsDir)
	if wd, err := os.Getwd(); err == nil {
		if err := skillsManager.SetProjectDir(wd); err != nil {
			log.Printf("[devo] Skills scan warning: %v", err)
		}
	}
	loop.SetSkillsManager(skillsManager)

	solidifier := skills.NewSolidifier(llm, skillsManager, store)
	loop.SetSolidifier(solidifier)

	handler := rest.NewHandler(store, loop, memoryManager, Version)
	handler.SetSkillsManager(skillsManager)

	wd, _ := os.Getwd()
	mcpManager := mcp.NewManager(wd)
	if err := mcpManager.ConnectAll(context.Background()); err != nil {
		log.Printf("[devo] MCP manager: some servers failed to connect: %v", err)
	}
	mcpManager.RegisterTools(toolRegistry)
	handler.SetMcpManager(mcpManager)

	if err := ensureProjectConfig(wd, skillsManager, mcpManager); err != nil {
		log.Printf("[devo] Project config init warning: %v", err)
	}

	handler.SetProjectDir(wd)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	serveWebUI(mux)

	log.Printf("[devo] Running crash recovery check...")
	if err := loop.RecoverCrashedSessions(); err != nil {
		log.Printf("[devo] Crash recovery scan failed (non-fatal): %v", err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[devo] Web server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[devo] Server error: %v", err)
		}
	}()

	waitForReady(baseURL, 10*time.Second)
	log.Printf("[devo] Server ready: %s", baseURL)

	if *webMode {
		openBrowser(baseURL)
	}

	if *tuiMode {
		log.Printf("[devo] Launching TUI...")
		tui.Launch(baseURL, Version)
		log.Printf("[devo] TUI exited, shutting down server...")
		server.Close()
		return
	}

	log.Printf("[devo] Server running on %s (press Ctrl+C to stop)", baseURL)
	select {}
}

func defaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[devo] Failed to get home directory: %v", err)
	}
	devoDir := filepath.Join(homeDir, ".devo")
	if err := os.MkdirAll(devoDir, 0755); err != nil {
		log.Fatalf("[devo] Failed to create .devo directory: %v", err)
	}
	return filepath.Join(devoDir, "devo.db")
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

func serveWebUI(mux *http.ServeMux) {
	webFS, err := webembed.StaticFS()
	if err != nil {
		webFS = os.DirFS("web/dist")
	}

	fileServer := http.FileServer(http.FS(webFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			fileServer.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		f, err := webFS.Open(path[1:])
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitForReady(baseURL string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	url := baseURL + "/api/v1/sessions"
	client := &http.Client{Timeout: 2 * time.Second}
	log.Printf("[devo] Waiting for server readiness (timeout: %v)...", timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("[devo] Server is ready.")
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Printf("[devo] Server did not become ready within %v", timeout)
	log.Fatalf("server did not become ready within %v", timeout)
}

func openBrowser(url string) {
	log.Printf("[devo] Opening browser: %s", url)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("[devo] Failed to open browser: %v", err)
	}
}
