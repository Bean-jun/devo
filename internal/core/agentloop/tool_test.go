package agentloop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devo/internal/config"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

type toolCallingMockClient struct {
	callCount int
}

func (m *toolCallingMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
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
						ToolName: "read_file",
						Params: map[string]interface{}{
							"path": "test.txt",
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

func (m *toolCallingMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}
func (m *toolCallingMockClient) TestConnection(ctx context.Context) error { return nil }

func TestAgentLoopWithToolCalling(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello from tool"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := newTestLoopWithTools(t, store, &toolCallingMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Read the file"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	eventTypes := []string{"thinking", "tool_call_request", "tool_result", "message_complete", "session_state_change"}
	for _, expectedType := range eventTypes {
		evt, ok := waitForEvent(ch, expectedType, 3*time.Second)
		if !ok {
			t.Fatalf("timed out waiting for event: %s", expectedType)
		}
		t.Logf("Received event: %s", evt.Type)
	}

	msgs, total, _ := store.GetMessages("sess-1", 0, 0)
	t.Logf("Total messages: %d", total)
	for i, m := range msgs {
		t.Logf("  msg[%d]: role=%s content=%s tool_calls=%d", i, m.Role, m.Content, len(m.ToolCalls))
	}

	hasToolCall := false
	hasToolResult := false
	for _, m := range msgs {
		if len(m.ToolCalls) > 0 {
			hasToolCall = true
		}
		if m.Role == session.RoleTool {
			hasToolResult = true
		}
	}

	if !hasToolCall {
		t.Error("expected a message with tool calls")
	}
	if !hasToolResult {
		t.Error("expected a tool result message")
	}
}

type multiToolMockClient struct {
	callCount int
}

func (m *multiToolMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		if m.callCount == 2 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-2",
						ToolName: "read_file",
						Params: map[string]interface{}{
							"path": "nested/file.txt",
						},
					},
				},
			}, nil
		}

		return &llmclient.CompleteResult{
			Text: "Final result after all tool calls",
		}, nil
	}

	if lastMsg.Role == session.RoleUser {
		return &llmclient.CompleteResult{
			ToolCalls: []session.ToolCall{
				{
					ID:       "call-1",
					ToolName: "list_files",
					Params:   map[string]interface{}{},
				},
			},
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
}

func (m *multiToolMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}
func (m *multiToolMockClient) TestConnection(ctx context.Context) error { return nil }

func TestAgentLoopWithMultipleToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "nested"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "nested", "file.txt"), []byte("nested content"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := newTestLoopWithTools(t, store, &multiToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Explore the project"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	for i := 0; i < 50; i++ {
		evt, ok := waitForEvent(ch, "", 500*time.Millisecond)
		if !ok {
			break
		}
		if evt.Type == "loop.loop_completed" {
			break
		}
	}

	msgs, total, _ := store.GetMessages("sess-1", 0, 0)
	if total < 4 {
		t.Fatalf("expected at least 4 messages (user+assistant+tool+assistant+tool+assistant), got %d", total)
	}

	toolCallCount := 0
	toolResultCount := 0
	for _, m := range msgs {
		if len(m.ToolCalls) > 0 {
			toolCallCount++
		}
		if m.Role == session.RoleTool {
			toolResultCount++
		}
	}

	if toolCallCount < 2 {
		t.Errorf("expected at least 2 tool call messages, got %d", toolCallCount)
	}
	if toolResultCount < 2 {
		t.Errorf("expected at least 2 tool result messages, got %d", toolResultCount)
	}
}

func TestAgentLoopWithoutToolExecutor(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := NewWithTools(store, &toolCallingMockClient{}, nil)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = t.TempDir()
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Read the file"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, ok := waitForEvent(ch, "tool_result", 3*time.Second)
	if !ok {
		t.Fatal("expected tool_result event even without executor")
	}
}

type limitedToolMockClient struct {
	callCount int
	maxCalls  int
}

func (m *limitedToolMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		if m.callCount >= m.maxCalls {
			return &llmclient.CompleteResult{
				Text: "I have completed all the tool calls.",
			}, nil
		}
		return &llmclient.CompleteResult{
			ToolCalls: []session.ToolCall{
				{
					ID:       fmt.Sprintf("call-%d", m.callCount),
					ToolName: "read_file",
					Params: map[string]interface{}{
						"path": "test.txt",
					},
				},
			},
		}, nil
	}

	if lastMsg.Role == session.RoleUser || lastMsg.Role == session.RoleSystem {
		if m.callCount >= m.maxCalls {
			return &llmclient.CompleteResult{
				Text: "No more tool calls needed.",
			}, nil
		}
		return &llmclient.CompleteResult{
			ToolCalls: []session.ToolCall{
				{
					ID:       fmt.Sprintf("call-%d", m.callCount),
					ToolName: "read_file",
					Params: map[string]interface{}{
						"path": "test.txt",
					},
				},
			},
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
}

func (m *limitedToolMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}
func (m *limitedToolMockClient) TestConnection(ctx context.Context) error { return nil }

func TestToolCallLimitReached(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test content"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := newTestLoopWithConfig(t, store, &limitedToolMockClient{maxCalls: 100}, toolRegistry, &config.Config{ToolCallLimit: 3})

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Do many operations"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	stateChangeCount := 0
	for {
		evt, ok := waitForEvent(ch, "", 3*time.Second)
		if !ok {
			break
		}
		if evt.Type == "session_state_change" {
			stateChangeCount++
			data, _ := evt.Data.(map[string]any)
			t.Logf("State change: %v", data)
			if data["reason"] == "tool_limit_reached" {
				t.Log("Got tool_limit_reached event")
				break
			}
		}
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle after tool limit reached, got %q", sess.State)
	}
	if sess.LastLoopTerminationReason != session.LoopTerminationToolLimitReached {
		t.Errorf("expected termination reason tool_limit_reached, got %q", sess.LastLoopTerminationReason)
	}
	if sess.ToolCallCount < 3 {
		t.Errorf("expected tool_call_count >= 3, got %d", sess.ToolCallCount)
	}

	msgs, total, _ := store.GetMessages("sess-1", 0, 0)
	t.Logf("Total messages: %d", total)
	hasLimitMessage := false
	for _, m := range msgs {
		if m.Role == session.RoleSystem && strings.Contains(m.Content, "达到上限暂停") {
			hasLimitMessage = true
		}
	}
	if !hasLimitMessage {
		t.Error("expected system message about tool call limit reached")
	}
}

func TestContinuationAfterToolLimit(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("continuation test"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := newTestLoopWithTools(t, store, &limitedToolMockClient{maxCalls: 100}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	loop.cfg.ToolCallLimit = 2
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Do operations"}); err != nil {
		t.Fatalf("first message: %v", err)
	}

	for {
		evt, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
		if !ok {
			break
		}
		data, _ := evt.Data.(map[string]any)
		if data["reason"] == "tool_limit_reached" {
			break
		}
	}

	sess, _ = store.Get("sess-1")
	if sess.LastLoopTerminationReason != session.LoopTerminationToolLimitReached {
		t.Fatal("expected tool_limit_reached termination reason")
	}

	msgsBeforeContinue, _, _ := store.GetMessages("sess-1", 0, 0)
	msgCountBefore := len(msgsBeforeContinue)

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "继续"}); err != nil {
		t.Fatalf("continuation message: %v", err)
	}

	msgsAfterContinue, _, _ := store.GetMessages("sess-1", 0, 0)
	msgCountAfter := len(msgsAfterContinue)

	hasContinuationMsg := false
	for i := msgCountBefore; i < msgCountAfter; i++ {
		m := msgsAfterContinue[i]
		t.Logf("New message %d: role=%s content=%s", i, m.Role, m.Content)
		if m.Role == session.RoleSystem && strings.Contains(m.Content, "从中断点继续") {
			hasContinuationMsg = true
		}
	}
	if !hasContinuationMsg {
		t.Error("expected continuation system message")
	}

	sess, _ = store.Get("sess-1")
	if sess.ToolCallCount != 0 {
		t.Errorf("expected tool_call_count reset to 0, got %d", sess.ToolCallCount)
	}
	if sess.LastLoopTerminationReason != "" {
		t.Errorf("expected termination reason cleared, got %q", sess.LastLoopTerminationReason)
	}

	for {
		evt, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
		if !ok {
			break
		}
		data, _ := evt.Data.(map[string]any)
		t.Logf("Continuation state change: %v", data)
		if data["reason"] == "tool_limit_reached" {
			break
		}
	}
}

func TestToolCallCountResetsOnNewLoop(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := newTestLoopWithTools(t, store, &limitedToolMockClient{maxCalls: 2}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	loop.cfg.ToolCallLimit = 1
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Do one thing"}); err != nil {
		t.Fatalf("first message: %v", err)
	}

	for {
		evt, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
		if !ok {
			break
		}
		data, _ := evt.Data.(map[string]any)
		if data["reason"] == "tool_limit_reached" {
			break
		}
	}

	sess, _ = store.Get("sess-1")
	t.Logf("After first loop: tool_call_count=%d, termination_reason=%s", sess.ToolCallCount, sess.LastLoopTerminationReason)

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "继续"}); err != nil {
		t.Fatalf("continuation: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.ToolCallCount != 0 {
		t.Errorf("expected tool_call_count reset to 0 on new loop, got %d", sess.ToolCallCount)
	}
}

func TestContinuationWithNewTask(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test content"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := newTestLoopWithTools(t, store, &limitedToolMockClient{maxCalls: 100}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	loop.cfg.ToolCallLimit = 2
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "First task"}); err != nil {
		t.Fatalf("first message: %v", err)
	}

	for {
		evt, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
		if !ok {
			break
		}
		data, _ := evt.Data.(map[string]any)
		if data["reason"] == "tool_limit_reached" {
			break
		}
	}

	msgsBeforeContinue, _, _ := store.GetMessages("sess-1", 0, 0)
	msgCountBefore := len(msgsBeforeContinue)

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Do a completely different task now"}); err != nil {
		t.Fatalf("new task message: %v", err)
	}

	for {
		evt, ok := waitForEvent(ch, "session_state_change", 3*time.Second)
		if !ok {
			break
		}
		data, _ := evt.Data.(map[string]any)
		if data["new_state"] == "Idle" {
			break
		}
	}

	msgsAfterContinue, _, _ := store.GetMessages("sess-1", 0, 0)
	msgCountAfter := len(msgsAfterContinue)

	hasContinuationMsg := false
	hasNewTaskMsg := false
	for i := msgCountBefore; i < msgCountAfter; i++ {
		m := msgsAfterContinue[i]
		if m.Role == session.RoleSystem && strings.Contains(m.Content, "从中断点继续") {
			hasContinuationMsg = true
		}
		if m.Role == session.RoleUser && m.Content == "Do a completely different task now" {
			hasNewTaskMsg = true
		}
	}
	if !hasContinuationMsg {
		t.Error("expected continuation context message even with new task")
	}
	if !hasNewTaskMsg {
		t.Error("expected new task user message")
	}
}
