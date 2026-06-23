package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"devo/internal/core/memory"
	"devo/internal/core/session"
)

type memoryItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Key       string `json:"key"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type listMemoryResponse struct {
	Memories []memoryItem `json:"memories"`
}

func (h *Handler) GetMemories(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	memoryType := memory.MemoryType(r.URL.Query().Get("type"))
	if memoryType != memory.TypeUser && memoryType != memory.TypeProject {
		writeError(w, http.StatusBadRequest, "type must be 'user' or 'project'")
		return
	}

	memories, err := h.memoryManager.List(memoryType, sess.WorkingDirectory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list memories")
		return
	}

	items := make([]memoryItem, 0, len(memories))
	for _, m := range memories {
		items = append(items, memoryItem{
			ID:        m.ID,
			Type:      string(m.Type),
			Key:       m.Key,
			Content:   m.Content,
			Source:    string(m.Source),
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	if items == nil {
		items = []memoryItem{}
	}

	writeJSON(w, http.StatusOK, listMemoryResponse{Memories: items})
}

type upsertMemoryRequest struct {
	Type    string `json:"type"`
	Key     string `json:"key"`
	Content string `json:"content"`
	Action  string `json:"action"`
}

type memoryResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Key       string `json:"key"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *Handler) UpsertMemory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req upsertMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type != "user" && req.Type != "project" {
		writeError(w, http.StatusBadRequest, "type must be 'user' or 'project'")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	var mem *memory.Memory
	switch req.Action {
	case "append":
		mem, err = h.memoryManager.Append(memory.MemoryType(req.Type), sess.WorkingDirectory, req.Key, req.Content, memory.SourceManual)
	default:
		mem, err = h.memoryManager.Upsert(memory.MemoryType(req.Type), sess.WorkingDirectory, req.Key, req.Content, memory.SourceManual)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save memory")
		return
	}

	eventBus, busErr := h.store.GetEventBus(sess.ID)
	if busErr == nil {
		eventBus.Publish("memory_updated", map[string]any{
			"memory_type": string(mem.Type),
			"key":         mem.Key,
			"summary":     truncate(mem.Content, 100),
		})
	}

	writeJSON(w, http.StatusOK, memoryResponse{
		ID:        mem.ID,
		Type:      string(mem.Type),
		Key:       mem.Key,
		Content:   mem.Content,
		Source:    string(mem.Source),
		CreatedAt: mem.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: mem.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	memoryID := r.PathValue("memory_id")

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	mem, err := h.memoryManager.Get(memoryID)
	if err != nil {
		if errors.Is(err, memory.ErrMemoryNotFound) {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.memoryManager.Delete(mem.Type, sess.WorkingDirectory, mem.Key); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete memory")
		return
	}

	eventBus, busErr := h.store.GetEventBus(sess.ID)
	if busErr == nil {
		eventBus.Publish("memory_updated", map[string]any{
			"memory_type": string(mem.Type),
			"key":         mem.Key,
			"summary":     "deleted",
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
