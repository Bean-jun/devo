package agentloop

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"devo/internal/core/session"
)

func newTestLoopContext(sessionID string, store session.SessionStore) *LoopContext {
	eventBus, _ := store.GetEventBus(sessionID)
	return &LoopContext{
		SessionID: sessionID,
		EventBus:  eventBus,
		CancelCh:  make(chan struct{}, 1),
		PauseCh:   make(chan struct{}, 1),
		ResumeCh:  make(chan struct{}, 1),
	}
}

func mockStateHandler(name string, nextState LoopState, shouldError bool, errMsg string) StateHandler {
	return func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		if shouldError {
			return LoopStateError, fmt.Errorf("%s", errMsg)
		}
		return nextState, nil
	}
}

func assertEventPublished(t *testing.T, ch chan session.Event, eventType string, timeout time.Duration) *session.Event {
	t.Helper()
	evt, ok := waitForEvent(ch, eventType, timeout)
	if !ok {
		t.Errorf("expected event %q within %v", eventType, timeout)
		return nil
	}
	return evt
}

func assertEventSequence(t *testing.T, ch chan session.Event, expectedTypes []string, timeout time.Duration) {
	t.Helper()
	received := make([]string, 0)
	deadline := time.After(timeout)
	for i := 0; i < len(expectedTypes); i++ {
		select {
		case <-deadline:
			t.Errorf("expected %d events, got %d: %v. Expected: %v", len(expectedTypes), len(received), received, expectedTypes)
			return
		case evt, ok := <-ch:
			if !ok {
				t.Errorf("channel closed after %d events: %v", len(received), received)
				return
			}
			received = append(received, evt.Type)
		}
	}
	for i, expected := range expectedTypes {
		if i < len(received) && received[i] != expected {
			t.Errorf("event[%d]: expected %q, got %q", i, expected, received[i])
		}
	}
}

func sendCancelSignal(lc *LoopContext) {
	select {
	case lc.CancelCh <- struct{}{}:
	default:
	}
}

func sendPauseSignal(lc *LoopContext) {
	select {
	case lc.PauseCh <- struct{}{}:
	default:
	}
}

func sendResumeSignal(lc *LoopContext) {
	select {
	case lc.ResumeCh <- struct{}{}:
	default:
	}
}

func drainEvents(ch chan session.Event, timeout time.Duration) []session.Event {
	events := make([]session.Event, 0)
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			return events
		case evt, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, evt)
			if evt.Type == "message_complete" || evt.Type == "loop.loop_completed" || evt.Type == "loop.cancelled" {
				return events
			}
		}
	}
}

func newTestStateMachine() *StateMachine {
	sm := NewStateMachine()
	executed := &syncOrder{}
	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		executed.append("preparing")
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		executed.append("thinking")
		return LoopStateEvaluatingResult, nil
	})
	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		executed.append("evaluating_result")
		return LoopStateTextResponse, nil
	})
	sm.Register(LoopStateTextResponse, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		executed.append("text_response")
		return LoopStateIdle, nil
	})
	return sm
}

type syncOrder struct {
	mu    sync.Mutex
	order []string
}

func (s *syncOrder) append(v string) {
	s.mu.Lock()
	s.order = append(s.order, v)
	s.mu.Unlock()
}

func (s *syncOrder) get() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.order))
	copy(result, s.order)
	return result
}
