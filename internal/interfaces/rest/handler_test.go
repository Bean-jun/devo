package rest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestPostMessage_AcceptedAsync(t *testing.T) {
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

	start := time.Now()
	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if elapsed > 500*time.Millisecond {
		t.Errorf("POST /messages should return immediately (async), took %v", elapsed)
	}

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	var result map[string]bool
	json.NewDecoder(resp.Body).Decode(&result)
	if !result["accepted"] {
		t.Errorf("expected accepted: true in response")
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

	if resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("send message: expected 202, got %d", resp2.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	resp3, _ := http.Get(server.URL + "/api/v1/sessions/" + created.ID + "/messages")
	defer resp3.Body.Close()

	var msgsResp getMessagesResponse
	json.NewDecoder(resp3.Body).Decode(&msgsResp)

	if msgsResp.Total != 2 {
		t.Errorf("expected 2 messages in history after async processing, got %d", msgsResp.Total)
	}

	resp4, _ := http.Get(server.URL + "/api/v1/sessions/" + created.ID)
	defer resp4.Body.Close()

	var sessResp getSessionResponse
	json.NewDecoder(resp4.Body).Decode(&sessResp)

	if sessResp.State != "Idle" {
		t.Errorf("session should be Idle after message completes, got %q", sessResp.State)
	}
}

func TestSSEConnect(t *testing.T) {
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

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/sess-test-1/events", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE connect failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type: text/event-stream, got %q", resp.Header.Get("Content-Type"))
	}

	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache")
	}
}

func TestSSEIncrementDecrementConnectionCount(t *testing.T) {
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

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.ActiveSSEConnections != 0 {
		t.Errorf("expected 0 connections initially, got %d", sessGot.ActiveSSEConnections)
	}

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/sess-test-1/events", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE connect failed: %v", err)
	}

	sessGot, _ = store.Get("sess-test-1")
	if sessGot.ActiveSSEConnections != 1 {
		t.Errorf("expected 1 connection after connect, got %d", sessGot.ActiveSSEConnections)
	}

	resp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	sessGot, _ = store.Get("sess-test-1")
	if sessGot.ActiveSSEConnections != 0 {
		t.Errorf("expected 0 connections after disconnect, got %d", sessGot.ActiveSSEConnections)
	}
}

func TestSSEReceiveEvents(t *testing.T) {
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

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/sess-test-1/events", nil)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE connect failed: %v", err)
	}
	defer resp.Body.Close()

	postBody := map[string]string{"content": "Hello SSE"}
	jsonBody, _ := json.Marshal(postBody)
	postResp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("POST message failed: %v", err)
	}
	postResp.Body.Close()

	if postResp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", postResp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	events := make(map[string]bool)
	expectedEvents := []string{"thinking", "message_complete", "session_state_change"}
	deadline := time.After(2 * time.Second)

readLoop:
	for {
		if len(events) >= len(expectedEvents) {
			break
		}
		select {
		case <-deadline:
			break readLoop
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break readLoop
				}
				t.Logf("read error: %v", err)
				break readLoop
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "event: ") {
				eventType := strings.TrimPrefix(line, "event: ")
				events[eventType] = true
			}
		}
	}

	for _, expected := range expectedEvents {
		if !events[expected] {
			t.Errorf("expected event %q not received", expected)
		}
	}

	if len(events) < len(expectedEvents) {
		t.Logf("Received events: %v", events)
	}
}

func TestSSELastEventIDReplay(t *testing.T) {
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

	eventBus, _ := store.GetEventBus("sess-test-1")
	eventBus.Publish("test1", map[string]string{"n": "1"})
	eventBus.Publish("test2", map[string]string{"n": "2"})
	eventBus.Publish("test3", map[string]string{"n": "3"})

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/sess-test-1/events", nil)
	req.Header.Set("Last-Event-ID", "1")
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE connect failed: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	receivedCount := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "id: ") || strings.HasPrefix(line, "event: ") || strings.HasPrefix(line, "data: ") {
			receivedCount++
		}
	}

	receivedEvents := receivedCount / 3
	if receivedEvents != 2 {
		t.Errorf("expected 2 events after ID 1, got %d events", receivedEvents)
	}
}

func TestSSENotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/nonexistent/events", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestMultipleSSEConnectionsSameSession(t *testing.T) {
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

	req1, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/sess-test-1/events", nil)
	req2, _ := http.NewRequest("GET", server.URL+"/api/v1/sessions/sess-test-1/events", nil)
	client := &http.Client{}

	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("conn 1 failed: %v", err)
	}
	defer resp1.Body.Close()

	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("conn 2 failed: %v", err)
	}
	defer resp2.Body.Close()

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.ActiveSSEConnections != 2 {
		t.Errorf("expected 2 active connections, got %d", sessGot.ActiveSSEConnections)
	}

	postBody := map[string]string{"content": "Hello two listeners"}
	jsonBody, _ := json.Marshal(postBody)
	postResp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	postResp.Body.Close()

	if postResp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", postResp.StatusCode)
	}
}
