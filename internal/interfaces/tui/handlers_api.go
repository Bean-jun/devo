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
		a.activeSession = sess

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
		a.ready = true
		a.focusInput()
		return tea.Batch(a.connectSSECmd(), a.refreshSessionCmd())

	case "message_sent":
		return nil

	case "sessions_listed":
		sessions := msg.Data.([]types.SessionInfo)
		a.sessions = sessions
		a.sidebar.SetSessions(sessions)
		return nil

	case "session_loaded":
		sess := msg.Data.(*types.SessionInfo)
		a.activeSession = sess
		a.sidebar.ActiveID = sess.ID
		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		if sess.TokenUsage.Total > 0 {
			a.statusBar.TokenUsage = fmt.Sprintf("%d token (↑%d ↓%d)",
				sess.TokenUsage.Total, sess.TokenUsage.Input, sess.TokenUsage.Output)
		}
		a.focusInput()
		return tea.Batch(a.connectSSECmd(), a.loadMessagesCmd(sess.ID))

	case "messages_loaded":
		msgs := msg.Data.([]types.Message)
		a.msgs = msgs
		a.chatView.MessageView.SetMessages(msgs)
		return nil

	case "approval_done":
		a.state = StateProcessing
		a.statusBar.SessionState = "Processing"
		a.chatView.Processing = true
		return nil

	case "pause_done", "resume_done", "cancel_done":
		return a.refreshSessionCmd()

	case "trust_set", "policy_set":
		a.toast.Show("设置已更新", false)
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
