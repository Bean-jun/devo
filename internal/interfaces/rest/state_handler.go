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

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateIdle.ToSnakeCase()})
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

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StatePaused.ToSnakeCase()})
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

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateProcessing.ToSnakeCase()})
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

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateCompleted.ToSnakeCase()})
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

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateArchived.ToSnakeCase()})
}
