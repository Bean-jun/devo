package session

import (
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test Session",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}

	err := store.Create(sess)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, err := store.Get("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if got.ID != sess.ID {
		t.Errorf("expected ID %q, got %q", sess.ID, got.ID)
	}
	if got.Title != sess.Title {
		t.Errorf("expected Title %q, got %q", sess.Title, got.Title)
	}
	if got.State != sess.State {
		t.Errorf("expected State %q, got %q", sess.State, got.State)
	}
}

func TestCreateDuplicate(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}

	if err := store.Create(sess); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	err := store.Create(sess)
	if err != ErrSessionConflict {
		t.Errorf("expected ErrSessionConflict, got: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestUpdate(t *testing.T) {
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

	sess.Title = "Updated Title"
	sess.State = StateToolExecuting

	err := store.Update(sess)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := store.Get("sess-1")
	if got.Title != "Updated Title" {
		t.Errorf("expected Title %q, got %q", "Updated Title", got.Title)
	}
	if got.State != StateToolExecuting {
		t.Errorf("expected State %q, got %q", StateToolExecuting, got.State)
	}
}

func TestUpdateNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.Update(&Session{ID: "nonexistent"})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID("test")
	id2 := GenerateID("test")

	if id1 == id2 {
		t.Error("expected different IDs")
	}
}

func TestSessionMutationIsolation(t *testing.T) {
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

	got, _ := store.Get("sess-1")
	got.Title = "mutated via get"

	gotAgain, _ := store.Get("sess-1")
	if gotAgain.Title == "mutated via get" {
		t.Error("Get should return a copy, not a reference - store was mutated")
	}
}

func TestUpdatePreservesEventBus(t *testing.T) {
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

	originalEB, _ := store.GetEventBus("sess-1")

	sess.State = StateToolExecuting
	store.Update(sess)

	updatedEB, _ := store.GetEventBus("sess-1")
	if originalEB != updatedEB {
		t.Error("EventBus should be preserved across Update calls")
	}
}
