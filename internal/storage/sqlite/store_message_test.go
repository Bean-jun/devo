package sqlite

import (
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestGormStoreAddAndGetMessages(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	msg1 := session.Message{ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now()}
	msg2 := session.Message{ID: "msg-2", Role: session.RoleAssistant, Content: "Hi there", CreatedAt: time.Now()}

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

func TestGormStoreGetMessagesPagination(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	for i := 0; i < 5; i++ {
		store.AddMessage("sess-1", session.Message{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleUser,
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

func TestGormStoreAddMessageSessionNotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.AddMessage("nonexistent", session.Message{})
	if err != session.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}
