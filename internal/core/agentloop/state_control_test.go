package agentloop

import (
	"context"
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestProcessMessageNotIdle(t *testing.T) {
	loop, store := setupTestLoop()
	sess := createTestSession(store, "sess-1")

	sess.State = session.StateProcessing
	store.Update(sess)

	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error when session is not idle")
	}
}

func TestProcessMessageArchivedRejected(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateArchived
	store.Update(sess)

	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error when posting to archived session")
	}
}

func TestProcessMessagePausedAutoResume(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StatePaused
	store.Update(sess)

	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello from paused")
	if err != nil {
		t.Fatalf("expected no error (auto-resume), got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateProcessing {
		t.Errorf("expected state Processing after auto-resume from Paused, got %q", sess.State)
	}
}

func TestProcessMessageFromProcessing(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	store.Update(sess)

	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error when processing is already running")
	}
}

func TestProcessMessageFromAwaitingApproval(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateAwaitingApproval
	store.Update(sess)

	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error when session is awaiting approval")
	}
}

func TestPauseFromProcessing(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	store.Update(sess)

	err := loop.Pause("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if !sess.PauseRequested {
		t.Error("expected PauseRequested to be true")
	}
}

func TestPauseFromNonProcessing(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	err := loop.Pause("sess-1")
	if err == nil {
		t.Fatal("expected error when pausing from Idle state")
	}
}

func TestPauseSessionNotFound(t *testing.T) {
	loop, _ := setupTestLoop()

	err := loop.Pause("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestResumeFromPaused(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StatePaused
	store.Update(sess)

	err := loop.Resume("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateProcessing {
		t.Errorf("expected state Processing after resume, got %q", sess.State)
	}
}

func TestResumeFromNonPaused(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	err := loop.Resume("sess-1")
	if err == nil {
		t.Fatal("expected error when resuming from Idle state")
	}
}

func TestResumeSessionNotFound(t *testing.T) {
	loop, _ := setupTestLoop()

	err := loop.Resume("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestResumePublishesEvent(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StatePaused
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.Resume("sess-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "session_state_change", 1*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}

	data, _ := evt.Data.(map[string]any)
	if data["new_state"] != string(session.StateProcessing) {
		t.Errorf("expected new_state Processing, got %v", data["new_state"])
	}
	if data["reason"] != "resumed" {
		t.Errorf("expected reason resumed, got %v", data["reason"])
	}
}

func TestCancelFromProcessing(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	store.Update(sess)

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if !sess.CancelRequested {
		t.Error("expected CancelRequested to be true")
	}
}

func TestCancelFromAwaitingApproval(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateAwaitingApproval
	store.Update(sess)

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if !sess.CancelRequested {
		t.Error("expected CancelRequested to be true")
	}
}

func TestCancelFromNonCancellable(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	err := loop.Cancel("sess-1")
	if err == nil {
		t.Fatal("expected error when cancelling from Idle state")
	}
}

func TestCancelFromPaused(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StatePaused
	store.Update(sess)

	err := loop.Cancel("sess-1")
	if err == nil {
		t.Fatal("expected error when cancelling from Paused state")
	}
}

func TestCancelSessionNotFound(t *testing.T) {
	loop, _ := setupTestLoop()

	err := loop.Cancel("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestCompleteFromIdle(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	err := loop.Complete("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateCompleted {
		t.Errorf("expected state Completed, got %q", sess.State)
	}
}

func TestCompleteFromPaused(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StatePaused
	store.Update(sess)

	err := loop.Complete("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateCompleted {
		t.Errorf("expected state Completed, got %q", sess.State)
	}
}

func TestCompleteFromProcessing(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	store.Update(sess)

	err := loop.Complete("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateCompleted {
		t.Errorf("expected state Completed after complete from Processing, got %q", sess.State)
	}
	if sess.CancelRequested {
		t.Error("expected CancelRequested to be cleared after complete")
	}
}

func TestCompleteFromArchived(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateArchived
	store.Update(sess)

	err := loop.Complete("sess-1")
	if err == nil {
		t.Fatal("expected error when completing from Archived state")
	}
}

func TestCompletePublishesEvent(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.Complete("sess-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "session_state_change", 1*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}

	data, _ := evt.Data.(map[string]any)
	if data["new_state"] != string(session.StateCompleted) {
		t.Errorf("expected new_state Completed, got %v", data["new_state"])
	}
}

func TestCompleteSessionNotFound(t *testing.T) {
	loop, _ := setupTestLoop()

	err := loop.Complete("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestArchiveFromCompleted(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateCompleted
	store.Update(sess)

	err := loop.Archive("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateArchived {
		t.Errorf("expected state Archived, got %q", sess.State)
	}
}

func TestArchiveFromNonCompleted(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	err := loop.Archive("sess-1")
	if err == nil {
		t.Fatal("expected error when archiving from Idle state")
	}
}

func TestArchiveFromProcessing(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	store.Update(sess)

	err := loop.Archive("sess-1")
	if err == nil {
		t.Fatal("expected error when archiving from Processing state")
	}
}

func TestArchivePublishesEvent(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateCompleted
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	if err := loop.Archive("sess-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "session_state_change", 1*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}

	data, _ := evt.Data.(map[string]any)
	if data["new_state"] != string(session.StateArchived) {
		t.Errorf("expected new_state Archived, got %v", data["new_state"])
	}
	if data["reason"] != "archived" {
		t.Errorf("expected reason archived, got %v", data["reason"])
	}
}

func TestArchiveSessionNotFound(t *testing.T) {
	loop, _ := setupTestLoop()

	err := loop.Archive("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestCheckControlFlagsCancel(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	sess.CancelRequested = true
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	stopped := loop.checkControlFlags("sess-1", eventBus)
	if !stopped {
		t.Fatal("expected checkControlFlags to return true when cancelled")
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle after cancel, got %q", sess.State)
	}
	if sess.CancelRequested {
		t.Error("expected CancelRequested to be cleared")
	}

	evt, ok := waitForEvent(ch, "session_state_change", 1*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}

	data, _ := evt.Data.(map[string]any)
	if data["reason"] != "cancelled" {
		t.Errorf("expected reason cancelled, got %v", data["reason"])
	}
}

func TestCheckControlFlagsPause(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	sess.PauseRequested = true
	store.Update(sess)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	stopped := loop.checkControlFlags("sess-1", eventBus)
	if !stopped {
		t.Fatal("expected checkControlFlags to return true when paused")
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StatePaused {
		t.Errorf("expected state Paused after pause, got %q", sess.State)
	}
	if sess.PauseRequested {
		t.Error("expected PauseRequested to be cleared")
	}

	evt, ok := waitForEvent(ch, "session_state_change", 1*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}

	data, _ := evt.Data.(map[string]any)
	if data["reason"] != "paused" {
		t.Errorf("expected reason paused, got %v", data["reason"])
	}
}

func TestCheckControlFlagsNoFlags(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	eventBus, _ := store.GetEventBus("sess-1")

	stopped := loop.checkControlFlags("sess-1", eventBus)
	if stopped {
		t.Fatal("expected checkControlFlags to return false when no flags set")
	}
}

func TestFullStateFlowIdleToArchived(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	sess, _ := store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Fatalf("expected Idle, got %q", sess.State)
	}

	if err := loop.Complete("sess-1"); err != nil {
		t.Fatalf("complete from Idle: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateCompleted {
		t.Fatalf("expected Completed, got %q", sess.State)
	}

	if err := loop.Archive("sess-1"); err != nil {
		t.Fatalf("archive from Completed: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateArchived {
		t.Fatalf("expected Archived, got %q", sess.State)
	}

	err := loop.ProcessMessage(context.Background(), "sess-1", "Hello")
	if err == nil {
		t.Fatal("expected error when posting to archived session")
	}
}

func TestKillChildProcess(t *testing.T) {
	killChildProcess(-1)
	killChildProcess(0)
	killChildProcess(99999999)
}

func TestCancelClearsChildPID(t *testing.T) {
	loop, store := setupTestLoop()

	pid := 99999
	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	sess.ChildPID = &pid
	store.Update(sess)

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after cancel")
	}
}

func TestCompleteCancelsRunningProcess(t *testing.T) {
	loop, store := setupTestLoop()

	pid := 99999
	sess := createTestSession(store, "sess-1")
	sess.State = session.StateProcessing
	sess.ChildPID = &pid
	store.Update(sess)

	err := loop.Complete("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateCompleted {
		t.Errorf("expected state Completed, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after complete")
	}
}
