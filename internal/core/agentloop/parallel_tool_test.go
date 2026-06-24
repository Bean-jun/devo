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

type parallelToolMockClient struct {
	callCount int
}

func (m *parallelToolMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "All tasks completed successfully",
		}, nil
	}

	return &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{
				ID:       "call-1",
				ToolName: "list_files",
				Params:   map[string]interface{}{"path": "."},
			},
			{
				ID:       "call-2",
				ToolName: "read_file",
				Params:   map[string]interface{}{"path": "test.txt"},
			},
			{
				ID:       "call-3",
				ToolName: "list_files",
				Params:   map[string]interface{}{"path": "subdir"},
			},
		},
	}, nil
}

func (m *parallelToolMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}

func TestParallelToolExecution_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("parallel test"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, &parallelToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-parallel")
	sess, _ := store.Get("sess-parallel")
	sess.WorkingDirectory = tmpDir
	sess.MaxConcurrentToolCalls = 3
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-parallel")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-parallel", "Do parallel tasks"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	toolCallCount := 0
	toolResultCount := 0
	toolProgressCount := 0
	for _, evt := range events {
		switch evt.Type {
		case "tool_call_request":
			toolCallCount++
		case "tool_result":
			toolResultCount++
		case "tool_progress":
			toolProgressCount++
		}
	}

	if toolCallCount != 3 {
		t.Errorf("expected 3 tool_call_request events, got %d", toolCallCount)
	}
	if toolResultCount != 3 {
		t.Errorf("expected 3 tool_result events, got %d", toolResultCount)
	}
	if toolProgressCount < 3 {
		t.Errorf("expected at least 3 tool_progress events, got %d", toolProgressCount)
	}

	msgs, _, _ := store.GetMessages("sess-parallel", 0, 0)
	hasToolCall := false
	toolResultMessages := 0
	for _, m := range msgs {
		if len(m.ToolCalls) > 0 {
			hasToolCall = true
		}
		if m.Role == session.RoleTool {
			toolResultMessages++
		}
	}
	if !hasToolCall {
		t.Error("expected a message with tool calls")
	}
	if toolResultMessages != 3 {
		t.Errorf("expected 3 tool result messages, got %d", toolResultMessages)
	}
}

func TestParallelToolExecution_SerialFallback(t *testing.T) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, &parallelToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-serial")
	sess, _ := store.Get("sess-serial")
	sess.WorkingDirectory = t.TempDir()
	sess.MaxConcurrentToolCalls = 1
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-serial")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-serial", "Do tasks"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	toolCallCount := 0
	toolResultCount := 0
	for _, evt := range events {
		switch evt.Type {
		case "tool_call_request":
			toolCallCount++
		case "tool_result":
			toolResultCount++
		}
	}

	if toolCallCount != 3 {
		t.Errorf("expected 3 tool_call_request events, got %d", toolCallCount)
	}
	if toolResultCount != 3 {
		t.Errorf("expected 3 tool_result events, got %d", toolResultCount)
	}
}

type approvalParallelMockClient struct {
	callCount int
}

func (m *approvalParallelMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++
	return &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{
				ID:       "call-1",
				ToolName: "list_files",
				Params:   map[string]interface{}{"path": "."},
			},
			{
				ID:       "call-2",
				ToolName: "write_file",
				Params:   map[string]interface{}{"path": "new.txt", "content": "hello"},
			},
		},
	}, nil
}

func (m *approvalParallelMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}

func TestParallelToolExecution_WithApproval(t *testing.T) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, &approvalParallelMockClient{}, toolRegistry)

	createTestSession(store, "sess-approval-parallel")
	sess, _ := store.Get("sess-approval-parallel")
	sess.WorkingDirectory = t.TempDir()
	sess.MaxConcurrentToolCalls = 2
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-approval-parallel")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-approval-parallel", "Write a file"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	approvalFound := false
	for {
		evt, ok := waitForEvent(ch, "", 3*time.Second)
		if !ok {
			break
		}
		if evt.Type == "approval_required" {
			approvalFound = true
			data, _ := evt.Data.(map[string]any)
			approvalID, _ := data["approval_id"].(string)
			if approvalID != "" {
				loop.ResolveApproval("sess-approval-parallel", approvalID, "reject")
			}
			break
		}
		if evt.Type == "message_complete" {
			break
		}
	}

	if !approvalFound {
		t.Error("expected approval_required event when mixed tool calls include write_file")
	}
}

func TestParallelToolExecution_ToolCallLimit(t *testing.T) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, &parallelToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-parallel-limit")
	sess, _ := store.Get("sess-parallel-limit")
	sess.WorkingDirectory = t.TempDir()
	sess.MaxConcurrentToolCalls = 3
	sess.ToolCallLimit = 2
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-parallel-limit")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-parallel-limit", "Do tasks"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	toolLimitFound := false
	for _, evt := range events {
		if evt.Type == "session_state_change" {
			data, _ := evt.Data.(map[string]any)
			if reason, ok := data["reason"].(string); ok && reason == "tool_limit_reached" {
				toolLimitFound = true
			}
		}
	}

	if !toolLimitFound {
		t.Error("expected tool_limit_reached session state change")
	}
}

func TestParallelToolExecution_FileChange(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)

	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, &parallelToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-file-change")
	sess, _ := store.Get("sess-file-change")
	sess.WorkingDirectory = tmpDir
	sess.MaxConcurrentToolCalls = 3
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-file-change")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-file-change", "Do tasks"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	drainEvents(ch, 5*time.Second)
}

func TestParallelToolExecution_SingleTool(t *testing.T) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := NewWithTools(store, &parallelToolMockClient{}, toolRegistry)

	createTestSession(store, "sess-single-parallel")
	sess, _ := store.Get("sess-single-parallel")
	sess.WorkingDirectory = t.TempDir()
	sess.MaxConcurrentToolCalls = 5
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-single-parallel")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-single-parallel", "Do tasks"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	events := drainEvents(ch, 5*time.Second)

	toolCallCount := 0
	toolResultCount := 0
	for _, evt := range events {
		switch evt.Type {
		case "tool_call_request":
			toolCallCount++
		case "tool_result":
			toolResultCount++
		}
	}

	if toolCallCount != 3 {
		t.Errorf("expected 3 tool_call_request events, got %d", toolCallCount)
	}
	if toolResultCount != 3 {
		t.Errorf("expected 3 tool_result events, got %d", toolResultCount)
	}
}
