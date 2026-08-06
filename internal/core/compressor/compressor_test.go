package compressor

import (
	"context"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func setupTestCompressor() (*Compressor, *session.InMemoryStore) {
	store := session.NewInMemoryStore()
	comp := New(llmclient.NewMockClient(), store)
	return comp, store
}

func createTestSession(store *session.InMemoryStore, id string, maxContextTokens, keepRecent int) *session.Session {
	sess := &session.Session{
		ID:               id,
		Title:            "Test Session",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		MaxContextTokens: maxContextTokens,
		KeepRecent:       keepRecent,
	}
	store.Create(sess)
	return sess
}

func addMessages(store *session.InMemoryStore, sessionID string, count int, role session.Role) {
	for i := 0; i < count; i++ {
		msg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      role,
			Content:   "Message number " + string(rune('0'+i%10)),
			CreatedAt: time.Now(),
		}
		store.AddMessage(sessionID, msg)
	}
}

func addMixedMessages(store *session.InMemoryStore, sessionID string, count int) {
	for i := 0; i < count; i++ {
		var role session.Role
		switch i % 3 {
		case 0:
			role = session.RoleUser
		case 1:
			role = session.RoleAssistant
		case 2:
			role = session.RoleSystem
		}
		msg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      role,
			Content:   "This is a mixed message for testing compression with various roles.",
			CreatedAt: time.Now(),
		}
		store.AddMessage(sessionID, msg)
	}
}

func addLongMessages(store *session.InMemoryStore, sessionID string, count int, role session.Role) {
	longContent := "This is a very long message designed to consume a significant number of tokens. " +
		"It contains detailed technical discussion about software architecture, design patterns, " +
		"and implementation strategies. The conversation covers topics such as microservices, " +
		"database optimization, API design, testing methodologies, and deployment pipelines. " +
		"Each message is carefully crafted to simulate real-world development discussions that " +
		"would naturally occur between a developer and an AI coding assistant during a complex " +
		"software engineering task. The content includes code snippets, configuration examples, " +
		"and detailed explanations of technical decisions. This approach ensures that the token " +
		"count per message is high enough to trigger compression thresholds in test scenarios " +
		"without requiring an excessive number of individual messages."
	for i := 0; i < count; i++ {
		msg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      role,
			Content:   longContent,
			CreatedAt: time.Now(),
		}
		store.AddMessage(sessionID, msg)
	}
}

func TestSelectMessagesToCompress(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "user 1"},
		{ID: "m2", Role: session.RoleAssistant, Content: "assistant 1"},
		{ID: "m3", Role: session.RoleSystem, Content: "system 1"},
		{ID: "m4", Role: session.RoleUser, Content: "user 2"},
		{ID: "m5", Role: session.RoleAssistant, Content: "assistant 2"},
		{ID: "m6", Role: session.RoleUser, Content: "user recent"},
		{ID: "m7", Role: session.RoleAssistant, Content: "assistant recent"},
	}

	remaining, toCompress := selectMessagesToCompress(msgs, 2)

	if len(toCompress) == 0 {
		t.Fatal("expected some messages to compress")
	}

	if len(toCompress) != 5 {
		t.Errorf("expected 5 messages to compress, got %d", len(toCompress))
	}

	recentFound := false
	for _, msg := range remaining {
		if msg.ID == "m6" || msg.ID == "m7" {
			recentFound = true
		}
	}
	if !recentFound {
		t.Error("recent messages should be in remaining")
	}

	if len(remaining) != 2 {
		t.Errorf("expected 2 messages remaining, got %d", len(remaining))
	}
}

func TestSelectMessagesToCompressAllSystem(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleSystem, Content: "system 1"},
		{ID: "m2", Role: session.RoleSystem, Content: "system 2"},
		{ID: "m3", Role: session.RoleSystem, Content: "system 3"},
	}

	remaining, toCompress := selectMessagesToCompress(msgs, 1)

	if len(toCompress) != 2 {
		t.Errorf("expected 2 messages to compress, got %d", len(toCompress))
	}
	if len(remaining) != 1 {
		t.Errorf("expected 1 message remaining, got %d", len(remaining))
	}
}

func TestSelectMessagesToCompressKeepAll(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "user 1"},
		{ID: "m2", Role: session.RoleAssistant, Content: "assistant 1"},
	}

	remaining, toCompress := selectMessagesToCompress(msgs, 5)

	if len(toCompress) != 0 {
		t.Errorf("expected no compression when keep_recent >= len(msgs), got %d", len(toCompress))
	}
	if len(remaining) != 2 {
		t.Errorf("expected all messages remaining, got %d", len(remaining))
	}
}

func TestSelectMessagesToCompressToolPairAlignment(t *testing.T) {
	tc := func(id string) session.ToolCall {
		return session.ToolCall{ID: id, ToolName: "test", Params: map[string]interface{}{}}
	}

	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "write a file"},
		{ID: "m2", Role: session.RoleAssistant, Content: "ok", ToolCalls: []session.ToolCall{tc("tc1")}},
		{ID: "m3", Role: session.RoleTool, Content: "file written", ToolCallID: "tc1"},
		{ID: "m4", Role: session.RoleAssistant, Content: "done!"},
		{ID: "m5", Role: session.RoleUser, Content: "now read it"},
		{ID: "m6", Role: session.RoleAssistant, Content: "ok", ToolCalls: []session.ToolCall{tc("tc2")}},
		{ID: "m7", Role: session.RoleTool, Content: "content here", ToolCallID: "tc2"},
		{ID: "m8", Role: session.RoleAssistant, Content: "here you go"},
	}

	remaining, _ := selectMessagesToCompress(msgs, 2)

	if len(remaining) == 0 {
		t.Fatal("expected some messages to remain")
	}

	remainingIDs := make(map[string]bool)
	for _, msg := range remaining {
		remainingIDs[msg.ID] = true
	}

	if !remainingIDs["m7"] {
		t.Error("m7 (tool result) should be in remaining")
	}
	if !remainingIDs["m8"] {
		t.Error("m8 (assistant final) should be in remaining")
	}

	if !remainingIDs["m6"] {
		t.Error("m6 (tool call for tc2) should be in remaining because m7 references tc2")
	}

	if remainingIDs["m3"] {
		t.Error("m3 (tool result for tc1) should NOT be in remaining, it's part of the previous turn")
	}
}

func TestSelectMessagesToCompressToolPairAlignmentMultiLevel(t *testing.T) {
	tc := func(id string) session.ToolCall {
		return session.ToolCall{ID: id, ToolName: "test", Params: map[string]interface{}{}}
	}

	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "do task"},
		{ID: "m2", Role: session.RoleAssistant, Content: "step 1", ToolCalls: []session.ToolCall{tc("tc1")}},
		{ID: "m3", Role: session.RoleTool, Content: "result 1", ToolCallID: "tc1"},
		{ID: "m4", Role: session.RoleAssistant, Content: "step 2", ToolCalls: []session.ToolCall{tc("tc2")}},
		{ID: "m5", Role: session.RoleTool, Content: "result 2", ToolCallID: "tc2"},
		{ID: "m6", Role: session.RoleAssistant, Content: "all done!"},
	}

	remaining, _ := selectMessagesToCompress(msgs, 1)

	if len(remaining) == 0 {
		t.Fatal("expected some messages to remain")
	}

	remainingIDs := make(map[string]bool)
	for _, msg := range remaining {
		remainingIDs[msg.ID] = true
	}

	if !remainingIDs["m6"] {
		t.Error("m6 should be in remaining")
	}

	if !remainingIDs["m5"] {
		t.Error("m5 (tool result for tc2) should be in remaining because m6 follows it and m4 references tc2")
	}

	if !remainingIDs["m4"] {
		t.Error("m4 (tool call for tc2) should be in remaining because m5 references tc2")
	}

	if remainingIDs["m3"] {
		t.Error("m3 should NOT be in remaining, tc1 is not referenced by remaining")
	}
}

func TestCompressNoopBelowThreshold(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 500, 5)

	addMessages(store, "sess-1", 5, session.RoleUser)

	eventBus, _ := store.GetEventBus("sess-1")

	result, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when below threshold, got: %+v", result)
	}
}

func TestCompressTriggersAboveThreshold(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 5, 2)

	addMixedMessages(store, "sess-1", 10)

	eventBus, _ := store.GetEventBus("sess-1")

	result, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected compression result when above threshold")
	}
	if result.CompressedCount == 0 {
		t.Error("expected some messages to be compressed")
	}
	if result.SummaryText == "" {
		t.Error("expected non-empty summary text")
	}
}

func TestCompressUpdatesSessionState(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 5, 2)

	addMessages(store, "sess-1", 10, session.RoleUser)

	eventBus, _ := store.GetEventBus("sess-1")

	_, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.Get("sess-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}

	if sess.CompressionCount != 1 {
		t.Errorf("expected compression_count=1, got %d", sess.CompressionCount)
	}
}

func TestCompressEmitsSSEEvent(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 5, 2)

	addMessages(store, "sess-1", 10, session.RoleUser)

	eventBus, _ := store.GetEventBus("sess-1")
	ch, unsubscribe := eventBus.Subscribe()
	defer unsubscribe()

	_, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt := <-ch:
		if evt.Type != "context_compressed" {
			t.Errorf("expected context_compressed event, got %q", evt.Type)
		}
		data, ok := evt.Data.(map[string]any)
		if !ok {
			t.Fatal("expected map data in event")
		}
		if data["compressed_count"] == nil {
			t.Error("expected compressed_count in event data")
		}
		if data["tokens_removed"] == nil {
			t.Error("expected tokens_removed in event data")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for context_compressed event")
	}
}

func TestCompressMultipleRounds(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 3, 1)

	addMessages(store, "sess-1", 5, session.RoleUser)
	eventBus, _ := store.GetEventBus("sess-1")

	result1, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("first compress: %v", err)
	}
	if result1 == nil {
		t.Fatal("expected first compression to trigger")
	}

	addMessages(store, "sess-1", 5, session.RoleUser)

	result2, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("second compress: %v", err)
	}
	if result2 == nil {
		t.Fatal("expected second compression to trigger")
	}

	sess, _ := store.Get("sess-1")
	if sess.CompressionCount != 2 {
		t.Errorf("expected compression_count=2, got %d", sess.CompressionCount)
	}
}

func TestCompressEmptyMessages(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 1, 1)

	eventBus, _ := store.GetEventBus("sess-1")

	result, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty messages, got: %+v", result)
	}
}

func TestCompressDefaultThresholds(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 0, 0)

	addMessages(store, "sess-1", 10, session.RoleUser)

	eventBus, _ := store.GetEventBus("sess-1")

	result, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Error("expected nil result when below default threshold (60) with 10 messages")
	}
}

func TestCompressWithDefaultThresholdsTriggered(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 0, 0)

	addLongMessages(store, "sess-1", 600, session.RoleUser)

	eventBus, _ := store.GetEventBus("sess-1")

	result, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected compression when above default threshold (60)")
	}
}

func TestCompressTokenEstimation(t *testing.T) {
	msgs := []session.Message{
		{ID: "abc", Role: session.RoleUser, Content: "Hello, this is a test message with some content"},
		{ID: "def", Role: session.RoleAssistant, Content: "This is a response with more content to estimate tokens"},
	}

	tokens := estimateTokens(msgs)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
}

func TestCompressSessionNotFound(t *testing.T) {
	comp, _ := setupTestCompressor()
	eventBus := session.NewEventBus(100)

	_, err := comp.Compress(context.Background(), "nonexistent", eventBus, 0)
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestCompressSummaryContainsContent(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 3, 2)

	addMessages(store, "sess-1", 8, session.RoleUser)

	eventBus, _ := store.GetEventBus("sess-1")

	result, err := comp.Compress(context.Background(), "sess-1", eventBus, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected compression result")
	}

	msgs, _, _ := store.GetMessages("sess-1", 0, 0)

	summaryFound := false
	for _, msg := range msgs {
		if msg.Role == session.RoleSystem && msg.ID != "" {
			if len(msg.Content) > 0 {
				summaryFound = true
				break
			}
		}
	}
	if !summaryFound {
		t.Error("expected summary system message to be added to messages")
	}
}

func TestCompressNilEventBus(t *testing.T) {
	comp, store := setupTestCompressor()
	createTestSession(store, "sess-1", 5, 2)

	addMessages(store, "sess-1", 10, session.RoleUser)

	result, err := comp.Compress(context.Background(), "sess-1", nil, 0)
	if err != nil {
		t.Fatalf("unexpected error with nil event bus: %v", err)
	}
	if result == nil {
		t.Fatal("expected compression result even with nil event bus")
	}
}

func TestEstimateContextTokens(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "Hello, this is a test message with some content"},
		{ID: "m2", Role: session.RoleAssistant, Content: "This is a response with more content to estimate tokens"},
	}

	tokens := EstimateContextTokens(msgs)
	if tokens <= 0 {
		t.Errorf("expected positive token count from EstimateContextTokens, got %d", tokens)
	}
}
