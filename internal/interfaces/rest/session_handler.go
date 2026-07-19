package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"devo/internal/config"
	"devo/internal/core/approval"
	"devo/internal/core/session"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/pathsec"
)

type createSessionRequest struct {
	WorkingDirectory       string `json:"working_directory"`
	Title                  string `json:"title,omitempty"`
	ApprovalTimeoutSeconds int    `json:"approval_timeout_seconds,omitempty"`
	SystemPromptOverride   string `json:"system_prompt_override,omitempty"`
	ToolCallLimit          int    `json:"tool_call_limit,omitempty"`
	MaxContextTokens       int    `json:"max_context_tokens,omitempty"`
	KeepRecent             int    `json:"keep_recent,omitempty"`
}

type createSessionResponse struct {
	ID                        string             `json:"id"`
	Title                     string             `json:"title"`
	WorkingDirectory          string             `json:"working_directory"`
	State                     string             `json:"state"`
	CreatedAt                 string             `json:"created_at"`
	TrustLevel                string             `json:"trust_level"`
	ApprovalPolicy            map[string]string  `json:"approval_policy,omitempty"`
	ApprovalTimeoutSeconds    int                `json:"approval_timeout_seconds"`
	TokenUsage                session.TokenUsage `json:"token_usage"`
	MaxContextTokens          int                `json:"max_context_tokens"`
	MaxConcurrentToolCalls    int                `json:"max_concurrent_tool_calls"`
	MaxConcurrentSubprocesses int                `json:"max_concurrent_subprocesses"`
	CurrentContextTokens      int                `json:"current_context_tokens"`
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkingDirectory == "" {
		writeError(w, http.StatusBadRequest, "working_directory is required")
		return
	}

	workingDir := pathsec.NormalizePath(req.WorkingDirectory)

	info, err := os.Stat(workingDir)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "working_directory does not exist or is not accessible")
		return
	}

	title := req.Title

	now := time.Now()
	timeoutSeconds := req.ApprovalTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}

	toolCallLimit := req.ToolCallLimit
	maxContextTokens := req.MaxContextTokens
	keepRecent := req.KeepRecent

	cfg, _ := config.LoadFullConfig(workingDir)
	if cfg != nil {
		if toolCallLimit <= 0 {
			toolCallLimit = cfg.ToolCallLimit
		}
		if maxContextTokens <= 0 {
			maxContextTokens = cfg.MaxContextTokens
		}
		if keepRecent <= 0 {
			keepRecent = cfg.KeepRecent
		}
	}
	if toolCallLimit <= 0 {
		toolCallLimit = 50
	}
	if maxContextTokens <= 0 {
		maxContextTokens = 128000
	}
	if keepRecent <= 0 {
		keepRecent = 30
	}

	sess := &session.Session{
		ID:                     session.GenerateID("sess"),
		Title:                  title,
		WorkingDirectory:       workingDir,
		State:                  session.StateIdle,
		CreatedAt:              now,
		LastActiveAt:           now,
		TrustLevel:             string(approval.TrustNormal),
		ApprovalPolicy:         make(map[string]string),
		ApprovalTimeoutSeconds: timeoutSeconds,
		SystemPromptOverride:   req.SystemPromptOverride,
		ToolCallLimit:          toolCallLimit,
		MaxContextTokens:       maxContextTokens,
		KeepRecent:             keepRecent,
	}
	sess.CurrentContextTokens = h.loop.EstimateInitialContextTokens(sess)

	if err := h.store.Create(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{
		ID:                        sess.ID,
		Title:                     sess.Title,
		WorkingDirectory:          sess.WorkingDirectory,
		State:                     sess.State.ToSnakeCase(),
		CreatedAt:                 sess.CreatedAt.Format(time.RFC3339),
		TrustLevel:                sess.TrustLevel,
		ApprovalPolicy:            sess.ApprovalPolicy,
		ApprovalTimeoutSeconds:    sess.ApprovalTimeoutSeconds,
		TokenUsage:                sess.TokenUsage,
		MaxContextTokens:          sess.MaxContextTokens,
		MaxConcurrentToolCalls:    sess.MaxConcurrentToolCalls,
		MaxConcurrentSubprocesses: sess.MaxConcurrentSubprocesses,
		CurrentContextTokens:      sess.CurrentContextTokens,
	})
}

type getSessionResponse struct {
	ID                        string             `json:"id"`
	Title                     string             `json:"title"`
	WorkingDirectory          string             `json:"working_directory"`
	State                     string             `json:"state"`
	CreatedAt                 string             `json:"created_at"`
	LastActiveAt              string             `json:"last_active_at"`
	TrustLevel                string             `json:"trust_level"`
	ApprovalPolicy            map[string]string  `json:"approval_policy,omitempty"`
	ApprovalTimeoutSeconds    int                `json:"approval_timeout_seconds"`
	ToolCallLimit             int                `json:"tool_call_limit"`
	ToolCallCount             int                `json:"tool_call_count"`
	TokenUsage                session.TokenUsage `json:"token_usage"`
	MaxContextTokens          int                `json:"max_context_tokens"`
	KeepRecent                int                `json:"keep_recent"`
	MaxConcurrentToolCalls    int                `json:"max_concurrent_tool_calls"`
	MaxConcurrentSubprocesses int                `json:"max_concurrent_subprocesses"`
	CurrentContextTokens      int                `json:"current_context_tokens"`
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	logging.Debug(r.Context(), "get session",
		"session_id", id,
		"state", string(sess.State),
	)

	if err := validateWorkingDirectory(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, getSessionResponse{
		ID:                        sess.ID,
		Title:                     sess.Title,
		WorkingDirectory:          sess.WorkingDirectory,
		State:                     sess.State.ToSnakeCase(),
		CreatedAt:                 sess.CreatedAt.Format(time.RFC3339),
		LastActiveAt:              sess.LastActiveAt.Format(time.RFC3339),
		TrustLevel:                sess.TrustLevel,
		ApprovalPolicy:            sess.ApprovalPolicy,
		ApprovalTimeoutSeconds:    sess.ApprovalTimeoutSeconds,
		ToolCallLimit:             sess.ToolCallLimit,
		ToolCallCount:             sess.ToolCallCount,
		TokenUsage:                sess.TokenUsage,
		MaxContextTokens:          sess.MaxContextTokens,
		KeepRecent:                sess.KeepRecent,
		MaxConcurrentToolCalls:    sess.MaxConcurrentToolCalls,
		MaxConcurrentSubprocesses: sess.MaxConcurrentSubprocesses,
		CurrentContextTokens:      sess.CurrentContextTokens,
	})
}

type listSessionsItem struct {
	ID                   string             `json:"id"`
	Title                string             `json:"title"`
	ProjectPath          string             `json:"project_path"`
	State                string             `json:"state"`
	CreatedAt            string             `json:"created_at"`
	LastActiveAt         string             `json:"last_active_at"`
	MessageCount         int                `json:"message_count"`
	TokenUsage           session.TokenUsage `json:"token_usage"`
	CurrentContextTokens int                `json:"current_context_tokens"`
	TrustLevel           string             `json:"trust_level"`
}

type listSessionsResponse struct {
	Sessions []listSessionsItem `json:"sessions"`
	Total    int                `json:"total"`
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	project := r.URL.Query().Get("project")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}

	offset := 0
	if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}

	sessions, total, err := h.store.ListSessions(status, project, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	items := make([]listSessionsItem, len(sessions))
	for i, s := range sessions {
		items[i] = listSessionsItem{
			ID:                   s.ID,
			Title:                s.Title,
			ProjectPath:          s.WorkingDirectory,
			State:                s.State.ToSnakeCase(),
			CreatedAt:            s.CreatedAt.Format(time.RFC3339),
			LastActiveAt:         s.LastActiveAt.Format(time.RFC3339),
			MessageCount:         s.MessageCount,
			TokenUsage:           s.TokenUsage,
			CurrentContextTokens: s.CurrentContextTokens,
			TrustLevel:           string(s.TrustLevel),
		}
	}

	writeJSON(w, http.StatusOK, listSessionsResponse{
		Sessions: items,
		Total:    total,
	})
}

type renameSessionRequest struct {
	Title string `json:"title"`
}

type renameSessionResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (h *Handler) RenameSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req renameSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	sess.Title = req.Title
	if err := h.store.Update(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rename session")
		return
	}

	writeJSON(w, http.StatusOK, renameSessionResponse{
		ID:    sess.ID,
		Title: sess.Title,
	})
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// 停止会话中所有后台进程
	if h.bgProcManager != nil {
		errs := h.bgProcManager.StopSession(id)
		if len(errs) > 0 {
			logging.Warn(r.Context(), "failed to stop background processes",
				"session_id", id,
				"errors", errs,
			)
		}
	}

	if err := h.store.DeleteSession(sess.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"id":      sess.ID,
		"deleted": "true",
	})
}

type updateConfigRequest struct {
	ToolCallLimit             *int `json:"tool_call_limit,omitempty"`
	MaxContextTokens          *int `json:"max_context_tokens,omitempty"`
	KeepRecent                *int `json:"keep_recent,omitempty"`
	MaxConcurrentToolCalls    *int `json:"max_concurrent_tool_calls,omitempty"`
	MaxConcurrentSubprocesses *int `json:"max_concurrent_subprocesses,omitempty"`
}

type updateConfigResponse struct {
	ToolCallLimit             int `json:"tool_call_limit"`
	MaxContextTokens          int `json:"max_context_tokens"`
	KeepRecent                int `json:"keep_recent"`
	MaxConcurrentToolCalls    int `json:"max_concurrent_tool_calls"`
	MaxConcurrentSubprocesses int `json:"max_concurrent_subprocesses"`
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ToolCallLimit != nil {
		if *req.ToolCallLimit <= 0 {
			writeError(w, http.StatusBadRequest, "tool_call_limit must be greater than 0")
			return
		}
		if err := h.loop.UpdateConfig(id, *req.ToolCallLimit); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if req.MaxContextTokens != nil || req.KeepRecent != nil {
		if err := h.loop.UpdateContextConfig(id, req.MaxContextTokens, req.KeepRecent); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if req.MaxConcurrentToolCalls != nil || req.MaxConcurrentSubprocesses != nil {
		if err := h.loop.UpdateConcurrencyConfig(id, req.MaxConcurrentToolCalls, req.MaxConcurrentSubprocesses); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	sess, err := h.store.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, updateConfigResponse{
		ToolCallLimit:             sess.ToolCallLimit,
		MaxContextTokens:          sess.MaxContextTokens,
		KeepRecent:                sess.KeepRecent,
		MaxConcurrentToolCalls:    sess.MaxConcurrentToolCalls,
		MaxConcurrentSubprocesses: sess.MaxConcurrentSubprocesses,
	})
}
