package agentloop

import (
	"context"
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestTokenUsageEventPublished(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-token-1")

	eventBus, _ := store.GetEventBus("sess-token-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-token-1", session.Message{Content: "Hello, world!"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "token_usage", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for token_usage event")
	}

	data, ok := evt.Data.(map[string]interface{})
	if !ok {
		t.Fatal("token_usage event data is not a map")
	}

	if _, hasStep := data["step"]; !hasStep {
		t.Error("token_usage event missing 'step' field")
	}
	if _, hasInput := data["input_tokens"]; !hasInput {
		t.Error("token_usage event missing 'input_tokens' field")
	}
	if _, hasOutput := data["output_tokens"]; !hasOutput {
		t.Error("token_usage event missing 'output_tokens' field")
	}
	if _, hasTotal := data["session_total_tokens"]; !hasTotal {
		t.Error("token_usage event missing 'session_total_tokens' field")
	}
	if _, hasSessionInput := data["session_input_tokens"]; !hasSessionInput {
		t.Error("token_usage event missing 'session_input_tokens' field")
	}
	if _, hasSessionOutput := data["session_output_tokens"]; !hasSessionOutput {
		t.Error("token_usage event missing 'session_output_tokens' field")
	}
}

func TestMessageCompleteHasTotalStepTokens(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-token-2")

	eventBus, _ := store.GetEventBus("sess-token-2")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-token-2", session.Message{Content: "Hello!"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "message_complete", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for message_complete event")
	}

	data, ok := evt.Data.(map[string]interface{})
	if !ok {
		t.Fatal("message_complete event data is not a map")
	}

	totalStepTokens, hasTokens := data["total_step_tokens"]
	if !hasTokens {
		t.Error("message_complete event missing 'total_step_tokens' field")
	}
	if totalStepTokens == nil {
		t.Error("total_step_tokens is nil")
	}
}

func TestSessionTokenUsageAccumulates(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-token-3")

	if err := loop.ProcessMessage(context.Background(), "sess-token-3", session.Message{Content: "Hello!"}); err != nil {
		t.Fatalf("first message: %v", err)
	}

	if err := waitForIdle(store, "sess-token-3", 5*time.Second); err != nil {
		t.Fatalf("first round did not finish: %v", err)
	}

	sess, _ := store.Get("sess-token-3")
	if sess.TokenUsage.Total <= 0 {
		t.Errorf("session token usage should be > 0 after LLM call, got total=%d", sess.TokenUsage.Total)
	}
	if sess.TokenUsage.Input <= 0 {
		t.Errorf("session token usage input should be > 0, got input=%d", sess.TokenUsage.Input)
	}
	if sess.TokenUsage.Output <= 0 {
		t.Errorf("session token usage output should be > 0, got output=%d", sess.TokenUsage.Output)
	}

	firstUsage := sess.TokenUsage

	if err := loop.ProcessMessage(context.Background(), "sess-token-3", session.Message{Content: "Second message"}); err != nil {
		t.Fatalf("second message: %v", err)
	}

	if err := waitForIdle(store, "sess-token-3", 5*time.Second); err != nil {
		t.Fatalf("second round did not finish: %v", err)
	}

	sess, _ = store.Get("sess-token-3")
	if sess.TokenUsage.Total <= firstUsage.Total {
		t.Errorf("session token usage should increase after second LLM call: first=%d, now=%d",
			firstUsage.Total, sess.TokenUsage.Total)
	}
}

func TestTokenUsageStepsStored(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-token-4")

	if err := loop.ProcessMessage(context.Background(), "sess-token-4", session.Message{Content: "Hello!"}); err != nil {
		t.Fatalf("message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	steps, err := store.GetUsageSteps("sess-token-4")
	if err != nil {
		t.Fatalf("GetUsageSteps failed: %v", err)
	}

	if len(steps) == 0 {
		t.Fatal("expected at least 1 usage step to be stored")
	}

	if steps[0].InputTokens <= 0 {
		t.Errorf("step input tokens should be > 0, got %d", steps[0].InputTokens)
	}
	if steps[0].OutputTokens <= 0 {
		t.Errorf("step output tokens should be > 0, got %d", steps[0].OutputTokens)
	}
	if steps[0].Source == "" {
		t.Error("step source should not be empty")
	}
}

func TestTokenUsageMonotonicallyIncreasing(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-token-5")

	eventBus, _ := store.GetEventBus("sess-token-5")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-token-5", session.Message{Content: "Hello!"}); err != nil {
		t.Fatalf("message: %v", err)
	}

	var lastTotal float64
	var receivedTokenUsage bool

	for {
		evt, ok := waitForEvent(ch, "", 2*time.Second)
		if !ok {
			break
		}

		if evt.Type == "token_usage" {
			receivedTokenUsage = true
			data, ok := evt.Data.(map[string]interface{})
			if !ok {
				continue
			}
			total, ok := data["session_total_tokens"].(float64)
			if !ok {
				continue
			}
			if total < lastTotal {
				t.Errorf("session_total_tokens should be monotonically increasing: last=%f, current=%f", lastTotal, total)
			}
			lastTotal = total
		}

		if evt.Type == "session_state_change" {
			data, ok := evt.Data.(map[string]any)
			if ok && data["reason"] == "completed" {
				break
			}
		}
	}

	if !receivedTokenUsage {
		t.Error("no token_usage event received")
	}
}
