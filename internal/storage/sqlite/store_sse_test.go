package sqlite

import (
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestGormStoreSSEConnectionCounter(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-sse",
		Title:            "SSE Test",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := store.IncrementSSEConnections("sess-sse"); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := store.IncrementSSEConnections("sess-sse"); err != nil {
		t.Fatalf("increment: %v", err)
	}

	got, _ := store.Get("sess-sse")
	if got.ActiveSSEConnections != 2 {
		t.Errorf("expected 2 connections, got %d", got.ActiveSSEConnections)
	}

	if err := store.DecrementSSEConnections("sess-sse"); err != nil {
		t.Fatalf("decrement: %v", err)
	}

	got, _ = store.Get("sess-sse")
	if got.ActiveSSEConnections != 1 {
		t.Errorf("expected 1 connection after decrement, got %d", got.ActiveSSEConnections)
	}
}
