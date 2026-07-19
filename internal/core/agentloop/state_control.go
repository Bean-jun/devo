package agentloop

import (
	"fmt"
	"time"

	"devo/internal/core/session"
)

func (l *Loop) Pause(sessionID string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StateToolExecuting {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotProcessing, sess.State)
	}

	lc, ok := l.activeLoops.Load(sessionID)
	if !ok {
		oldState := string(sess.State)
		sess.State = session.StatePaused
		sess.PauseRequested = true
		if err := l.store.Update(sess); err != nil {
			return fmt.Errorf("update pause flag: %w", err)
		}

		eventBus, err := l.store.GetEventBus(sessionID)
		if err == nil {
			eventBus.Publish("session_state_change", map[string]any{
				"old_state": session.State(oldState).ToSnakeCase(),
				"new_state": session.StatePaused.ToSnakeCase(),
				"reason":    "paused",
			})
		}
		return nil
	}

	loopCtx := lc.(*LoopContext)
	select {
	case loopCtx.PauseCh <- struct{}{}:
	default:
	}

	oldState := string(sess.State)
	sess.State = session.StatePaused
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session state to paused: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StatePaused.ToSnakeCase(),
			"reason":    "paused",
		})
	}

	return nil
}

func (l *Loop) Resume(sessionID string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StatePaused {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotPaused, sess.State)
	}

	oldState := string(sess.State)
	sess.State = session.StateToolExecuting
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session state: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateToolExecuting.ToSnakeCase(),
			"reason":    "resumed",
		})
	}

	lc, ok := l.activeLoops.Load(sessionID)
	if !ok {
		return nil
	}

	loopCtx := lc.(*LoopContext)
	select {
	case loopCtx.ResumeCh <- struct{}{}:
	default:
	}

	return nil
}

func (l *Loop) Cancel(sessionID string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StateThinking && sess.State != session.StateToolExecuting && sess.State != session.StateAwaitingApproval && sess.State != session.StatePaused {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotCancellable, sess.State)
	}

	// Stop every background process owned by this session via the manager
	// (kills the process group, closes pipes, unregisters). Sync-mode python
	// has already exited by the time we get here; nothing to clean up for it.
	l.stopSessionBackgrounds(sessionID)

	sess, err = l.store.Get(sessionID)
	if err == nil {
		sess.ChildPID = nil
		sess.BackgroundPIDs = nil
		l.store.Update(sess)
	}

	oldState := string(sess.State)
	sess.State = session.StateIdle
	sess.CancelRequested = true
	sess.PauseRequested = false
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session state to idle after cancel: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateIdle.ToSnakeCase(),
			"reason":    "cancelled",
		})
	}

	lc, ok := l.activeLoops.Load(sessionID)
	if !ok {
		return nil
	}

	loopCtx := lc.(*LoopContext)
	select {
	case loopCtx.CancelCh <- struct{}{}:
	default:
	}

	return nil
}

func (l *Loop) Complete(sessionID string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State == session.StateArchived {
		return fmt.Errorf("%w: session is archived", session.ErrSessionArchived)
	}

	if sess.State == session.StateThinking || sess.State == session.StateToolExecuting || sess.State == session.StateAwaitingApproval {
		l.stopSessionBackgrounds(sessionID)

		sess, err := l.store.Get(sessionID)
		if err == nil {
			sess.ChildPID = nil
			sess.BackgroundPIDs = nil
			l.store.Update(sess)
		}

		lc, ok := l.activeLoops.Load(sessionID)
		if ok {
			loopCtx := lc.(*LoopContext)
			select {
			case loopCtx.CancelCh <- struct{}{}:
			default:
			}
		} else {
			sess.CancelRequested = true
			sess.PauseRequested = false
			l.store.Update(sess)
		}
	}

	sess, err = l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session after cancel: %w", err)
	}

	oldState := string(sess.State)
	sess.State = session.StateCompleted
	sess.CancelRequested = false
	sess.PauseRequested = false
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session to completed: %w", err)
	}

	l.archiveManager.AppendSystemMessage(sessionID, fmt.Sprintf("会话已于 %s 被标记为完成。", sess.LastActiveAt.Format(time.RFC3339)))

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateCompleted.ToSnakeCase(),
			"reason":    "completed",
		})
	}

	return nil
}

func (l *Loop) Archive(sessionID string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StateCompleted {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotCompleted, sess.State)
	}

	oldState := string(sess.State)
	sess.State = session.StateArchived
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session to archived: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateArchived.ToSnakeCase(),
			"reason":    "archived",
		})
	}

	return nil
}

func (l *Loop) UpdateConfig(sessionID string, toolCallLimit int) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if toolCallLimit <= 0 {
		return fmt.Errorf("tool_call_limit must be greater than 0")
	}

	sess.ToolCallLimit = toolCallLimit
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session config: %w", err)
	}

	return nil
}

func (l *Loop) UpdateContextConfig(sessionID string, maxContextTokens, keepRecent *int) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if maxContextTokens != nil {
		if *maxContextTokens <= 0 {
			return fmt.Errorf("max_context_tokens must be greater than 0")
		}
		sess.MaxContextTokens = *maxContextTokens
	}

	if keepRecent != nil {
		if *keepRecent <= 0 {
			return fmt.Errorf("keep_recent must be greater than 0")
		}
		sess.KeepRecent = *keepRecent
	}

	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session context config: %w", err)
	}

	return nil
}

func (l *Loop) UpdateConcurrencyConfig(sessionID string, maxConcurrentToolCalls, maxConcurrentSubprocesses *int) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if maxConcurrentToolCalls != nil {
		if *maxConcurrentToolCalls < 0 {
			return fmt.Errorf("max_concurrent_tool_calls must be >= 0")
		}
		sess.MaxConcurrentToolCalls = *maxConcurrentToolCalls
	}

	if maxConcurrentSubprocesses != nil {
		if *maxConcurrentSubprocesses < 0 {
			return fmt.Errorf("max_concurrent_subprocesses must be >= 0")
		}
		sess.MaxConcurrentSubprocesses = *maxConcurrentSubprocesses
	}

	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session concurrency config: %w", err)
	}

	return nil
}
