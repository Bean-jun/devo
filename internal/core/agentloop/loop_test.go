package agentloop

import (
	"context"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func setupTestLoop() (*Loop, *session.InMemoryStore) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())
	return loop, store
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

func TestProcessMessageReturnsImmediately(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	start := time.Now()
	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello, world!")
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

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Hello, world!"); err != nil {
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

	err := loop.ProcessMessage(context.Background(), "nonexistent", "Hello")
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

	if err := loop.ProcessMessage(context.Background(), "sess-1", "First message"); err != nil {
		t.Fatalf("first message: %v", err)
	}

	_, ok := waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for first round to complete")
	}

	time.Sleep(50 * time.Millisecond)

	if err := loop.ProcessMessage(context.Background(), "sess-1", "Second message"); err != nil {
		t.Fatalf("second message: %v", err)
	}

	_, ok = waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for second round to complete")
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

	loop.ProcessMessage(context.Background(), "sess-1", "Hello")

	_, ok := waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("timed out waiting for session_state_change")
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

func TestProcessMessageConflictDuringProcessing(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	slowClient := &slowLLMClient{}
	loop := New(store, slowClient)

	if err := loop.ProcessMessage(context.Background(), "sess-1", "First"); err != nil {
		t.Fatalf("first message: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateProcessing {
		t.Errorf("expected state Processing after first message starts processing")
	}

	err := loop.ProcessMessage(context.Background(), "sess-1", "Second")
	if err == nil {
		t.Fatal("expected error when session is Processing")
	}
}

type errorLLMClient struct{}

func (e *errorLLMClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	return nil, context.DeadlineExceeded
}

func TestStateRevertsOnLLMError(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	failingClient := &errorLLMClient{}
	loop := New(store, failingClient)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	loop.ProcessMessage(context.Background(), "sess-1", "Hello")

	_, ok := waitForEvent(ch, "thinking", 2*time.Second)
	if !ok {
		t.Fatal("expected thinking event even on error")
	}

	time.Sleep(100 * time.Millisecond)

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

	loop.ProcessMessage(context.Background(), "sess-1", "Hello")

	expectedOrder := []string{"thinking", "message_complete", "session_state_change"}
	received := make([]string, 0, 3)

	for range expectedOrder {
		evt, ok := waitForEvent(ch, "", 2*time.Second)
		if !ok {
			t.Fatal("timed out waiting for events")
		}
		received = append(received, evt.Type)
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
