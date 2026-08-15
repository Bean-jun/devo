package agentloop

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

func setupTestLoop() (*Loop, *session.InMemoryStore) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())
	return loop, store
}

func newTestLoopWithTools(t *testing.T, store *session.InMemoryStore, client llmclient.Client, toolRegistry *tools.Registry) *Loop {
	loop := NewWithTools(store, client, toolRegistry)
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() {
			loop.WaitForCompletion()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	})
	return loop
}

func createTestSession(store *session.InMemoryStore, id string) *session.Session {
	sess := &session.Session{
		ID:               id,
		Title:            "Test Session",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)
	return sess
}

func waitForEvent(ch chan session.Event, eventType string, timeout time.Duration) (*session.Event, bool) {
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			return nil, false
		case evt, ok := <-ch:
			if !ok {
				return nil, false
			}
			if eventType == "" || evt.Type == eventType {
				return &evt, true
			}
		}
	}
}

func waitForIdle(store *session.InMemoryStore, sessionID string, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("timed out waiting for session %s to become Idle", sessionID)
		default:
			sess, err := store.Get(sessionID)
			if err != nil {
				return err
			}
			if sess.State == session.StateIdle {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestProcessMessageReturnsImmediately(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	start := time.Now()
	err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Hello, world!"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("ProcessMessage should return immediately, but took %v", elapsed)
	}
}

func TestProcessMessagePublishesEvents(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Hello, world!"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	eventTypes := []string{"thinking", "message_complete", "session_state_change"}
	for _, expectedType := range eventTypes {
		evt, ok := waitForEvent(ch, expectedType, 2*time.Second)
		if !ok {
			t.Fatalf("timed out waiting for event: %s", expectedType)
		}
		t.Logf("Received event: %s with data: %v", evt.Type, evt.Data)
	}

	msgs, total, _ := store.GetMessages("sess-1", 0, 0)
	if total != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", total)
	}
	if msgs[0].Role != session.RoleUser {
		t.Errorf("expected first message to be user, got %q", msgs[0].Role)
	}
	if msgs[1].Role != session.RoleAssistant {
		t.Errorf("expected second message to be assistant, got %q", msgs[1].Role)
	}
}

func TestProcessMessageSessionNotFound(t *testing.T) {
	loop, _ := setupTestLoop()

	err := loop.ProcessMessage(context.Background(), "nonexistent", session.Message{Content: "Hello"})
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestMultiTurnConversation(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "First message"}); err != nil {
		t.Fatalf("first message: %v", err)
	}

	_, ok := waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for first round to start")
	}

	if err := waitForIdle(store, "sess-1", 5*time.Second); err != nil {
		t.Fatalf("first round did not finish: %v", err)
	}

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Second message"}); err != nil {
		t.Fatalf("second message: %v", err)
	}

	_, ok = waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for second round to start")
	}

	if err := waitForIdle(store, "sess-1", 5*time.Second); err != nil {
		t.Fatalf("second round did not finish: %v", err)
	}

	msgs, total, _ := store.GetMessages("sess-1", 0, 0)
	if total != 4 {
		t.Fatalf("expected 4 messages after 2 turns, got %d", total)
	}

	expectedRoles := []session.Role{
		session.RoleUser, session.RoleAssistant,
		session.RoleUser, session.RoleAssistant,
	}
	for i, m := range msgs {
		if m.Role != expectedRoles[i] {
			t.Errorf("message %d: expected role %q, got %q", i, expectedRoles[i], m.Role)
		}
	}
}

func TestStateTransitions(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("initial state should be Idle, got %q", sess.State)
	}

	loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Hello"})

	_, ok := waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change (start)")
	}

	if err := waitForIdle(store, "sess-1", 5*time.Second); err != nil {
		t.Fatalf("loop did not finish: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("final state should be Idle, got %q", sess.State)
	}
}

type slowLLMClient struct{}

func (s *slowLLMClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	time.Sleep(200 * time.Millisecond)
	return &llmclient.CompleteResult{Text: "Slow reply"}, nil
}

func (s *slowLLMClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := s.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}
	callback(llmclient.StreamEvent{Type: "done", FullText: result.Text, ToolCalls: result.ToolCalls, TokenUsage: result.TokenUsage})
	return nil
}
func (s *slowLLMClient) TestConnection(ctx context.Context) error { return nil }

func TestProcessMessageConflictDuringProcessing(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	slowClient := &slowLLMClient{}
	loop := New(store, slowClient)

	if err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "First"}); err != nil {
		t.Fatalf("first message: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateThinking {
		t.Errorf("expected state Thinking after first message starts processing")
	}

	err := loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Second"})
	if err == nil {
		t.Fatal("expected error when session is Thinking")
	}
}

type errorLLMClient struct{}

func (e *errorLLMClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	return nil, context.DeadlineExceeded
}

func (e *errorLLMClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	callback(llmclient.StreamEvent{Type: "error", Err: context.DeadlineExceeded})
	return context.DeadlineExceeded
}
func (e *errorLLMClient) TestConnection(ctx context.Context) error { return nil }

func TestStateRevertsOnLLMError(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	failingClient := &errorLLMClient{}
	loop := New(store, failingClient)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Hello"})

	_, ok := waitForEvent(ch, "loop.thinking_started", 2*time.Second)
	if !ok {
		t.Fatal("expected thinking event even on error")
	}

	time.Sleep(500 * time.Millisecond)

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("state should revert to Idle on error, got %q", sess.State)
	}
}

func TestEventTypesOrder(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	loop.ProcessMessage(context.Background(), "sess-1", session.Message{Content: "Hello"})

	expectedOrder := []string{"loop.thinking_started", "token_usage", "loop.thinking_complete", "loop.loop_completed"}
	received := make([]string, 0, 4)

	for i := 0; i < 50; i++ {
		evt, ok := waitForEvent(ch, "", 2*time.Second)
		if !ok {
			break
		}
		for _, expected := range expectedOrder {
			if evt.Type == expected {
				received = append(received, evt.Type)
				break
			}
		}
		if len(received) >= len(expectedOrder) {
			break
		}
	}

	for i, expected := range expectedOrder {
		if i >= len(received) {
			t.Errorf("missing event %q at position %d", expected, i)
			continue
		}
		if received[i] != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, received[i])
		}
	}
}

func TestStateCompletedAndArchivedDefined(t *testing.T) {
	if session.StateCompleted != "Completed" {
		t.Errorf("expected StateCompleted to be 'Completed', got %q", session.StateCompleted)
	}
	if session.StateArchived != "Archived" {
		t.Errorf("expected StateArchived to be 'Archived', got %q", session.StateArchived)
	}
}

func TestEventChannelIsolation(t *testing.T) {
	store := session.NewInMemoryStore()

	createTestSession(store, "sess-a")
	createTestSession(store, "sess-b")

	ebA, _ := store.GetEventBus("sess-a")
	ebB, _ := store.GetEventBus("sess-b")

	chA, unsubA := ebA.Subscribe()
	defer unsubA()
	chB, unsubB := ebB.Subscribe()
	defer unsubB()

	ebA.Publish("test_a", map[string]string{"session": "a"})
	ebB.Publish("test_b", map[string]string{"session": "b"})

	evtA, ok := waitForEvent(chA, "test_a", 500*time.Millisecond)
	if !ok {
		t.Fatal("session A should receive its own event")
	}
	if evtA.Data.(map[string]string)["session"] != "a" {
		t.Error("session A received wrong event data")
	}

	evtB, ok := waitForEvent(chB, "test_b", 500*time.Millisecond)
	if !ok {
		t.Fatal("session B should receive its own event")
	}
	if evtB.Data.(map[string]string)["session"] != "b" {
		t.Error("session B received wrong event data")
	}

	select {
	case evt := <-chA:
		if evt.Type == "test_b" {
			t.Error("session A should NOT receive session B's events")
		}
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case evt := <-chB:
		if evt.Type == "test_a" {
			t.Error("session B should NOT receive session A's events")
		}
	case <-time.After(100 * time.Millisecond):
	}
}

func TestConcurrentSessionProcessing(t *testing.T) {
	store := session.NewInMemoryStore()

	for i := 0; i < 5; i++ {
		sess := &session.Session{
			ID:               fmt.Sprintf("sess-concurrent-%d", i),
			Title:            fmt.Sprintf("Concurrent Test %d", i),
			WorkingDirectory: fmt.Sprintf("/tmp/test-%d", i),
			State:            session.StateIdle,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		}
		store.Create(sess)
	}

	loop := New(store, llmclient.NewMockClient())

	var wg sync.WaitGroup
	errs := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("sess-concurrent-%d", idx)
			err := loop.ProcessMessage(context.Background(), sessionID, session.Message{Content: fmt.Sprintf("Message %d", idx)})
			if err != nil {
				errs <- fmt.Errorf("session %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	time.Sleep(200 * time.Millisecond)

	for i := 0; i < 5; i++ {
		sessionID := fmt.Sprintf("sess-concurrent-%d", i)
		msgs, total, err := store.GetMessages(sessionID, 0, 0)
		if err != nil {
			t.Errorf("session %d: get messages failed: %v", i, err)
			continue
		}
		if total < 2 {
			t.Errorf("session %d: expected at least 2 messages, got %d", i, total)
			continue
		}
		if msgs[0].Role != session.RoleUser {
			t.Errorf("session %d: first message should be user, got %q", i, msgs[0].Role)
		}
	}
}

func TestConcurrentSessionStateIsolation(t *testing.T) {
	store := session.NewInMemoryStore()

	createTestSession(store, "sess-a")
	createTestSession(store, "sess-b")

	slowClient := &slowLLMClient{}
	loop := New(store, slowClient)

	if err := loop.ProcessMessage(context.Background(), "sess-a", session.Message{Content: "Message A"}); err != nil {
		t.Fatalf("session A: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	sessA, _ := store.Get("sess-a")
	if sessA.State != session.StateThinking {
		t.Errorf("session A should be Thinking, got %q", sessA.State)
	}

	sessB, _ := store.Get("sess-b")
	if sessB.State != session.StateIdle {
		t.Errorf("session B should remain Idle, got %q", sessB.State)
	}

	if err := loop.ProcessMessage(context.Background(), "sess-b", session.Message{Content: "Message B"}); err != nil {
		t.Fatalf("session B: %v", err)
	}
}

func TestUpdateConcurrencyConfig(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	loop := New(store, llmclient.NewMockClient())

	maxToolCalls := 5
	maxSubprocesses := 3
	err := loop.UpdateConcurrencyConfig("sess-1", &maxToolCalls, &maxSubprocesses)
	if err != nil {
		t.Fatalf("UpdateConcurrencyConfig failed: %v", err)
	}

	sess, _ := store.Get("sess-1")
	if sess.MaxConcurrentToolCalls != 5 {
		t.Errorf("expected MaxConcurrentToolCalls 5, got %d", sess.MaxConcurrentToolCalls)
	}
	if sess.MaxConcurrentSubprocesses != 3 {
		t.Errorf("expected MaxConcurrentSubprocesses 3, got %d", sess.MaxConcurrentSubprocesses)
	}
}

func TestUpdateConcurrencyConfigNotFound(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	maxToolCalls := 5
	err := loop.UpdateConcurrencyConfig("nonexistent", &maxToolCalls, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestUpdateConcurrencyConfigNegativeValue(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	loop := New(store, llmclient.NewMockClient())

	badValue := -1
	err := loop.UpdateConcurrencyConfig("sess-1", nil, &badValue)
	if err == nil {
		t.Fatal("expected error for negative max_concurrent_subprocesses")
	}
}
