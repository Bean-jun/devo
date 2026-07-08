package agentloop

import (
	"context"
	"fmt"
	"log"
	"sync"

	"devo/internal/core/session"
)

type LoopState string

const (
	LoopStateIdle             LoopState = "idle"
	LoopStatePreparing        LoopState = "preparing"
	LoopStateThinking         LoopState = "thinking"
	LoopStateEvaluatingResult LoopState = "evaluating_result"
	LoopStateToolExecuting    LoopState = "tool_executing"
	LoopStateAwaitingApproval LoopState = "awaiting_approval"
	LoopStateTextResponse     LoopState = "text_response"
	LoopStatePaused           LoopState = "paused"
	LoopStateCancelled        LoopState = "cancelled"
	LoopStateError            LoopState = "error"
)

type StateHandler func(ctx context.Context, lc *LoopContext) (LoopState, error)

type StateMachine struct {
	handlers map[LoopState]StateHandler
	mu       sync.RWMutex
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		handlers: make(map[LoopState]StateHandler),
	}
}

func (sm *StateMachine) Register(state LoopState, handler StateHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers[state] = handler
}

func (sm *StateMachine) Run(ctx context.Context, lc *LoopContext) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("state machine panic for session %s: %v", lc.SessionID, r)
			lc.TerminationReason = "panic"
			lc.EventBus.Publish("error", map[string]any{
				"message": fmt.Sprintf("panic: %v", r),
			})
		}
	}()

	currentState := LoopStatePreparing

	for currentState != LoopStateIdle && currentState != LoopStateCancelled {
		select {
		case <-lc.CancelCh:
			lc.TerminationReason = "cancelled"
			lc.EventBus.Publish("loop.state_change", map[string]any{
				"old_state": string(currentState),
				"new_state": string(LoopStateCancelled),
			})
			lc.EventBus.Publish("loop.cancelled", map[string]any{
				"session_id":   lc.SessionID,
				"cancelled_at": string(currentState),
				"reason":       "user_requested",
			})
			return

		case <-lc.PauseCh:
			lc.PausedInState = currentState
			lc.EventBus.Publish("loop.state_change", map[string]any{
				"old_state": string(currentState),
				"new_state": string(LoopStatePaused),
			})
			lc.EventBus.Publish("loop.paused", map[string]any{
				"session_id": lc.SessionID,
				"paused_at":  string(currentState),
				"reason":     "user_requested",
			})

			select {
			case <-lc.ResumeCh:
				currentState = lc.PausedInState
				lc.EventBus.Publish("loop.state_change", map[string]any{
					"old_state": string(LoopStatePaused),
					"new_state": string(currentState),
				})
				continue
			case <-lc.CancelCh:
				lc.TerminationReason = "cancelled"
				lc.EventBus.Publish("loop.state_change", map[string]any{
					"old_state": string(LoopStatePaused),
					"new_state": string(LoopStateCancelled),
				})
				lc.EventBus.Publish("loop.cancelled", map[string]any{
					"session_id":   lc.SessionID,
					"cancelled_at": string(LoopStatePaused),
					"reason":       "user_requested",
				})
				return
			}

		default:
		}

		sm.mu.RLock()
		handler, ok := sm.handlers[currentState]
		sm.mu.RUnlock()

		if !ok {
			log.Printf("state machine: no handler registered for state %s", currentState)
			lc.TerminationReason = "error"
			lc.EventBus.Publish("error", map[string]any{
				"message": fmt.Sprintf("no handler for state: %s", currentState),
			})
			return
		}

		nextState, err := handler(ctx, lc)

		if err != nil {
			log.Printf("state machine error in state %s: %v", currentState, err)
			lc.TerminationReason = "error"
			lc.EventBus.Publish("error", map[string]any{
				"message": err.Error(),
			})
			lc.EventBus.Publish("session_state_change", map[string]any{
				"old_state": session.State(currentState).ToSnakeCase(),
				"new_state": session.StateIdle.ToSnakeCase(),
				"reason":    "error",
			})
			lc.EventBus.Publish("loop.loop_completed", nil)
			lc.EventBus.Publish("message_complete", map[string]any{
				"session_id": lc.SessionID,
				"reason":     "error",
			})
			currentState = LoopStateError
			nextState = LoopStateIdle
		}

		if nextState == LoopStatePaused {
			lc.EventBus.Publish("loop.state_change", map[string]any{
				"old_state": string(currentState),
				"new_state": string(LoopStatePaused),
			})
			lc.EventBus.Publish("loop.paused", map[string]any{
				"session_id": lc.SessionID,
				"paused_at":  string(currentState),
				"reason":     "handler_requested",
			})

			select {
			case <-lc.ResumeCh:
				currentState = lc.PausedInState
				lc.EventBus.Publish("loop.state_change", map[string]any{
					"old_state": string(LoopStatePaused),
					"new_state": string(currentState),
				})
				continue
			case <-lc.CancelCh:
				lc.TerminationReason = "cancelled"
				lc.EventBus.Publish("loop.state_change", map[string]any{
					"old_state": string(LoopStatePaused),
					"new_state": string(LoopStateCancelled),
				})
				lc.EventBus.Publish("loop.cancelled", map[string]any{
					"session_id":   lc.SessionID,
					"cancelled_at": string(LoopStatePaused),
					"reason":       "user_requested",
				})
				return
			}
		}

		if nextState == LoopStateCancelled {
			lc.TerminationReason = "cancelled"
			lc.EventBus.Publish("loop.state_change", map[string]any{
				"old_state": string(currentState),
				"new_state": string(LoopStateCancelled),
			})
			lc.EventBus.Publish("loop.cancelled", map[string]any{
				"session_id":   lc.SessionID,
				"cancelled_at": string(currentState),
				"reason":       "handler_requested",
			})
			return
		}

		lc.EventBus.Publish("loop.state_change", map[string]any{
			"old_state": string(currentState),
			"new_state": string(nextState),
		})

		currentState = nextState
	}
}
