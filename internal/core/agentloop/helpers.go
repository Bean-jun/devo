package agentloop

import (
	"fmt"
	"path/filepath"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/tools"
)

func (l *Loop) rejectionMessage(tc session.ToolCall) session.Message {
	return session.Message{
		ID:         session.GenerateID("msg"),
		Role:       session.RoleTool,
		Content:    "操作被用户拒绝",
		ToolCallID: tc.ID,
		CreatedAt:  time.Now(),
	}
}

func (l *Loop) toolResultToMessage(tr *tools.ToolResult) session.Message {
	content := tr.Content
	if !tr.Success {
		content = "错误: " + tr.Error
	}
	return session.Message{
		ID:         session.GenerateID("msg"),
		Role:       session.RoleTool,
		Content:    content,
		ToolCallID: tr.ToolCallID,
		CreatedAt:  time.Now(),
	}
}

func (l *Loop) incrementToolCallCount(sessionID string, eventBus *session.EventBus) (shouldStop bool) {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return false
	}

	sess.ToolCallCount++
	l.store.Update(sess)

	if sess.ToolCallCount >= l.cfg.ToolCallLimit {
		sess, err = l.store.Get(sessionID)
		if err != nil {
			return false
		}

		limitMsg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleSystem,
			Content:   fmt.Sprintf("AI 已进行 %d 次工具调用，达到上限暂停。任务尚未完成，是否继续？", sess.ToolCallCount),
			CreatedAt: time.Now(),
		}
		l.store.AddMessage(sessionID, limitMsg)
		l.archiveManager.AppendSystemMessage(sessionID, limitMsg.Content)

		oldState := string(sess.State)
		sess.State = session.StateIdle
		sess.LastLoopTerminationReason = session.LoopTerminationToolLimitReached
		sess.LastActiveAt = time.Now()
		l.store.Update(sess)

		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateIdle.ToSnakeCase(),
			"reason":    "tool_limit_reached",
		})

		return true
	}

	return false
}

// stopSessionBackgrounds kills every background process owned by sessionID via
// the BackgroundProcessManager. Returns when all kill goroutines have completed.
// No-op if the manager isn't configured (e.g. in unit tests).
func (l *Loop) stopSessionBackgrounds(sessionID string) {
	if l.bgManager == nil {
		return
	}
	_ = l.bgManager.StopSession(sessionID)
}

func (l *Loop) getLockPath(workingDir, toolName string, params map[string]interface{}) string {
	switch toolName {
	case "write_file", "edit_file":
		if path, ok := params["path"].(string); ok && path != "" {
			return filepath.Join(workingDir, path)
		}
		return workingDir
	case "exec_python":
		return workingDir
	default:
		return ""
	}
}
