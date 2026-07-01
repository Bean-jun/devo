package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"devo/internal/core/session"
)

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
		State:            session.StateThinking,
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
	if sessGot.State != session.StateThinking {
		t.Errorf("expected state Thinking after auto-resume, got %q", sessGot.State)
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

	if sessResp.State != "idle" {
		t.Errorf("session should be idle after message completes, got %q", sessResp.State)
	}
}
