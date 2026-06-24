package tui

import (
	"devo/internal/interfaces/tui/messages"
	"devo/internal/interfaces/tui/types"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleAPIResponse(msg messages.APIResponse) tea.Cmd {
	if msg.Err != nil {
		Log("[API] Response error (kind=%s): %v", msg.Kind, msg.Err)
		a.toast.Show(msg.Err.Error(), true)
		return nil
	}

	Log("[API] Response OK: kind=%s", msg.Kind)

	switch msg.Kind {
	case "init_session":
		sess := msg.Data.(*types.SessionInfo)
		if a.sseClient != nil {
			a.sseClient.Disconnect()
		}
		a.activeSession = sess
		a.msgs = nil
		a.chatView.MessageView.SetMessages(nil)
		a.chatView.InputArea.TokenUsage = ""
		a.chatView.InputArea.ContextUsage = ""

		found := false
		for i, s := range a.sessions {
			if s.ID == sess.ID {
				sess.NormalizeState()
				a.sessions[i] = *sess
				found = true
				break
			}
		}
		if !found {
			sess.NormalizeState()
			a.sessions = append(a.sessions, *sess)
		}

		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		if sess.TokenUsage.Total > 0 {
			a.chatView.InputArea.TokenUsage = fmt.Sprintf("Tokens %s (↑%s ↓%s)",
				formatTokens(sess.TokenUsage.Total), formatTokens(sess.TokenUsage.Input), formatTokens(sess.TokenUsage.Output))
		}
		a.updateContextUsage(sess)
		a.ready = true
		a.focusInput()
		return tea.Batch(a.connectSSECmd(), a.refreshSessionCmd())

	case "message_sent":
		return nil

	case "sessions_listed":
		sessions := msg.Data.([]types.SessionInfo)
		a.sessions = sessions
		if a.showSessionPicker {
			a.sessionPicker.Sessions = sessions
		}
		return nil

	case "session_loaded":
		sess := msg.Data.(*types.SessionInfo)
		sess.NormalizeState()
		a.activeSession = sess
		a.msgs = nil
		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		a.chatView.InputArea.TokenUsage = ""
		a.chatView.InputArea.ContextUsage = ""
		if sess.TokenUsage.Total > 0 {
			a.chatView.InputArea.TokenUsage = fmt.Sprintf("Tokens %s (↑%s ↓%s)",
				formatTokens(sess.TokenUsage.Total), formatTokens(sess.TokenUsage.Input), formatTokens(sess.TokenUsage.Output))
		}
		a.updateContextUsage(sess)
		a.focusInput()
		return tea.Batch(a.connectSSECmd(), a.loadMessagesCmd(sess.ID))

	case "messages_loaded":
		msgs := msg.Data.([]types.Message)
		a.msgs = msgs
		a.chatView.MessageView.SetMessages(msgs)
		if a.activeSession != nil {
			a.updateContextUsage(a.activeSession)
		}
		return nil

	case "rollback_messages_loaded":
		msgs := msg.Data.([]types.Message)
		a.msgs = msgs
		a.showRollbackPicker = true
		a.buildRollbackItems()
		return nil

	case "approval_done":
		a.state = StateProcessing
		a.statusBar.SessionState = "processing"
		a.chatView.Processing = true
		return nil

	case "pause_done", "resume_done", "cancel_done":
		return a.refreshSessionCmd()

	case "rename_done":
		if msg.Err != nil {
			a.toast.Show(fmt.Sprintf("重命名失败: %v", msg.Err), true)
		} else {
			a.toast.Show("重命名成功", false)
		}
		return a.refreshSessionCmd()

	case "rollback_done":
		result := msg.Data.(*types.RollbackResult)
		toastMsg := fmt.Sprintf("已回滚，删除了 %d 条消息", result.DeletedCount)
		if result.Adjusted {
			toastMsg += "（吸附已自动调整）"
		}
		a.toast.Show(toastMsg, false)
		return a.loadMessagesCmd(a.activeSession.ID)

	case "trust_set", "policy_set":
		a.toast.Show("设置已更新", false)
		return nil

	case "archive_done":
		result := msg.Data.(*types.SyncArchiveResult)
		toastMsg := fmt.Sprintf("会话存档已导出: %s", result.ArchivePath)
		a.toast.Show(toastMsg, false)
		return nil
	}

	return nil
}

func (a *App) handleApprovalDecision(msg messages.ApprovalDecision) tea.Cmd {
	pending := a.approvalModal.Request
	a.approvalModal.Hide()

	if msg.Approved {
		a.toast.Show("已批准，正在执行...", false)
		if pending != nil {
			return a.sendApprovalCmd(pending.ApprovalID, true)
		}
	} else {
		a.toast.Show("已拒绝", false)
		if pending != nil {
			return a.sendApprovalCmd(pending.ApprovalID, false)
		}
	}
	return nil
}

func (a *App) updateContextUsage(sess *types.SessionInfo) {
	if sess != nil && sess.TokenUsage.Input > 0 {
		a.chatView.InputArea.ContextUsage = fmt.Sprintf("context %s", formatTokens(sess.TokenUsage.Input))
	}
}

func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	v := float64(n) / 1000
	if v >= 10 {
		return fmt.Sprintf("%.0fK", v)
	}
	return fmt.Sprintf("%.1fK", v)
}
