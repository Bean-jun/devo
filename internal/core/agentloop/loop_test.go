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

func TestProcessMessage(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	ctx := context.Background()
	msg, err := loop.ProcessMessage(ctx, "sess-1", "Hello, world!")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msg.Role != session.RoleAssistant {
		t.Errorf("expected assistant role, got %q", msg.Role)
	}
	if msg.Content == "" {
		t.Error("expected non-empty response")
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

	ctx := context.Background()
	_, err := loop.ProcessMessage(ctx, "nonexistent", "Hello")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestProcessMessageNotIdle(t *testing.T) {
	loop, store := setupTestLoop()
	sess := createTestSession(store, "sess-1")

	sess.State = session.StateProcessing
	store.Update(sess)

	ctx := context.Background()
	_, err := loop.ProcessMessage(ctx, "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error when session is not idle")
	}
}

func TestMultiTurnConversation(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	ctx := context.Background()

	msg1, err := loop.ProcessMessage(ctx, "sess-1", "First message")
	if err != nil {
		t.Fatalf("first message failed: %v", err)
	}
	t.Logf("Reply 1: %s", msg1.Content)

	msg2, err := loop.ProcessMessage(ctx, "sess-1", "Second message")
	if err != nil {
		t.Fatalf("second message failed: %v", err)
	}
	t.Logf("Reply 2: %s", msg2.Content)

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

	ctx := context.Background()

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("initial state should be Idle, got %q", sess.State)
	}

	loop.ProcessMessage(ctx, "sess-1", "Hello")

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("final state should be Idle, got %q", sess.State)
	}
}

func TestStateRevertsOnLLMError(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "sess-1")

	failingClient := &errorLLMClient{}
	loop := New(store, failingClient)

	ctx := context.Background()
	_, err := loop.ProcessMessage(ctx, "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error from failing LLM")
	}

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("state should revert to Idle on error, got %q", sess.State)
	}
}

type errorLLMClient struct{}

func (e *errorLLMClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (string, error) {
	return "", context.DeadlineExceeded
}
