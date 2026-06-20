package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/messages"
)

func (a *App) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if a.showRollbackPicker {
		switch key {
		case "esc":
			a.showRollbackPicker = false
			a.rollbackPicker.Hide()
			a.chatView.InputArea.Reset()
			a.focusInput()
			return nil
		case "enter":
			msgID := a.rollbackPicker.SelectedMessageID()
			a.showRollbackPicker = false
			a.rollbackPicker.Hide()
			a.chatView.InputArea.Reset()
			a.focusInput()
			if msgID != "" {
				return a.rollbackCmd(msgID)
			}
			return nil
		case "up":
			a.rollbackPicker.CursorUp()
			return nil
		case "down":
			a.rollbackPicker.CursorDown()
			return nil
		}
		return nil
	}

	if a.showCommandPalette {
		switch key {
		case "esc":
			a.showCommandPalette = false
			a.commandPalette.Hide()
			a.chatView.InputArea.Reset()
			a.focusInput()
			return nil
		case "enter":
			action := a.commandPalette.SelectedAction()
			a.showCommandPalette = false
			a.commandPalette.Hide()
			a.chatView.InputArea.Reset()
			a.focusInput()
			return a.executeCommand(action)
		case "up":
			a.commandPalette.CursorUp()
			return nil
		case "down":
			a.commandPalette.CursorDown()
			return nil
		}
		return nil
	}

	if a.state == StateAwaitingApproval {
		switch key {
		case "y", "Y":
			return func() tea.Msg { return messages.ApprovalDecision{Approved: true} }
		case "n", "N", "esc":
			return func() tea.Msg { return messages.ApprovalDecision{Approved: false} }
		case "d", "D":
			a.approvalModal.Update(msg)
			return nil
		}
		return nil
	}

	switch key {
	case "ctrl+c":
		return a.cancelCmd()

	case "ctrl+q":
		a.state = StateQuitting
		return tea.Quit

	case "ctrl+b":
		a.showSidebar = !a.showSidebar
		a.layout()
		return nil

	case "enter":
		if a.chatView.InputArea.Focused() && a.state == StateReady {
			content := strings.TrimSpace(a.chatView.InputArea.Value())
			if content != "" {
				if strings.HasPrefix(content, "/") {
					if strings.TrimSpace(content) == "/rollback" {
						a.chatView.InputArea.Reset()
						a.chatView.Blur()
						return a.loadAllMessagesCmd(a.activeSession.ID)
					}
					a.showCommandPalette = true
					a.commandPalette.Show()
					a.chatView.Blur()
					return nil
				}
				return a.sendMessageCmd(content)
			}
		} else if a.showSidebar {
			selected := a.sidebar.SelectedSession()
			if selected != nil && selected.ID != a.sidebar.ActiveID {
				return a.switchSessionCmd(selected.ID)
			}
		}

	case "/":
		if a.chatView.InputArea.Focused() && a.state == StateReady {
			if strings.TrimSpace(a.chatView.InputArea.Value()) == "" {
				a.showCommandPalette = true
				a.commandPalette.Show()
				a.chatView.Blur()
				return nil
			}
		}

	case "esc":
		if a.chatView.InputArea.Focused() {
			a.chatView.Blur()
			a.statusBar.Mode = "Sessions"
		} else {
			a.chatView.Focus()
			a.statusBar.Mode = ""
		}

	case "up":
		if a.showSidebar && !a.chatView.InputArea.Focused() {
			a.sidebar.CursorUp()
		}

	case "down":
		if a.showSidebar && !a.chatView.InputArea.Focused() {
			a.sidebar.CursorDown()
		}
	}

	return nil
}

func (a *App) buildRollbackItems() {
	var items []components.RollbackItem
	for i := len(a.msgs) - 1; i >= 0; i-- {
		msg := a.msgs[i]
		if msg.Role != "user" {
			continue
		}
		if msg.Content == "" {
			continue
		}
		timePrefix := formatTimeForDisplay(msg.CreatedAt)
		displayText := timePrefix + " " + truncateForDisplay(msg.Content, 60)
		items = append(items, components.RollbackItem{
			MessageID:   msg.ID,
			Content:     msg.Content,
			DisplayText: displayText,
			CreatedAt:   msg.CreatedAt,
		})
	}
	a.rollbackPicker.Show(items)
}

func (a *App) executeCommand(action string) tea.Cmd {
	switch action {
	case "new":
		return a.newSessionCmd()
	case "rollback":
		return a.loadAllMessagesCmd(a.activeSession.ID)
	case "cancel":
		return a.cancelCmd()
	case "usage":
		usage := a.statusBar.TokenUsage
		if usage == "" || usage == "0 tok" {
			usage = "暂无 Token 消耗数据"
		}
		a.toast.Show("Token 用量: "+usage, false)
		return nil
	case "pause":
		if a.activeSession != nil {
			switch a.activeSession.State {
			case "Processing":
				return a.pauseCmd()
			case "Paused":
				return a.resumeCmd()
			}
		}
		return nil
	case "clear":
		return tea.ClearScreen
	case "quit":
		a.state = StateQuitting
		return tea.Quit
	case "export":
		if a.activeSession != nil {
			return a.exportArchiveCmd()
		}
		return nil
	}
	return nil
}

func truncateForDisplay(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	runes := []rune(text)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return text
}

func formatTimeForDisplay(createdAt string) string {
	if createdAt == "" {
		return ""
	}
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, createdAt); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return ""
}
