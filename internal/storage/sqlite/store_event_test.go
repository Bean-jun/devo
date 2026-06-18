package sqlite

import (
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"

	"devo/internal/core/session"

	gormsqlite "github.com/glebarez/sqlite"
)

func TestGormStoreEventBusCreatedOnCreate(t *testing.T) {
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

	eventBus, err := store.GetEventBus("sess-1")
	if err != nil {
		t.Fatalf("expected event bus to exist, got: %v", err)
	}
	if eventBus == nil {
		t.Fatal("expected non-nil event bus")
	}
}

func TestGormStoreGetEventBusNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetEventBus("nonexistent")
	if err != session.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestGormStoreAddAndGetEvents(t *testing.T) {
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

	evt1 := session.Event{
		ID:        1,
		Type:      "event1",
		Data:      map[string]string{"n": "1"},
		CreatedAt: time.Now(),
	}
	evt2 := session.Event{
		ID:        2,
		Type:      "event2",
		Data:      map[string]string{"n": "2"},
		CreatedAt: time.Now(),
	}

	if err := store.AddEvent("sess-1", evt1); err != nil {
		t.Fatalf("add event 1: %v", err)
	}
	if err := store.AddEvent("sess-1", evt2); err != nil {
		t.Fatalf("add event 2: %v", err)
	}

	events, err := store.GetEvents("sess-1", 0)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "event1" {
		t.Errorf("expected event1, got %s", events[0].Type)
	}
	if events[1].Type != "event2" {
		t.Errorf("expected event2, got %s", events[1].Type)
	}
}

func TestGormStoreGetEventsSinceID(t *testing.T) {
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

	store.AddEvent("sess-1", session.Event{ID: 1, Type: "event1", Data: map[string]string{"n": "1"}, CreatedAt: time.Now()})
	store.AddEvent("sess-1", session.Event{ID: 2, Type: "event2", Data: map[string]string{"n": "2"}, CreatedAt: time.Now()})
	store.AddEvent("sess-1", session.Event{ID: 3, Type: "event3", Data: map[string]string{"n": "3"}, CreatedAt: time.Now()})

	events, err := store.GetEvents("sess-1", 1)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events after ID 1, got %d", len(events))
	}
	if events[0].Type != "event2" {
		t.Errorf("expected event2, got %s", events[0].Type)
	}
}

func TestGormStoreEventPersistenceAcrossRestarts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "event_persist.db")

	dir := t.TempDir()

	func() {
		db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open db: %v", err)
		}

		store, err := NewGormStore(db)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		sess := &session.Session{
			ID:               "sess-event",
			Title:            "Event Persist Test",
			WorkingDirectory: dir,
			State:            session.StateIdle,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		}
		store.Create(sess)

		store.AddEvent("sess-event", session.Event{ID: 1, Type: "thinking", Data: map[string]string{"message": "processing"}, CreatedAt: time.Now()})
		store.AddEvent("sess-event", session.Event{ID: 2, Type: "message_complete", Data: map[string]string{"full_text": "done"}, CreatedAt: time.Now()})
	}()

	func() {
		db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to reopen db: %v", err)
		}

		store, err := NewGormStore(db)
		if err != nil {
			t.Fatalf("failed to recreate store: %v", err)
		}
		defer store.Close()

		events, err := store.GetEvents("sess-event", 0)
		if err != nil {
			t.Fatalf("get events after restart: %v", err)
		}

		if len(events) != 2 {
			t.Fatalf("expected 2 events after restart, got %d", len(events))
		}
		if events[0].Type != "thinking" {
			t.Errorf("expected first event 'thinking', got %q", events[0].Type)
		}
		if events[1].Type != "message_complete" {
			t.Errorf("expected second event 'message_complete', got %q", events[1].Type)
		}
	}()
}
