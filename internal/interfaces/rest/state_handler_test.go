package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"devo/internal/core/session"
)

func TestCancel_FromProcessing(t *testing.T) {
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
	if result["state"] != string(session.StateIdle.ToSnakeCase()) {
		t.Errorf("expected state Idle, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if !sessGot.CancelRequested {
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
	if result["state"] != string(session.StateIdle.ToSnakeCase()) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestCancel_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/sessions/nonexistent/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
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
		State:            session.StateToolExecuting,
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
	if result["state"] != string(session.StatePaused.ToSnakeCase()) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/pause", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestPause_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/sessions/nonexistent/pause", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
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
	if result["state"] != string(session.StateToolExecuting.ToSnakeCase()) {
		t.Errorf("expected state tool_executing, got %q", result["state"])
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateToolExecuting {
		t.Errorf("expected state ToolExecuting in store, got %q", sessGot.State)
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/resume", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestResume_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/sessions/nonexistent/resume", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
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
	if result["state"] != string(session.StateCompleted.ToSnakeCase()) {
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
		State:            session.StateThinking,
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
	if result["state"] != string(session.StateCompleted.ToSnakeCase()) {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestComplete_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/sessions/nonexistent/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
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
	if result["state"] != session.StateArchived.ToSnakeCase() {
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/archive", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestArchive_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/v1/sessions/nonexistent/archive", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
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

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("complete request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d", resp.StatusCode)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateCompleted {
		t.Fatalf("expected Completed, got %q", sessGot.State)
	}

	resp, err = http.Post(server.URL+"/api/v1/sessions/sess-test-1/archive", "application/json", nil)
	if err != nil {
		t.Fatalf("archive request failed: %v", err)
	}
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
