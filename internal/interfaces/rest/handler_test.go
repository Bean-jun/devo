package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"devo/internal/core/agentloop"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func setupTestServer() (*httptest.Server, *session.InMemoryStore) {
	store := session.NewInMemoryStore()
	llm := llmclient.NewMockClient()
	loop := agentloop.New(store, llm)
	handler := NewHandler(store, loop)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), store
}

func TestCreateSession_Success(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()

	body := map[string]string{
		"working_directory": tmpDir,
		"title":             "My Test Session",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result createSessionResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID == "" {
		t.Error("expected non-empty id")
	}
	if result.State != "Idle" {
		t.Errorf("expected state Idle, got %q", result.State)
	}
	if result.Title != "My Test Session" {
		t.Errorf("expected title 'My Test Session', got %q", result.Title)
	}
	if result.WorkingDirectory != tmpDir {
		t.Errorf("expected working_directory %q, got %q", tmpDir, result.WorkingDirectory)
	}
	if result.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestCreateSession_DirectoryNotExist(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]string{
		"working_directory": "/nonexistent/path/12345",
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSession_FileNotDirectory(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	body := map[string]string{
		"working_directory": tmpFile,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for file path, got %d", resp.StatusCode)
	}
}

func TestCreateSession_DefaultTitle(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()

	body := map[string]string{
		"working_directory": tmpDir,
	}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	var result createSessionResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Title != tmpDir {
		t.Errorf("expected default title %q, got %q", tmpDir, result.Title)
	}
}

func TestGetSession_Success(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	sess.CreatedAt = sess.CreatedAt.UTC()
	sess.LastActiveAt = sess.LastActiveAt.UTC()
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-test-1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result getSessionResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID != "sess-test-1" {
		t.Errorf("expected id sess-test-1, got %q", result.ID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/sessions/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPostMessage_Success(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	body := map[string]string{"content": "Hello, Devo!"}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result postMessageResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Role != "assistant" {
		t.Errorf("expected role assistant, got %q", result.Role)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if result.MessageID == "" {
		t.Error("expected non-empty message_id")
	}
}

func TestPostMessage_SessionNotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]string{"content": "Hello"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/messages", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPostMessage_ConflictWhenProcessing(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateProcessing,
	}
	store.Create(sess)

	body := map[string]string{"content": "Hello"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestGetMessages_Success(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	store.AddMessage("sess-test-1", session.Message{
		ID:      "msg-1",
		Role:    session.RoleUser,
		Content: "Hello",
	})
	store.AddMessage("sess-test-1", session.Message{
		ID:      "msg-2",
		Role:    session.RoleAssistant,
		Content: "Hi there",
	})

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-test-1/messages")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result getMessagesResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result.Messages))
	}
	if result.Messages[0].Role != "user" {
		t.Errorf("expected first message role user, got %q", result.Messages[0].Role)
	}
	if result.Messages[1].Role != "assistant" {
		t.Errorf("expected second message role assistant, got %q", result.Messages[1].Role)
	}
}

func TestGetMessages_Pagination(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	for i := 0; i < 5; i++ {
		store.AddMessage("sess-test-1", session.Message{
			ID:      "msg",
			Role:    session.RoleUser,
			Content: "msg",
		})
	}

	resp, _ := http.Get(server.URL + "/api/v1/sessions/sess-test-1/messages?limit=2&offset=1")
	defer resp.Body.Close()

	var result getMessagesResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
	if len(result.Messages) != 2 {
		t.Errorf("expected 2 messages with limit=2, got %d", len(result.Messages))
	}
}

func TestGetMessages_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Get(server.URL + "/api/v1/sessions/nonexistent/messages")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestFullConversationFlow(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()

	createBody := map[string]string{
		"working_directory": tmpDir,
		"title":             "Integration Test",
	}
	createJSON, _ := json.Marshal(createBody)
	resp, _ := http.Post(server.URL+"/api/v1/sessions", "application/json", bytes.NewReader(createJSON))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session: expected 201, got %d", resp.StatusCode)
	}

	var created createSessionResponse
	json.NewDecoder(resp.Body).Decode(&created)

	msgBody := map[string]string{"content": "First question"}
	msgJSON, _ := json.Marshal(msgBody)
	resp2, _ := http.Post(server.URL+"/api/v1/sessions/"+created.ID+"/messages", "application/json", bytes.NewReader(msgJSON))
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("send message: expected 200, got %d", resp2.StatusCode)
	}

	var msgResp postMessageResponse
	json.NewDecoder(resp2.Body).Decode(&msgResp)

	if msgResp.Role != "assistant" {
		t.Errorf("expected assistant role, got %q", msgResp.Role)
	}
	if msgResp.Content == "" {
		t.Error("expected non-empty reply")
	}

	resp3, _ := http.Get(server.URL + "/api/v1/sessions/" + created.ID + "/messages")
	defer resp3.Body.Close()

	var msgsResp getMessagesResponse
	json.NewDecoder(resp3.Body).Decode(&msgsResp)

	if msgsResp.Total != 2 {
		t.Errorf("expected 2 messages in history, got %d", msgsResp.Total)
	}

	resp4, _ := http.Get(server.URL + "/api/v1/sessions/" + created.ID)
	defer resp4.Body.Close()

	var sessResp getSessionResponse
	json.NewDecoder(resp4.Body).Decode(&sessResp)

	if sessResp.State != "Idle" {
		t.Errorf("session should be Idle after message completes, got %q", sessResp.State)
	}
}
