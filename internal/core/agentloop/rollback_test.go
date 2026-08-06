package agentloop

import (
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestRollbackNormalMessage(t *testing.T) {
	loop, store := setupTestLoop()
	sess := createTestSession(store, "sess-1")

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now().Add(-10 * time.Minute)},
		{ID: "msg-2", Role: session.RoleAssistant, Content: "Hi there!", CreatedAt: time.Now().Add(-9 * time.Minute)},
		{ID: "msg-3", Role: session.RoleUser, Content: "Write code", CreatedAt: time.Now().Add(-8 * time.Minute)},
		{ID: "msg-4", Role: session.RoleAssistant, Content: "Sure, here's code", CreatedAt: time.Now().Add(-7 * time.Minute)},
	}
	for _, m := range msgs {
		store.AddMessage("sess-1", m)
	}

	sess.State = session.StateIdle
	store.Update(sess)

	result, err := loop.Rollback("sess-1", "msg-3")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Adjusted {
		t.Error("expected no adjustment for normal message")
	}

	remaining, total, _ := store.GetMessages("sess-1", 0, 0)
	if total != 3 {
		t.Fatalf("expected 3 messages (2 original + 1 system), got %d", total)
	}

	if remaining[0].ID != "msg-1" {
		t.Errorf("expected msg-1 first, got %s", remaining[0].ID)
	}
	if remaining[1].ID != "msg-2" {
		t.Errorf("expected msg-2 second, got %s", remaining[1].ID)
	}
	if remaining[2].Role != session.RoleSystem {
		t.Errorf("expected last message to be system message, got %s", remaining[2].Role)
	}
}

func TestRollbackToolCallRequestAdsorption(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Write a file", CreatedAt: time.Now().Add(-10 * time.Minute)},
		{
			ID:        "msg-2",
			Role:      session.RoleAssistant,
			Content:   "",
			ToolCalls: []session.ToolCall{{ID: "tc-1", ToolName: "write_file", Params: map[string]interface{}{"path": "test.txt"}}},
			CreatedAt: time.Now().Add(-9 * time.Minute),
		},
		{ID: "msg-3", Role: session.RoleTool, Content: "File created", ToolCallID: "tc-1", CreatedAt: time.Now().Add(-8 * time.Minute)},
		{ID: "msg-4", Role: session.RoleAssistant, Content: "Done!", CreatedAt: time.Now().Add(-7 * time.Minute)},
	}
	for _, m := range msgs {
		store.AddMessage("sess-1", m)
	}

	result, err := loop.Rollback("sess-1", "msg-2")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Adjusted {
		t.Fatal("expected adjustment info for tool call request message")
	}
	if result.ActualRollbackMessageID != "msg-2" {
		t.Errorf("expected rollback to msg-2 (the assistant tool call itself), got %s", result.ActualRollbackMessageID)
	}

	remaining, total, _ := store.GetMessages("sess-1", 0, 0)
	if total != 2 {
		t.Fatalf("expected 2 messages (msg-1 + system), got %d", total)
	}

	if remaining[0].ID != "msg-1" {
		t.Errorf("expected msg-1 first, got %s", remaining[0].ID)
	}
	if remaining[1].Role != session.RoleSystem {
		t.Errorf("expected system message last, got %s", remaining[1].Role)
	}

	sysMsg := remaining[1]
	if sysMsg.Content == "" {
		t.Error("system message should have content")
	}
}

func TestRollbackToolResultAdsorption(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Write a file", CreatedAt: time.Now().Add(-10 * time.Minute)},
		{
			ID:        "msg-2",
			Role:      session.RoleAssistant,
			Content:   "",
			ToolCalls: []session.ToolCall{{ID: "tc-1", ToolName: "write_file", Params: map[string]interface{}{"path": "test.txt"}}},
			CreatedAt: time.Now().Add(-9 * time.Minute),
		},
		{ID: "msg-3", Role: session.RoleTool, Content: "File created", ToolCallID: "tc-1", CreatedAt: time.Now().Add(-8 * time.Minute)},
		{ID: "msg-4", Role: session.RoleAssistant, Content: "Done!", CreatedAt: time.Now().Add(-7 * time.Minute)},
	}
	for _, m := range msgs {
		store.AddMessage("sess-1", m)
	}

	result, err := loop.Rollback("sess-1", "msg-3")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Adjusted {
		t.Fatal("expected adjustment for tool result message")
	}
	if result.ActualRollbackMessageID != "msg-2" {
		t.Errorf("expected rollback to msg-2 (assistant tool call before tool result), got %s", result.ActualRollbackMessageID)
	}

	remaining, total, _ := store.GetMessages("sess-1", 0, 0)
	if total != 2 {
		t.Fatalf("expected 2 messages (msg-1 + system), got %d", total)
	}

	if remaining[0].ID != "msg-1" {
		t.Errorf("expected msg-1 first, got %s", remaining[0].ID)
	}
	if remaining[1].Role != session.RoleSystem {
		t.Errorf("expected system message last, got %s", remaining[1].Role)
	}
}

func TestRollbackFileStateWarning(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	fileModTime := time.Now().Add(-5 * time.Minute)
	msgTime := time.Now().Add(-10 * time.Minute)

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: msgTime},
		{
			ID:        "msg-2",
			Role:      session.RoleAssistant,
			Content:   "",
			ToolCalls: []session.ToolCall{{ID: "tc-1", ToolName: "write_file", Params: map[string]interface{}{"path": "src/main.go"}}},
			CreatedAt: msgTime.Add(1 * time.Minute),
		},
		{ID: "msg-3", Role: session.RoleTool, Content: "File created", ToolCallID: "tc-1", CreatedAt: msgTime.Add(2 * time.Minute)},
		{ID: "msg-4", Role: session.RoleAssistant, Content: "Done!", CreatedAt: msgTime.Add(3 * time.Minute)},
	}
	for _, m := range msgs {
		store.AddMessage("sess-1", m)
	}

	store.RecordFileModification(session.FileModificationRecord{
		SessionID:         "sess-1",
		FilePath:          "src/main.go",
		ModifiedAt:        fileModTime,
		CausedByMessageID: "msg-2",
	})

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	_, err := loop.Rollback("sess-1", "msg-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "file_state_warning", 1*time.Second)
	if !ok {
		t.Fatal("expected file_state_warning event")
	}

	data, _ := evt.Data.(map[string]any)
	if data["message"] == nil || data["message"].(string) == "" {
		t.Error("file_state_warning should have message")
	}

	files, _ := data["affected_files"].([]string)
	if len(files) == 0 {
		t.Error("expected affected files in warning")
	}
}

func TestRollbackNoFileWarningWhenNoModifications(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	msgTime := time.Now().Add(-10 * time.Minute)

	msgs := []session.Message{
		{ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: msgTime},
		{ID: "msg-2", Role: session.RoleAssistant, Content: "Hi!", CreatedAt: msgTime.Add(1 * time.Minute)},
	}
	for _, m := range msgs {
		store.AddMessage("sess-1", m)
	}

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	_, err := loop.Rollback("sess-1", "msg-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, ok := waitForEvent(ch, "file_state_warning", 500*time.Millisecond)
	if ok {
		t.Error("expected no file_state_warning when no file modifications")
	}
}

func TestRollbackStateResetToIdle(t *testing.T) {
	loop, store := setupTestLoop()
	sess := createTestSession(store, "sess-1")

	sess.State = session.StateThinking
	store.Update(sess)

	store.AddMessage("sess-1", session.Message{
		ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now(),
	})

	_, err := loop.Rollback("sess-1", "msg-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.State != session.StateIdle {
		t.Errorf("expected state Idle after rollback, got %s", sess.State)
	}
}

func TestRollbackArchivedSessionRejected(t *testing.T) {
	loop, store := setupTestLoop()
	sess := createTestSession(store, "sess-1")

	sess.State = session.StateArchived
	store.Update(sess)

	_, err := loop.Rollback("sess-1", "msg-1")
	if err == nil {
		t.Fatal("expected error when rolling back archived session")
	}
}

func TestRollbackMessageNotFound(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	_, err := loop.Rollback("sess-1", "nonexistent-msg")
	if err == nil {
		t.Fatal("expected error for nonexistent message")
	}
}

func TestRollbackCanContinueConversation(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	store.AddMessage("sess-1", session.Message{
		ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now(),
	})
	store.AddMessage("sess-1", session.Message{
		ID: "msg-2", Role: session.RoleAssistant, Content: "Hi!", CreatedAt: time.Now(),
	})

	_, err := loop.Rollback("sess-1", "msg-1")
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	err = loop.ProcessMessage(nil, "sess-1", session.Message{Content: "New message after rollback"})
	if err != nil {
		t.Fatalf("expected to be able to continue conversation after rollback, got: %v", err)
	}
}

func TestRollbackPublishesStateChangeEvent(t *testing.T) {
	loop, store := setupTestLoop()
	createTestSession(store, "sess-1")

	store.AddMessage("sess-1", session.Message{
		ID: "msg-1", Role: session.RoleUser, Content: "Hello", CreatedAt: time.Now(),
	})

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	_, err := loop.Rollback("sess-1", "msg-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	evt, ok := waitForEvent(ch, "session_state_change", 1*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event after rollback")
	}

	data, _ := evt.Data.(map[string]any)
	if data["new_state"] != session.StateIdle.ToSnakeCase() {
		t.Errorf("expected new_state Idle, got %v", data["new_state"])
	}
	if data["reason"] != "rollback" {
		t.Errorf("expected reason rollback, got %v", data["reason"])
	}
}
