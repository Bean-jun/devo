package agentloop

import (
	"context"
	"fmt"

	"devo/internal/core/session"
	"devo/internal/pkg/logging"
)

const crashRecoverySystemMessage = "系统检测到上次服务异常中断，当前会话已重置。未完成的工具调用已丢弃，请检查文件状态。"

func (l *Loop) RecoverCrashedSessions() error {
	ctx := context.Background()
	sessions, total, err := l.store.ListSessions("all", "", 0, 0)
	if err != nil {
		return fmt.Errorf("list sessions for crash recovery: %w", err)
	}

	if total > len(sessions) {
		sessions, _, err = l.store.ListSessions("all", "", total, 0)
		if err != nil {
			return fmt.Errorf("list all sessions for crash recovery: %w", err)
		}
	}

	for i := range sessions {
		sess := &sessions[i]

		switch sess.State {
		case session.StateThinking, session.StateToolExecuting, session.StateAwaitingApproval:
			l.recoverSession(ctx, sess)

		case session.StatePaused, session.StateIdle:
		}
	}

	return nil
}

func (l *Loop) recoverSession(ctx context.Context, sess *session.Session) {
	oldState := string(sess.State)

	// Note: we intentionally do NOT kill processes via sess.ChildPID /
	// sess.BackgroundPIDs here. Those fields are leftover from the pre-refactor
	// design (where exec_python emitted __DEVO_CHILD_PID__ / __DEVO_BG_PID__
	// markers). After devo crashed, those PIDs may have been recycled by the OS
	// and killing them risks hitting an unrelated process. New background
	// processes started under the new design are tracked in-memory by the
	// BackgroundProcessManager and are killed on App.Shutdown; anything that
	// survives a crash is left for the user or OS to handle.

	sess.State = session.StateIdle
	sess.ChildPID = nil
	sess.BackgroundPIDs = nil
	sess.CancelRequested = false
	sess.PauseRequested = false

	if err := l.store.Update(sess); err != nil {
		logging.Warn(ctx, "crash recovery: failed to update session",
			"session_id", sess.ID,
			"error", err,
		)
		return
	}

	sysMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleSystem,
		Content:   crashRecoverySystemMessage,
		CreatedAt: sess.LastActiveAt,
	}
	if err := l.store.AddMessage(sess.ID, sysMsg); err != nil {
		logging.Warn(ctx, "crash recovery: failed to add system message",
			"session_id", sess.ID,
			"error", err,
		)
	}

	l.archiveManager.AppendSystemMessage(sess.ID, crashRecoverySystemMessage)

	eventBus, err := l.store.GetEventBus(sess.ID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateIdle.ToSnakeCase(),
			"reason":    "error",
		})
	}

	logging.Info(ctx, "crash recovery: recovered session",
		"session_id", sess.ID,
		"old_state", oldState,
		"new_state", "Idle",
	)
}
