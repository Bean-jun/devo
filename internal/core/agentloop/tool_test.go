package agentloop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestAgentLoopWithToolCalling(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello from tool"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, &toolCallingMockClient{}, toolRegistry)

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

func TestAgentLoopWithMultipleToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "nested"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "nested", "file.txt"), []byte("nested content"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := NewWithTools(store, &multiToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Explore the project"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	for i := 0; i < 10; i++ {
		_, ok := waitForEvent(ch, "", 500*time.Millisecond)
		if !ok {
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
	loop := New(store, &toolCallingMockClient{})

	createTestSession(store, "sess-1")
	sess, _ := store.Get("sess-1")
	sess.WorkingDirectory = t.TempDir()
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Read the file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, ok := waitForEvent(ch, "tool_result", 3*time.Second)
	if !ok {
		t.Fatal("expected tool_result event even without executor")
	}
}
