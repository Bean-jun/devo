package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"devo/internal/core/session"
)

func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		TargetMessageID string `json:"target_message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TargetMessageID == "" {
		writeError(w, http.StatusBadRequest, "target_message_id is required")
		return
	}

	result, err := h.loop.Rollback(id, req.TargetMessageID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		if errors.Is(err, session.ErrSessionArchived) {
			writeError(w, http.StatusConflict, "session is archived")
			return
		}
		if errors.Is(err, session.ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, "message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
