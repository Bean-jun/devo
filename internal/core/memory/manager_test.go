package memory

import (
	"os"
	"path/filepath"
	"testing"

	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
)

func TestUpsert_NewMemory(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	mem, err := mgr.Upsert(TypeUser, "/tmp/test", "editor_pref", "I prefer 4 spaces", SourceManual)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.Key != "editor_pref" {
		t.Errorf("expected key editor_pref, got %s", mem.Key)
	}
	if mem.Content != "I prefer 4 spaces" {
		t.Errorf("expected content 'I prefer 4 spaces', got %s", mem.Content)
	}
	if mem.Type != TypeUser {
		t.Errorf("expected type user, got %s", mem.Type)
	}
}

func TestUpsert_UpdateExisting(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	_, err := mgr.Upsert(TypeUser, "/tmp/test", "editor_pref", "I prefer 4 spaces", SourceManual)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mem, err := mgr.Upsert(TypeUser, "/tmp/test", "editor_pref", "I prefer 2 spaces", SourceManual)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.Content != "I prefer 2 spaces" {
		t.Errorf("expected content 'I prefer 2 spaces', got %s", mem.Content)
	}

	sections, _ := mgr.List(TypeUser, "/tmp/test")
	if len(sections) != 1 {
		t.Errorf("expected 1 memory, got %d", len(sections))
	}
}

func TestAppend(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	_, err := mgr.Upsert(TypeUser, "/tmp/test", "notes", "line1", SourceManual)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = mgr.Append(TypeUser, "/tmp/test", "notes", "line2", SourceManual)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sections, _ := mgr.List(TypeUser, "/tmp/test")
	if len(sections) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(sections))
	}
	if sections[0].Content != "line1\nline2" {
		t.Errorf("expected content 'line1\\nline2', got %q", sections[0].Content)
	}
}

func TestGet(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	mem, _ := mgr.Upsert(TypeUser, "/tmp/test", "test_key", "test content", SourceManual)

	got, err := mgr.Get(mem.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Key != "test_key" {
		t.Errorf("expected key test_key, got %s", got.Key)
	}
}

func TestGet_NotFound(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	_, err := mgr.Get("nonexistent")
	if err != ErrMemoryNotFound {
		t.Errorf("expected ErrMemoryNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	mem, _ := mgr.Upsert(TypeUser, "/tmp/test", "temp_key", "temp content", SourceManual)

	err := mgr.Delete(mem.Type, "/tmp/test", mem.Key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = mgr.Get(mem.ID)
	if err != ErrMemoryNotFound {
		t.Errorf("expected ErrMemoryNotFound after delete, got %v", err)
	}
}

func TestList_ByType(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	wd := filepath.Join(t.TempDir(), "project1")
	os.MkdirAll(wd, 0755)

	mgr.Upsert(TypeUser, "/tmp/test", "key1", "val1", SourceManual)
	mgr.Upsert(TypeUser, "/tmp/test", "key2", "val2", SourceManual)
	mgr.Upsert(TypeProject, wd, "proj_key", "proj_val", SourceManual)

	userList, _ := mgr.List(TypeUser, "/tmp/test")
	if len(userList) != 2 {
		t.Errorf("expected 2 user memories, got %d", len(userList))
	}

	projList, _ := mgr.List(TypeProject, wd)
	if len(projList) != 1 {
		t.Errorf("expected 1 project memory, got %d", len(projList))
	}
}

func TestGetRelevantMemories(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	wd := filepath.Join(t.TempDir(), "project2")
	os.MkdirAll(wd, 0755)

	mgr.Upsert(TypeUser, "/tmp/test", "editor_pref", "4 spaces", SourceManual)
	mgr.Upsert(TypeProject, wd, "api_port", "8080", SourceManual)

	result := mgr.GetRelevantMemories(wd, "sess-1")
	if result == "" {
		t.Error("expected non-empty result")
	}
	if result != "" {
		t.Logf("GetRelevantMemories:\n%s", result)
	}
}

func TestDetectAutoUpdate(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	drafts := mgr.DetectAutoUpdate("记住这个 用4空格缩进", "sess-1")
	if len(drafts) == 0 {
		t.Error("expected drafts for '记住这个'")
	}

	drafts = mgr.DetectAutoUpdate("hello world", "sess-1")
	if len(drafts) != 0 {
		t.Error("expected no drafts for normal message")
	}
}

func TestProjectKey(t *testing.T) {
	store, _ := NewFileStore(t.TempDir())
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	mgr := NewManager(store, pathLock, approvalMgr)

	key1 := mgr.ProjectKey("/tmp/project-a")
	key2 := mgr.ProjectKey("/tmp/project-a")
	key3 := mgr.ProjectKey("/tmp/project-b")

	if key1 != key2 {
		t.Error("same path should produce same key")
	}
	if key1 == key3 {
		t.Error("different paths should produce different keys")
	}
}
