package session

import (
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test Session",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
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

func TestCreateDuplicate(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}

	if err := store.Create(sess); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	err := store.Create(sess)
	if err != ErrSessionConflict {
		t.Errorf("expected ErrSessionConflict, got: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	sess.Title = "Updated Title"
	sess.State = StateToolExecuting

	err := store.Update(sess)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := store.Get("sess-1")
	if got.Title != "Updated Title" {
		t.Errorf("expected Title %q, got %q", "Updated Title", got.Title)
	}
	if got.State != StateToolExecuting {
		t.Errorf("expected State %q, got %q", StateToolExecuting, got.State)
	}
}

func TestUpdateNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.Update(&Session{ID: "nonexistent"})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID("test")
	id2 := GenerateID("test")

	if id1 == id2 {
		t.Error("expected different IDs")
	}
}

func TestSessionMutationIsolation(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	got, _ := store.Get("sess-1")
	got.Title = "mutated via get"

	gotAgain, _ := store.Get("sess-1")
	if gotAgain.Title == "mutated via get" {
		t.Error("Get should return a copy, not a reference - store was mutated")
	}
}

func TestUpdatePreservesEventBus(t *testing.T) {
	store := NewInMemoryStore()

	sess := &Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		State:            StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	store.Create(sess)

	originalEB, _ := store.GetEventBus("sess-1")

	sess.State = StateToolExecuting
	store.Update(sess)

	updatedEB, _ := store.GetEventBus("sess-1")
	if originalEB != updatedEB {
		t.Error("EventBus should be preserved across Update calls")
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    State
		expected string
	}{
		{StateIdle, "idle"},
		{StateThinking, "thinking"},
		{StateToolExecuting, "tool_executing"},
		{StateAwaitingApproval, "awaiting_approval"},
		{StatePaused, "paused"},
		{StateCompleted, "completed"},
		{StateArchived, "archived"},
	}

	for _, tt := range tests {
		result := tt.input.ToSnakeCase()
		if result != tt.expected {
			t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestListSessions(t *testing.T) {
	store := NewInMemoryStore()

	sess1 := &Session{ID: "sess-1", Title: "Test 1", WorkingDirectory: "/tmp/proj1", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()}
	sess2 := &Session{ID: "sess-2", Title: "Test 2", WorkingDirectory: "/tmp/proj2", State: StateThinking, CreatedAt: time.Now(), LastActiveAt: time.Now()}
	sess3 := &Session{ID: "sess-3", Title: "Test 3", WorkingDirectory: "/tmp/proj1", State: StateCompleted, CreatedAt: time.Now(), LastActiveAt: time.Now()}

	store.Create(sess1)
	store.Create(sess2)
	store.Create(sess3)

	t.Run("all", func(t *testing.T) {
		sessions, total, err := store.ListSessions("all", "", 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Errorf("expected 3 sessions, got %d", total)
		}
		if len(sessions) != 3 {
			t.Errorf("expected 3 sessions, got %d", len(sessions))
		}
	})

	t.Run("filter_by_status", func(t *testing.T) {
		sessions, total, err := store.ListSessions("Idle", "", 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1 session, got %d", total)
		}
		if len(sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(sessions))
		}
	})

	t.Run("filter_by_project", func(t *testing.T) {
		_, total, err := store.ListSessions("", "/tmp/proj1", 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2 sessions, got %d", total)
		}
	})

	t.Run("limit_offset", func(t *testing.T) {
		sessions, total, err := store.ListSessions("all", "", 1, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(sessions))
		}
	})

	t.Run("offset_beyond", func(t *testing.T) {
		sessions, _, err := store.ListSessions("all", "", 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions, got %d", len(sessions))
		}
	})
}

func TestListUniqueWorkspaces(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/proj1", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.Create(&Session{ID: "sess-2", WorkingDirectory: "/tmp/proj2", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.Create(&Session{ID: "sess-3", WorkingDirectory: "/tmp/proj1", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	dirs, err := store.ListUniqueWorkspaces()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dirs) != 2 {
		t.Errorf("expected 2 unique workspaces, got %d", len(dirs))
	}
}

func TestDeleteByWorkspace(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/proj1", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.Create(&Session{ID: "sess-2", WorkingDirectory: "/tmp/proj2", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.Create(&Session{ID: "sess-3", WorkingDirectory: "/tmp/proj1", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	count, err := store.DeleteByWorkspace("/tmp/proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	_, err = store.Get("sess-1")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
	_, err = store.Get("sess-2")
	if err != nil {
		t.Errorf("sess-2 should still exist: %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	err := store.DeleteSession("sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.Get("sess-1")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	err := store.DeleteSession("nonexistent")
	if err != nil {
		t.Errorf("DeleteSession should not return error for nonexistent: %v", err)
	}
}

func TestAddAndGetEvents(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	err := store.AddEvent("sess-1", Event{Type: "test", Data: "data1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	store.AddEvent("sess-1", Event{Type: "test", Data: "data2"})

	events, err := store.GetEvents("sess-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestAddEvent_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	err := store.AddEvent("nonexistent", Event{Type: "test"})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestGetEvents_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	_, err := store.GetEvents("nonexistent", 0)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestUsageSteps(t *testing.T) {
	store := NewInMemoryStore()

	err := store.AddUsageStep("sess-1", 1, 100, 50, "estimated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	store.AddUsageStep("sess-1", 2, 200, 100, "actual")

	steps, err := store.GetUsageSteps("sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", steps[0].InputTokens)
	}
	if steps[1].OutputTokens != 100 {
		t.Errorf("expected 100 output tokens, got %d", steps[1].OutputTokens)
	}
}

func TestUpdateSessionUsage(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	err := store.UpdateSessionUsage("sess-1", 100, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, _ := store.Get("sess-1")
	if sess.TokenUsage.Input != 100 {
		t.Errorf("expected 100 input tokens, got %d", sess.TokenUsage.Input)
	}
	if sess.TokenUsage.Output != 50 {
		t.Errorf("expected 50 output tokens, got %d", sess.TokenUsage.Output)
	}
	if sess.TokenUsage.Total != 150 {
		t.Errorf("expected 150 total tokens, got %d", sess.TokenUsage.Total)
	}
}

func TestUpdateSessionUsage_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	err := store.UpdateSessionUsage("nonexistent", 100, 50)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestGetUsageStats(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/proj1", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.Create(&Session{ID: "sess-2", WorkingDirectory: "/tmp/proj2", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	store.UpdateSessionUsage("sess-1", 100, 50)
	store.UpdateSessionUsage("sess-2", 200, 100)

	t.Run("by_project", func(t *testing.T) {
		result, err := store.GetUsageStats("project", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Groups) != 2 {
			t.Errorf("expected 2 groups, got %d", len(result.Groups))
		}
		if result.Summary.Input != 300 {
			t.Errorf("expected 300 total input, got %d", result.Summary.Input)
		}
	})

	t.Run("by_session", func(t *testing.T) {
		result, err := store.GetUsageStats("session", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Groups) != 2 {
			t.Errorf("expected 2 groups, got %d", len(result.Groups))
		}
	})

	t.Run("by_date", func(t *testing.T) {
		result, err := store.GetUsageStats("date", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Summary.Total != 450 {
			t.Errorf("expected 450 total, got %d", result.Summary.Total)
		}
	})

	t.Run("default", func(t *testing.T) {
		result, err := store.GetUsageStats("", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Groups) != 1 {
			t.Errorf("expected 1 group, got %d", len(result.Groups))
		}
	})

	t.Run("filter_by_project", func(t *testing.T) {
		result, err := store.GetUsageStats("project", "", "/tmp/proj1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Groups) != 1 {
			t.Errorf("expected 1 group, got %d", len(result.Groups))
		}
	})
}

func TestClose(t *testing.T) {
	store := NewInMemoryStore()
	err := store.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetMessageByID(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.AddMessage("sess-1", Message{ID: "msg-1", Role: RoleUser, Content: "hello"})
	store.AddMessage("sess-1", Message{ID: "msg-2", Role: RoleAssistant, Content: "hi"})

	t.Run("found", func(t *testing.T) {
		msg, err := store.GetMessageByID("sess-1", "msg-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.Content != "hello" {
			t.Errorf("expected 'hello', got %q", msg.Content)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := store.GetMessageByID("sess-1", "msg-999")
		if err != ErrMessageNotFound {
			t.Errorf("expected ErrMessageNotFound, got %v", err)
		}
	})

	t.Run("session_not_found", func(t *testing.T) {
		_, err := store.GetMessageByID("nonexistent", "msg-1")
		if err != ErrSessionNotFound {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})
}

func TestDeleteMessages(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.AddMessage("sess-1", Message{ID: "msg-1", Role: RoleUser, Content: "hello"})
	store.AddMessage("sess-1", Message{ID: "msg-2", Role: RoleAssistant, Content: "hi"})
	store.AddMessage("sess-1", Message{ID: "msg-3", Role: RoleUser, Content: "bye"})

	deleted, err := store.DeleteMessages("sess-1", []string{"msg-1", "msg-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	_, err = store.GetMessageByID("sess-1", "msg-1")
	if err != ErrMessageNotFound {
		t.Errorf("expected ErrMessageNotFound, got %v", err)
	}

	msg, _ := store.GetMessageByID("sess-1", "msg-2")
	if msg == nil {
		t.Error("msg-2 should still exist")
	}
}

func TestDeleteMessages_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	_, err := store.DeleteMessages("nonexistent", []string{"msg-1"})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestDeleteMessagesAfter(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.AddMessage("sess-1", Message{ID: "msg-1", Role: RoleUser, Content: "hello"})
	store.AddMessage("sess-1", Message{ID: "msg-2", Role: RoleAssistant, Content: "hi"})
	store.AddMessage("sess-1", Message{ID: "msg-3", Role: RoleUser, Content: "bye"})

	deleted, err := store.DeleteMessagesAfter("sess-1", "msg-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted (msg-2 and msg-3), got %d", deleted)
	}

	msgs, _, _ := store.GetMessages("sess-1", 0, 0)
	if len(msgs) != 1 {
		t.Errorf("expected 1 message remaining, got %d", len(msgs))
	}
}

func TestDeleteMessagesAfter_NotFound(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	_, err := store.DeleteMessagesAfter("sess-1", "msg-999")
	if err != ErrMessageNotFound {
		t.Errorf("expected ErrMessageNotFound, got %v", err)
	}

	_, err = store.DeleteMessagesAfter("nonexistent", "msg-1")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestFileModifications(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	err := store.RecordFileModification(FileModificationRecord{
		SessionID:         "sess-1",
		FilePath:          "/tmp/test/file1.go",
		ModifiedAt:        time.Now(),
		CausedByMessageID: "msg-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records, err := store.GetFileModifications("sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	if records[0].FilePath != "/tmp/test/file1.go" {
		t.Errorf("expected '/tmp/test/file1.go', got %q", records[0].FilePath)
	}
}

func TestRecordFileModification_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	err := store.RecordFileModification(FileModificationRecord{
		SessionID: "nonexistent",
		FilePath:  "/tmp/test/file.go",
	})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestDeleteFileModificationsAfter(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	now := time.Now()
	store.RecordFileModification(FileModificationRecord{
		SessionID: "sess-1", FilePath: "/tmp/test/file1.go", ModifiedAt: now.Add(-2 * time.Hour),
	})
	store.RecordFileModification(FileModificationRecord{
		SessionID: "sess-1", FilePath: "/tmp/test/file2.go", ModifiedAt: now,
	})

	err := store.DeleteFileModificationsAfter("sess-1", now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records, _ := store.GetFileModifications("sess-1")
	if len(records) != 1 {
		t.Errorf("expected 1 record remaining, got %d", len(records))
	}
}

func TestGetLastMessages(t *testing.T) {
	store := NewInMemoryStore()

	store.Create(&Session{ID: "sess-1", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})
	store.Create(&Session{ID: "sess-2", WorkingDirectory: "/tmp/test", State: StateIdle, CreatedAt: time.Now(), LastActiveAt: time.Now()})

	store.AddMessage("sess-1", Message{ID: "msg-1", Role: RoleUser, Content: "hello from 1", CreatedAt: time.Now()})
	store.AddMessage("sess-1", Message{ID: "msg-2", Role: RoleAssistant, Content: "assistant reply", CreatedAt: time.Now()})
	store.AddMessage("sess-2", Message{ID: "msg-3", Role: RoleUser, Content: "hello from 2", CreatedAt: time.Now()})

	result, err := store.GetLastMessages([]string{"sess-1", "sess-2", "sess-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if result["sess-1"].Content != "hello from 1" {
		t.Errorf("expected 'hello from 1', got %q", result["sess-1"].Content)
	}
	if result["sess-2"].Content != "hello from 2" {
		t.Errorf("expected 'hello from 2', got %q", result["sess-2"].Content)
	}
}

func TestSession_AgentID(t *testing.T) {
	sess := Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: "/tmp/test",
		AgentID:          "python-expert",
		State:            StateIdle,
	}
	if sess.AgentID != "python-expert" {
		t.Errorf("expected AgentID 'python-expert', got %q", sess.AgentID)
	}
}

func TestSession_MaxConcurrentFields(t *testing.T) {
	sess := Session{
		ID:                        "sess-1",
		MaxConcurrentToolCalls:    3,
		MaxConcurrentSubprocesses: 10,
	}
	if sess.MaxConcurrentToolCalls != 3 {
		t.Errorf("expected MaxConcurrentToolCalls 3, got %d", sess.MaxConcurrentToolCalls)
	}
	if sess.MaxConcurrentSubprocesses != 10 {
		t.Errorf("expected MaxConcurrentSubprocesses 10, got %d", sess.MaxConcurrentSubprocesses)
	}
}

func TestSession_ApprovalPolicy(t *testing.T) {
	sess := Session{
		ID:             "sess-1",
		ApprovalPolicy: map[string]string{"write_file": "always_approve", "execute": "always_ask"},
	}
	if sess.ApprovalPolicy["write_file"] != "always_approve" {
		t.Errorf("expected 'always_approve', got %q", sess.ApprovalPolicy["write_file"])
	}
}

func TestSession_ApprovalTimeoutSeconds(t *testing.T) {
	sess := Session{
		ID:                     "sess-1",
		ApprovalTimeoutSeconds: 120,
	}
	if sess.ApprovalTimeoutSeconds != 120 {
		t.Errorf("expected 120, got %d", sess.ApprovalTimeoutSeconds)
	}
}

func TestMessage_Fields(t *testing.T) {
	msg := Message{
		ID:           "msg-1",
		Role:         RoleUser,
		Content:      "hello",
		Reasoning:    "thinking...",
		ToolCalls:    []ToolCall{{ID: "tc-1", ToolName: "read_file", Params: map[string]interface{}{"path": "/tmp/test"}}},
		ToolCallID:   "tool-1",
		ContentParts: []ContentPart{{Type: "text", Text: "hello"}},
		CreatedAt:    time.Now(),
	}
	if msg.ID != "msg-1" {
		t.Errorf("expected ID 'msg-1', got %q", msg.ID)
	}
	if msg.Role != RoleUser {
		t.Errorf("expected RoleUser, got %q", msg.Role)
	}
	if len(msg.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	if len(msg.ContentParts) != 1 {
		t.Errorf("expected 1 content part, got %d", len(msg.ContentParts))
	}
}

func TestErrors(t *testing.T) {
	errors := []error{
		ErrSessionNotFound,
		ErrSessionNotIdle,
		ErrSessionConflict,
		ErrSessionArchived,
		ErrSessionNotPaused,
		ErrSessionNotCompleted,
		ErrSessionNotProcessing,
		ErrSessionNotCancellable,
		ErrMessageNotFound,
	}

	for _, e := range errors {
		if e.Error() == "" {
			t.Errorf("expected non-empty error message for %v", e)
		}
	}
}

func TestUsageStepRecord(t *testing.T) {
	record := UsageStepRecord{
		SessionID:    "sess-1",
		StepSeq:      1,
		InputTokens:  100,
		OutputTokens: 50,
		Source:       "estimated",
		CreatedAt:    time.Now(),
	}
	if record.StepSeq != 1 {
		t.Errorf("expected StepSeq 1, got %d", record.StepSeq)
	}
}

func TestFileModificationRecord(t *testing.T) {
	now := time.Now()
	record := FileModificationRecord{
		SessionID:         "sess-1",
		FilePath:          "/tmp/test/file.go",
		ModifiedAt:        now,
		CausedByMessageID: "msg-1",
	}
	if record.FilePath != "/tmp/test/file.go" {
		t.Errorf("expected '/tmp/test/file.go', got %q", record.FilePath)
	}
}

func TestLoopTerminationReason(t *testing.T) {
	reasons := []LoopTerminationReason{
		LoopTerminationCompleted,
		LoopTerminationCancelled,
		LoopTerminationToolLimitReached,
		LoopTerminationError,
	}

	for _, r := range reasons {
		if string(r) == "" {
			t.Errorf("expected non-empty termination reason")
		}
	}
}

func TestSession_ArchivePath(t *testing.T) {
	sess := Session{
		ID:          "sess-1",
		ArchivePath: "/tmp/archive/sess-1.md",
	}
	if sess.ArchivePath != "/tmp/archive/sess-1.md" {
		t.Errorf("expected ArchivePath '/tmp/archive/sess-1.md', got %q", sess.ArchivePath)
	}
}

func TestSession_ActiveSkills(t *testing.T) {
	sess := Session{
		ID:           "sess-1",
		ActiveSkills: []string{"skill-1", "skill-2"},
	}
	if len(sess.ActiveSkills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(sess.ActiveSkills))
	}
}

func TestSession_CachedDirectorySummary(t *testing.T) {
	summary := DirectorySummary{
		Content:     "test content",
		GeneratedAt: time.Now(),
		Valid:       true,
	}
	sess := Session{
		ID:                     "sess-1",
		CachedDirectorySummary: &summary,
	}
	if sess.CachedDirectorySummary == nil {
		t.Fatal("expected non-nil CachedDirectorySummary")
	}
	if sess.CachedDirectorySummary.Content != "test content" {
		t.Errorf("expected 'test content', got %q", sess.CachedDirectorySummary.Content)
	}
}
