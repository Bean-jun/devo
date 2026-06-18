package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"devo/internal/core/agentloop"
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
	mux.HandleFunc("GET /api/v1/sessions/{id}", h.GetSession)
	mux.HandleFunc("POST /api/v1/sessions/{id}/messages", h.PostMessage)
	mux.HandleFunc("GET /api/v1/sessions/{id}/messages", h.GetMessages)
	mux.HandleFunc("GET /api/v1/sessions/{id}/events", h.SSEEvents)
}

type createSessionRequest struct {
	WorkingDirectory string `json:"working_directory"`
	Title            string `json:"title,omitempty"`
}

type createSessionResponse struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	WorkingDirectory string `json:"working_directory"`
	State            string `json:"state"`
	CreatedAt        string `json:"created_at"`
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
	sess := &session.Session{
		ID:               session.GenerateID("sess"),
		Title:            title,
		WorkingDirectory: req.WorkingDirectory,
		State:            session.StateIdle,
		CreatedAt:        now,
		LastActiveAt:     now,
	}

	if err := h.store.Create(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{
		ID:               sess.ID,
		Title:            sess.Title,
		WorkingDirectory: sess.WorkingDirectory,
		State:            string(sess.State),
		CreatedAt:        sess.CreatedAt.Format(time.RFC3339),
	})
}

type getSessionResponse struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	WorkingDirectory string `json:"working_directory"`
	State            string `json:"state"`
	CreatedAt        string `json:"created_at"`
	LastActiveAt     string `json:"last_active_at"`
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
		ID:               sess.ID,
		Title:            sess.Title,
		WorkingDirectory: sess.WorkingDirectory,
		State:            string(sess.State),
		CreatedAt:        sess.CreatedAt.Format(time.RFC3339),
		LastActiveAt:     sess.LastActiveAt.Format(time.RFC3339),
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
	defer h.store.DecrementSSEConnections(id)

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
