package session

import (
	"sync"
	"time"
)

const DefaultEventHistorySize = 200

type Event struct {
	ID        int64     `json:"id"`
	Type      string    `json:"event"`
	Data      any       `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

type EventBus struct {
	mu           sync.RWMutex
	history      []Event
	maxHistory   int
	nextID       int64
	subscribers  map[int64]chan Event
	subIDCounter int64
}

func NewEventBus(maxHistory int) *EventBus {
	if maxHistory <= 0 {
		maxHistory = DefaultEventHistorySize
	}
	return &EventBus{
		history:     make([]Event, 0, maxHistory),
		maxHistory:  maxHistory,
		subscribers: make(map[int64]chan Event),
	}
}

func (eb *EventBus) Publish(eventType string, data any) {
	eb.mu.Lock()

	eb.nextID++
	evt := Event{
		ID:        eb.nextID,
		Type:      eventType,
		Data:      data,
		CreatedAt: time.Now(),
	}

	if len(eb.history) >= eb.maxHistory {
		eb.history = eb.history[1:]
	}
	eb.history = append(eb.history, evt)

	subs := make([]chan Event, 0, len(eb.subscribers))
	for _, ch := range eb.subscribers {
		subs = append(subs, ch)
	}

	eb.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (eb *EventBus) Subscribe() (chan Event, func()) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subIDCounter++
	id := eb.subIDCounter
	ch := make(chan Event, 64)
	eb.subscribers[id] = ch

	unsubscribe := func() {
		eb.mu.Lock()
		defer eb.mu.Unlock()
		if _, ok := eb.subscribers[id]; ok {
			delete(eb.subscribers, id)
			close(ch)
		}
	}

	return ch, unsubscribe
}

func (eb *EventBus) GetHistory(sinceID int64) []Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	var result []Event
	for _, evt := range eb.history {
		if evt.ID > sinceID {
			result = append(result, evt)
		}
	}
	return result
}
