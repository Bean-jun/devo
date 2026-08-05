package rest

import (
	"encoding/json"
	"net/http"
	"testing"

	"devo/internal/core/session"
)

func TestCompact_FromIdle(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		MaxContextTokens: 128000,
		KeepRecent:       30,
	}
	store.Create(sess)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/compact", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if _, ok := result["compressed_count"]; !ok {
		t.Error("expected compressed_count in response")
	}
	if _, ok := result["tokens_removed"]; !ok {
		t.Error("expected tokens_removed in response")
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateIdle {
		t.Errorf("expected state to remain idle, got %s", sessGot.State)
	}
}

func TestCompact_FromProcessing(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateThinking,
		MaxContextTokens: 128000,
		KeepRecent:       30,
	}
	store.Create(sess)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/compact", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for non-idle state, got %d", resp.StatusCode)
	}
}

func TestCompact_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/compact", "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
