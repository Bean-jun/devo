package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"devo/internal/core/agentloop"
	"devo/internal/core/session"
	"devo/internal/interfaces/rest"
	"devo/internal/interfaces/tui"
	"devo/internal/storage/sqlite"
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/tools"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	tuiMode := flag.Bool("tui", true, "Launch TUI mode")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DEVO_DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home directory: %v", err)
		}
		devoDir := filepath.Join(homeDir, ".devo")
		if err := os.MkdirAll(devoDir, 0755); err != nil {
			log.Fatalf("failed to create .devo directory: %v", err)
		}
		dbPath = filepath.Join(devoDir, "devo.db")
	}

	db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	store, err := sqlite.NewGormStore(db)
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	var _ session.SessionStore = store

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.SearchCodebaseTool{})
	toolRegistry.Register(&tools.WriteFileTool{})
	toolRegistry.Register(tools.NewExecuteCommandTool())

	llm := providers.NewClientFromEnv(toolRegistry)
	loop := agentloop.NewWithTools(store, llm, toolRegistry)
	handler := rest.NewHandler(store, loop)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	if *tuiMode {
		fmt.Fprintf(os.Stderr, "[devo] TUI mode: initializing server components...\n")

		tuiPort, err := findFreePort()
		if err != nil {
			log.Fatalf("find free port: %v", err)
		}
		fmt.Fprintf(os.Stderr, "[devo] Allocated port: %d\n", tuiPort)

		server := &http.Server{
			Addr:    fmt.Sprintf("127.0.0.1:%d", tuiPort),
			Handler: mux,
		}

		go func() {
			fmt.Fprintf(os.Stderr, "[devo] HTTP server goroutine starting on :%d\n", tuiPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "[devo] Server error: %v\n", err)
			}
		}()

		baseURL := fmt.Sprintf("http://127.0.0.1:%d", tuiPort)
		waitForReady(baseURL, 10*time.Second)

		fmt.Fprintf(os.Stderr, "[devo] Server ready, launching TUI...\n")
		tui.Launch(baseURL)

		fmt.Fprintf(os.Stderr, "[devo] TUI exited, shutting down...\n")
		server.Close()
		return
	}

	log.Printf("Devo server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
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
	fmt.Fprintf(os.Stderr, "[devo] Waiting for server readiness (timeout: %v)...\n", timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Fprintf(os.Stderr, "[devo] Server is ready.\n")
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "[devo] Server did not become ready within %v\n", timeout)
	log.Fatalf("server did not become ready within %v", timeout)
}
