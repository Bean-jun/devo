package rest

import (
	"errors"
	"net/http"

	"devo/internal/core/session"
)

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if sess, err := h.store.Get(id); err == nil {
		if err := h.getAgent(sess).Cancel(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		if err := h.getDefaultAgent().Cancel(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateIdle.ToSnakeCase()})
}

func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if sess, err := h.store.Get(id); err == nil {
		if err := h.getAgent(sess).Pause(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		if err := h.getDefaultAgent().Pause(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StatePaused.ToSnakeCase()})
}

func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if sess, err := h.store.Get(id); err == nil {
		if err := h.getAgent(sess).Resume(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		if err := h.getDefaultAgent().Resume(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateToolExecuting.ToSnakeCase()})
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if sess, err := h.store.Get(id); err == nil {
		if err := h.getAgent(sess).Complete(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		if err := h.getDefaultAgent().Complete(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": session.StateCompleted.ToSnakeCase()})
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if sess, err := h.store.Get(id); err == nil {
		if err := h.getAgent(sess).Archive(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		if err := h.getDefaultAgent().Archive(id); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
