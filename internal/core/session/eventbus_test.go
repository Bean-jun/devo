package session

import (
	"testing"
	"time"
)

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
