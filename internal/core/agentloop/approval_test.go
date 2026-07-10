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

func (m *approvalMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
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

func (m *rejectionMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
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

func (m *readOnlyMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
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

	if approvalID, ok := data["approval_id"].(string); ok && approvalID != "" {
		loop.ResolveApproval("sess-1", approvalID, "reject")
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

func (m *policyAutoApprovalMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
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

type overwriteApprovalMockClient struct {
	callCount int
}

func (m *overwriteApprovalMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
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
							"path":    "existing.txt",
							"content": "new line1\nnew line2\n",
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

func (m *overwriteApprovalMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}

func TestApprovalRequest_WriteFileDiff(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("old line1\nold line2\n"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &overwriteApprovalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Overwrite the file"); err != nil {
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

	details, ok := data["details"].(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if details["path"] != "existing.txt" {
		t.Errorf("expected path 'existing.txt', got %v", details["path"])
	}

	diff, ok := details["diff"].(string)
	if !ok || diff == "" {
		t.Fatal("expected diff in details for overwrite operation")
	}

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}
	if !strings.Contains(diff, "old line") {
		t.Error("diff should mention old content")
	}
	if !strings.Contains(diff, "new line") {
		t.Error("diff should mention new content")
	}

	t.Logf("Diff:\n%s", diff)

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

	time.Sleep(50 * time.Millisecond)
}

type editApprovalMockClient struct {
	callCount int
}

func (m *editApprovalMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		if strings.Contains(lastMsg.Content, "错误") {
			return &llmclient.CompleteResult{
				Text: "The edit failed: " + lastMsg.Content,
			}, nil
		}
		return &llmclient.CompleteResult{
			Text: "Edit completed: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "edit_file",
						Params: map[string]interface{}{
							"path":    "app.go",
							"mode":    "replace",
							"old_str": "old_func",
							"new_str": "new_func",
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

func (m *editApprovalMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}

func TestApprovalRequest_EditFileDiff(t *testing.T) {
	tmpDir := t.TempDir()
	originalContent := "package main\n\nfunc old_func() {\n\treturn\n}\n"
	os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte(originalContent), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.EditFileTool{})

	loop := NewWithTools(store, &editApprovalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Edit the file"); err != nil {
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

	details, ok := data["details"].(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if details["path"] != "app.go" {
		t.Errorf("expected path 'app.go', got %v", details["path"])
	}
	if details["mode"] != "replace" {
		t.Errorf("expected mode 'replace', got %v", details["mode"])
	}

	diff, ok := details["diff"].(string)
	if !ok || diff == "" {
		t.Fatal("expected diff in details for edit_file operation")
	}

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}
	if !strings.Contains(diff, "old_func") {
		t.Error("diff should mention old_func")
	}
	if !strings.Contains(diff, "new_func") {
		t.Error("diff should mention new_func")
	}

	t.Logf("Edit Diff:\n%s", diff)

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

	fileData, err := os.ReadFile(filepath.Join(tmpDir, "app.go"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileData), "new_func") {
		t.Error("file should contain new_func after edit")
	}
}

type editFailureMockClient struct {
	callCount int
}

func (m *editFailureMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		if strings.Contains(lastMsg.Content, "错误") {
			return &llmclient.CompleteResult{
				Text: "The edit failed, I'll try a different approach.",
			}, nil
		}
		return &llmclient.CompleteResult{
			Text: "Edit completed: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "edit_file",
						Params: map[string]interface{}{
							"path":    "dup.txt",
							"mode":    "replace",
							"old_str": "hello",
							"new_str": "hi",
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

func (m *editFailureMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}

func TestApprovalRequest_EditFileDiffFailureNoApproval(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "dup.txt"), []byte("hello\nhello\n"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.EditFileTool{})

	loop := NewWithTools(store, &editFailureMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Edit the file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var evt session.Event
	var foundApproval, foundToolResult bool
	timeout := time.After(3 * time.Second)
loop:
	for {
		select {
		case <-timeout:
			break loop
		case e, ok := <-ch:
			if !ok {
				break loop
			}
			if e.Type == "approval_required" {
				foundApproval = true
			}
			if e.Type == "tool_result" {
				evt = e
				foundToolResult = true
				break loop
			}
		}
	}

	if foundApproval {
		t.Fatal("should NOT have received approval_required event for failed edit")
	}
	if !foundToolResult {
		t.Fatal("timed out waiting for tool_result event")
	}

	toolResultData, ok := evt.Data.(map[string]any)
	if !ok {
		t.Fatal("expected tool_result data to be a map")
	}
	if toolResultData["success"] != false {
		t.Error("expected tool_result success=false for failed edit")
	}

	_, ok = waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}

	time.Sleep(100 * time.Millisecond)

	data, _ := os.ReadFile(filepath.Join(tmpDir, "dup.txt"))
	if string(data) != "hello\nhello\n" {
		t.Error("file should NOT have been modified")
	}
}

type execCommandApprovalMockClient struct {
	callCount int
}

func (m *execCommandApprovalMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "Command executed: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "exec_python",
						Params: map[string]interface{}{
							"code":            "print('hello')",
							"timeout_seconds": float64(60),
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

func (m *execCommandApprovalMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}

func TestApprovalRequest_ExecPythonContext(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewExecPythonTool(nil))

	loop := NewWithTools(store, &execCommandApprovalMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Run a command"); err != nil {
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

	details, ok := data["details"].(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if details["code"] != "print('hello')" {
		t.Errorf("expected code 'print('hello')', got %v", details["code"])
	}
	if details["timeout_seconds"] != float64(60) {
		t.Errorf("expected timeout_seconds 60, got %v", details["timeout_seconds"])
	}

	cmdCtx, ok := details["command_context"].(map[string]any)
	if !ok {
		t.Fatal("expected command_context in details")
	}

	if cmdCtx["working_directory"] != tmpDir {
		t.Errorf("expected working_directory %s, got %v", tmpDir, cmdCtx["working_directory"])
	}
	if cmdCtx["invocation"] == nil {
		t.Error("expected invocation in command_context")
	}

	if mode, ok := cmdCtx["mode"].(string); !ok || (mode != "sync" && mode != "background") {
		t.Errorf("expected valid mode (sync/background), got %v", cmdCtx["mode"])
	}

	t.Logf("command_context: %+v", cmdCtx)

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
}

func TestAwaitingApprovalHandler_NoPendingToolCall(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-no-pending")
	lc := newTestLoopContext("sess-no-pending", store)

	nextState, err := loop.awaitingApprovalHandler(context.Background(), lc)
	if err == nil {
		t.Fatal("expected error for no pending tool call")
	}
	if nextState != LoopStateError {
		t.Errorf("expected next state Error, got %s", nextState)
	}
}

func TestAwaitingApprovalHandler_CancelWhileWaiting(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-cancel-approval")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, llmclient.NewMockClient(), toolRegistry)
	lc := newTestLoopContext("sess-cancel-approval", store)

	sess, _ := store.Get("sess-cancel-approval")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	tc := session.ToolCall{
		ID:       "call-1",
		ToolName: "write_file",
		Params:   map[string]interface{}{"path": "newfile.txt", "content": "test"},
	}
	lc.PendingToolCall = &tc

	sendCancelSignal(lc)

	nextState, err := loop.awaitingApprovalHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateCancelled {
		t.Errorf("expected next state Cancelled, got %s", nextState)
	}
}

func TestAwaitingApprovalHandler_Approved(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-approve")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, llmclient.NewMockClient(), toolRegistry)
	lc := newTestLoopContext("sess-approve", store)

	sess, _ := store.Get("sess-approve")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	tc := session.ToolCall{
		ID:       "call-1",
		ToolName: "write_file",
		Params:   map[string]interface{}{"path": "newfile.txt", "content": "test"},
	}
	lc.PendingToolCall = &tc

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	go func() {
		evt, ok := waitForEvent(ch, "approval_required", 3*time.Second)
		if !ok {
			return
		}
		data, _ := evt.Data.(map[string]any)
		approvalID, _ := data["approval_id"].(string)
		loop.ResolveApproval("sess-approve", approvalID, "approve")
	}()

	nextState, err := loop.awaitingApprovalHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateToolExecuting {
		t.Errorf("expected next state ToolExecuting, got %s", nextState)
	}
}
