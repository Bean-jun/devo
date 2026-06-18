package rest

import (
	"encoding/json"
	"errors"
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
	mux.HandleFunc("GET /api/v1/sessions/{id}", h.GetSession)
	mux.HandleFunc("POST /api/v1/sessions/{id}/messages", h.PostMessage)
	mux.HandleFunc("GET /api/v1/sessions/{id}/messages", h.GetMessages)
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

	writeJSON(w, http.StatusOK, getSessionResponse{
		ID:               sess.ID,
		Title:            sess.Title,
		WorkingDirectory: sess.WorkingDirectory,
		State:            string(sess.State),
		CreatedAt:        sess.CreatedAt.Format(time.RFC3339),
		LastActiveAt:     sess.LastActiveAt.Format(time.RFC3339),
	})
}

type postMessageRequest struct {
	Content string `json:"content"`
}

type postMessageResponse struct {
	MessageID string `json:"message_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
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

	msg, err := h.loop.ProcessMessage(r.Context(), id, req.Content)
	if err != nil {
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

	writeJSON(w, http.StatusOK, postMessageResponse{
		MessageID: msg.ID,
		Role:      string(msg.Role),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.Format(time.RFC3339),
	})
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
