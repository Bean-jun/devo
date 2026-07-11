package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"devo/internal/core/session"
	"devo/internal/pkg/logging"
)

func (h *Handler) SSEEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	eventBus, err := h.store.GetEventBus(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.store.IncrementSSEConnections(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to increment connections")
		return
	}
	defer func() {
		h.store.DecrementSSEConnections(id)
		sess, err := h.store.Get(id)
		if err == nil && (sess.State == session.StateThinking || sess.State == session.StateToolExecuting || sess.State == session.StateAwaitingApproval) && sess.ActiveSSEConnections <= 0 {
			eventBus.Publish("session_state_change", map[string]any{
				"old_state": sess.State.ToSnakeCase(),
				"new_state": session.StatePaused.ToSnakeCase(),
				"reason":    "sse_disconnected",
			})
			sess.State = session.StatePaused
			sess.LastActiveAt = time.Now()
			h.store.Update(sess)
		}
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		logging.Warn(r.Context(), "sse handler: flusher not supported")
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		sinceID, parseErr := strconv.ParseInt(lastEventID, 10, 64)
		if parseErr == nil {
			historyEvents := eventBus.GetHistory(sinceID)
			for _, evt := range historyEvents {
				writeSSEEvent(w, flusher, evt)
			}
		}
	}

	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	var mcpCh chan session.Event
	var mcpUnsubscribe func()
	if h.mcpManager != nil {
		mcpCh, mcpUnsubscribe = h.mcpManager.GlobalEventBus().Subscribe()
		defer mcpUnsubscribe()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, evt)
		case evt, ok := <-mcpCh:
			if !ok {
				mcpCh = nil
				continue
			}
			writeSSEEvent(w, flusher, evt)
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, evt session.Event) {
	dataJSON, _ := json.Marshal(evt.Data)
	wrapJSON, _ := json.Marshal(map[string]any{
		"type": evt.Type,
		"data": evt.Data,
	})
	_ = dataJSON
	fmt.Fprintf(w, "id: %d\n", evt.ID)
	fmt.Fprintf(w, "data: %s\n\n", string(wrapJSON))
	flusher.Flush()
}
