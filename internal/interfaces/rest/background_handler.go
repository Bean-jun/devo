package rest

import (
	"errors"
	"net/http"
	"strconv"

	"devo/internal/pkg/logging"
)

// ListBackgroundProcesses returns the background processes currently registered
// for this session. PIDs only stay in the manager while the process is alive -
// when it exits, the pipe-reader goroutine unregisters it. So this list is the
// authoritative "what's running right now" view for the UI.
func (h *Handler) ListBackgroundProcesses(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.store.Get(id); err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if h.bgProcManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"processes": []any{}})
		return
	}

	procs := h.bgProcManager.List(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"processes": procs,
	})
}

// StopBackgroundProcess kills a background process by PID. The PID must belong
// to the session identified by {id} - stopping another session's process is
// rejected. Returns 404 if the PID isn't registered (already exited or never
// existed in this process's manager).
func (h *Handler) StopBackgroundProcess(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.store.Get(id); err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	pidStr := r.PathValue("pid")
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		writeError(w, http.StatusBadRequest, "invalid pid")
		return
	}

	if h.bgProcManager == nil {
		writeError(w, http.StatusServiceUnavailable, "background process manager not configured")
		return
	}

	// Verify the PID belongs to this session before stopping - otherwise a UI
	// bug or stale request could let session A kill session B's process.
	procs := h.bgProcManager.List(id)
	owned := false
	for _, p := range procs {
		if p.PID == pid {
			owned = true
			break
		}
	}
	if !owned {
		writeError(w, http.StatusNotFound, "no background process with this pid belongs to the session")
		return
	}

	if err := h.bgProcManager.Stop(pid); err != nil {
		logging.Warn(r.Context(), "stop background process failed",
			"session_id", id,
			"pid", pid,
			"error", err,
		)
		if errors.Is(err, errAlreadyStopped) {
			writeError(w, http.StatusNotFound, "process already exited")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pid":    pid,
		"status": "stopped",
	})
}

// errAlreadyStopped is returned by Stop when the process exited before the kill
// signal landed. The manager doesn't currently distinguish this case, so this
// stays unused-but-reserved for a future finer-grained error type.
var errAlreadyStopped = errors.New("process already stopped")
