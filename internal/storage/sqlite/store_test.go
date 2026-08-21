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
	sess.State = session.StateToolExecuting

	err := store.Update(sess)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := store.Get("sess-1")
	if got.Title != "Updated Title" {
		t.Errorf("expected Title %q, got %q", "Updated Title", got.Title)
	}
	if got.State != session.StateToolExecuting {
		t.Errorf("expected State %q, got %q", session.StateToolExecuting, got.State)
	}
}

func TestGormStoreUpdateNotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.Update(&session.Session{ID: "nonexistent"})
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
		State:            session.StateThinking,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	sessions, total, err := store.ListSessions("all", "", "", 10, 0)
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
		State:            session.StateThinking,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	})

	sessions, total, err := store.ListSessions("Idle", "", "", 10, 0)
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

	sessions, total, err := store.ListSessions("", dir1, "", 10, 0)
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

	sessions, total, err := store.ListSessions("", "", "", 2, 0)
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
