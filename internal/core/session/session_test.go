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
	sess.State = StateProcessing

	err := store.Update(sess)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := store.Get("sess-1")
	if got.Title != "Updated Title" {
		t.Errorf("expected Title %q, got %q", "Updated Title", got.Title)
	}
	if got.State != StateProcessing {
		t.Errorf("expected State %q, got %q", StateProcessing, got.State)
	}
}

func TestUpdateNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.Update(&Session{ID: "nonexistent"})
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestAddAndGetMessages(t *testing.T) {
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

	msg1 := Message{ID: "msg-1", Role: RoleUser, Content: "Hello", CreatedAt: time.Now()}
	msg2 := Message{ID: "msg-2", Role: RoleAssistant, Content: "Hi there", CreatedAt: time.Now()}

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

func TestGetMessagesPagination(t *testing.T) {
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

	for i := 0; i < 5; i++ {
		store.AddMessage("sess-1", Message{
			ID:        GenerateID("msg"),
			Role:      RoleUser,
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

func TestAddMessageSessionNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.AddMessage("nonexistent", Message{})
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

func TestEventBusCreatedOnSessionCreate(t *testing.T) {
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

	eventBus, err := store.GetEventBus("sess-1")
	if err != nil {
		t.Fatalf("expected event bus to exist, got: %v", err)
	}
	if eventBus == nil {
		t.Fatal("expected non-nil event bus")
	}
}

func TestGetEventBusNotFound(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.GetEventBus("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestEventBusPublishAndSubscribe(t *testing.T) {
	eb := NewEventBus(DefaultEventHistorySize)

	ch, unsubscribe := eb.Subscribe()
	defer unsubscribe()

	eb.Publish("test_event", map[string]string{"key": "value"})

	select {
	case evt := <-ch:
		if evt.Type != "test_event" {
			t.Errorf("expected event type 'test_event', got %q", evt.Type)
		}
		data, ok := evt.Data.(map[string]string)
		if !ok {
			t.Fatal("expected data to be map[string]string")
		}
		if data["key"] != "value" {
			t.Errorf("expected data key 'value', got %q", data["key"])
		}
	default:
		t.Fatal("expected to receive event")
	}
}

func TestEventBusHistory(t *testing.T) {
	eb := NewEventBus(DefaultEventHistorySize)

	eb.Publish("event1", map[string]string{"n": "1"})
	eb.Publish("event2", map[string]string{"n": "2"})
	eb.Publish("event3", map[string]string{"n": "3"})

	history := eb.GetHistory(0)
	if len(history) != 3 {
		t.Fatalf("expected 3 events in history, got %d", len(history))
	}

	history = eb.GetHistory(1)
	if len(history) != 2 {
		t.Fatalf("expected 2 events after ID 1, got %d", len(history))
	}
	if history[0].Type != "event2" {
		t.Errorf("expected event2, got %s", history[0].Type)
	}

	history = eb.GetHistory(3)
	if len(history) != 0 {
		t.Fatalf("expected 0 events after ID 3, got %d", len(history))
	}
}

func TestEventBusHistoryRolling(t *testing.T) {
	eb := NewEventBus(3)

	for i := 0; i < 5; i++ {
		eb.Publish("event", map[string]int{"n": i})
	}

	history := eb.GetHistory(0)
	if len(history) != 3 {
		t.Fatalf("expected 3 events in rolling history, got %d", len(history))
	}
}

func TestEventBusMultipleSubscribers(t *testing.T) {
	eb := NewEventBus(DefaultEventHistorySize)

	ch1, unsub1 := eb.Subscribe()
	defer unsub1()
	ch2, unsub2 := eb.Subscribe()
	defer unsub2()

	eb.Publish("event", map[string]string{"msg": "hello"})

	for i, ch := range []chan Event{ch1, ch2} {
		select {
		case evt := <-ch:
			if evt.Type != "event" {
				t.Errorf("subscriber %d: expected event type 'event', got %q", i, evt.Type)
			}
		default:
			t.Errorf("subscriber %d: expected to receive event", i)
		}
	}
}

func TestSSEConnectionsIncrementDecrement(t *testing.T) {
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

	if err := store.IncrementSSEConnections("sess-1"); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := store.IncrementSSEConnections("sess-1"); err != nil {
		t.Fatalf("increment: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.ActiveSSEConnections != 2 {
		t.Errorf("expected 2 connections, got %d", sess.ActiveSSEConnections)
	}

	if err := store.DecrementSSEConnections("sess-1"); err != nil {
		t.Fatalf("decrement: %v", err)
	}

	sess, _ = store.Get("sess-1")
	if sess.ActiveSSEConnections != 1 {
		t.Errorf("expected 1 connection after decrement, got %d", sess.ActiveSSEConnections)
	}
}

func TestSSEConnectionsDecrementBelowZero(t *testing.T) {
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

	store.DecrementSSEConnections("sess-1")

	sess, _ = store.Get("sess-1")
	if sess.ActiveSSEConnections < 0 {
		t.Errorf("ActiveSSEConnections should not go below 0, got %d", sess.ActiveSSEConnections)
	}
}

func TestSSEConnectionsNotFound(t *testing.T) {
	store := NewInMemoryStore()

	err := store.IncrementSSEConnections("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}

	err = store.DecrementSSEConnections("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
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

	sess.State = StateProcessing
	store.Update(sess)

	updatedEB, _ := store.GetEventBus("sess-1")
	if originalEB != updatedEB {
		t.Error("EventBus should be preserved across Update calls")
	}
}
