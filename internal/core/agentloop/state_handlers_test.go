package agentloop

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

func TestPreparingHandler_BasicMessages(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-preparing")
	lc := newTestLoopContext("sess-preparing", store)

	store.AddMessage("sess-preparing", session.Message{
		ID:        "msg-1",
		Role:      session.RoleUser,
		Content:   "Hello",
		CreatedAt: time.Now(),
	})

	nextState, err := loop.preparingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateThinking {
		t.Errorf("expected next state Thinking, got %s", nextState)
	}
	if len(lc.ActiveMsgs) == 0 {
		t.Error("expected ActiveMsgs to be populated")
	}
	if lc.DynamicPrompt == "" {
		t.Error("expected DynamicPrompt to be set")
	}
}

func TestPreparingHandler_WithCompression(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-compress")
	lc := newTestLoopContext("sess-compress", store)

	store.AddMessage("sess-compress", session.Message{
		ID:        "msg-1",
		Role:      session.RoleUser,
		Content:   "Hello",
		CreatedAt: time.Now(),
	})

	nextState, err := loop.preparingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateThinking {
		t.Errorf("expected next state Thinking, got %s", nextState)
	}
}

func TestPreparingHandler_GetMessagesError(t *testing.T) {
	loop, store := setupTestLoop()

	lc := newTestLoopContext("nonexistent", store)

	nextState, err := loop.preparingHandler(context.Background(), lc)
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
	if nextState != LoopStateError {
		t.Errorf("expected next state Error, got %s", nextState)
	}
}

func TestThinkingHandler_StreamingTokens(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-think")
	lc := newTestLoopContext("sess-think", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	lc.ActiveMsgs = []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello"},
	}
	lc.DynamicPrompt = "You are a helpful assistant."

	nextState, err := loop.thinkingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateEvaluatingResult {
		t.Errorf("expected next state EvaluatingResult, got %s", nextState)
	}

	streamingCompleteFound := false
	for {
		evt, ok := waitForEvent(ch, "", 1*time.Second)
		if !ok {
			break
		}
		if evt.Type == "streaming_complete" {
			streamingCompleteFound = true
		}
	}
	if !streamingCompleteFound {
		t.Error("expected streaming_complete event")
	}
}

func TestThinkingHandler_CompleteWithToolCalls(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-toolcalls")

	mockClient := &toolCallMockClient{
		toolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "read_file", Params: map[string]interface{}{"path": "test.txt"}},
		},
	}
	loop := New(store, mockClient)
	lc := newTestLoopContext("sess-toolcalls", store)

	lc.ActiveMsgs = []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Read test.txt"},
	}
	lc.DynamicPrompt = "You are a helpful assistant."

	nextState, err := loop.thinkingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateEvaluatingResult {
		t.Errorf("expected next state EvaluatingResult, got %s", nextState)
	}
	if len(lc.LLMResult.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(lc.LLMResult.ToolCalls))
	}
}

func TestThinkingHandler_CompleteWithTextOnly(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-textonly")
	lc := newTestLoopContext("sess-textonly", store)

	lc.ActiveMsgs = []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello"},
	}
	lc.DynamicPrompt = "You are a helpful assistant."

	nextState, err := loop.thinkingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateEvaluatingResult {
		t.Errorf("expected next state EvaluatingResult, got %s", nextState)
	}
	if lc.LLMResult == nil {
		t.Fatal("expected LLMResult to be set")
	}
	if lc.LLMResult.Text == "" {
		t.Error("expected text content")
	}
}

func TestThinkingHandler_StreamError(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-streamerr")

	mockClient := &errorMockClient{}
	loop := New(store, mockClient)
	lc := newTestLoopContext("sess-streamerr", store)

	lc.ActiveMsgs = []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello"},
	}
	lc.DynamicPrompt = "You are a helpful assistant."

	nextState, err := loop.thinkingHandler(context.Background(), lc)
	if err == nil {
		t.Fatal("expected error from stream error")
	}
	if nextState != LoopStateError {
		t.Errorf("expected next state Error, got %s", nextState)
	}
}

func TestEvaluatingResultHandler_WithToolCalls(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-eval-tools")
	lc := newTestLoopContext("sess-eval-tools", store)

	lc.LLMResult = &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "read_file", Params: map[string]interface{}{"path": "test.txt"}},
		},
	}

	nextState, err := loop.evaluatingResultHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateToolExecuting {
		t.Errorf("expected next state ToolExecuting, got %s", nextState)
	}
}

func TestEvaluatingResultHandler_WithTextOnly(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-eval-text")
	lc := newTestLoopContext("sess-eval-text", store)

	lc.LLMResult = &llmclient.CompleteResult{
		Text: "Hello, how can I help?",
	}

	nextState, err := loop.evaluatingResultHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateTextResponse {
		t.Errorf("expected next state TextResponse, got %s", nextState)
	}
}

func TestToolExecutingHandler_SingleTool(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-tool-single")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := NewWithTools(store, llmclient.NewMockClient(), toolRegistry)
	lc := newTestLoopContext("sess-tool-single", store)

	sess, _ := store.Get("sess-tool-single")
	sess.WorkingDirectory = tmpDir
	store.Update(sess)

	lc.LLMResult = &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "list_files", Params: map[string]interface{}{
				"path": tmpDir,
			}},
		},
	}

	nextState, err := loop.toolExecutingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStatePreparing {
		t.Errorf("expected next state Preparing, got %s", nextState)
	}
}

func TestToolExecutingHandler_UnknownTool(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-unknown-tool")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ReadFileTool{})

	loop := NewWithTools(store, llmclient.NewMockClient(), toolRegistry)
	lc := newTestLoopContext("sess-unknown-tool", store)

	lc.LLMResult = &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "nonexistent_tool", Params: map[string]interface{}{}},
		},
	}

	nextState, err := loop.toolExecutingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStatePreparing {
		t.Errorf("expected next state Preparing, got %s", nextState)
	}
}

func TestToolExecutingHandler_NoToolExecutor(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-noexec")
	lc := newTestLoopContext("sess-noexec", store)

	lc.LLMResult = &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "write_file", Params: map[string]interface{}{"path": "test.txt"}},
		},
	}

	nextState, err := loop.toolExecutingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStatePreparing {
		t.Errorf("expected next state Preparing, got %s", nextState)
	}
}

func TestToolExecutingHandler_CancelDuringExecution(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-cancel-tool")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	loop := NewWithTools(store, llmclient.NewMockClient(), toolRegistry)
	lc := newTestLoopContext("sess-cancel-tool", store)

	sendCancelSignal(lc)

	lc.LLMResult = &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "write_file", Params: map[string]interface{}{"path": "test.txt"}},
		},
	}

	nextState, err := loop.toolExecutingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateCancelled {
		t.Errorf("expected next state Cancelled, got %s", nextState)
	}
}

func TestToolExecutingHandler_ToolCallLimitReached(t *testing.T) {
	tmpDir := t.TempDir()

	store := session.NewInMemoryStore()
	createTestSession(store, "sess-limit")

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.ListFilesTool{})

	loop := NewWithTools(store, llmclient.NewMockClient(), toolRegistry)
	lc := newTestLoopContext("sess-limit", store)

	sess, _ := store.Get("sess-limit")
	sess.WorkingDirectory = tmpDir
	sess.ToolCallLimit = 1
	store.Update(sess)

	lc.LLMResult = &llmclient.CompleteResult{
		ToolCalls: []session.ToolCall{
			{ID: "call-1", ToolName: "list_files", Params: map[string]interface{}{
				"path": tmpDir,
			}},
		},
	}

	nextState, err := loop.toolExecutingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateIdle {
		t.Errorf("expected next state Idle (tool limit reached), got %s", nextState)
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

func TestTextResponseHandler_SavesMessage(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-text")
	lc := newTestLoopContext("sess-text", store)

	lc.LLMResult = &llmclient.CompleteResult{
		Text: "Hello, how can I help you?",
	}

	nextState, err := loop.textResponseHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateIdle {
		t.Errorf("expected next state Idle, got %s", nextState)
	}

	msgs, _, err := store.GetMessages("sess-text", 0, 0)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	found := false
	for _, msg := range msgs {
		if msg.Content == "Hello, how can I help you?" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected assistant message to be saved")
	}
}

func TestTextResponseHandler_PublishesEvents(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-text-events")
	lc := newTestLoopContext("sess-text-events", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	lc.LLMResult = &llmclient.CompleteResult{
		Text: "Hello!",
		TokenUsage: &tokenmeter.TokenUsage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
			Source:       tokenmeter.SourceEstimated,
		},
	}

	_, err := loop.textResponseHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	messageCompleteFound := false
	sessionStateChangeFound := false
	loopCompletedFound := false

	for {
		evt, ok := waitForEvent(ch, "", 1*time.Second)
		if !ok {
			break
		}
		switch evt.Type {
		case "message_complete":
			messageCompleteFound = true
		case "session_state_change":
			data, _ := evt.Data.(map[string]any)
			if data["reason"] == "completed" {
				sessionStateChangeFound = true
			}
		case "loop.loop_completed":
			loopCompletedFound = true
		}
	}

	if !messageCompleteFound {
		t.Error("expected message_complete event")
	}
	if !sessionStateChangeFound {
		t.Error("expected session_state_change event with reason=completed")
	}
	if !loopCompletedFound {
		t.Error("expected loop.loop_completed event")
	}
}

func TestStateMachine_RegisterHandlers(t *testing.T) {
	loop, _ := setupTestLoop()
	sm := NewStateMachine()
	loop.registerHandlers(sm)

	requiredStates := []LoopState{
		LoopStatePreparing,
		LoopStateThinking,
		LoopStateEvaluatingResult,
		LoopStateToolExecuting,
		LoopStateAwaitingApproval,
		LoopStateTextResponse,
	}

	for _, state := range requiredStates {
		sm.mu.RLock()
		_, ok := sm.handlers[state]
		sm.mu.RUnlock()
		if !ok {
			t.Errorf("expected handler for state %s", state)
		}
	}
}

type toolCallMockClient struct {
	toolCalls []session.ToolCall
}

func (m *toolCallMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	return &llmclient.CompleteResult{
		ToolCalls: m.toolCalls,
	}, nil
}

func (m *toolCallMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	callback(llmclient.StreamEvent{Type: "done", FullText: "", ToolCalls: m.toolCalls})
	return nil
}

type errorMockClient struct{}

func (m *errorMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	return nil, errors.New("llm error")
}

func (m *errorMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	callback(llmclient.StreamEvent{Type: "error", Err: errors.New("stream error")})
	return errors.New("stream error")
}

func TestPrepareHandler_AssemblesPrompt(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-assemble")
	lc := newTestLoopContext("sess-assemble", store)

	store.AddMessage("sess-assemble", session.Message{
		ID:        "msg-1",
		Role:      session.RoleUser,
		Content:   "Hello, world!",
		CreatedAt: time.Now(),
	})

	_, err := loop.preparingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if lc.DynamicPrompt == "" {
		t.Error("expected DynamicPrompt to be assembled")
	}
	if !strings.Contains(lc.DynamicPrompt, "devo") || !strings.Contains(lc.DynamicPrompt, "You are") {
		t.Logf("DynamicPrompt: %s", lc.DynamicPrompt)
	}
}
