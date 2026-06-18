package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"devo/internal/core/agentloop"
	"devo/internal/core/approval"
	"devo/internal/core/session"
)

type Handler struct {
	store session.SessionStore
	loop  *agentloop.Loop
}

func NewHandler(store session.SessionStore, loop *agentloop.Loop) *Handler {
	return &Handler{store: store, loop: loop}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/sessions", h.CreateSession)
	mux.HandleFunc("GET /api/v1/sessions", h.ListSessions)
	mux.HandleFunc("GET /api/v1/sessions/{id}/files", h.GetFiles)
	mux.HandleFunc("GET /api/v1/sessions/{id}", h.GetSession)
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
}

type createSessionRequest struct {
	WorkingDirectory       string `json:"working_directory"`
	Title                  string `json:"title,omitempty"`
	ApprovalTimeoutSeconds int    `json:"approval_timeout_seconds,omitempty"`
}

type createSessionResponse struct {
	ID                     string            `json:"id"`
	Title                  string            `json:"title"`
	WorkingDirectory       string            `json:"working_directory"`
	State                  string            `json:"state"`
	CreatedAt              string            `json:"created_at"`
	TrustLevel             string            `json:"trust_level"`
	ApprovalPolicy         map[string]string `json:"approval_policy,omitempty"`
	ApprovalTimeoutSeconds int               `json:"approval_timeout_seconds"`
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

	info, err := os.Stat(req.WorkingDirectory)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "working_directory does not exist or is not accessible")
		return
	}

	title := req.Title
	if title == "" {
		title = req.WorkingDirectory
	}

	now := time.Now()
	timeoutSeconds := req.ApprovalTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}

	sess := &session.Session{
		ID:                     session.GenerateID("sess"),
		Title:                  title,
		WorkingDirectory:       req.WorkingDirectory,
		State:                  session.StateIdle,
		CreatedAt:              now,
		LastActiveAt:           now,
		TrustLevel:             string(approval.TrustNormal),
		ApprovalPolicy:         make(map[string]string),
		ApprovalTimeoutSeconds: timeoutSeconds,
	}

	if err := h.store.Create(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{
		ID:                     sess.ID,
		Title:                  sess.Title,
		WorkingDirectory:       sess.WorkingDirectory,
		State:                  string(sess.State),
		CreatedAt:              sess.CreatedAt.Format(time.RFC3339),
		TrustLevel:             sess.TrustLevel,
		ApprovalPolicy:         sess.ApprovalPolicy,
		ApprovalTimeoutSeconds: sess.ApprovalTimeoutSeconds,
	})
}

type getSessionResponse struct {
	ID                     string            `json:"id"`
	Title                  string            `json:"title"`
	WorkingDirectory       string            `json:"working_directory"`
	State                  string            `json:"state"`
	CreatedAt              string            `json:"created_at"`
	LastActiveAt           string            `json:"last_active_at"`
	TrustLevel             string            `json:"trust_level"`
	ApprovalPolicy         map[string]string `json:"approval_policy,omitempty"`
	ApprovalTimeoutSeconds int               `json:"approval_timeout_seconds"`
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

	if err := validateWorkingDirectory(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, getSessionResponse{
		ID:                     sess.ID,
		Title:                  sess.Title,
		WorkingDirectory:       sess.WorkingDirectory,
		State:                  string(sess.State),
		CreatedAt:              sess.CreatedAt.Format(time.RFC3339),
		LastActiveAt:           sess.LastActiveAt.Format(time.RFC3339),
		TrustLevel:             sess.TrustLevel,
		ApprovalPolicy:         sess.ApprovalPolicy,
		ApprovalTimeoutSeconds: sess.ApprovalTimeoutSeconds,
	})
}

type listSessionsItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ProjectPath  string `json:"project_path"`
	State        string `json:"state"`
	CreatedAt    string `json:"created_at"`
	LastActiveAt string `json:"last_active_at"`
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
			ID:           s.ID,
			Title:        s.Title,
			ProjectPath:  s.WorkingDirectory,
			State:        string(s.State),
			CreatedAt:    s.CreatedAt.Format(time.RFC3339),
			LastActiveAt: s.LastActiveAt.Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, listSessionsResponse{
		Sessions: items,
		Total:    total,
	})
}

type postMessageRequest struct {
	Content string `json:"content"`
}

func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req postMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	if err := h.loop.ProcessMessage(r.Context(), id, req.Content); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		if errors.Is(err, session.ErrSessionArchived) {
			writeError(w, http.StatusConflict, "session is archived")
			return
		}
		if errors.Is(err, session.ErrSessionNotIdle) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

type messageItem struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type getMessagesResponse struct {
	Messages []messageItem `json:"messages"`
	Total    int           `json:"total"`
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 0
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

	msgs, total, err := h.store.GetMessages(id, limit, offset)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	items := make([]messageItem, len(msgs))
	for i, m := range msgs {
		items[i] = messageItem{
			ID:        m.ID,
			Role:      string(m.Role),
			Content:   m.Content,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, getMessagesResponse{
		Messages: items,
		Total:    total,
	})
}

func (h *Handler) SSEEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	eventBus, err := h.store.GetEventBus(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.store.IncrementSSEConnections(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to increment connections")
		return
	}
	defer func() {
		h.store.DecrementSSEConnections(id)
		sess, err := h.store.Get(id)
		if err == nil && sess.State == session.StateProcessing && sess.ActiveSSEConnections <= 0 {
			eventBus.Publish("session_state_change", map[string]any{
				"old_state": string(session.StateProcessing),
				"new_state": string(session.StatePaused),
				"reason":    "sse_disconnected",
			})
			sess.State = session.StatePaused
			sess.LastActiveAt = time.Now()
			h.store.Update(sess)
		}
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		sinceID, parseErr := strconv.ParseInt(lastEventID, 10, 64)
		if parseErr == nil {
			historyEvents := eventBus.GetHistory(sinceID)
			for _, evt := range historyEvents {
				writeSSEEvent(w, flusher, evt)
			}
		}
	}

	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, evt)
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, evt session.Event) {
	dataJSON, _ := json.Marshal(evt.Data)
	fmt.Fprintf(w, "id: %d\n", evt.ID)
	fmt.Fprintf(w, "event: %s\n", evt.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(dataJSON))
	flusher.Flush()
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

type fileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type getFilesResponse struct {
	Type    string      `json:"type"`
	Entries []fileEntry `json:"entries,omitempty"`
	Content string      `json:"content,omitempty"`
	Size    int64       `json:"size,omitempty"`
}

func (h *Handler) GetFiles(w http.ResponseWriter, r *http.Request) {
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

	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "."
	}

	absWorkDir, err := filepath.Abs(sess.WorkingDirectory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var absPath string
	if filepath.IsAbs(requestedPath) {
		absPath = requestedPath
	} else {
		absPath = filepath.Join(absWorkDir, requestedPath)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	if !strings.HasPrefix(absPath, absWorkDir) {
		writeError(w, http.StatusBadRequest, "path is outside working directory")
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path does not exist or is not accessible")
		return
	}

	if info.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read directory")
			return
		}

		var fileEntries []fileEntry
		for _, entry := range entries {
			entryInfo, err := entry.Info()
			if err != nil {
				continue
			}
			entryType := "file"
			if entry.IsDir() {
				entryType = "dir"
			}
			fileEntries = append(fileEntries, fileEntry{
				Name: entry.Name(),
				Type: entryType,
				Size: entryInfo.Size(),
			})
		}

		writeJSON(w, http.StatusOK, getFilesResponse{
			Type:    "dir",
			Entries: fileEntries,
		})
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	writeJSON(w, http.StatusOK, getFilesResponse{
		Type:    "file",
		Content: string(data),
		Size:    info.Size(),
	})
}

type approveRequest struct {
	Decision string `json:"decision"`
}

type approveResponse struct {
	ApprovalID string `json:"approval_id"`
	Decision   string `json:"decision"`
	ResolvedAt string `json:"resolved_at"`
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	approvalID := r.PathValue("approval_id")

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if sess.State != session.StateAwaitingApproval {
		writeError(w, http.StatusConflict, "session is not in AwaitingApproval state")
		return
	}

	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Decision != "approve" && req.Decision != "reject" {
		writeError(w, http.StatusBadRequest, "decision must be 'approve' or 'reject'")
		return
	}

	if err := h.loop.ResolveApproval(id, approvalID, req.Decision); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "has expired") {
			writeError(w, http.StatusConflict, errMsg)
			return
		}
		writeError(w, http.StatusConflict, errMsg)
		return
	}

	writeJSON(w, http.StatusOK, approveResponse{
		ApprovalID: approvalID,
		Decision:   req.Decision,
		ResolvedAt: time.Now().Format(time.RFC3339),
	})
}

type setTrustLevelRequest struct {
	TrustLevel string `json:"trust_level"`
}

func (h *Handler) SetTrustLevel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req setTrustLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !approval.IsValidTrustLevel(req.TrustLevel) {
		writeError(w, http.StatusBadRequest, "trust_level must be one of: low, normal, elevated")
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

	sess.TrustLevel = req.TrustLevel
	sess.LastActiveAt = time.Now()

	if err := h.store.Update(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"trust_level": req.TrustLevel})
}

type setApprovalPolicyRequest map[string]string

func (h *Handler) SetApprovalPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req setApprovalPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for opType, policyLevel := range req {
		if !approval.IsValidOperationType(opType) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid operation_type: %s", opType))
			return
		}
		if !approval.IsValidPolicyLevel(policyLevel) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid policy_level: %s", policyLevel))
			return
		}
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

	if sess.ApprovalPolicy == nil {
		sess.ApprovalPolicy = make(map[string]string)
	}

	for opType, policyLevel := range req {
		sess.ApprovalPolicy[opType] = policyLevel
	}

	sess.LastActiveAt = time.Now()

	if err := h.store.Update(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approval_policy": sess.ApprovalPolicy,
	})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.loop.Cancel(id); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": string(session.StateIdle)})
}

func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.loop.Pause(id); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": string(session.StatePaused)})
}

func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.loop.Resume(id); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": string(session.StateProcessing)})
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.loop.Complete(id); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": string(session.StateCompleted)})
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.loop.Archive(id); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": string(session.StateArchived)})
}
