package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"devo/internal/core/session"
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
		if err == nil && sess.State == session.StateProcessing && sess.ActiveSSEConnections <= 0 {
			eventBus.Publish("session_state_change", map[string]any{
				"old_state": string(session.StateProcessing),
				"new_state": string(session.StatePaused),
				"reason":    "sse_disconnected",
			})
			sess.State = session.StatePaused
			sess.LastActiveAt = time.Now()
			h.store.Update(sess)
		}
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
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

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, evt)
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, evt session.Event) {
	dataJSON, _ := json.Marshal(evt.Data)
	fmt.Fprintf(w, "id: %d\n", evt.ID)
	fmt.Fprintf(w, "event: %s\n", evt.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(dataJSON))
	flusher.Flush()
}
