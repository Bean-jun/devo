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
