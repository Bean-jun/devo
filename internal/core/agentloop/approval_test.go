package agentloop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

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

	if lastMsg.Role == session.RoleUser {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "write_file",
						Params: map[string]interface{}{
							"path":    "newfile.txt",
							"content": "Hello from approval test",
						},
					},
				},
			}, nil
		}

		return &llmclient.CompleteResult{
			Text: "Echo: " + lastMsg.Content,
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
}

func TestAgentLoopWithApproval_Approve(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &approvalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Create a new file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required event")
	}

	data, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_required data to be a map")
	}

	approvalID, ok := data["approval_id"].(string)
	if !ok || approvalID == "" {
		t.Fatal("expected approval_id in approval_required event")
	}

	t.Logf("received approval_required event with approval_id=%s operation_type=%s", approvalID, data["operation_type"])

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateAwaitingApproval {
		t.Errorf("expected session state AwaitingApproval, got %s", sess.State)
	}

	if err := loop.ResolveApproval("sess-1", approvalID, "approve"); err != nil {
		t.Fatalf("failed to resolve approval: %v", err)
	}

	evt, ok = waitForEvent(ch, "approval_resolved", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_resolved event")
	}

	_, ok = waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected final state Idle, got %s", sess.State)
	}

	fileData, err := os.ReadFile(filepath.Join(tmpDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("file should have been created: %v", err)
	}
	if string(fileData) != "Hello from approval test" {
		t.Errorf("expected file content 'Hello from approval test', got %q", string(fileData))
	}
}

type rejectionMockClient struct {
	callCount int
}

func (m *rejectionMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "The tool was rejected by the user: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "write_file",
						Params: map[string]interface{}{
							"path":    "should_not_exist.txt",
							"content": "This should not be written",
						},
					},
				},
			}, nil
		}

		return &llmclient.CompleteResult{
			Text: "Echo: " + lastMsg.Content,
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
}

func TestAgentLoopWithApproval_Reject(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &rejectionMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Create a file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required event")
	}

	data, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_required data to be a map")
	}

	approvalID, ok := data["approval_id"].(string)
	if !ok || approvalID == "" {
		t.Fatal("expected approval_id in approval_required event")
	}

	if err := loop.ResolveApproval("sess-1", approvalID, "reject"); err != nil {
		t.Fatalf("failed to resolve approval: %v", err)
	}

	_, ok = waitForEvent(ch, "approval_resolved", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_resolved event")
	}

	_, ok = waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	_, err := os.Stat(filepath.Join(tmpDir, "should_not_exist.txt"))
	if err == nil {
		t.Fatal("file should NOT have been created after rejection")
	}

	msgs, total, _ := store.GetMessages("sess-1", 0, 0)
	t.Logf("Total messages after rejection: %d", total)
	hasRejection := false
	for _, m := range msgs {
		t.Logf("  msg: role=%s content=%s", m.Role, m.Content)
		if m.Role == session.RoleTool && strings.Contains(m.Content, "拒绝") {
			hasRejection = true
		}
	}
	if !hasRejection {
		t.Error("expected a tool message indicating rejection")
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected final state Idle, got %s", sess.State)
	}
}

type readOnlyMockClient struct {
	callCount int
}

func (m *readOnlyMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "I read the file: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "read_file",
						Params: map[string]interface{}{
							"path": "test.txt",
						},
					},
				},
			}, nil
		}

		return &llmclient.CompleteResult{
			Text: "Done.",
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
}

func TestAgentLoop_ReadOnlyToolsNoApproval(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("read-only content"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, &readOnlyMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Read the file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected final state Idle, got %s", sess.State)
	}

	msgs, _, _ := store.GetMessages("sess-1", 0, 0)
	hasToolResult := false
	for _, m := range msgs {
		if m.Role == session.RoleTool && strings.Contains(m.Content, "read-only content") {
			hasToolResult = true
		}
	}
	if !hasToolResult {
		t.Error("expected tool result message with file content")
	}
}

func TestResolveApproval_InvalidDecision(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	err := loop.ResolveApproval("sess-1", "approval-1", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid decision")
	}
}

func TestResolveApproval_NotFound(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	err := loop.ResolveApproval("sess-1", "nonexistent", "approve")
	if err == nil {
		t.Fatal("expected error for nonexistent approval")
	}
}

func TestApprovalRequiredEventFields(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &approvalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	loop.ProcessMessage(context.Background(), "sess-1", "Create a file")

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required event")
	}

	data, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_required data to be a map")
	}

	requiredFields := []string{"approval_id", "operation_type", "risk_level", "details"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("approval_required event missing field: %s", field)
		}
	}

	if data["risk_level"] != "medium" {
		t.Errorf("expected risk_level 'medium', got %v", data["risk_level"])
	}
}

type policyAutoApprovalMockClient struct {
	callCount int
}

func (m *policyAutoApprovalMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
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
							"path":    "auto_approved.txt",
							"content": "This file was auto-approved",
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

func TestAgentLoop_PolicySessionTrust_AutoApprove(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &policyAutoApprovalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	sess.ApprovalPolicy = map[string]string{
		"file_write_new": "session_trust",
	}
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Create a file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "approval_auto", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_auto event")
	}

	data, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_auto data to be a map")
	}
	if data["operation_type"] != "file_write_new" {
		t.Errorf("expected operation_type file_write_new, got %v", data["operation_type"])
	}

	_, ok = waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	fileData, err := os.ReadFile(filepath.Join(tmpDir, "auto_approved.txt"))
	if err != nil {
		t.Fatalf("file should have been created: %v", err)
	}
	if string(fileData) != "This file was auto-approved" {
		t.Errorf("expected file content 'This file was auto-approved', got %q", string(fileData))
	}

	msgs, _, _ := store.GetMessages("sess-1", 0, 0)
	hasSystemNote := false
	for _, m := range msgs {
		if m.Role == session.RoleSystem && strings.Contains(m.Content, "session_trust") {
			hasSystemNote = true
		}
	}
	if !hasSystemNote {
		t.Error("expected a system message noting auto-approval via session_trust")
	}
}

func TestAgentLoop_PolicyAlwaysAsk_StillRequiresApproval(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &approvalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	sess.ApprovalPolicy = map[string]string{
		"file_write_new": "always_ask",
	}
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Create a file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required event (always_ask policy should still require approval)")
	}

	data, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_required data to be a map")
	}

	approvalID, ok := data["approval_id"].(string)
	if !ok || approvalID == "" {
		t.Fatal("expected approval_id in approval_required event")
	}

	if err := loop.ResolveApproval("sess-1", approvalID, "approve"); err != nil {
		t.Fatalf("failed to resolve approval: %v", err)
	}

	_, ok = waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	_, err := os.Stat(filepath.Join(tmpDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("file should have been created after approval: %v", err)
	}
}

func TestApprovalTimeout_AutoReject(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &rejectionMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	sess.ApprovalTimeoutSeconds = 1
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Create a file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_required event")
	}

	data, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_required data to be a map")
	}
	approvalID, ok := data["approval_id"].(string)
	if !ok || approvalID == "" {
		t.Fatal("expected approval_id in approval_required event")
	}

	evt, ok = waitForEvent(ch, "approval_resolved", 5*time.Second)
	if !ok {
		t.Fatal("timed out waiting for approval_resolved event (timeout should trigger)")
	}

	resolvedData, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected approval_resolved data to be a map")
	}
	if resolvedData["decision"] != "reject" {
		t.Errorf("expected decision 'reject' on timeout, got %v", resolvedData["decision"])
	}
	if resolvedData["source"] != "timeout" {
		t.Errorf("expected source 'timeout', got %v", resolvedData["source"])
	}

	_, ok = waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	_, err := os.Stat(filepath.Join(tmpDir, "should_not_exist.txt"))
	if err == nil {
		t.Fatal("file should NOT have been created after timeout rejection")
	}

	err = loop.ResolveApproval("sess-1", approvalID, "approve")
	if err == nil {
		t.Fatal("expected error when approving an already-resolved (timed-out) approval")
	}
}

func TestResolveApproval_Expired(t *testing.T) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &approvalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.ApprovalTimeoutSeconds = 1
	store.Update(sess)

	req := loop.approvalManager.CreateRequest("sess-1", "tool-1", "file_write_new", "medium", map[string]any{"path": "test.txt"})
	pastTime := time.Now().Add(-1 * time.Second)
	loop.approvalManager.SetTimeout(req.ID, pastTime)

	err := loop.ResolveApproval("sess-1", req.ID, "approve")
	if err == nil {
		t.Fatal("expected error for expired approval")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' in error message, got: %v", err)
	}
}
