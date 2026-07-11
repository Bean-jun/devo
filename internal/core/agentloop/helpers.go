package agentloop

import (
	"fmt"
	"path/filepath"
	"time"

	"devo/internal/config"
	"devo/internal/core/session"
	"devo/internal/pkg/process"
	"devo/internal/taskexec/tools"
)

func killAllBackgroundPIDs(pids []int) {
	for _, pid := range pids {
		process.KillProcessGroup(pid)
	}
}

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
	if sess.ToolCallLimit <= 0 {
		sess.ToolCallLimit = config.DefaultToolCallLimit
	}
	l.store.Update(sess)

	if sess.ToolCallCount >= sess.ToolCallLimit {
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

func (l *Loop) recordChildPID(sessionID, toolName string, tr *tools.ToolResult) {
	if toolName != "exec_python" {
		return
	}

	pid := extractPIDFromContent(tr.Content)
	if pid <= 0 {
		return
	}

	sess, err := l.store.Get(sessionID)
	if err != nil {
		return
	}

	bgPID := extractBGPIDFromContent(tr.Content)

	if bgPID > 0 {
		sess.BackgroundPIDs = append(sess.BackgroundPIDs, bgPID)
	} else {
		sess.ChildPID = &pid
	}

	l.store.Update(sess)
}

func extractBGPIDFromContent(content string) int {
	marker := "__DEVO_BG_PID__="
	idx := findLastIndex(content, marker)
	if idx < 0 {
		return 0
	}

	start := idx + len(marker)
	end := start
	for end < len(content) && content[end] >= '0' && content[end] <= '9' {
		end++
	}

	if end == start {
		return 0
	}

	pid := 0
	for i := start; i < end; i++ {
		pid = pid*10 + int(content[i]-'0')
	}
	return pid
}

func extractPIDFromContent(content string) int {
	marker := "__DEVO_CHILD_PID__="
	idx := findLastIndex(content, marker)
	if idx < 0 {
		return 0
	}

	start := idx + len(marker)
	end := start
	for end < len(content) && content[end] >= '0' && content[end] <= '9' {
		end++
	}

	if end == start {
		return 0
	}

	pid := 0
	for i := start; i < end; i++ {
		pid = pid*10 + int(content[i]-'0')
	}
	return pid
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
