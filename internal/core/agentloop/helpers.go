package agentloop

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/tools"
)

func killChildProcess(pid int) {
	if pid <= 0 {
		return
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid))
	} else {
		cmd = exec.Command("kill", "-9", fmt.Sprintf("%d", pid))
	}

	if err := cmd.Run(); err != nil {
		log.Printf("failed to kill child process %d: %v", pid, err)
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
		sess.ToolCallLimit = session.DefaultToolCallLimit
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

		oldState := string(sess.State)
		sess.State = session.StateIdle
		sess.LastLoopTerminationReason = session.LoopTerminationToolLimitReached
		sess.LastActiveAt = time.Now()
		l.store.Update(sess)

		eventBus.Publish("session_state_change", map[string]any{
			"old_state": oldState,
			"new_state": string(session.StateIdle),
			"reason":    "tool_limit_reached",
		})

		return true
	}

	return false
}

func (l *Loop) handleLoopError(sessionID string, err error) {
	log.Printf("agent loop error for session %s: %v", sessionID, err)

	sess, getErr := l.store.Get(sessionID)
	if getErr != nil {
		return
	}

	sess.State = session.StateIdle
	l.store.Update(sess)
}

func (l *Loop) recordChildPID(sessionID, toolName string, tr *tools.ToolResult) {
	if toolName != "execute_command" {
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

	sess.ChildPID = &pid
	l.store.Update(sess)
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
	case "execute_command":
		return workingDir
	default:
		return ""
	}
}
