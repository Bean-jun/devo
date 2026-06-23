package rest

import (
	"encoding/json"
	"net/http"
	"os"

	"devo/internal/core/agentloop"
	"devo/internal/core/memory"
	"devo/internal/core/session"
)

type Handler struct {
	store         session.SessionStore
	loop          *agentloop.Loop
	memoryManager *memory.Manager
	version       string
}

func NewHandler(store session.SessionStore, loop *agentloop.Loop, memoryManager *memory.Manager, version string) *Handler {
	return &Handler{store: store, loop: loop, memoryManager: memoryManager, version: version}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/version", h.GetVersion)
	mux.HandleFunc("GET /api/v1/workspace", h.GetWorkspace)
	mux.HandleFunc("POST /api/v1/sessions", h.CreateSession)
	mux.HandleFunc("GET /api/v1/sessions", h.ListSessions)
	mux.HandleFunc("GET /api/v1/sessions/{id}/files", h.GetFiles)
	mux.HandleFunc("GET /api/v1/sessions/{id}", h.GetSession)
	mux.HandleFunc("PUT /api/v1/sessions/{id}", h.RenameSession)
	mux.HandleFunc("PUT /api/v1/sessions/{id}/config", h.UpdateConfig)
	mux.HandleFunc("POST /api/v1/sessions/{id}/messages", h.PostMessage)
	mux.HandleFunc("GET /api/v1/sessions/{id}/messages", h.GetMessages)
	mux.HandleFunc("GET /api/v1/sessions/{id}/events", h.SSEEvents)
	mux.HandleFunc("POST /api/v1/sessions/{id}/approve/{approval_id}", h.Approve)
	mux.HandleFunc("PUT /api/v1/sessions/{id}/trust", h.SetTrustLevel)
	mux.HandleFunc("PUT /api/v1/sessions/{id}/approval-policy", h.SetApprovalPolicy)
	mux.HandleFunc("POST /api/v1/sessions/{id}/cancel", h.Cancel)
	mux.HandleFunc("POST /api/v1/sessions/{id}/pause", h.Pause)
	mux.HandleFunc("POST /api/v1/sessions/{id}/resume", h.Resume)
	mux.HandleFunc("POST /api/v1/sessions/{id}/complete", h.Complete)
	mux.HandleFunc("POST /api/v1/sessions/{id}/archive", h.Archive)
	mux.HandleFunc("GET /api/v1/sessions/{id}/usage", h.GetSessionUsage)
	mux.HandleFunc("GET /api/v1/usage/stats", h.GetUsageStats)
	mux.HandleFunc("POST /api/v1/sessions/{id}/rollback", h.Rollback)
	mux.HandleFunc("GET /api/v1/sessions/{id}/archive", h.GetArchive)
	mux.HandleFunc("POST /api/v1/sessions/{id}/sync-archive", h.SyncArchive)
	mux.HandleFunc("GET /api/v1/sessions/{id}/memory", h.GetMemories)
	mux.HandleFunc("POST /api/v1/sessions/{id}/memory", h.UpsertMemory)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}/memory/{memory_id}", h.DeleteMemory)
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": h.version,
	})
}

func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	wd, err := os.Getwd()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get working directory")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"working_directory": wd,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func validateWorkingDirectory(sess *session.Session) error {
	info, err := os.Stat(sess.WorkingDirectory)
	if err != nil || !info.IsDir() {
		if sess.State != session.StatePaused {
			sess.State = session.StatePaused
			return nil
		}
	}
	return nil
}
