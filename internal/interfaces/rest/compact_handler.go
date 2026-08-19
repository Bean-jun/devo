package rest

import (
	"errors"
	"net/http"

	"devo/internal/core/session"
)

func (h *Handler) Compact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, err := h.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	result, err := h.getAgent(sess).Compact(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	compressedCount := 0
	tokensRemoved := 0
	if result != nil {
		compressedCount = result.CompressedCount
		tokensRemoved = result.TokensRemoved
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"compressed_count": compressedCount,
		"tokens_removed":   tokensRemoved,
	})
}
