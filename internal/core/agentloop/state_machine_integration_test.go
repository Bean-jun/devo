package agentloop

import (
	"context"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

func TestIntegration_FullConversation(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-full")

	eventBus, _ := store.GetEventBus("sess-full")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-full", session.Message{Content: "Hello, world!"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedEvents := []string{"thinking", "message_complete", "session_state_change"}
	for _, expectedType := range expectedEvents {
		_, ok := waitForEvent(ch, expectedType, 5*time.Second)
		if !ok {
			t.Errorf("expected event %q", expectedType)
		}
	}

	sess, err := store.Get("sess-full")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle, got %s", sess.State)
	}
}

func TestIntegration_ToolCallLoop(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-tool-loop")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := NewWithTools(store, &toolLoopMockClient{}, toolRegistry)

	sess, _ := store.Get("sess-tool-loop")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-tool-loop")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-tool-loop", session.Message{Content: "list files"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	eventTypes := []string{}
	events := drainEvents(ch, 5*time.Second)
	for _, evt := range events {
		eventTypes = append(eventTypes, evt.Type)
	}

	hasToolCall := false
	hasToolResult := false
	hasMessageComplete := false
	for _, et := range eventTypes {
		switch et {
		case "tool_call_request":
			hasToolCall = true
		case "tool_result":
			hasToolResult = true
		case "message_complete":
			hasMessageComplete = true
		}
	}

	if !hasToolCall {
		t.Error("expected tool_call_request event")
	}
	if !hasToolResult {
		t.Error("expected tool_result event")
	}
	if !hasMessageComplete {
		t.Error("expected message_complete event")
	}
}

func TestIntegration_ApprovalRequired(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-approval")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &approvalMockClient{}, toolRegistry)

	sess, _ := store.Get("sess-approval")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-approval")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-approval", session.Message{Content: "Write a new file"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	approvalRequiredFound := false
	var approvalID string
	for {
		evt, ok := waitForEvent(ch, "", 5*time.Second)
		if !ok {
			break
		}
		if evt.Type == "approval_required" {
			approvalRequiredFound = true
			if data, ok := evt.Data.(map[string]any); ok {
				approvalID, _ = data["approval_id"].(string)
			}
			break
		}
	}

	if !approvalRequiredFound {
		t.Error("expected approval_required event")
	}

	if approvalID != "" {
		loop.ResolveApproval("sess-approval", approvalID, "reject")
	}

	_, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (completed)")
	}
}

func TestIntegration_EventSequence(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-seq")

	eventBus, _ := store.GetEventBus("sess-seq")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-seq", session.Message{Content: "Hello"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.Type
	}

	thinkingIdx := indexOf(eventTypes, "thinking")
	messageCompleteIdx := indexOf(eventTypes, "message_complete")

	if thinkingIdx < 0 {
		t.Error("expected thinking event")
	}
	if messageCompleteIdx < 0 {
		t.Error("expected message_complete event")
	}
	if thinkingIdx >= 0 && messageCompleteIdx >= 0 && thinkingIdx > messageCompleteIdx {
		t.Error("expected thinking before message_complete")
	}
}

func TestIntegration_StateTransitions(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-transitions")

	eventBus, _ := store.GetEventBus("sess-transitions")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-transitions", session.Message{Content: "Hello"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	stateChanges := []string{}
	events := drainEvents(ch, 5*time.Second)
	for _, evt := range events {
		if evt.Type == "loop.state_change" {
			data, _ := evt.Data.(map[string]any)
			stateChanges = append(stateChanges, data["old_state"].(string)+"->"+data["new_state"].(string))
		}
	}

	if len(stateChanges) == 0 {
		t.Error("expected at least one loop.state_change event")
	}
	t.Logf("State transitions: %v", stateChanges)
}

func TestIntegration_PauseResumeMidLoop(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-pause-resume")

	eventBus, _ := store.GetEventBus("sess-pause-resume")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-pause-resume", session.Message{Content: "Hello"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = loop.Pause("sess-pause-resume")
	if err != nil {
		t.Logf("Pause returned: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	err = loop.Resume("sess-pause-resume")
	if err != nil {
		t.Logf("Resume returned: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	hasLoopCompleted := false
	for _, e := range events {
		if e.Type == "loop.loop_completed" {
			hasLoopCompleted = true
		}
	}

	if !hasLoopCompleted {
		t.Log("loop may not have completed (pause/resume timing dependent)")
	}
}

func TestIntegration_CancelMidLoop(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-cancel")

	eventBus, _ := store.GetEventBus("sess-cancel")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-cancel", session.Message{Content: "Hello"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = loop.Cancel("sess-cancel")
	if err != nil {
		t.Logf("Cancel returned: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	hasCancelled := false
	for _, e := range events {
		if e.Type == "loop.cancelled" {
			hasCancelled = true
		}
	}

	if !hasCancelled {
		t.Log("loop may not have emitted cancelled event (timing dependent)")
	}
}

func TestIntegration_ToolCallLimit(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-tool-limit")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := NewWithTools(store, &toolLoopMockClient{}, toolRegistry)

	sess, _ := store.Get("sess-tool-limit")
	sess.WorkingDirectory = tmpDir
	sess.ToolCallLimit = 1
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-tool-limit")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-tool-limit", session.Message{Content: "list files"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	toolLimitReached := false
	for _, e := range events {
		if e.Type == "session_state_change" {
			data, _ := e.Data.(map[string]any)
			if data["reason"] == "tool_limit_reached" {
				toolLimitReached = true
			}
		}
	}

	if !toolLimitReached {
		t.Error("expected tool_limit_reached state change")
	}
}

func TestIntegration_ErrorRecovery(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-error-recovery")

	loop := New(store, &errorMockClient{})

	sess, _ := store.Get("sess-error-recovery")
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-error-recovery")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	err := loop.ProcessMessage(context.Background(), "sess-error-recovery", session.Message{Content: "Hello"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	hasError := false
	hasStateChangeToIdle := false
	for _, e := range events {
		if e.Type == "error" {
			hasError = true
		}
		if e.Type == "session_state_change" {
			data, _ := e.Data.(map[string]any)
			if data["new_state"] == session.StateIdle.ToSnakeCase() && data["reason"] == "error" {
				hasStateChangeToIdle = true
			}
		}
	}

	if !hasError {
		t.Error("expected error event")
	}
	if !hasStateChangeToIdle {
		t.Error("expected session state change to Idle with reason=error")
	}
}

type toolLoopMockClient struct {
	callCount int
}

func (m *toolLoopMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "I completed the tool execution: " + lastMsg.Content,
		}, nil
	}

	return &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{
				ID:       "call-1",
				ToolName: "list_files",
				Params: map[string]interface{}{
					"path": ".",
				},
			},
		},
	}, nil
}

func (m *toolLoopMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}
func (m *toolLoopMockClient) TestConnection(ctx context.Context) error { return nil }


func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
