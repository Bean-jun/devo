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
		a.statusBar.TokenUsage = "0 token"
		a.statusBar.ContextUsage = ""

		found := false
		for i, s := range a.sessions {
			if s.ID == sess.ID {
				a.sessions[i] = *sess
				found = true
				break
			}
		}
		if !found {
			a.sessions = append(a.sessions, *sess)
		}

		a.sidebar.SetSessions(a.sessions)
		a.sidebar.ActiveID = sess.ID
		for i, s := range a.sessions {
			if s.ID == sess.ID {
				a.sidebar.Cursor = i
				break
			}
		}
		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		a.updateContextUsage(sess)
		a.ready = true
		a.focusInput()
		return tea.Batch(a.connectSSECmd(), a.refreshSessionCmd())

	case "message_sent":
		return nil

	case "sessions_listed":
		sessions := msg.Data.([]types.SessionInfo)
		a.sessions = sessions
		a.sidebar.SetSessions(sessions)
		if a.activeSession != nil {
			a.sidebar.ActiveID = a.activeSession.ID
			for i, s := range sessions {
				if s.ID == a.activeSession.ID {
					a.sidebar.Cursor = i
					break
				}
			}
		}
		return nil

	case "session_loaded":
		sess := msg.Data.(*types.SessionInfo)
		a.activeSession = sess
		a.msgs = nil
		a.sidebar.ActiveID = sess.ID
		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		a.statusBar.TokenUsage = ""
		a.statusBar.ContextUsage = ""
		if sess.TokenUsage.Total > 0 {
			a.statusBar.TokenUsage = fmt.Sprintf("%d token (↑%d ↓%d)",
				sess.TokenUsage.Total, sess.TokenUsage.Input, sess.TokenUsage.Output)
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
		a.statusBar.SessionState = "Processing"
		a.chatView.Processing = true
		return nil

	case "pause_done", "resume_done", "cancel_done":
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
	maxTokens := sess.MaxContextTokens
	if maxTokens <= 0 {
		maxTokens = 128000
	}
	used := a.estimateContextTokens()
	if used == 0 && len(a.msgs) > 0 {
		used = len(a.msgs) * 50
	}
	pct := float64(used) / float64(maxTokens) * 100
	a.statusBar.ContextUsage = fmt.Sprintf("上下文 %s/%s (%.1f%%)",
		formatTokens(used), formatTokens(maxTokens), pct)
}

func (a *App) estimateContextTokens() int {
	total := 0
	for _, msg := range a.msgs {
		if msg.Content == "" {
			continue
		}
		tokens := (len(msg.Content) + 3) / 4
		if tokens < 1 {
			tokens = 1
		}
		total += tokens
	}
	return total
}

func formatTokens(n int) string {
	return fmt.Sprintf("%.1fK", float64(n)/1000)
}
