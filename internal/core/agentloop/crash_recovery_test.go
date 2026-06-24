package agentloop

import (
	"context"
	"fmt"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func TestRecoverCrashedSessions_ProcessingToIdle(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	pid := 99999
	sess := &session.Session{
		ID:               "sess-processing",
		Title:            "Processing Session",
		WorkingDirectory: "/tmp/test-processing",
		State:            session.StateProcessing,
		ChildPID:         &pid,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-processing")
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after recovery")
	}
	if sess.CancelRequested {
		t.Error("expected CancelRequested to be false after recovery")
	}
	if sess.PauseRequested {
		t.Error("expected PauseRequested to be false after recovery")
	}
}

func TestRecoverCrashedSessions_AwaitingApprovalToIdle(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	pid := 88888
	sess := &session.Session{
		ID:               "sess-awaiting",
		Title:            "Awaiting Approval Session",
		WorkingDirectory: "/tmp/test-awaiting",
		State:            session.StateAwaitingApproval,
		ChildPID:         &pid,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-awaiting")
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after recovery")
	}
}

func TestRecoverCrashedSessions_SystemMessageInserted(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	sess := &session.Session{
		ID:               "sess-sysmsg",
		Title:            "System Message Test",
		WorkingDirectory: "/tmp/test-sysmsg",
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	msgs, total, err := store.GetMessages("sess-sysmsg", 0, 0)
	if err != nil {
		t.Fatalf("get messages failed: %v", err)
	}
	if total == 0 {
		t.Fatal("expected at least one system message")
	}
	if msgs[0].Role != session.RoleSystem {
		t.Errorf("expected first message to be system, got %q", msgs[0].Role)
	}
	if msgs[0].Content != crashRecoverySystemMessage {
		t.Errorf("unexpected system message content: %q", msgs[0].Content)
	}
}

func TestRecoverCrashedSessions_PausedUnchanged(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	sess := &session.Session{
		ID:               "sess-paused",
		Title:            "Paused Session",
		WorkingDirectory: "/tmp/test-paused",
		State:            session.StatePaused,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-paused")
	if sess.State != session.StatePaused {
		t.Errorf("expected Paused state unchanged, got %q", sess.State)
	}

	msgs, _, _ := store.GetMessages("sess-paused", 0, 0)
	if len(msgs) != 0 {
		t.Errorf("expected no messages for Paused session, got %d", len(msgs))
	}
}

func TestRecoverCrashedSessions_IdleUnchanged(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	sess := &session.Session{
		ID:               "sess-idle",
		Title:            "Idle Session",
		WorkingDirectory: "/tmp/test-idle",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-idle")
	if sess.State != session.StateIdle {
		t.Errorf("expected Idle state unchanged, got %q", sess.State)
	}

	msgs, _, _ := store.GetMessages("sess-idle", 0, 0)
	if len(msgs) != 0 {
		t.Errorf("expected no messages for Idle session, got %d", len(msgs))
	}
}

func TestRecoverCrashedSessions_MixedStates(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	pid := 77777
	create := []*session.Session{
		{
			ID: "sess-1", Title: "Processing", WorkingDirectory: "/tmp/p1",
			State: session.StateProcessing, ChildPID: &pid,
			CreatedAt: time.Now(), LastActiveAt: time.Now(),
		},
		{
			ID: "sess-2", Title: "AwaitingApproval", WorkingDirectory: "/tmp/p2",
			State:     session.StateAwaitingApproval,
			CreatedAt: time.Now(), LastActiveAt: time.Now(),
		},
		{
			ID: "sess-3", Title: "Paused", WorkingDirectory: "/tmp/p3",
			State:     session.StatePaused,
			CreatedAt: time.Now(), LastActiveAt: time.Now(),
		},
		{
			ID: "sess-4", Title: "Idle", WorkingDirectory: "/tmp/p4",
			State:     session.StateIdle,
			CreatedAt: time.Now(), LastActiveAt: time.Now(),
		},
		{
			ID: "sess-5", Title: "Completed", WorkingDirectory: "/tmp/p5",
			State:     session.StateCompleted,
			CreatedAt: time.Now(), LastActiveAt: time.Now(),
		},
		{
			ID: "sess-6", Title: "Archived", WorkingDirectory: "/tmp/p6",
			State:     session.StateArchived,
			CreatedAt: time.Now(), LastActiveAt: time.Now(),
		},
	}
	for _, s := range create {
		store.Create(s)
	}

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess1, _ := store.Get("sess-1")
	if sess1.State != session.StateIdle {
		t.Errorf("sess-1: expected Idle, got %q", sess1.State)
	}
	if sess1.ChildPID != nil {
		t.Error("sess-1: expected ChildPID nil")
	}

	sess2, _ := store.Get("sess-2")
	if sess2.State != session.StateIdle {
		t.Errorf("sess-2: expected Idle, got %q", sess2.State)
	}

	sess3, _ := store.Get("sess-3")
	if sess3.State != session.StatePaused {
		t.Errorf("sess-3: expected Paused unchanged, got %q", sess3.State)
	}

	sess4, _ := store.Get("sess-4")
	if sess4.State != session.StateIdle {
		t.Errorf("sess-4: expected Idle unchanged, got %q", sess4.State)
	}

	sess5, _ := store.Get("sess-5")
	if sess5.State != session.StateCompleted {
		t.Errorf("sess-5: expected Completed unchanged, got %q", sess5.State)
	}

	sess6, _ := store.Get("sess-6")
	if sess6.State != session.StateArchived {
		t.Errorf("sess-6: expected Archived unchanged, got %q", sess6.State)
	}
}

func TestRecoverCrashedSessions_SSEEventPublished(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	sess := &session.Session{
		ID:               "sess-event",
		Title:            "Event Test",
		WorkingDirectory: "/tmp/test-event",
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	eb, err := store.GetEventBus("sess-event")
	if err != nil {
		t.Fatalf("get event bus: %v", err)
	}

	events := eb.GetHistory(0)
	found := false
	for _, evt := range events {
		if evt.Type == "session_state_change" {
			data, ok := evt.Data.(map[string]any)
			if ok && data["reason"] == "error" {
				found = true
				if data["new_state"] != session.StateIdle.ToSnakeCase() {
					t.Errorf("expected new_state Idle, got %v", data["new_state"])
				}
				if data["old_state"] != session.StateProcessing.ToSnakeCase() {
					t.Errorf("expected old_state Processing, got %v", data["old_state"])
				}
			}
		}
	}
	if !found {
		t.Error("expected session_state_change event with reason=error")
	}
}

func TestRecoverCrashedSessions_CanContinueAfterRecovery(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	sess := &session.Session{
		ID:               "sess-continue",
		Title:            "Continue Test",
		WorkingDirectory: "/tmp/test-continue",
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-continue")
	if sess.State != session.StateIdle {
		t.Fatalf("expected Idle after recovery, got %q", sess.State)
	}

	err := loop.ProcessMessage(context.Background(), "sess-continue", "Hello after crash")
	if err != nil {
		t.Fatalf("ProcessMessage after recovery failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	msgs, total, _ := store.GetMessages("sess-continue", 0, 0)
	if total < 3 {
		t.Errorf("expected at least 3 messages (system + user + assistant), got %d", total)
	}

	hasSystem := false
	hasUser := false
	for _, msg := range msgs {
		if msg.Role == session.RoleSystem && msg.Content == crashRecoverySystemMessage {
			hasSystem = true
		}
		if msg.Role == session.RoleUser && msg.Content == "Hello after crash" {
			hasUser = true
		}
	}
	if !hasSystem {
		t.Error("expected crash recovery system message")
	}
	if !hasUser {
		t.Error("expected user message after recovery")
	}
}

func TestRecoverCrashedSessions_EmptyStore(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions on empty store failed: %v", err)
	}
}

func TestRecoverCrashedSessions_NoChildPID(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	sess := &session.Session{
		ID:               "sess-nochild",
		Title:            "No Child PID",
		WorkingDirectory: "/tmp/test-nochild",
		State:            session.StateProcessing,
		ChildPID:         nil,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-nochild")
	if sess.State != session.StateIdle {
		t.Errorf("expected Idle, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to remain nil")
	}
}

func TestRecoverCrashedSessions_WithBackgroundPIDs(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	pid := 99999
	sess := &session.Session{
		ID:               "sess-bg-crash",
		Title:            "Background PID Crash",
		WorkingDirectory: "/tmp/test-bg-crash",
		State:            session.StateProcessing,
		ChildPID:         &pid,
		BackgroundPIDs:   []int{100001, 100002},
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-bg-crash")
	if sess.State != session.StateIdle {
		t.Errorf("expected Idle, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after recovery")
	}
	if len(sess.BackgroundPIDs) != 0 {
		t.Errorf("expected BackgroundPIDs to be empty, got %v", sess.BackgroundPIDs)
	}
}

func TestRecoverCrashedSessions_InvalidChildPID(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	invalidPID := -1
	sess := &session.Session{
		ID:               "sess-invalid",
		Title:            "Invalid PID",
		WorkingDirectory: "/tmp/test-invalid",
		State:            session.StateProcessing,
		ChildPID:         &invalidPID,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions failed: %v", err)
	}

	sess, _ = store.Get("sess-invalid")
	if sess.State != session.StateIdle {
		t.Errorf("expected Idle, got %q", sess.State)
	}
	if sess.ChildPID != nil {
		t.Error("expected ChildPID to be nil after recovery")
	}
}

func TestRecoverCrashedSessions_Pagination(t *testing.T) {
	store := session.NewInMemoryStore()
	loop := New(store, llmclient.NewMockClient())

	for i := 0; i < 150; i++ {
		sess := &session.Session{
			ID:               fmt.Sprintf("sess-pag-%d", i),
			Title:            fmt.Sprintf("Page Test %d", i),
			WorkingDirectory: fmt.Sprintf("/tmp/pag-%d", i),
			State:            session.StateProcessing,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		}
		store.Create(sess)
	}

	if err := loop.RecoverCrashedSessions(); err != nil {
		t.Fatalf("RecoverCrashedSessions with pagination failed: %v", err)
	}

	for i := 0; i < 150; i++ {
		sess, err := store.Get(fmt.Sprintf("sess-pag-%d", i))
		if err != nil {
			t.Errorf("sess-pag-%d: not found", i)
			continue
		}
		if sess.State != session.StateIdle {
			t.Errorf("sess-pag-%d: expected Idle, got %q", i, sess.State)
		}
	}
}
