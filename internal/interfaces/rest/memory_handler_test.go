package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devo/internal/config"
	"devo/internal/core/agent"
	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func setupMemoryTestServer(t *testing.T) (*httptest.Server, *session.InMemoryStore) {
	t.Helper()
	store := session.NewInMemoryStore()
	llm := llmclient.NewMockClient()
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memStore, err := memory.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("create file store: %v", err)
	}
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)

	ag := agent.New(
		agent.Config{ID: "test", Name: "Test", Description: "Test agent", SystemPrompt: "", Tools: nil},
		store, llm, nil, config.DefaultConfig(),
		approvalMgr, memManager, nil, nil, nil, nil,
	)
	registry := agent.NewRegistry(ag)

	handler := NewHandler(HandlerDeps{Store: store, AgentRegistry: registry, Version: "0.0.1", MemoryManager: memManager})

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return httptest.NewServer(mux), store
}

func TestGetMemories_Empty(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-1", tmpDir)

	resp, err := http.Get(server.URL + "/api/v1/sessions/" + sess.ID + "/memory?type=user")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result listMemoryResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Memories) != 0 {
		t.Errorf("expected 0 memories, got %d", len(result.Memories))
	}
}

func TestUpsertMemory_User(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-2", tmpDir)

	body := map[string]string{
		"type":    "user",
		"key":     "editor_pref",
		"content": "I prefer 4 spaces",
	}
	bodyBytes, _ := json.Marshal(body)

	resp, err := http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var memResp memoryResponse
	json.NewDecoder(resp.Body).Decode(&memResp)
	if memResp.Key != "editor_pref" {
		t.Errorf("expected key editor_pref, got %s", memResp.Key)
	}
}

func TestUpsertMemory_Project(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-3", tmpDir)

	body := map[string]string{
		"type":    "project",
		"key":     "api_port",
		"content": "8080",
	}
	bodyBytes, _ := json.Marshal(body)

	resp, err := http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpsertMemory_Append(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-4", tmpDir)

	body1 := map[string]string{
		"type":    "user",
		"key":     "notes",
		"content": "line1",
	}
	bodyBytes1, _ := json.Marshal(body1)
	http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes1),
	)

	body2 := map[string]string{
		"type":    "user",
		"key":     "notes",
		"content": "line2",
		"action":  "append",
	}
	bodyBytes2, _ := json.Marshal(body2)
	resp, err := http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes2),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var memResp memoryResponse
	json.NewDecoder(resp.Body).Decode(&memResp)
	if memResp.Content != "line1\nline2" {
		t.Errorf("expected content 'line1\\nline2', got %q", memResp.Content)
	}
}

func TestUpsertMemory_InvalidType(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-5", tmpDir)

	body := map[string]string{
		"type":    "invalid",
		"key":     "test",
		"content": "test",
	}
	bodyBytes, _ := json.Marshal(body)

	resp, err := http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpsertMemory_MissingKey(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-6", tmpDir)

	body := map[string]string{
		"type":    "user",
		"content": "test",
	}
	bodyBytes, _ := json.Marshal(body)

	resp, err := http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDeleteMemory(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-7", tmpDir)

	body := map[string]string{
		"type":    "user",
		"key":     "temp_key",
		"content": "temp content",
	}
	bodyBytes, _ := json.Marshal(body)
	resp, _ := http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes),
	)

	var memResp memoryResponse
	json.NewDecoder(resp.Body).Decode(&memResp)
	resp.Body.Close()

	req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/sessions/"+sess.ID+"/memory/"+memResp.ID, nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}
}

func TestDeleteMemory_NotFound(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-8", tmpDir)

	req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/sessions/"+sess.ID+"/memory/nonexistent", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetMemories_InvalidType(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-9", tmpDir)

	resp, err := http.Get(server.URL + "/api/v1/sessions/" + sess.ID + "/memory?type=invalid")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetMemories_ListAfterUpsert(t *testing.T) {
	server, store := setupMemoryTestServer(t)
	defer server.Close()

	tmpDir := t.TempDir()
	sess := createTestSession(store, "sess-mem-10", tmpDir)

	body := map[string]string{
		"type":    "user",
		"key":     "test_key",
		"content": "test content",
	}
	bodyBytes, _ := json.Marshal(body)
	http.Post(
		server.URL+"/api/v1/sessions/"+sess.ID+"/memory",
		"application/json",
		bytes.NewReader(bodyBytes),
	)

	resp, err := http.Get(server.URL + "/api/v1/sessions/" + sess.ID + "/memory?type=user")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result listMemoryResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(result.Memories))
	}
	if result.Memories[0].Content != "test content" {
		t.Errorf("expected content 'test content', got %q", result.Memories[0].Content)
	}
}

func createTestSession(store *session.InMemoryStore, id string, workingDir string) *session.Session {
	sess := &session.Session{
		ID:               id,
		Title:            "Test Session",
		WorkingDirectory: workingDir,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)
	return sess
}
