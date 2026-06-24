package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"devo/internal/core/session"
)

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
	if result.State != "idle" {
		t.Errorf("expected state idle, got %q", result.State)
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

	if result.Title != "" {
		t.Errorf("expected empty title, got %q", result.Title)
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
}

func TestCreateSession_DefaultTimeout(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()

	body := map[string]string{
		"working_directory": tmpDir,
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

	if result.ApprovalTimeoutSeconds != 300 {
		t.Errorf("expected default approval_timeout_seconds 300, got %d", result.ApprovalTimeoutSeconds)
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

func TestGetSession_IncludesToolCallFields(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ToolCallLimit:    50,
		ToolCallCount:    3,
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

	if result.ToolCallLimit != 50 {
		t.Errorf("expected tool_call_limit 50, got %d", result.ToolCallLimit)
	}
	if result.ToolCallCount != 3 {
		t.Errorf("expected tool_call_count 3, got %d", result.ToolCallCount)
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
		if s.State != "idle" {
			t.Errorf("expected all sessions to be idle, got %q for %s", s.State, s.ID)
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

func TestUpdateConfig_ToolCallLimit(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ToolCallLimit:    50,
	}
	store.Create(sess)

	body := map[string]int{"tool_call_limit": 80}
	jsonBody, _ := json.Marshal(body)

	resp := doPut(t, server.URL+"/api/v1/sessions/sess-test-1/config", jsonBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result updateConfigResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ToolCallLimit != 80 {
		t.Errorf("expected tool_call_limit 80, got %d", result.ToolCallLimit)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.ToolCallLimit != 80 {
		t.Errorf("expected tool_call_limit 80 in store, got %d", sessGot.ToolCallLimit)
	}
}

func TestUpdateConfig_InvalidValue(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ToolCallLimit:    50,
	}
	store.Create(sess)

	body := map[string]int{"tool_call_limit": 0}
	jsonBody, _ := json.Marshal(body)

	resp := doPut(t, server.URL+"/api/v1/sessions/sess-test-1/config", jsonBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for tool_call_limit=0, got %d", resp.StatusCode)
	}
}

func TestUpdateConfig_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]int{"tool_call_limit": 80}
	jsonBody, _ := json.Marshal(body)

	resp := doPut(t, server.URL+"/api/v1/sessions/nonexistent/config", jsonBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateConfig_MaxConcurrentToolCalls(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ToolCallLimit:    50,
	}
	store.Create(sess)

	body := map[string]int{"max_concurrent_tool_calls": 5}
	jsonBody, _ := json.Marshal(body)

	resp := doPut(t, server.URL+"/api/v1/sessions/sess-test-1/config", jsonBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result updateConfigResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.MaxConcurrentToolCalls != 5 {
		t.Errorf("expected max_concurrent_tool_calls 5, got %d", result.MaxConcurrentToolCalls)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.MaxConcurrentToolCalls != 5 {
		t.Errorf("expected max_concurrent_tool_calls 5 in store, got %d", sessGot.MaxConcurrentToolCalls)
	}
}

func TestUpdateConfig_MaxConcurrentSubprocesses(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ToolCallLimit:    50,
	}
	store.Create(sess)

	body := map[string]int{"max_concurrent_subprocesses": 3}
	jsonBody, _ := json.Marshal(body)

	resp := doPut(t, server.URL+"/api/v1/sessions/sess-test-1/config", jsonBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result updateConfigResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.MaxConcurrentSubprocesses != 3 {
		t.Errorf("expected max_concurrent_subprocesses 3, got %d", result.MaxConcurrentSubprocesses)
	}

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.MaxConcurrentSubprocesses != 3 {
		t.Errorf("expected max_concurrent_subprocesses 3 in store, got %d", sessGot.MaxConcurrentSubprocesses)
	}
}

func TestUpdateConfig_AllFields(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-test-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
		ToolCallLimit:    50,
	}
	store.Create(sess)

	body := map[string]int{
		"tool_call_limit":             100,
		"max_concurrent_tool_calls":   5,
		"max_concurrent_subprocesses": 3,
	}
	jsonBody, _ := json.Marshal(body)

	resp := doPut(t, server.URL+"/api/v1/sessions/sess-test-1/config", jsonBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result updateConfigResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ToolCallLimit != 100 {
		t.Errorf("expected tool_call_limit 100, got %d", result.ToolCallLimit)
	}
	if result.MaxConcurrentToolCalls != 5 {
		t.Errorf("expected max_concurrent_tool_calls 5, got %d", result.MaxConcurrentToolCalls)
	}
	if result.MaxConcurrentSubprocesses != 3 {
		t.Errorf("expected max_concurrent_subprocesses 3, got %d", result.MaxConcurrentSubprocesses)
	}
}

func TestGetSession_IncludesConcurrencyFields(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:                        "sess-test-1",
		Title:                     "Test",
		WorkingDirectory:          tmpDir,
		State:                     session.StateIdle,
		MaxConcurrentToolCalls:    5,
		MaxConcurrentSubprocesses: 3,
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
	if result.MaxConcurrentToolCalls != 5 {
		t.Errorf("expected max_concurrent_tool_calls 5, got %d", result.MaxConcurrentToolCalls)
	}
	if result.MaxConcurrentSubprocesses != 3 {
		t.Errorf("expected max_concurrent_subprocesses 3, got %d", result.MaxConcurrentSubprocesses)
	}
}
