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

	if sess.State != session.StateProcessing {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotProcessing, sess.State)
	}

	sess.PauseRequested = true
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update pause flag: %w", err)
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
	sess.State = session.StateProcessing
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session state: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": oldState,
			"new_state": string(session.StateProcessing),
			"reason":    "resumed",
		})
	}

	return nil
}

func (l *Loop) Cancel(sessionID string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StateProcessing && sess.State != session.StateAwaitingApproval {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotCancellable, sess.State)
	}

	childPID := sess.ChildPID
	bgPIDs := sess.BackgroundPIDs

	sess.CancelRequested = true
	sess.PauseRequested = false
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update cancel flag: %w", err)
	}

	if childPID != nil {
		killChildProcess(*childPID)
	}
	if len(bgPIDs) > 0 {
		killAllBackgroundPIDs(bgPIDs)
	}

	sess, err = l.store.Get(sessionID)
	if err == nil {
		sess.ChildPID = nil
		sess.BackgroundPIDs = nil
		l.store.Update(sess)
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

	if sess.State == session.StateProcessing || sess.State == session.StateAwaitingApproval {
		childPID := sess.ChildPID
		bgPIDs := sess.BackgroundPIDs
		sess.CancelRequested = true
		sess.PauseRequested = false
		if err := l.store.Update(sess); err != nil {
			return fmt.Errorf("update cancel flag: %w", err)
		}

		if childPID != nil {
			killChildProcess(*childPID)
		}
		if len(bgPIDs) > 0 {
			killAllBackgroundPIDs(bgPIDs)
		}

		sess, err := l.store.Get(sessionID)
		if err == nil {
			sess.ChildPID = nil
			sess.BackgroundPIDs = nil
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
			"old_state": oldState,
			"new_state": string(session.StateCompleted),
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
			"old_state": oldState,
			"new_state": string(session.StateArchived),
			"reason":    "archived",
		})
	}

	return nil
}

func (l *Loop) checkControlFlags(sessionID string, eventBus *session.EventBus) (shouldStop bool) {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return false
	}

	if sess.CancelRequested {
		oldState := string(sess.State)
		sess.State = session.StateIdle
		sess.CancelRequested = false
		sess.PauseRequested = false
		sess.LastActiveAt = time.Now()
		l.store.Update(sess)

		l.archiveManager.AppendSystemMessage(sessionID, fmt.Sprintf("会话已于 %s 被用户取消。", sess.LastActiveAt.Format(time.RFC3339)))

		eventBus.Publish("session_state_change", map[string]any{
			"old_state": oldState,
			"new_state": string(session.StateIdle),
			"reason":    "cancelled",
		})
		return true
	}

	if sess.PauseRequested {
		oldState := string(sess.State)
		sess.State = session.StatePaused
		sess.PauseRequested = false
		sess.CancelRequested = false
		sess.LastActiveAt = time.Now()
		l.store.Update(sess)

		eventBus.Publish("session_state_change", map[string]any{
			"old_state": oldState,
			"new_state": string(session.StatePaused),
			"reason":    "paused",
		})
		return true
	}

	return false
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
