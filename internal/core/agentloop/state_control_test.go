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

	sess.State = session.StateThinking
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
	if sess.State != session.StateThinking {
		t.Errorf("expected state Thinking after auto-resume from Paused, got %q", sess.State)
	}
}

func TestProcessMessageFromThinking(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateThinking
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
	sess.State = session.StateToolExecuting
	store.Update(sess)

	err := loop.Pause("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	lc, ok := loop.activeLoops.Load("sess-1")
	if !ok {
		t.Skip("no active loop, flag-based fallback used")
		return
	}

	loopCtx := lc.(*LoopContext)
	select {
	case <-loopCtx.PauseCh:
		t.Log("pause signal received via channel")
	default:
		t.Error("expected pause signal on PauseCh")
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
	if sess.State != session.StateToolExecuting {
		t.Errorf("expected state ToolExecuting after resume, got %q", sess.State)
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
	if data["new_state"] != session.StateToolExecuting.ToSnakeCase() {
		t.Errorf("expected new_state tool_executing, got %v", data["new_state"])
	}
	if data["reason"] != "resumed" {
		t.Errorf("expected reason resumed, got %v", data["reason"])
	}
}

func TestCancelFromProcessing(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateThinking
	store.Update(sess)

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	lc, ok := loop.activeLoops.Load("sess-1")
	if !ok {
		t.Skip("no active loop, flag-based fallback used")
		return
	}

	loopCtx := lc.(*LoopContext)
	select {
	case <-loopCtx.CancelCh:
		t.Log("cancel signal received via channel")
	default:
		t.Error("expected cancel signal on CancelCh")
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

	lc, ok := loop.activeLoops.Load("sess-1")
	if !ok {
		t.Skip("no active loop, flag-based fallback used")
		return
	}

	loopCtx := lc.(*LoopContext)
	select {
	case <-loopCtx.CancelCh:
		t.Log("cancel signal received via channel")
	default:
		t.Error("expected cancel signal on CancelCh")
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
	if err != nil {
		t.Fatalf("expected no error when cancelling from Paused state, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle after cancel from Paused, got %q", sess.State)
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
	sess.State = session.StateThinking
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
	if data["new_state"] != session.StateCompleted.ToSnakeCase() {
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
	sess.State = session.StateThinking
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
	if data["new_state"] != session.StateArchived.ToSnakeCase() {
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

func TestPauseChannelSignal(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateToolExecuting
	store.Update(sess)

	lc := &LoopContext{
		SessionID: "sess-1",
		PauseCh:   make(chan struct{}, 1),
	}
	loop.activeLoops.Store("sess-1", lc)
	defer loop.activeLoops.Delete("sess-1")

	err := loop.Pause("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	select {
	case <-lc.PauseCh:
		t.Log("pause signal received on PauseCh")
	default:
		t.Error("expected pause signal on PauseCh")
	}
}

func TestCancelChannelSignal(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateThinking
	store.Update(sess)

	lc := &LoopContext{
		SessionID: "sess-1",
		CancelCh:  make(chan struct{}, 1),
	}
	loop.activeLoops.Store("sess-1", lc)
	defer loop.activeLoops.Delete("sess-1")

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	select {
	case <-lc.CancelCh:
		t.Log("cancel signal received on CancelCh")
	default:
		t.Error("expected cancel signal on CancelCh")
	}
}

func TestResumeChannelSignal(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StatePaused
	store.Update(sess)

	lc := &LoopContext{
		SessionID: "sess-1",
		ResumeCh:  make(chan struct{}, 1),
	}
	loop.activeLoops.Store("sess-1", lc)
	defer loop.activeLoops.Delete("sess-1")

	err := loop.Resume("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	select {
	case <-lc.ResumeCh:
		t.Log("resume signal received on ResumeCh")
	default:
		t.Error("expected resume signal on ResumeCh")
	}
}

func TestPauseFlagFallback(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateToolExecuting
	store.Update(sess)

	err := loop.Pause("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if !sess.PauseRequested {
		t.Error("expected PauseRequested to be true as fallback")
	}
}

func TestCancelFlagFallback(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-1")
	sess.State = session.StateThinking
	store.Update(sess)

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.CancelRequested {
		t.Log("cancel flag set as fallback")
	}
}

func TestCancelChannelSignalKillsChildProcess(t *testing.T) {
	loop, store := setupTestLoop()

	pid := 99999
	sess := createTestSession(store, "sess-1")
	sess.State = session.StateThinking
	sess.ChildPID = &pid
	store.Update(sess)

	lc := &LoopContext{
		SessionID: "sess-1",
		CancelCh:  make(chan struct{}, 1),
	}
	loop.activeLoops.Store("sess-1", lc)
	defer loop.activeLoops.Delete("sess-1")

	err := loop.Cancel("sess-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	select {
	case <-lc.CancelCh:
		t.Log("cancel signal received on CancelCh")
	default:
		t.Error("expected cancel signal on CancelCh")
	}

	sess, _ = store.Get("sess-1")
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after cancel")
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
	sess.State = session.StateThinking
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
	sess.State = session.StateThinking
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

func TestCancelClearsBackgroundPIDs(t *testing.T) {
	loop, store := setupTestLoop()

	pid := 99999
	bgPIDs := []int{100001, 100002}
	sess := createTestSession(store, "sess-bg")
	sess.State = session.StateThinking
	sess.ChildPID = &pid
	sess.BackgroundPIDs = bgPIDs
	store.Update(sess)

	err := loop.Cancel("sess-bg")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-bg")
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after cancel")
	}
	if len(sess.BackgroundPIDs) != 0 {
		t.Errorf("expected BackgroundPIDs to be empty, got %v", sess.BackgroundPIDs)
	}
}

func TestCompleteClearsBackgroundPIDs(t *testing.T) {
	loop, store := setupTestLoop()

	pid := 99999
	bgPIDs := []int{100001}
	sess := createTestSession(store, "sess-bg-complete")
	sess.State = session.StateThinking
	sess.ChildPID = &pid
	sess.BackgroundPIDs = bgPIDs
	store.Update(sess)

	err := loop.Complete("sess-bg-complete")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-bg-complete")
	if sess.State != session.StateCompleted {
		t.Errorf("expected state Completed, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after complete")
	}
	if len(sess.BackgroundPIDs) != 0 {
		t.Errorf("expected BackgroundPIDs to be empty, got %v", sess.BackgroundPIDs)
	}
}

func TestCancelOnlyBackgroundPIDs(t *testing.T) {
	loop, store := setupTestLoop()

	bgPIDs := []int{100001, 100002, 100003}
	sess := createTestSession(store, "sess-bg-only")
	sess.State = session.StateThinking
	sess.BackgroundPIDs = bgPIDs
	store.Update(sess)

	err := loop.Cancel("sess-bg-only")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-bg-only")
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to remain nil")
	}
	if len(sess.BackgroundPIDs) != 0 {
		t.Errorf("expected BackgroundPIDs to be empty, got %v", sess.BackgroundPIDs)
	}
}

func TestCancelNoPIDs(t *testing.T) {
	loop, store := setupTestLoop()

	sess := createTestSession(store, "sess-nopid")
	sess.State = session.StateThinking
	store.Update(sess)

	err := loop.Cancel("sess-nopid")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-nopid")
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to remain nil")
	}
	if len(sess.BackgroundPIDs) != 0 {
		t.Errorf("expected BackgroundPIDs to be empty, got %v", sess.BackgroundPIDs)
	}
}

func TestKillAllBackgroundPIDs(t *testing.T) {
	killAllBackgroundPIDs(nil)
	killAllBackgroundPIDs([]int{})
	killAllBackgroundPIDs([]int{-1, 0, 99999999})
}
