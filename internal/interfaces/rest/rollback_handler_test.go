package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devo/internal/core/agentloop"
	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func TestRollbackHandler(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := agentloop.New(store, llmclient.NewMockClient())
	memStore, _ := memory.NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	store.AddMessage("sess-1", session.Message{
		ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now(),
	})
	store.AddMessage("sess-1", session.Message{
		ID: "msg-2", Role: session.RoleAssistant, Content: "Hi!", CreatedAt: time.Now(),
	})

	body := map[string]string{"target_message_id": "msg-1"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sess-1/rollback", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "sess-1")
	rec := httptest.NewRecorder()

	handler.Rollback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result struct {
		ActualRollbackMessageID string `json:"actual_rollback_message_id"`
		Adjusted                bool   `json:"adjusted"`
	}
	json.Unmarshal(rec.Body.Bytes(), &result)

	if result.ActualRollbackMessageID != "msg-1" {
		t.Errorf("expected actual_rollback_message_id msg-1, got %s", result.ActualRollbackMessageID)
	}
}

func TestRollbackHandlerMissingTarget(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := agentloop.New(store, llmclient.NewMockClient())
	memStore, _ := memory.NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	body := map[string]string{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sess-1/rollback", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "sess-1")
	rec := httptest.NewRecorder()

	handler.Rollback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRollbackHandlerSessionNotFound(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := agentloop.New(store, llmclient.NewMockClient())
	memStore, _ := memory.NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

	body := map[string]string{"target_message_id": "msg-1"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/nonexistent/rollback", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	handler.Rollback(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRollbackHandlerArchivedSession(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := agentloop.New(store, llmclient.NewMockClient())
	memStore, _ := memory.NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            session.StateArchived,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	body := map[string]string{"target_message_id": "msg-1"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sess-1/rollback", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "sess-1")
	rec := httptest.NewRecorder()

	handler.Rollback(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRollbackHandlerMessageNotFound(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := agentloop.New(store, llmclient.NewMockClient())
	memStore, _ := memory.NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	body := map[string]string{"target_message_id": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sess-1/rollback", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", "sess-1")
	rec := httptest.NewRecorder()

	handler.Rollback(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
