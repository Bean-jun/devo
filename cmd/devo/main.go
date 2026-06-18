package main

import (
	"log"
	"net/http"
	"os"

	"devo/internal/core/agentloop"
	"devo/internal/core/session"
	"devo/internal/interfaces/rest"
	"devo/internal/taskexec/llmclient"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := session.NewInMemoryStore()
	llm := llmclient.NewMockClient()
	loop := agentloop.New(store, llm)
	handler := rest.NewHandler(store, loop)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	log.Printf("Devo server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
