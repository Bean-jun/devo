package agentloop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestStateMachine_BasicFlow(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-basic")
	lc := newTestLoopContext("test-basic", store)

	sm := NewStateMachine()
	executed := make([]string, 0)
	var mu sync.Mutex

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "preparing")
		mu.Unlock()
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "thinking")
		mu.Unlock()
		return LoopStateEvaluatingResult, nil
	})
	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "evaluating_result")
		mu.Unlock()
		return LoopStateTextResponse, nil
	})
	sm.Register(LoopStateTextResponse, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "text_response")
		mu.Unlock()
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	mu.Lock()
	defer mu.Unlock()
	expected := []string{"preparing", "thinking", "evaluating_result", "text_response"}
	if len(executed) != len(expected) {
		t.Fatalf("expected %d states executed, got %d: %v", len(expected), len(executed), executed)
	}
	for i, s := range expected {
		if executed[i] != s {
			t.Errorf("step %d: expected %s, got %s", i, s, executed[i])
		}
	}
}

func TestStateMachine_ToolCallFlow(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-toolflow")
	lc := newTestLoopContext("test-toolflow", store)

	sm := NewStateMachine()
	executed := make([]string, 0)
	var mu sync.Mutex

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "preparing")
		mu.Unlock()
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "thinking")
		mu.Unlock()
		return LoopStateEvaluatingResult, nil
	})
	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "evaluating_result")
		mu.Unlock()
		return LoopStateToolExecuting, nil
	})

	toolCallCount := 0
	sm.Register(LoopStateToolExecuting, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "tool_executing")
		mu.Unlock()
		toolCallCount++
		if toolCallCount < 2 {
			return LoopStatePreparing, nil
		}
		return LoopStateTextResponse, nil
	})
	sm.Register(LoopStateTextResponse, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "text_response")
		mu.Unlock()
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	mu.Lock()
	defer mu.Unlock()
	expected := []string{"preparing", "thinking", "evaluating_result", "tool_executing",
		"preparing", "thinking", "evaluating_result", "tool_executing", "text_response"}
	if len(executed) != len(expected) {
		t.Fatalf("expected %d states executed, got %d: %v", len(expected), len(executed), executed)
	}
	for i, s := range expected {
		if executed[i] != s {
			t.Errorf("step %d: expected %s, got %s", i, s, executed[i])
		}
	}
}

func TestStateMachine_CancelDuringThinking(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-cancel-think")
	lc := newTestLoopContext("test-cancel-think", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		select {
		case lc.CancelCh <- struct{}{}:
		default:
		}
		time.Sleep(10 * time.Millisecond)
		return LoopStateEvaluatingResult, nil
	})
	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		t.Error("evaluating_result should not be reached after cancel")
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "loop.cancelled", 2*time.Second)
	if !ok {
		t.Fatal("expected loop.cancelled event")
	}
}

func TestStateMachine_CancelDuringToolExecution(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-cancel-tool")
	lc := newTestLoopContext("test-cancel-tool", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateToolExecuting, nil
	})
	sm.Register(LoopStateToolExecuting, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		select {
		case lc.CancelCh <- struct{}{}:
		default:
		}
		time.Sleep(10 * time.Millisecond)
		return LoopStatePreparing, nil
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "loop.cancelled", 2*time.Second)
	if !ok {
		t.Fatal("expected loop.cancelled event")
	}
}

func TestStateMachine_PauseAndResume(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-pause-resume")
	lc := newTestLoopContext("test-pause-resume", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()
	executed := make([]string, 0)
	var mu sync.Mutex

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "preparing")
		mu.Unlock()
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "thinking")
		mu.Unlock()

		select {
		case lc.PauseCh <- struct{}{}:
		default:
		}

		go func() {
			time.Sleep(50 * time.Millisecond)
			select {
			case lc.ResumeCh <- struct{}{}:
			default:
			}
		}()

		return LoopStateEvaluatingResult, nil
	})
	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "evaluating_result")
		mu.Unlock()
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	mu.Lock()
	defer mu.Unlock()

	_, ok := waitForEvent(ch, "loop.paused", 2*time.Second)
	if !ok {
		t.Fatal("expected loop.paused event")
	}

	if len(executed) != 3 {
		t.Fatalf("expected 3 states executed, got %d: %v", len(executed), executed)
	}
}

func TestStateMachine_PauseAndCancel(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-pause-cancel")
	lc := newTestLoopContext("test-pause-cancel", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		select {
		case lc.PauseCh <- struct{}{}:
		default:
		}

		go func() {
			time.Sleep(50 * time.Millisecond)
			select {
			case lc.CancelCh <- struct{}{}:
			default:
			}
		}()

		return LoopStateEvaluatingResult, nil
	})
	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		t.Error("evaluating_result should not be reached after cancel")
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "loop.cancelled", 2*time.Second)
	if !ok {
		t.Fatal("expected loop.cancelled event")
	}
}

func TestStateMachine_ErrorHandling(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-error")
	lc := newTestLoopContext("test-error", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateError, fmt.Errorf("llm service unavailable")
	})
	sm.Register(LoopStateError, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		t.Error("error handler should not be called directly")
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "error", 2*time.Second)
	if !ok {
		t.Fatal("expected error event")
	}

	_, ok = waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}
}

func TestStateMachine_HandlerNotFound(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-nohandler")
	lc := newTestLoopContext("test-nohandler", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStateEvaluatingResult, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "error", 2*time.Second)
	if !ok {
		t.Fatal("expected error event for missing handler")
	}
}

func TestStateMachine_PanicRecovery(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-panic")
	lc := newTestLoopContext("test-panic", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		panic("unexpected nil pointer")
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "error", 2*time.Second)
	if !ok {
		t.Fatal("expected error event after panic")
	}
}

func TestStateMachine_StateChangeEvents(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-statechange")
	lc := newTestLoopContext("test-statechange", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	stateChanges := 0
	for {
		evt, ok := waitForEvent(ch, "loop.state_change", 1*time.Second)
		if !ok {
			break
		}
		data, _ := evt.Data.(map[string]any)
		t.Logf("State change: %s -> %s", data["old_state"], data["new_state"])
		stateChanges++
	}

	if stateChanges < 2 {
		t.Errorf("expected at least 2 state change events, got %d", stateChanges)
	}
}

func TestStateMachine_ApprovalFlow(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-approval-flow")
	lc := newTestLoopContext("test-approval-flow", store)

	sm := NewStateMachine()
	executed := make([]string, 0)
	var mu sync.Mutex
	approvalCount := 0

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "preparing")
		mu.Unlock()
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "thinking")
		mu.Unlock()
		return LoopStateToolExecuting, nil
	})
	sm.Register(LoopStateToolExecuting, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "tool_executing")
		mu.Unlock()
		return LoopStateAwaitingApproval, nil
	})
	sm.Register(LoopStateAwaitingApproval, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		mu.Lock()
		executed = append(executed, "awaiting_approval")
		approvalCount++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		if approvalCount >= 2 {
			return LoopStateTextResponse, nil
		}
		return LoopStateToolExecuting, nil
	})
	sm.Register(LoopStateTextResponse, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateIdle, nil
	})

	sm.Run(context.Background(), lc)

	mu.Lock()
	defer mu.Unlock()
	expected := []string{"preparing", "thinking", "tool_executing", "awaiting_approval", "tool_executing", "awaiting_approval"}
	if len(executed) != len(expected) {
		t.Fatalf("expected %d states, got %d: %v", len(expected), len(executed), executed)
	}
	for i, s := range expected {
		if executed[i] != s {
			t.Errorf("step %d: expected %s, got %s", i, s, executed[i])
		}
	}
}

func TestStateMachine_ContextCancel(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-ctx-cancel")
	lc := newTestLoopContext("test-ctx-cancel", store)

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		time.Sleep(100 * time.Millisecond)
		return LoopStateIdle, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	sm.Run(ctx, lc)

	cancel()

	time.Sleep(50 * time.Millisecond)
}

func TestStateMachine_ConcurrentRegistration(t *testing.T) {
	sm := NewStateMachine()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			state := LoopState(fmt.Sprintf("state_%d", i))
			sm.Register(state, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
				return LoopStateIdle, nil
			})
		}(i)
	}
	wg.Wait()

	sm.mu.RLock()
	count := len(sm.handlers)
	sm.mu.RUnlock()

	if count != 100 {
		t.Errorf("expected 100 handlers, got %d", count)
	}
}

func TestStateMachine_ErrorHandlerReturnsError(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-err-return")
	lc := newTestLoopContext("test-err-return", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateIdle, errors.New("fatal error")
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "error", 2*time.Second)
	if !ok {
		t.Fatal("expected error event")
	}

	_, ok = waitForEvent(ch, "session_state_change", 2*time.Second)
	if !ok {
		t.Fatal("expected session_state_change event")
	}
}

func TestStateMachine_HandlerReturnsPaused(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-handler-paused")
	lc := newTestLoopContext("test-handler-paused", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()
	handlerCalled := false

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		lc.PausedInState = LoopStateThinking
		if !handlerCalled {
			handlerCalled = true
			return LoopStatePaused, nil
		}
		return LoopStateIdle, nil
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		lc.ResumeCh <- struct{}{}
	}()

	sm.Run(context.Background(), lc)

	if !handlerCalled {
		t.Error("thinking handler was not called")
	}

	var events []session.Event
	timeout := time.After(500 * time.Millisecond)
drainLoop:
	for {
		select {
		case <-timeout:
			break drainLoop
		case evt, ok := <-ch:
			if !ok {
				break drainLoop
			}
			events = append(events, evt)
		}
	}

	hasPaused := false
	stateChanges := make([]string, 0)
	for _, evt := range events {
		if evt.Type == "loop.paused" {
			hasPaused = true
		}
		if evt.Type == "loop.state_change" {
			data, _ := evt.Data.(map[string]any)
			oldS, _ := data["old_state"].(string)
			newS, _ := data["new_state"].(string)
			stateChanges = append(stateChanges, fmt.Sprintf("%s->%s", oldS, newS))
		}
	}

	if !hasPaused {
		t.Error("expected loop.paused event")
	}

	t.Logf("State changes: %v", stateChanges)
	if len(stateChanges) < 4 {
		t.Errorf("expected at least 4 state changes, got %d: %v", len(stateChanges), stateChanges)
	}
}

func TestStateMachine_HandlerReturnsCancelled(t *testing.T) {
	store := session.NewInMemoryStore()
	createTestSession(store, "test-handler-cancelled")
	lc := newTestLoopContext("test-handler-cancelled", store)

	ch, unsubscribe := lc.EventBus.Subscribe()
	defer unsubscribe()

	sm := NewStateMachine()

	sm.Register(LoopStatePreparing, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateThinking, nil
	})
	sm.Register(LoopStateThinking, func(ctx context.Context, lc *LoopContext) (LoopState, error) {
		return LoopStateCancelled, nil
	})

	sm.Run(context.Background(), lc)

	_, ok := waitForEvent(ch, "loop.cancelled", 2*time.Second)
	if !ok {
		t.Fatal("expected loop.cancelled event")
	}
}
