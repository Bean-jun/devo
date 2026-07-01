package agentloop

import (
	"fmt"
	"log"

	"devo/internal/core/session"
)

const crashRecoverySystemMessage = "系统检测到上次服务异常中断，当前会话已重置。未完成的工具调用已丢弃，请检查文件状态。"

func (l *Loop) RecoverCrashedSessions() error {
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
			l.recoverSession(sess)

		case session.StatePaused, session.StateIdle:
		}
	}

	return nil
}

func (l *Loop) recoverSession(sess *session.Session) {
	oldState := string(sess.State)

	if sess.ChildPID != nil {
		killChildProcess(*sess.ChildPID)
	}
	if len(sess.BackgroundPIDs) > 0 {
		killAllBackgroundPIDs(sess.BackgroundPIDs)
	}

	sess.State = session.StateIdle
	sess.ChildPID = nil
	sess.BackgroundPIDs = nil
	sess.CancelRequested = false
	sess.PauseRequested = false

	if err := l.store.Update(sess); err != nil {
		log.Printf("[crash-recovery] failed to update session %s: %v", sess.ID, err)
		return
	}

	sysMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleSystem,
		Content:   crashRecoverySystemMessage,
		CreatedAt: sess.LastActiveAt,
	}
	if err := l.store.AddMessage(sess.ID, sysMsg); err != nil {
		log.Printf("[crash-recovery] failed to add system message to session %s: %v", sess.ID, err)
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

	log.Printf("[crash-recovery] recovered session %s: %s -> Idle", sess.ID, oldState)
}
