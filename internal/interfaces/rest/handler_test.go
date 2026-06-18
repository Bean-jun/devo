package rest

import (
	"bufio"
	"bytes"
	"context"
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
	"devo/internal/taskexec/tools"
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

func setupTestServerWithTools() (*httptest.Server, *session.InMemoryStore, *agentloop.Loop) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	llm := &approvalMockClient{}
	loop := agentloop.NewWithTools(store, llm, toolRegistry)
	handler := NewHandler(store, loop)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), store, loop
}

type approvalMockClient struct {
	callCount int
}

func (m *approvalMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "I received the tool result: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser || lastMsg.Role == session.RoleSystem {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "write_file",
						Params: map[string]interface{}{
							"path":    "test_approve.txt",
							"content": "Hello from approve test",
						},
					},
				},
			}, nil
		}
		return &llmclient.CompleteResult{
			Text: "Task completed.",
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
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

func TestListSessionsAll(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	store.Create(&session.Session{
		ID:               "sess-list-1",
		Title:            "List Test 1",
		WorkingDirectory: tmpDir1,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})
	store.Create(&session.Session{
		ID:               "sess-list-2",
		Title:            "List Test 2",
		WorkingDirectory: tmpDir2,
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	resp, err := http.Get(server.URL + "/api/v1/sessions?status=all")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result listSessionsResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total < 2 {
		t.Errorf("expected at least 2 sessions, got %d", result.Total)
	}
	if len(result.Sessions) < 2 {
		t.Errorf("expected at least 2 sessions in response, got %d", len(result.Sessions))
	}
}

func TestListSessionsFilterByStatus(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	store.Create(&session.Session{
		ID:               "sess-filter-1",
		Title:            "Idle Session",
		WorkingDirectory: tmpDir1,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})
	store.Create(&session.Session{
		ID:               "sess-filter-2",
		Title:            "Processing Session",
		WorkingDirectory: tmpDir2,
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	resp, err := http.Get(server.URL + "/api/v1/sessions?status=Idle")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result listSessionsResponse
	json.NewDecoder(resp.Body).Decode(&result)

	for _, s := range result.Sessions {
		if s.State != "Idle" {
			t.Errorf("expected all sessions to be Idle, got %q for %s", s.State, s.ID)
		}
	}
}

func TestListSessionsPagination(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	for i := 0; i < 5; i++ {
		store.Create(&session.Session{
			ID:               session.GenerateID("sess"),
			Title:            "Test",
			WorkingDirectory: t.TempDir(),
			State:            session.StateIdle,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		})
	}

	resp, err := http.Get(server.URL + "/api/v1/sessions?limit=2&offset=0")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result listSessionsResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Sessions) > 2 {
		t.Errorf("expected at most 2 sessions with limit=2, got %d", len(result.Sessions))
	}
}

func TestListSessionsEmptyResult(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/sessions?status=Paused")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result listSessionsResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 0 {
		t.Errorf("expected 0 sessions, got %d", result.Total)
	}
	if len(result.Sessions) != 0 {
		t.Errorf("expected empty sessions array, got %d", len(result.Sessions))
	}
}

func TestSetTrustLevel_Success(t *testing.T) {
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

	body := map[string]string{"trust_level": "elevated"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/sess-test-1/trust", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["trust_level"] != "elevated" {
		t.Errorf("expected trust_level elevated, got %q", result["trust_level"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.TrustLevel != "elevated" {
		t.Errorf("expected trust_level elevated in store, got %q", sessGot.TrustLevel)
	}
}

func TestSetTrustLevel_InvalidValue(t *testing.T) {
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

	body := map[string]string{"trust_level": "super_trusted"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/sess-test-1/trust", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid trust_level, got %d", resp.StatusCode)
	}
}

func TestSetTrustLevel_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]string{"trust_level": "normal"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/nonexistent/trust", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSetApprovalPolicy_Success(t *testing.T) {
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

	body := map[string]string{
		"file_write_new":  "session_trust",
		"execute_command": "always_ask",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/sess-test-1/approval-policy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	policy, ok := result["approval_policy"].(map[string]interface{})
	if !ok {
		t.Fatal("expected approval_policy in response")
	}
	if policy["file_write_new"] != "session_trust" {
		t.Errorf("expected file_write_new to be session_trust, got %v", policy["file_write_new"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.ApprovalPolicy["file_write_new"] != "session_trust" {
		t.Errorf("expected file_write_new session_trust in store, got %q", sessGot.ApprovalPolicy["file_write_new"])
	}
}

func TestSetApprovalPolicy_PartialUpdate(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ApprovalPolicy: map[string]string{
			"file_write_new":  "always_ask",
			"execute_command": "always_ask",
		},
	}
	store.Create(sess)

	body := map[string]string{
		"file_write_new": "session_trust",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/sess-test-1/approval-policy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.ApprovalPolicy["file_write_new"] != "session_trust" {
		t.Errorf("expected file_write_new updated to session_trust, got %q", sessGot.ApprovalPolicy["file_write_new"])
	}
	if sessGot.ApprovalPolicy["execute_command"] != "always_ask" {
		t.Errorf("expected execute_command to remain always_ask, got %q", sessGot.ApprovalPolicy["execute_command"])
	}
}

func TestSetApprovalPolicy_InvalidOperationType(t *testing.T) {
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

	body := map[string]string{
		"invalid_op": "session_trust",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/sess-test-1/approval-policy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid operation_type, got %d", resp.StatusCode)
	}
}

func TestSetApprovalPolicy_InvalidPolicyLevel(t *testing.T) {
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

	body := map[string]string{
		"file_write_new": "super_trust",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/sess-test-1/approval-policy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid policy_level, got %d", resp.StatusCode)
	}
}

func TestSetApprovalPolicy_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]string{
		"file_write_new": "session_trust",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", server.URL+"/api/v1/sessions/nonexistent/approval-policy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateSession_WithApprovalTimeout(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()

	body := map[string]interface{}{
		"working_directory":        tmpDir,
		"approval_timeout_seconds": 120,
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

	if result.ApprovalTimeoutSeconds != 120 {
		t.Errorf("expected approval_timeout_seconds 120, got %d", result.ApprovalTimeoutSeconds)
	}
	if result.TrustLevel != "normal" {
		t.Errorf("expected default trust_level normal, got %q", result.TrustLevel)
	}
}

func TestCreateSession_DefaultTimeout(t *testing.T) {
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

	if result.ApprovalTimeoutSeconds != 300 {
		t.Errorf("expected default approval_timeout_seconds 300, got %d", result.ApprovalTimeoutSeconds)
	}
}

func TestApprove_ApproveDecision(t *testing.T) {
	server, store, loop := setupTestServerWithTools()
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
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	go loop.ProcessMessage(context.Background(), "sess-test-1", "Create a file")

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required")
	}
	data := evt.Data.(map[string]any)
	approvalID := data["approval_id"].(string)

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve/"+approvalID, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result approveResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Decision != "approve" {
		t.Errorf("expected decision 'approve', got %q", result.Decision)
	}
	if result.ApprovalID != approvalID {
		t.Errorf("expected approval_id %q, got %q", approvalID, result.ApprovalID)
	}
}

func TestApprove_RejectDecision(t *testing.T) {
	server, store, loop := setupTestServerWithTools()
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
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	go loop.ProcessMessage(context.Background(), "sess-test-1", "Create a file")

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required")
	}
	data := evt.Data.(map[string]any)
	approvalID := data["approval_id"].(string)

	body := map[string]string{"decision": "reject"}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve/"+approvalID, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result approveResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Decision != "reject" {
		t.Errorf("expected decision 'reject', got %q", result.Decision)
	}
}

func TestApprove_InvalidDecision(t *testing.T) {
	server, store, loop := setupTestServerWithTools()
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
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	go loop.ProcessMessage(context.Background(), "sess-test-1", "Create a file")

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required")
	}
	data := evt.Data.(map[string]any)
	approvalID := data["approval_id"].(string)

	body := map[string]string{"decision": "maybe"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve/"+approvalID, "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid decision, got %d", resp.StatusCode)
	}
}

func TestApprove_TimeoutReturns409(t *testing.T) {
	server, store, loop := setupTestServerWithTools()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:                     "sess-test-1",
		Title:                  "Test",
		WorkingDirectory:       tmpDir,
		State:                  session.StateIdle,
		ApprovalTimeoutSeconds: 1,
	}
	store.Create(sess)

	eventBus, _ := store.GetEventBus("sess-test-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	go loop.ProcessMessage(context.Background(), "sess-test-1", "Create a file")

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required")
	}
	data := evt.Data.(map[string]any)
	approvalID := data["approval_id"].(string)

	evt, ok = waitForEvent(ch, "approval_resolved", 5*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_resolved (timeout)")
	}
	resolvedData := evt.Data.(map[string]any)
	if resolvedData["source"] != "timeout" {
		t.Fatalf("expected timeout source, got %v", resolvedData["source"])
	}

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve/"+approvalID, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 after timeout, got %d", resp.StatusCode)
	}
}

func TestApprove_SessionNotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/approve/approval-1", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestApprove_NotAwaitingApproval(t *testing.T) {
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

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve/approval-1", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for non-AwaitingApproval state, got %d", resp.StatusCode)
	}
}

func TestCancel_FromProcessing(t *testing.T) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StateIdle) {
		t.Errorf("expected state Idle, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.CancelRequested {
		t.Error("CancelRequested should be true after cancel")
	}
}

func TestCancel_FromAwaitingApproval(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateAwaitingApproval,
	}
	store.Create(sess)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StateIdle) {
		t.Errorf("expected state Idle, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if !sessGot.CancelRequested {
		t.Error("expected CancelRequested to be true")
	}
}

func TestCancel_FromIdle(t *testing.T) {
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

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/cancel", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestCancel_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/cancel", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPause_FromProcessing(t *testing.T) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/pause", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StatePaused) {
		t.Errorf("expected state Paused, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if !sessGot.PauseRequested {
		t.Error("expected PauseRequested to be true")
	}
}

func TestPause_FromIdle(t *testing.T) {
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

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/pause", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestPause_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/pause", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestResume_FromPaused(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StatePaused,
	}
	store.Create(sess)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/resume", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StateProcessing) {
		t.Errorf("expected state Processing, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateProcessing {
		t.Errorf("expected state Processing in store, got %q", sessGot.State)
	}
}

func TestResume_FromIdle(t *testing.T) {
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

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/resume", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestResume_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/resume", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestComplete_FromIdle(t *testing.T) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StateCompleted) {
		t.Errorf("expected state Completed, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateCompleted {
		t.Errorf("expected state Completed in store, got %q", sessGot.State)
	}
}

func TestComplete_FromProcessing(t *testing.T) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StateCompleted) {
		t.Errorf("expected state Completed, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateCompleted {
		t.Errorf("expected state Completed in store, got %q", sessGot.State)
	}
}

func TestComplete_FromArchived(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateArchived,
	}
	store.Create(sess)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/complete", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestComplete_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/complete", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestArchive_FromCompleted(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateCompleted,
	}
	store.Create(sess)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/archive", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["state"] != string(session.StateArchived) {
		t.Errorf("expected state Archived, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateArchived {
		t.Errorf("expected state Archived in store, got %q", sessGot.State)
	}
}

func TestArchive_FromIdle(t *testing.T) {
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

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/archive", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestArchive_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/archive", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPostMessage_ArchivedReturns409(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateArchived,
	}
	store.Create(sess)

	body := map[string]string{"content": "Hello from archived"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for archived session, got %d", resp.StatusCode)
	}
}

func TestPostMessage_PausedAutoResumes(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StatePaused,
	}
	store.Create(sess)

	body := map[string]string{"content": "Hello from paused"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateProcessing {
		t.Errorf("expected state Processing after auto-resume, got %q", sessGot.State)
	}
}

func TestFullStateMachineViaREST(t *testing.T) {
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

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/complete", "application/json", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d", resp.StatusCode)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateCompleted {
		t.Fatalf("expected Completed, got %q", sessGot.State)
	}

	resp, _ = http.Post(server.URL+"/api/v1/sessions/sess-test-1/archive", "application/json", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("archive: expected 200, got %d", resp.StatusCode)
	}

	sessGot, _ = store.Get("sess-test-1")
	if sessGot.State != session.StateArchived {
		t.Fatalf("expected Archived, got %q", sessGot.State)
	}

	body := map[string]string{"content": "Try posting to archived"}
	jsonBody, _ := json.Marshal(body)
	resp, _ = http.Post(server.URL+"/api/v1/sessions/sess-test-1/messages", "application/json", bytes.NewReader(jsonBody))
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for archived, got %d", resp.StatusCode)
	}
}

func waitForEvent(ch chan session.Event, eventType string, timeout time.Duration) (*session.Event, bool) {
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			return nil, false
		case evt, ok := <-ch:
			if !ok {
				return nil, false
			}
			if eventType == "" || evt.Type == eventType {
				return &evt, true
			}
		}
	}
}
