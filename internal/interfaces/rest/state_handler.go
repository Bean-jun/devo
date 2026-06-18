package rest

import (
	"errors"
	"net/http"

	"devo/internal/core/session"
)

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
