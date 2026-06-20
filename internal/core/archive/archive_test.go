package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devo/internal/core/concurrency"
	"devo/internal/core/session"
)

func newTestSession(t *testing.T) *session.Session {
	t.Helper()
	return &session.Session{
		ID:               session.GenerateID("sess"),
		Title:            "Test Session",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage: session.TokenUsage{
			Input:  100,
			Output: 200,
			Total:  300,
		},
		CompressionCount: 0,
	}
}

func TestArchivePath(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	expected := filepath.Join(sess.WorkingDirectory, ".devo", "sessions", sess.ID+".md")

	path := am.ArchivePath(sess)
	if path != expected {
		t.Errorf("unexpected archive path: got %s, want %s", path, expected)
	}

	sess.ArchivePath = "/custom/path/archive.md"
	path = am.ArchivePath(sess)
	if path != "/custom/path/archive.md" {
		t.Errorf("custom archive path not respected: got %s", path)
	}
}

func TestAppendUserMessage(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendUserMessage(sess.ID, "Hello, World!"); err != nil {
		t.Fatalf("append user message: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	if !strings.Contains(string(content), "### 用户") {
		t.Error("archive missing user section header")
	}
	if !strings.Contains(string(content), "Hello, World!") {
		t.Error("archive missing user message content")
	}
}

func TestAppendAssistantMessage(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendAssistantMessage(sess.ID, "I'll help you with that."); err != nil {
		t.Fatalf("append assistant message: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	if !strings.Contains(string(content), "### 助手") {
		t.Error("archive missing assistant section header")
	}
	if !strings.Contains(string(content), "I'll help you with that.") {
		t.Error("archive missing assistant message content")
	}
}

func TestAppendSystemMessage(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendSystemMessage(sess.ID, "System notification."); err != nil {
		t.Fatalf("append system message: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	if !strings.Contains(string(content), "### 系统") {
		t.Error("archive missing system section header")
	}
	if !strings.Contains(string(content), "System notification.") {
		t.Error("archive missing system message content")
	}
}

func TestAppendToolCall(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	params := map[string]interface{}{
		"path":    "test.txt",
		"content": "hello",
	}

	if err := am.AppendToolCall(sess.ID, "write_file", params); err != nil {
		t.Fatalf("append tool call: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "[工具调用: write_file]") {
		t.Error("archive missing tool call annotation")
	}
	if !strings.Contains(text, "path=test.txt") {
		t.Error("archive missing tool call params")
	}

	if err := am.AppendToolCall(sess.ID, "read_file", nil); err != nil {
		t.Fatalf("append tool call with nil params: %v", err)
	}
}

func TestAppendToolResult(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendToolResult(sess.ID, "write_file", true, "File written successfully"); err != nil {
		t.Fatalf("append successful tool result: %v", err)
	}

	if err := am.AppendToolResult(sess.ID, "execute_command", false, "command failed"); err != nil {
		t.Fatalf("append failed tool result: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "结果(成功)") {
		t.Error("archive missing success result")
	}
	if !strings.Contains(text, "结果(失败)") {
		t.Error("archive missing failure result")
	}
}

func TestSyncArchive(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now()},
		{ID: "msg-2", Role: session.RoleAssistant, Content: "Hi there!", CreatedAt: time.Now()},
		{ID: "msg-3", Role: session.RoleUser, Content: "Write a file", CreatedAt: time.Now()},
		{ID: "msg-4", Role: session.RoleAssistant, Content: "", ToolCalls: []session.ToolCall{
			{ID: "tc-1", ToolName: "write_file", Params: map[string]interface{}{"path": "test.txt"}},
		}, CreatedAt: time.Now()},
		{ID: "msg-5", Role: session.RoleTool, Content: "File written successfully", ToolCallID: "tc-1", CreatedAt: time.Now()},
		{ID: "msg-6", Role: session.RoleAssistant, Content: "File has been written!", CreatedAt: time.Now()},
	}

	for _, msg := range msgs {
		if err := store.AddMessage(sess.ID, msg); err != nil {
			t.Fatalf("add message: %v", err)
		}
	}

	lastMsgID, err := am.SyncArchive(sess.ID)
	if err != nil {
		t.Fatalf("sync archive: %v", err)
	}

	if lastMsgID != "msg-6" {
		t.Errorf("unexpected last message ID: got %s, want msg-6", lastMsgID)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)

	checks := []struct {
		name   string
		needle string
	}{
		{"header", "# 会话存档"},
		{"session ID", sess.ID},
		{"state", string(sess.State)},
		{"working directory", sess.WorkingDirectory},
		{"token input", "输入 100"},
		{"token output", "输出 200"},
		{"token total", "合计 300"},
		{"compression count", "压缩次数"},
		{"user message", "Hello"},
		{"assistant message", "Hi there!"},
		{"tool call", "[工具调用: write_file]"},
		{"tool result", "File written successfully"},
		{"final assistant", "File has been written!"},
		{"dialog section", "## 对话历史"},
	}

	for _, c := range checks {
		if !strings.Contains(text, c.needle) {
			t.Errorf("archive missing '%s': expected '%s' in content", c.name, c.needle)
		}
	}
}

func TestSyncArchiveWithCompression(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	sess.CompressionCount = 2
	sess.CompressionState = &session.CompressionState{
		CompressedRanges: []session.CompressedRange{
			{StartMessageID: "msg-1", EndMessageID: "msg-3"},
		},
		Summaries: []session.CompressionSummary{
			{SummaryText: "First compression summary here.", CreatedAt: time.Now()},
			{SummaryText: "Second compression summary here.", CreatedAt: time.Now()},
		},
	}
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now()},
		{ID: "msg-2", Role: session.RoleAssistant, Content: "Hi", CreatedAt: time.Now()},
	}
	for _, msg := range msgs {
		store.AddMessage(sess.ID, msg)
	}

	_, err := am.SyncArchive(sess.ID)
	if err != nil {
		t.Fatalf("sync archive: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "## 上下文压缩摘要") {
		t.Error("archive missing compression summary appendix")
	}
	if !strings.Contains(text, "First compression summary here.") {
		t.Error("archive missing first compression summary")
	}
	if !strings.Contains(text, "Second compression summary here.") {
		t.Error("archive missing second compression summary")
	}
	if !strings.Contains(text, "压缩次数") {
		t.Error("archive missing compression count")
	}
}

func TestGetArchiveContent(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendUserMessage(sess.ID, "test content"); err != nil {
		t.Fatalf("append message: %v", err)
	}

	content, err := am.GetArchiveContent(sess.ID)
	if err != nil {
		t.Fatalf("get archive content: %v", err)
	}

	if content == nil {
		t.Fatal("expected non-nil content")
	}
	if !strings.Contains(string(content), "test content") {
		t.Error("archive content missing expected text")
	}

	nonExistentSess := &session.Session{
		ID:               "nonexistent",
		WorkingDirectory: t.TempDir(),
	}
	store.Create(nonExistentSess)
	content, err = am.GetArchiveContent("nonexistent")
	if err != nil {
		t.Fatalf("get archive content for non-existent file: %v", err)
	}
	if content != nil {
		t.Error("expected nil content for non-existent archive file")
	}
}

func TestArchiveExists(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	exists, err := am.ArchiveExists(sess.ID)
	if err != nil {
		t.Fatalf("check archive exists: %v", err)
	}
	if exists {
		t.Error("archive should not exist before any write")
	}

	if err := am.AppendSystemMessage(sess.ID, "test"); err != nil {
		t.Fatalf("append message: %v", err)
	}

	exists, err = am.ArchiveExists(sess.ID)
	if err != nil {
		t.Fatalf("check archive exists after write: %v", err)
	}
	if !exists {
		t.Error("archive should exist after write")
	}
}

func TestSyncArchiveOverwrite(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Original message", CreatedAt: time.Now()},
	}
	for _, msg := range msgs {
		store.AddMessage(sess.ID, msg)
	}

	if err := am.AppendUserMessage(sess.ID, "Manually added junk"); err != nil {
		t.Fatalf("append: %v", err)
	}

	archivePath := am.ArchivePath(sess)
	os.WriteFile(archivePath, []byte("Manual edit - should be overwritten"), 0644)

	_, err := am.SyncArchive(sess.ID)
	if err != nil {
		t.Fatalf("sync archive: %v", err)
	}

	content, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if strings.Contains(text, "Manual edit - should be overwritten") {
		t.Error("manual edit was not overwritten by sync")
	}
	if !strings.Contains(text, "Original message") {
		t.Error("sync archive missing database content")
	}
}

func TestEnsureArchivePath(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := am.EnsureArchivePath(sess); err != nil {
		t.Fatalf("ensure archive path: %v", err)
	}

	dir := filepath.Dir(am.ArchivePath(sess))
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat archive directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("archive directory is not a directory")
	}
}

func TestGetLastMessageID(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	id, err := am.GetLastMessageID(sess.ID)
	if err != nil {
		t.Fatalf("get last message ID (empty): %v", err)
	}
	if id != "" {
		t.Errorf("expected empty last message ID, got %s", id)
	}

	store.AddMessage(sess.ID, session.Message{ID: "msg-1", Role: session.RoleUser, Content: "a", CreatedAt: time.Now()})
	store.AddMessage(sess.ID, session.Message{ID: "msg-2", Role: session.RoleAssistant, Content: "b", CreatedAt: time.Now()})

	id, err = am.GetLastMessageID(sess.ID)
	if err != nil {
		t.Fatalf("get last message ID: %v", err)
	}
	if id != "msg-2" {
		t.Errorf("expected last message ID msg-2, got %s", id)
	}
}

func TestRenderArchiveSystemMessages(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleSystem, Content: "System started", CreatedAt: time.Now()},
		{ID: "msg-2", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now()},
		{ID: "msg-3", Role: session.RoleSystem, Content: "Mid-conversation system note", CreatedAt: time.Now()},
		{ID: "msg-4", Role: session.RoleAssistant, Content: "Response", CreatedAt: time.Now()},
	}

	for _, msg := range msgs {
		store.AddMessage(sess.ID, msg)
	}

	_, err := am.SyncArchive(sess.ID)
	if err != nil {
		t.Fatalf("sync archive: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "System started") {
		t.Error("archive missing system message")
	}
	if !strings.Contains(text, "Mid-conversation system note") {
		t.Error("archive missing mid-conversation system message")
	}
}

func TestRenderArchiveToolMessages(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleAssistant, Content: "", ToolCalls: []session.ToolCall{
			{ID: "tc-1", ToolName: "execute_command", Params: map[string]interface{}{"command": "ls"}},
		}, CreatedAt: time.Now()},
		{ID: "msg-2", Role: session.RoleTool, Content: "files: a.txt, b.txt", ToolCallID: "tc-1", CreatedAt: time.Now()},
	}

	for _, msg := range msgs {
		store.AddMessage(sess.ID, msg)
	}

	_, err := am.SyncArchive(sess.ID)
	if err != nil {
		t.Fatalf("sync archive: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "[工具调用: execute_command]") {
		t.Error("archive missing tool call in render")
	}
	if !strings.Contains(text, "command=ls") {
		t.Error("archive missing tool call params in render")
	}
	if !strings.Contains(text, "files: a.txt, b.txt") {
		t.Error("archive missing tool result in render")
	}
}

func TestAppendNonBlocking(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	archivePath := am.ArchivePath(sess)
	dirPath := filepath.Dir(archivePath)
	parentDir := filepath.Dir(dirPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("create parent dir: %v", err)
	}
	if err := os.WriteFile(dirPath, []byte("block"), 0644); err != nil {
		t.Fatalf("create blocking file: %v", err)
	}
	defer os.Remove(dirPath)

	err := am.AppendUserMessage(sess.ID, "test")
	if err == nil {
		t.Error("expected error when archive directory path is a file, but got nil")
	}
}
