package tui

import (
	"strings"

	"devo/internal/interfaces/tui/messages"
	"devo/internal/interfaces/tui/types"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) initSessionCmd() tea.Cmd {
	return func() tea.Msg {
		dirName := a.workingDir
		if idx := strings.LastIndex(dirName, "/"); idx >= 0 {
			dirName = dirName[idx+1:]
		}
		if idx := strings.LastIndex(dirName, "\\"); idx >= 0 {
			dirName = dirName[idx+1:]
		}

		Log("[API] POST /api/v1/sessions working_directory=%s title=%s", a.workingDir, dirName)
		sess, err := a.apiClient.CreateSession(a.workingDir, dirName)
		if err != nil {
			Log("[API] CreateSession failed: %v", err)
			return messages.APIResponse{Kind: "init_session", Err: err}
		}

		Log("[API] CreateSession OK: id=%s", sess.ID)
		return messages.APIResponse{Kind: "init_session", Data: sess}
	}
}

func (a *App) sendMessageCmd(content string) tea.Cmd {
	userMsg := types.Message{
		Role:    "user",
		Content: content,
	}
	a.msgs = append(a.msgs, userMsg)
	a.chatView.MessageView.AddMessage(userMsg)
	a.chatView.InputArea.Reset()
	a.state = StateProcessing
	a.statusBar.SessionState = "Processing"
	a.chatView.Processing = true
	if a.activeSession != nil {
		a.updateContextUsage(a.activeSession)
	}

	return func() tea.Msg {
		err := a.apiClient.SendMessage(a.activeSession.ID, content)
		if err != nil {
			return messages.APIResponse{Kind: "message_sent", Err: err}
		}
		return messages.APIResponse{Kind: "message_sent"}
	}
}

func (a *App) sendApprovalCmd(approvalID string, approved bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if approved {
			err = a.apiClient.Approve(a.activeSession.ID, approvalID)
		} else {
			err = a.apiClient.Reject(a.activeSession.ID, approvalID)
		}
		if err != nil {
			return messages.APIResponse{Kind: "approval_done", Err: err}
		}
		return messages.APIResponse{Kind: "approval_done"}
	}
}

func (a *App) cancelCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.apiClient.Cancel(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "cancel_done", Err: err}
		}
		return messages.APIResponse{Kind: "cancel_done"}
	}
}

func (a *App) pauseCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.apiClient.Pause(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "pause_done", Err: err}
		}
		return messages.APIResponse{Kind: "pause_done"}
	}
}

func (a *App) resumeCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.apiClient.Resume(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "resume_done", Err: err}
		}
		return messages.APIResponse{Kind: "resume_done"}
	}
}

func (a *App) switchSessionCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		if a.sseClient != nil {
			a.sseClient.Disconnect()
		}

		sess, err := a.apiClient.GetSession(sessionID)
		if err != nil {
			return messages.APIResponse{Kind: "session_loaded", Err: err}
		}

		return messages.APIResponse{Kind: "session_loaded", Data: sess}
	}
}

func (a *App) newSessionCmd() tea.Cmd {
	return func() tea.Msg {
		dirName := a.workingDir
		if idx := strings.LastIndex(dirName, "/"); idx >= 0 {
			dirName = dirName[idx+1:]
		}
		if idx := strings.LastIndex(dirName, "\\"); idx >= 0 {
			dirName = dirName[idx+1:]
		}

		sess, err := a.apiClient.CreateSession(a.workingDir, dirName)
		if err != nil {
			return messages.APIResponse{Kind: "init_session", Err: err}
		}

		return messages.APIResponse{Kind: "init_session", Data: sess}
	}
}

func (a *App) refreshSessionCmd() tea.Cmd {
	return func() tea.Msg {
		sessions, err := a.apiClient.ListSessions(20, 0)
		if err != nil {
			return messages.APIResponse{Kind: "sessions_listed", Err: err}
		}

		if a.activeSession != nil {
			sess, err := a.apiClient.GetSession(a.activeSession.ID)
			if err == nil {
				a.activeSession = sess
				a.statusBar.SessionState = sess.State
			}
		}

		return messages.APIResponse{Kind: "sessions_listed", Data: sessions}
	}
}

func (a *App) loadMessagesCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := a.apiClient.GetMessages(sessionID, 50, 0)
		if err != nil {
			return messages.APIResponse{Kind: "messages_loaded", Err: err}
		}
		return messages.APIResponse{Kind: "messages_loaded", Data: msgs}
	}
}

func (a *App) loadAllMessagesCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := a.apiClient.GetMessages(sessionID, 1000, 0)
		if err != nil {
			return messages.APIResponse{Kind: "rollback_messages_loaded", Err: err}
		}
		return messages.APIResponse{Kind: "rollback_messages_loaded", Data: msgs}
	}
}

func (a *App) rollbackCmd(targetMessageID string) tea.Cmd {
	return func() tea.Msg {
		result, err := a.apiClient.Rollback(a.activeSession.ID, targetMessageID)
		if err != nil {
			return messages.APIResponse{Kind: "rollback_done", Err: err}
		}
		return messages.APIResponse{Kind: "rollback_done", Data: result}
	}
}

func (a *App) exportArchiveCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := a.apiClient.SyncArchive(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "archive_done", Err: err}
		}
		return messages.APIResponse{Kind: "archive_done", Data: result}
	}
}
