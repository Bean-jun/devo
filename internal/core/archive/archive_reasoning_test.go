package archive

import (
	"os"
	"strings"
	"testing"
	"time"

	"devo/internal/core/concurrency"
	"devo/internal/core/session"
)

func TestAppendAssistantMessageWithReasoning(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendAssistantMessageWithReasoning(sess.ID, "最终答案", "思考过程ABC"); err != nil {
		t.Fatalf("append assistant message with reasoning: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)

	if !strings.Contains(text, "### 助手") {
		t.Error("archive missing assistant header")
	}
	if !strings.Contains(text, "<details>") {
		t.Error("archive missing <details> block for reasoning")
	}
	if !strings.Contains(text, "<summary>💭 思考过程</summary>") {
		t.Error("archive missing reasoning summary")
	}
	if !strings.Contains(text, "思考过程ABC") {
		t.Error("archive missing reasoning content")
	}
	if !strings.Contains(text, "最终答案") {
		t.Error("archive missing assistant content after reasoning")
	}

	if strings.Index(text, "<details>") >= strings.Index(text, "最终答案") {
		t.Error("reasoning block should appear before final content")
	}
}

func TestAppendAssistantMessageWithReasoning_EmptyReasoning(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendAssistantMessageWithReasoning(sess.ID, "no reasoning here", ""); err != nil {
		t.Fatalf("append assistant message without reasoning: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if strings.Contains(text, "<details>") {
		t.Error("archive should not contain <details> when reasoning is empty")
	}
	if !strings.Contains(text, "no reasoning here") {
		t.Error("archive missing content")
	}
}

func TestAppendAssistantMessage_BackwardCompatible(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := am.AppendAssistantMessage(sess.ID, "legacy call"); err != nil {
		t.Fatalf("legacy AppendAssistantMessage failed: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	if !strings.Contains(string(content), "legacy call") {
		t.Error("legacy AppendAssistantMessage lost content")
	}
	if strings.Contains(string(content), "<details>") {
		t.Error("legacy AppendAssistantMessage should not emit details block")
	}
}

func TestSyncArchive_RendersReasoning(t *testing.T) {
	store := session.NewInMemoryStore()
	am := NewArchiveManager(store, concurrency.NewPathLockManager())

	sess := newTestSession(t)
	if err := store.Create(sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "explain Go interfaces", CreatedAt: time.Now()},
		{
			ID:         "msg-2",
			Role:       session.RoleAssistant,
			Content:    "An interface in Go is a type that specifies a method set...",
			Reasoning:  "用户问 Go 接口，我应该从基本定义、隐式实现、应用场景三个角度回答。",
			CreatedAt:  time.Now(),
		},
	}

	for _, msg := range msgs {
		if err := store.AddMessage(sess.ID, msg); err != nil {
			t.Fatalf("add message: %v", err)
		}
	}

	if _, err := am.SyncArchive(sess.ID); err != nil {
		t.Fatalf("sync archive: %v", err)
	}

	content, err := os.ReadFile(am.ArchivePath(sess))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "<details>") {
		t.Error("synced archive missing <details> for reasoning")
	}
	if !strings.Contains(text, "用户问 Go 接口") {
		t.Error("synced archive missing reasoning content")
	}
	if !strings.Contains(text, "An interface in Go") {
		t.Error("synced archive missing assistant content")
	}
}
