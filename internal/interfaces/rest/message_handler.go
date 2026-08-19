package rest

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/imageproc"
)

type postMessageRequest struct {
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req postMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" && len(req.Images) == 0 {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	msg := session.Message{
		Content: req.Content,
	}

	if len(req.Images) > 0 {
		compressed := make([]string, 0, len(req.Images))
		for _, img := range req.Images {
			result, err := imageproc.Compress(img, nil)
			if err != nil {
				log.Printf("image compress warning: %v", err)
				compressed = append(compressed, img)
				continue
			}
			compressed = append(compressed, result.DataURL)
		}
		req.Images = compressed

		msg.ContentParts = make([]session.ContentPart, 0, len(req.Images)+1)
		for _, img := range req.Images {
			msg.ContentParts = append(msg.ContentParts, session.ContentPart{
				Type: "image_url",
				URL:  img,
			})
		}
		if req.Content != "" {
			msg.ContentParts = append(msg.ContentParts, session.ContentPart{
				Type: "text",
				Text: req.Content,
			})
		}
	}

	sess, err := h.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if err := h.getAgent(sess).ProcessMessage(r.Context(), id, msg); err != nil {
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
	ID           string                `json:"id"`
	Role         string                `json:"role"`
	Content      string                `json:"content"`
	ContentParts []session.ContentPart `json:"content_parts,omitempty"`
	CreatedAt    string                `json:"created_at"`
	ToolCalls    []session.ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string                `json:"tool_call_id,omitempty"`
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
			ID:           m.ID,
			Role:         string(m.Role),
			Content:      m.Content,
			ContentParts: m.ContentParts,
			CreatedAt:    m.CreatedAt.Format(time.RFC3339),
			ToolCalls:    m.ToolCalls,
			ToolCallID:   m.ToolCallID,
		}
	}

	writeJSON(w, http.StatusOK, getMessagesResponse{
		Messages: items,
		Total:    total,
	})
}
