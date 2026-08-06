package sqlite

import (
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"

	"devo/internal/core/session"

	gormsqlite "github.com/glebarez/sqlite"
)

func TestMessageReasoning_PersistAndLoad(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-reasoning",
		Title:            "Reasoning Test",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	assistantMsg := session.Message{
		ID:        "msg-with-reasoning",
		Role:      session.RoleAssistant,
		Content:   "最终答案",
		Reasoning: "先分析问题，然后推理出答案",
		CreatedAt: time.Now(),
	}
	if err := store.AddMessage("sess-reasoning", assistantMsg); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	msgs, total, err := store.GetMessages("sess-reasoning", 0, 0)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Reasoning != "先分析问题，然后推理出答案" {
		t.Errorf("expected reasoning '先分析问题，然后推理出答案', got %q", msgs[0].Reasoning)
	}
	if msgs[0].Content != "最终答案" {
		t.Errorf("expected content '最终答案', got %q", msgs[0].Content)
	}
}

func TestMessageReasoning_EmptyByDefault(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-no-reasoning",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	msg := session.Message{
		ID:        "msg-plain",
		Role:      session.RoleUser,
		Content:   "hello",
		CreatedAt: time.Now(),
	}
	if err := store.AddMessage("sess-no-reasoning", msg); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	msgs, _, err := store.GetMessages("sess-no-reasoning", 0, 0)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if msgs[0].Reasoning != "" {
		t.Errorf("expected empty reasoning, got %q", msgs[0].Reasoning)
	}
}

func TestMessageReasoning_PersistenceAcrossRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reasoning-persist.db")

	dir := t.TempDir()

	func() {
		db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		store, err := NewGormStore(db)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		defer store.Close()

		store.Create(&session.Session{
			ID:               "sess-persist-r",
			WorkingDirectory: dir,
			State:            session.StateIdle,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		})

		store.AddMessage("sess-persist-r", session.Message{
			ID:        "msg-1",
			Role:      session.RoleAssistant,
			Content:   "answer",
			Reasoning: "thinking before answer",
			CreatedAt: time.Now(),
		})
	}()

	func() {
		db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("reopen db: %v", err)
		}
		store, err := NewGormStore(db)
		if err != nil {
			t.Fatalf("recreate store: %v", err)
		}
		defer store.Close()

		msgs, _, err := store.GetMessages("sess-persist-r", 0, 0)
		if err != nil {
			t.Fatalf("GetMessages after restart: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		if msgs[0].Reasoning != "thinking before answer" {
			t.Errorf("expected reasoning 'thinking before answer', got %q", msgs[0].Reasoning)
		}
	}()
}
