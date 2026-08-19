package agentloop

import (
	"context"
	"errors"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

type reasoningMockClient struct{}

func (m *reasoningMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	return &llmclient.CompleteResult{Text: "ok"}, nil
}

func (m *reasoningMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	callback(llmclient.StreamEvent{Type: "reasoning_token", Reasoning: "first ", FullReasoning: "first "})
	callback(llmclient.StreamEvent{Type: "reasoning_token", Reasoning: "thought", FullReasoning: "first thought"})
	callback(llmclient.StreamEvent{Type: "token", Token: "Hello", FullText: "Hello"})
	callback(llmclient.StreamEvent{Type: "token", Token: " world", FullText: "Hello world"})
	callback(llmclient.StreamEvent{
		Type:          "done",
		FullText:      "Hello world",
		FullReasoning: "first thought",
		FinishReason:  "stop",
	})
	return nil
}
func (m *reasoningMockClient) TestConnection(ctx context.Context) error { return nil }

func TestThinkingHandler_PublishesReasoningEvents(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-reasoning")
	loop := NewWithTools(store, &reasoningMockClient{}, nil)
	lc := newTestLoopContext("sess-reasoning", store)

	lc.ActiveMsgs = []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hi"},
	}
	lc.DynamicPrompt = "system"

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	nextState, err := loop.thinkingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if nextState != LoopStateEvaluatingResult {
		t.Errorf("expected EvaluatingResult, got %s", nextState)
	}

	var reasoningTokens []string
	var reasoningComplete bool
	var streamingComplete bool
	deadline := time.After(1 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout. reasoning_tokens=%v reasoning_complete=%v streaming_complete=%v", reasoningTokens, reasoningComplete, streamingComplete)
		case evt, ok := <-ch:
			if !ok {
				return
			}
			switch evt.Type {
			case "reasoning_token":
				if data, ok := evt.Data.(map[string]any); ok {
					if token, _ := data["token"].(string); token != "" {
						reasoningTokens = append(reasoningTokens, token)
					}
				}
			case "reasoning_complete":
				reasoningComplete = true
				if data, ok := evt.Data.(map[string]any); ok {
					if full, _ := data["full_reasoning"].(string); full != "first thought" {
						t.Errorf("expected full_reasoning 'first thought', got %q", full)
					}
				}
			case "streaming_complete":
				streamingComplete = true
			}
			if reasoningComplete && streamingComplete && len(reasoningTokens) == 2 {
				return
			}
		}
	}
}

func TestThinkingHandler_LLMResultCapturesReasoning(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-reasoning-result")
	loop := NewWithTools(store, &reasoningMockClient{}, nil)
	lc := newTestLoopContext("sess-reasoning-result", store)

	lc.ActiveMsgs = []session.Message{{ID: "msg-1", Role: session.RoleUser, Content: "Hi"}}
	lc.DynamicPrompt = "system"

	_, err := loop.thinkingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lc.LLMResult == nil {
		t.Fatal("expected LLMResult to be set")
	}
	if lc.LLMResult.Reasoning != "first thought" {
		t.Errorf("expected reasoning 'first thought', got %q", lc.LLMResult.Reasoning)
	}
	if lc.LLMResult.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %q", lc.LLMResult.Text)
	}
	if lc.ReasoningBuilder.String() != "first thought" {
		t.Errorf("expected ReasoningBuilder 'first thought', got %q", lc.ReasoningBuilder.String())
	}
}

func TestTextResponseHandler_PersistsReasoning(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-text-reasoning")
	loop := NewWithTools(store, &reasoningMockClient{}, nil)
	lc := newTestLoopContext("sess-text-reasoning", store)

	lc.LLMResult = &llmclient.CompleteResult{
		Text:      "answer",
		Reasoning: "thinking",
	}

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	nextState, err := loop.textResponseHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextState != LoopStateIdle {
		t.Errorf("expected Idle, got %s", nextState)
	}

	msgs, _, err := store.GetMessages("sess-text-reasoning", 0, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Reasoning != "thinking" {
		t.Errorf("expected reasoning 'thinking', got %q", msgs[0].Reasoning)
	}
	if msgs[0].Content != "answer" {
		t.Errorf("expected content 'answer', got %q", msgs[0].Content)
	}

	var messageComplete bool
	deadline := time.After(500 * time.Millisecond)
	for !messageComplete {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for message_complete event")
		case evt := <-ch:
			if evt.Type == "message_complete" {
				messageComplete = true
				if data, ok := evt.Data.(map[string]any); ok {
					if full, _ := data["full_reasoning"].(string); full != "thinking" {
						t.Errorf("expected full_reasoning 'thinking', got %q", full)
					}
				}
			}
		}
	}
}

func TestReasoningBuilderResetsBetweenTurns(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-reset")
	loop := NewWithTools(store, &reasoningMockClient{}, nil)
	lc := newTestLoopContext("sess-reset", store)

	lc.ActiveMsgs = []session.Message{{ID: "msg-1", Role: session.RoleUser, Content: "Hi"}}
	lc.DynamicPrompt = "system"

	lc.ReasoningBuilder.WriteString("previous turn reasoning")
	_, err := loop.thinkingHandler(context.Background(), lc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lc.ReasoningBuilder.String() != "first thought" {
		t.Errorf("expected builder reset to 'first thought', got %q", lc.ReasoningBuilder.String())
	}
}

type noReasoningErrorMockClient struct{}

func (m *noReasoningErrorMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	return nil, errors.New("fail")
}

func (m *noReasoningErrorMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	callback(llmclient.StreamEvent{Type: "error", Err: errors.New("stream failed")})
	return errors.New("stream failed")
}
func (m *noReasoningErrorMockClient) TestConnection(ctx context.Context) error { return nil }

func TestThinkingHandler_NoReasoningCompleteOnEmpty(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-empty-reasoning")
	loop := NewWithTools(store, &noReasoningErrorMockClient{}, nil)
	lc := newTestLoopContext("sess-empty-reasoning", store)

	lc.ActiveMsgs = []session.Message{{ID: "msg-1", Role: session.RoleUser, Content: "Hi"}}
	lc.DynamicPrompt = "system"

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	_, _ = loop.thinkingHandler(context.Background(), lc)

	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case <-deadline:
			return
		case evt := <-ch:
			if evt.Type == "reasoning_complete" {
				t.Error("did not expect reasoning_complete event when no reasoning was emitted")
			}
		}
	}
}
