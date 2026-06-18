package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"devo/internal/core/session"
)

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

	loop.ProcessMessage(nil, "sess-test-1", "Write test_approve.txt")

	time.Sleep(200 * time.Millisecond)

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateAwaitingApproval {
		t.Fatalf("expected state AwaitingApproval, got %q", sessGot.State)
	}

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["decision"] != "approved" {
		t.Errorf("expected decision approved, got %v", result["decision"])
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

	loop.ProcessMessage(nil, "sess-test-1", "Write test_approve.txt")

	time.Sleep(200 * time.Millisecond)

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateAwaitingApproval {
		t.Fatalf("expected state AwaitingApproval, got %q", sessGot.State)
	}

	body := map[string]string{"decision": "reject"}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["decision"] != "rejected" {
		t.Errorf("expected decision rejected, got %v", result["decision"])
	}
}

func TestApprove_InvalidDecision(t *testing.T) {
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

	body := map[string]string{"decision": "maybe"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestApprove_TimeoutReturns409(t *testing.T) {
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

	loop.ProcessMessage(nil, "sess-test-1", "Write test_approve.txt")

	time.Sleep(200 * time.Millisecond)

	sessGot, _ := store.Get("sess-test-1")
	if sessGot.State != session.StateAwaitingApproval {
		t.Fatalf("expected state AwaitingApproval, got %q", sessGot.State)
	}

	sessGot.ApprovalTimeoutSeconds = 0
	store.Update(sessGot)

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestApprove_SessionNotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	body := map[string]string{"decision": "approve"}
	jsonBody, _ := json.Marshal(body)

	resp, _ := http.Post(server.URL+"/api/v1/sessions/nonexistent/approve", "application/json", bytes.NewReader(jsonBody))
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

	resp, _ := http.Post(server.URL+"/api/v1/sessions/sess-test-1/approve", "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}
