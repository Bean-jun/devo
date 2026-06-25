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
		{ID: "m6", Role: session.RoleTool, Content: "tool result"},
		{ID: "m7", Role: session.RoleUser, Content: "user recent"},
		{ID: "m8", Role: session.RoleAssistant, Content: "assistant recent"},
	}

	remaining, toCompress := selectMessagesToCompress(msgs, 2)

	if len(toCompress) == 0 {
		t.Fatal("expected some messages to compress")
	}

	for _, msg := range toCompress {
		if msg.Role == session.RoleSystem || msg.Role == session.RoleTool {
			t.Errorf("system/tool messages should not be compressed, got role %q", msg.Role)
		}
	}

	recentFound := false
	for _, msg := range remaining {
		if msg.ID == "m7" || msg.ID == "m8" {
			recentFound = true
		}
	}
	if !recentFound {
		t.Error("recent messages should be in remaining")
	}

	systemFound := false
	for _, msg := range remaining {
		if msg.ID == "m3" {
			systemFound = true
		}
	}
	if !systemFound {
		t.Error("system messages should be preserved in remaining")
	}
}

func TestSelectMessagesToCompressAllSystem(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleSystem, Content: "system 1"},
		{ID: "m2", Role: session.RoleSystem, Content: "system 2"},
		{ID: "m3", Role: session.RoleSystem, Content: "system 3"},
	}

	remaining, toCompress := selectMessagesToCompress(msgs, 1)

	if len(toCompress) != 0 {
		t.Errorf("expected no messages to compress when all are system, got %d", len(toCompress))
	}
	if len(remaining) != 3 {
		t.Errorf("expected all messages remaining, got %d", len(remaining))
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
	if sess.CompressionState == nil {
		t.Fatal("expected non-nil compression state")
	}
	if len(sess.CompressionState.CompressedRanges) != 1 {
		t.Errorf("expected 1 compressed range, got %d", len(sess.CompressionState.CompressedRanges))
	}
	if len(sess.CompressionState.Summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(sess.CompressionState.Summaries))
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

func TestFilterActiveMessages(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "compressed user message"},
		{ID: "m2", Role: session.RoleAssistant, Content: "compressed assistant message"},
		{ID: "m3", Role: session.RoleSystem, Content: "system message"},
		{ID: "m4", Role: session.RoleUser, Content: "active user message 1"},
		{ID: "m5", Role: session.RoleAssistant, Content: "active assistant message 1"},
		{ID: "m6", Role: session.RoleUser, Content: "active user message 2"},
	}

	compressionState := &session.CompressionState{
		CompressedRanges: []session.CompressedRange{
			{StartMessageID: "m1", EndMessageID: "m2"},
		},
	}

	filtered := FilterActiveMessages(msgs, compressionState)

	if len(filtered) != 4 {
		t.Errorf("expected 4 messages after filtering, got %d", len(filtered))
	}

	for _, msg := range filtered {
		if msg.ID == "m1" || msg.ID == "m2" {
			t.Errorf("compressed message %s should not be in active messages", msg.ID)
		}
	}

	if filtered[0].ID != "m3" {
		t.Errorf("expected m3 (system) to be first, got %s", filtered[0].ID)
	}
}

func TestFilterActiveMessagesNilState(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "message 1"},
		{ID: "m2", Role: session.RoleAssistant, Content: "message 2"},
	}

	filtered := FilterActiveMessages(msgs, nil)

	if len(filtered) != 2 {
		t.Errorf("expected all messages when no compression state, got %d", len(filtered))
	}
}

func TestFilterActiveMessagesEmptyRanges(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "message 1"},
	}

	compressionState := &session.CompressionState{
		CompressedRanges: []session.CompressedRange{},
	}

	filtered := FilterActiveMessages(msgs, compressionState)

	if len(filtered) != 1 {
		t.Errorf("expected all messages when empty ranges, got %d", len(filtered))
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
	if len(sess.CompressionState.CompressedRanges) != 2 {
		t.Errorf("expected 2 compressed ranges, got %d", len(sess.CompressionState.CompressedRanges))
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
