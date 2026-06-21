package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"devo/internal/core/session"
)

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
	ID         string             `json:"id"`
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	CreatedAt  string             `json:"created_at"`
	ToolCalls  []session.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
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
			ID:         m.ID,
			Role:       string(m.Role),
			Content:    m.Content,
			CreatedAt:  m.CreatedAt.Format(time.RFC3339),
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}

	writeJSON(w, http.StatusOK, getMessagesResponse{
		Messages: items,
		Total:    total,
	})
}
