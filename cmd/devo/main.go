package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"devo/internal/core/agentloop"
	"devo/internal/core/session"
	"devo/internal/interfaces/rest"
	"devo/internal/storage/sqlite"
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/tools"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
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

	log.Printf("Devo server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
