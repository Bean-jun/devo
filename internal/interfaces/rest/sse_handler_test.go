package rest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"devo/internal/core/session"
)

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
			if strings.HasPrefix(line, "data: ") {
				dataStr := strings.TrimPrefix(line, "data: ")
				var wrapper struct {
					Type string `json:"type"`
				}
				if json.Unmarshal([]byte(dataStr), &wrapper) == nil {
					events[wrapper.Type] = true
				}
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
		if strings.HasPrefix(line, "id: ") || strings.HasPrefix(line, "data: ") {
			receivedCount++
		}
	}

	receivedEvents := receivedCount / 2
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
