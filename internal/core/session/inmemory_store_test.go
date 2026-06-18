package session

import (
	"testing"
	"time"
)

func TestAddAndGetMessages(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	msg1 := Message{ID: "msg-1", Role: RoleUser, Content: "Hello", CreatedAt: time.Now()}
	msg2 := Message{ID: "msg-2", Role: RoleAssistant, Content: "Hi there", CreatedAt: time.Now()}

	if err := store.AddMessage("sess-1", msg1); err != nil {
		t.Fatalf("add message 1: %v", err)
	}
	if err := store.AddMessage("sess-1", msg2); err != nil {
		t.Fatalf("add message 2: %v", err)
	}

	msgs, total, err := store.GetMessages("sess-1", 0, 0)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello" {
		t.Errorf("expected first message %q, got %q", "Hello", msgs[0].Content)
	}
	if msgs[1].Content != "Hi there" {
		t.Errorf("expected second message %q, got %q", "Hi there", msgs[1].Content)
	}
}

func TestGetMessagesPagination(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	for i := 0; i < 5; i++ {
		store.AddMessage("sess-1", Message{
			ID:        GenerateID("msg"),
			Role:      RoleUser,
			Content:   "message",
			CreatedAt: time.Now(),
		})
	}

	t.Run("limit only", func(t *testing.T) {
		msgs, total, _ := store.GetMessages("sess-1", 2, 0)
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(msgs) != 2 {
			t.Errorf("expected 2 messages with limit=2, got %d", len(msgs))
		}
	})

	t.Run("offset only", func(t *testing.T) {
		msgs, total, _ := store.GetMessages("sess-1", 0, 3)
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(msgs) != 2 {
			t.Errorf("expected 2 messages with offset=3, got %d", len(msgs))
		}
	})

	t.Run("offset beyond total", func(t *testing.T) {
		msgs, total, _ := store.GetMessages("sess-1", 0, 10)
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(msgs) != 0 {
			t.Errorf("expected 0 messages with offset=10, got %d", len(msgs))
		}
	})
}

func TestAddMessageSessionNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.AddMessage("nonexistent", Message{})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestSSEConnectionsIncrementDecrement(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := store.IncrementSSEConnections("sess-1"); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := store.IncrementSSEConnections("sess-1"); err != nil {
		t.Fatalf("increment: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.ActiveSSEConnections != 2 {
		t.Errorf("expected 2 connections, got %d", sess.ActiveSSEConnections)
	}

	if err := store.DecrementSSEConnections("sess-1"); err != nil {
		t.Fatalf("decrement: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.ActiveSSEConnections != 1 {
		t.Errorf("expected 1 connection after decrement, got %d", sess.ActiveSSEConnections)
	}
}

func TestSSEConnectionsDecrementBelowZero(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	store.DecrementSSEConnections("sess-1")

	sess, _ = store.Get("sess-1")
	if sess.ActiveSSEConnections < 0 {
		t.Errorf("ActiveSSEConnections should not go below 0, got %d", sess.ActiveSSEConnections)
	}
}

func TestSSEConnectionsNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.IncrementSSEConnections("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}

	err = store.DecrementSSEConnections("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}
