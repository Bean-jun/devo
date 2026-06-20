package rest

import (
	"net/http"
	"path/filepath"
)

func (h *Handler) GetArchive(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	sess, err := h.store.Get(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	content, err := h.loop.GetArchiveContent(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read archive: "+err.Error())
		return
	}

	if content == nil {
		writeError(w, http.StatusNotFound, "archive not found")
		return
	}

	filename := filepath.Base(sess.ArchivePath)
	if filename == "" {
		filename = sessionID + ".md"
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func (h *Handler) SyncArchive(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	lastMessageID, err := h.loop.SyncArchive(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to sync archive: "+err.Error())
		return
	}

	sess, _ := h.store.Get(sessionID)

	writeJSON(w, http.StatusOK, map[string]string{
		"archive_path":    sess.ArchivePath,
		"last_message_id": lastMessageID,
	})
}
