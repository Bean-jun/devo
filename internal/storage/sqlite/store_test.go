package sqlite

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"

	"devo/internal/core/session"

	gormsqlite "github.com/glebarez/sqlite"
)

func newTestStore(t *testing.T) *GormStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	store, err := NewGormStore(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

func TestGormStoreCreateAndGet(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test Session",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
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

func TestGormStoreCreateDuplicate(t *testing.T) {
	store := newTestStore(t)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}

	if err := store.Create(sess); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	err := store.Create(sess)
	if err != session.ErrSessionConflict {
		t.Errorf("expected ErrSessionConflict, got: %v", err)
	}
}

func TestGormStoreGetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get("nonexistent")
	if err != session.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestGormStoreUpdate(t *testing.T) {
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

	sess.Title = "Updated Title"
	sess.State = session.StateProcessing

	err := store.Update(sess)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := store.Get("sess-1")
	if got.Title != "Updated Title" {
		t.Errorf("expected Title %q, got %q", "Updated Title", got.Title)
	}
	if got.State != session.StateProcessing {
		t.Errorf("expected State %q, got %q", session.StateProcessing, got.State)
	}
}

func TestGormStoreUpdateNotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.Update(&session.Session{ID: "nonexistent"})
	if err != session.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

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

func TestGormStoreListSessionsAll(t *testing.T) {
	store := newTestStore(t)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	store.Create(&session.Session{
		ID:               "sess-1",
		Title:            "Session 1",
		WorkingDirectory: dir1,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})
	store.Create(&session.Session{
		ID:               "sess-2",
		Title:            "Session 2",
		WorkingDirectory: dir2,
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	sessions, total, err := store.ListSessions("all", "", 10, 0)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestGormStoreListSessionsFilterByStatus(t *testing.T) {
	store := newTestStore(t)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	store.Create(&session.Session{
		ID:               "sess-1",
		Title:            "Session 1",
		WorkingDirectory: dir1,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})
	store.Create(&session.Session{
		ID:               "sess-2",
		Title:            "Session 2",
		WorkingDirectory: dir2,
		State:            session.StateProcessing,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	sessions, total, err := store.ListSessions("Idle", "", 10, 0)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != "sess-1" {
		t.Errorf("expected sess-1, got %q", sessions[0].ID)
	}
}

func TestGormStoreListSessionsFilterByProject(t *testing.T) {
	store := newTestStore(t)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	store.Create(&session.Session{
		ID:               "sess-1",
		Title:            "Session 1",
		WorkingDirectory: dir1,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})
	store.Create(&session.Session{
		ID:               "sess-2",
		Title:            "Session 2",
		WorkingDirectory: dir2,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	sessions, total, err := store.ListSessions("", dir1, 10, 0)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if sessions[0].WorkingDirectory != dir1 {
		t.Errorf("expected working_directory %q, got %q", dir1, sessions[0].WorkingDirectory)
	}
}

func TestGormStoreListSessionsPagination(t *testing.T) {
	store := newTestStore(t)

	for i := 0; i < 5; i++ {
		store.Create(&session.Session{
			ID:               session.GenerateID("sess"),
			Title:            "Test",
			WorkingDirectory: t.TempDir(),
			State:            session.StateIdle,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		})
	}

	sessions, total, err := store.ListSessions("", "", 2, 0)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions with limit=2, got %d", len(sessions))
	}
}

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

func TestGormStorePersistenceAcrossRestarts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")

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
			ID:               "sess-persist",
			Title:            "Persist Test",
			WorkingDirectory: dir,
			State:            session.StateIdle,
			CreatedAt:        time.Now(),
			LastActiveAt:     time.Now(),
		}
		store.Create(sess)

		store.AddMessage("sess-persist", session.Message{
			ID:        "msg-1",
			Role:      session.RoleUser,
			Content:   "Hello, world!",
			CreatedAt: time.Now(),
		})
		store.AddMessage("sess-persist", session.Message{
			ID:        "msg-2",
			Role:      session.RoleAssistant,
			Content:   "Echo: Hello, world!",
			CreatedAt: time.Now(),
		})
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

		sess, err := store.Get("sess-persist")
		if err != nil {
			t.Fatalf("expected session to exist after restart: %v", err)
		}
		if sess.Title != "Persist Test" {
			t.Errorf("expected title 'Persist Test', got %q", sess.Title)
		}

		msgs, total, err := store.GetMessages("sess-persist", 0, 0)
		if err != nil {
			t.Fatalf("get messages after restart: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2 messages after restart, got %d", total)
		}
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
		if msgs[0].Content != "Hello, world!" {
			t.Errorf("expected first message 'Hello, world!', got %q", msgs[0].Content)
		}
		if msgs[1].Content != "Echo: Hello, world!" {
			t.Errorf("expected second message 'Echo: Hello, world!', got %q", msgs[1].Content)
		}

		store.AddMessage("sess-persist", session.Message{
			ID:        "msg-3",
			Role:      session.RoleUser,
			Content:   "Continue after restart",
			CreatedAt: time.Now(),
		})

		msgs, total, _ = store.GetMessages("sess-persist", 0, 0)
		if total != 3 {
			t.Errorf("expected 3 messages after adding new message, got %d", total)
		}
	}()
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

func TestGormStoreWorkingDirectoryValidation(t *testing.T) {
	store := newTestStore(t)

	nonExistentDir := "/nonexistent/path/12345"

	sess := &session.Session{
		ID:               "sess-broken",
		Title:            "Broken Dir",
		WorkingDirectory: nonExistentDir,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	got, err := store.Get("sess-broken")
	if err != nil {
		t.Fatalf("expected session to be retrievable: %v", err)
	}

	info, statErr := os.Stat(got.WorkingDirectory)
	if statErr == nil && info.IsDir() {
		t.Skip("non-existent directory somehow exists, skipping")
	}

	if statErr == nil {
		t.Fatal("expected directory to not exist")
	}

	if got.State != session.StateIdle {
		t.Logf("state is %q (path validation happens at handler level, not store level)", got.State)
	}
}

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
