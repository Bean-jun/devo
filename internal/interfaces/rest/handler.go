package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"devo/internal/core/agentloop"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/core/skills"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/tools"
)

type Handler struct {
	store         session.SessionStore
	loop          *agentloop.Loop
	memoryManager *memory.Manager
	skillsManager *skills.Manager
	mcpManager    *mcp.Manager
	bgProcManager *tools.BackgroundProcessManager
	version       string
	projectDir    string
	userConfigDir string
	llmConfigured bool
}

func NewHandler(store session.SessionStore, loop *agentloop.Loop, memoryManager *memory.Manager, version string) *Handler {
	return &Handler{store: store, loop: loop, memoryManager: memoryManager, version: version}
}

func (h *Handler) SetBgProcManager(mgr *tools.BackgroundProcessManager) {
	h.bgProcManager = mgr
}

func (h *Handler) SetMcpManager(mgr *mcp.Manager) {
	h.mcpManager = mgr
}

func (h *Handler) SetSkillsManager(sm *skills.Manager) {
	h.skillsManager = sm
}

func (h *Handler) SetUserConfigDir(dir string) {
	h.userConfigDir = dir
}

func (h *Handler) SetProjectDir(dir string) {
	h.projectDir = dir
}

func (h *Handler) SetLLMConfigured(configured bool) {
	h.llmConfigured = configured
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.GetHealth)
	mux.HandleFunc("GET /api/v1/version", h.GetVersion)

	mux.HandleFunc("GET /api/v1/current-workspace", h.GetCurrentWorkspace)
	mux.HandleFunc("POST /api/v1/current-workspace", h.SetCurrentWorkspace)
	mux.HandleFunc("GET /api/v1/workspace", h.GetWorkspaces)
	mux.HandleFunc("DELETE /api/v1/workspace", h.DeleteWorkspace)

	mux.HandleFunc("GET /api/v1/files", h.GetWorkspaceFiles)

	mux.HandleFunc("GET /api/v1/sessions", h.ListSessions)
	mux.HandleFunc("POST /api/v1/sessions", h.CreateSession)
	mux.HandleFunc("GET /api/v1/sessions/{id}", h.GetSession)
	mux.HandleFunc("PUT /api/v1/sessions/{id}", h.RenameSession)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", h.DeleteSession)

	mux.HandleFunc("PUT /api/v1/sessions/{id}/config", h.UpdateConfig)
	mux.HandleFunc("POST /api/v1/sessions/{id}/messages", h.PostMessage)
	mux.HandleFunc("GET /api/v1/sessions/{id}/messages", h.GetMessages)
	mux.HandleFunc("GET /api/v1/sessions/{id}/events", h.SSEEvents)

	mux.HandleFunc("POST /api/v1/sessions/{id}/approve/{approval_id}", h.Approve)
	mux.HandleFunc("PUT /api/v1/sessions/{id}/trust", h.SetTrustLevel)
	mux.HandleFunc("PUT /api/v1/sessions/{id}/approval-policy", h.SetApprovalPolicy)

	// 全局有效
	mux.HandleFunc("GET /api/v1/user/approval-policy", h.GetUserApprovalPolicy)
	mux.HandleFunc("PUT /api/v1/user/approval-policy", h.SetUserApprovalPolicy)

	mux.HandleFunc("POST /api/v1/sessions/{id}/cancel", h.Cancel)
	mux.HandleFunc("POST /api/v1/sessions/{id}/pause", h.Pause)
	mux.HandleFunc("POST /api/v1/sessions/{id}/resume", h.Resume)
	mux.HandleFunc("POST /api/v1/sessions/{id}/rollback", h.Rollback)
	mux.HandleFunc("POST /api/v1/sessions/{id}/complete", h.Complete)

	mux.HandleFunc("GET /api/v1/sessions/{id}/archive", h.GetArchive)
	mux.HandleFunc("POST /api/v1/sessions/{id}/archive", h.Archive)
	mux.HandleFunc("POST /api/v1/sessions/{id}/sync-archive", h.SyncArchive)

	mux.HandleFunc("GET /api/v1/sessions/{id}/usage", h.GetSessionUsage)
	mux.HandleFunc("GET /api/v1/usage/stats", h.GetUsageStats)

	mux.HandleFunc("GET /api/v1/sessions/{id}/memory", h.GetMemories)
	mux.HandleFunc("POST /api/v1/sessions/{id}/memory", h.UpsertMemory)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}/memory/{memory_id}", h.DeleteMemory)

	mux.HandleFunc("POST /api/v1/sessions/{id}/skills", h.SetSessionSkills)
	mux.HandleFunc("GET /api/v1/skills", h.GetSkills)
	mux.HandleFunc("POST /api/v1/skills/reload", h.ReloadSkills)
	mux.HandleFunc("POST /api/v1/skills/install", h.InstallSkill)
	mux.HandleFunc("DELETE /api/v1/skills/{name}", h.DeleteSkillByName)
	mux.HandleFunc("POST /api/v1/sessions/{id}/solidify", h.SolidifySession)

	mux.HandleFunc("GET /api/v1/mcp/tools", h.GetMcpTools)
	mux.HandleFunc("GET /api/v1/mcp/servers", h.GetMcpServers)
	mux.HandleFunc("POST /api/v1/mcp/servers", h.AddMcpServer)
	mux.HandleFunc("POST /api/v1/mcp/servers/{id}/toggle", h.ToggleMcpServer)
	mux.HandleFunc("DELETE /api/v1/mcp/servers/{id}", h.RemoveMcpServer)

	mux.HandleFunc("GET /api/v1/project/config", h.GetProjectConfig)
	mux.HandleFunc("PUT /api/v1/project/config", h.SetProjectConfig)

	mux.HandleFunc("GET /api/v1/config/status", h.GetConfigStatus)
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": h.version,
	})
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": h.version,
	})
}

func (h *Handler) GetConfigStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"llm_configured": h.llmConfigured,
	})
}

func (h *Handler) GetCurrentWorkspace(w http.ResponseWriter, r *http.Request) {
	wd, err := os.Getwd()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get working directory")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"working_directory": wd,
	})
}

func (h *Handler) SetCurrentWorkspace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkingDirectory string `json:"working_directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.WorkingDirectory == "" {
		writeError(w, http.StatusBadRequest, "working_directory is required")
		return
	}
	info, err := os.Stat(req.WorkingDirectory)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "working_directory does not exist or is not accessible")
		return
	}
	if err := os.Chdir(req.WorkingDirectory); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to switch working directory")
		return
	}

	if h.skillsManager != nil {
		if err := h.skillsManager.SetProjectDir(req.WorkingDirectory); err != nil {
			logging.Warn(r.Context(), "skills scan warning after workspace switch",
				"error", err,
			)
		}
	}

	if h.mcpManager != nil {
		if err := h.mcpManager.SetProjectDir(req.WorkingDirectory); err != nil {
			logging.Warn(r.Context(), "mcp reconnect warning after workspace switch",
				"error", err,
			)
		}
	}

	h.projectDir = req.WorkingDirectory

	writeJSON(w, http.StatusOK, map[string]string{
		"working_directory": req.WorkingDirectory,
	})
}

func (h *Handler) GetWorkspaces(w http.ResponseWriter, r *http.Request) {
	dirs, err := h.store.ListUniqueWorkspaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}

	type workspaceEntry struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Path   string `json:"path"`
		Exists bool   `json:"exists"`
	}
	workspaces := make([]workspaceEntry, 0, len(dirs))
	for _, d := range dirs {
		name := d
		if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
			name = name[idx+1:]
		}
		_, statErr := os.Stat(d)
		workspaces = append(workspaces, workspaceEntry{
			ID:     d,
			Name:   name,
			Path:   d,
			Exists: statErr == nil,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workspaces": workspaces,
	})
}

func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	n, err := h.store.DeleteByWorkspace(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": n,
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
