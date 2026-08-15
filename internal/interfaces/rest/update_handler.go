package rest

import (
	"net/http"

	"devo/internal/update"
)

func (h *Handler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	result, err := update.CheckForUpdate(h.version)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "update check failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
