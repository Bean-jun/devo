package tui

import (
	"fmt"
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

	if a.showSessionPicker {
		switch key {
		case "esc":
			a.showSessionPicker = false
			a.sessionPicker.Hide()
			a.focusInput()
			return nil
		case "enter":
			selected := a.sessionPicker.SelectedSession()
			a.showSessionPicker = false
			a.sessionPicker.Hide()
			a.focusInput()
			if selected != nil && selected.ID != a.activeSession.ID {
				return a.switchSessionCmd(selected.ID)
			}
			return nil
		case "up":
			a.sessionPicker.CursorUp()
			return nil
		case "down":
			a.sessionPicker.CursorDown()
			return nil
		case "backspace":
			if len(a.sessionPicker.Query) > 0 {
				a.sessionPicker.Query = a.sessionPicker.Query[:len(a.sessionPicker.Query)-1]
			}
			return nil
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= ' ' {
				a.sessionPicker.Query += string(msg.Runes[0])
			}
			return nil
		}
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
			if len(a.commandPalette.Items) == 0 {
				return nil
			}
			label := a.commandPalette.SelectedLabel()
			a.showCommandPalette = false
			a.commandPalette.Hide()
			a.chatView.InputArea.SetValue(label + " ")
			a.chatView.Focus()
			a.focusInput()
			return nil
		case "up":
			a.commandPalette.CursorUp()
			return nil
		case "down":
			a.commandPalette.CursorDown()
			return nil
		case "backspace":
			if len(a.commandPalette.Query) > 0 {
				a.commandPalette.Query = a.commandPalette.Query[:len(a.commandPalette.Query)-1]
				a.commandPalette.Filter()
			}
			return nil
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= ' ' {
				a.commandPalette.Query += string(msg.Runes[0])
				a.commandPalette.Filter()
			}
			return nil
		}
	}

	if a.state == StateAwaitingApproval {
		if a.yoloMode {
			return func() tea.Msg { return messages.ApprovalDecision{Approved: true} }
		}
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

	case "alt+y":
		a.keyConsumed = true
		return a.toggleYOLO()

	case "enter":
		if a.chatView.InputArea.Focused() && (a.state == StateReady || a.state == StateCancelled) {
			if a.chatView.InputArea.IsPasteActive() {
				return nil
			}
			content := strings.TrimSpace(a.chatView.InputArea.Value())
			if content != "" {
				a.pushInputHistory(content)
				if strings.HasPrefix(content, "/") {
					return a.executeSlashCommand(content)
				}
				return a.sendMessageCmd(content)
			}
			expanded := false
			for i := range a.chatView.MessageView.ToolCards {
				card := &a.chatView.MessageView.ToolCards[i]
				if !card.Expanded {
					a.chatView.MessageView.ToggleToolCardExpanded(i)
					expanded = true
					break
				}
			}
			if !expanded {
				for i := range a.chatView.MessageView.ToolCards {
					if a.chatView.MessageView.ToolCards[i].Expanded {
						a.chatView.MessageView.ToggleToolCardExpanded(i)
					}
				}
			}
		}

	case "/":
		if a.chatView.InputArea.Focused() && (a.state == StateReady || a.state == StateCancelled) {
			if strings.TrimSpace(a.chatView.InputArea.Value()) == "" {
				a.showCommandPalette = true
				a.commandPalette.Show()
				a.chatView.Blur()
				return nil
			}
		}

	case "esc":
		if a.showHelpPanel {
			a.showHelpPanel = false
			a.helpPanel.Hide()
			return nil
		}
		if a.state == StateToolExecuting {
			return a.pauseCmd()
		}
		if a.state == StatePaused || a.statusBar.SessionState == "paused" {
			return a.cancelCmd()
		}
		if a.state == StateThinking || a.state == StateProcessing {
			return a.cancelCmd()
		}
		if a.chatView.InputArea.Focused() {
			a.chatView.Blur()
		} else {
			a.chatView.Focus()
		}

	case "shift+up":
		if a.chatView.InputArea.Focused() && !a.showCommandPalette {
			a.historyPrev()
		}

	case "shift+down":
		if a.chatView.InputArea.Focused() && !a.showCommandPalette {
			a.historyNext()
		}

	case "up":
	case "down":
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

func (a *App) executeSlashCommand(input string) tea.Cmd {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.TrimPrefix(parts[0], "/")
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	a.chatView.InputArea.Reset()

	switch cmd {
	case "new":
		return a.newSessionCmd(arg)

	case "rename":
		if a.activeSession == nil {
			return nil
		}
		return a.renameSessionCmd(arg)

	case "switch":
		a.sessionPicker.Sessions = a.sessions
		if a.activeSession != nil {
			a.sessionPicker.ActiveID = a.activeSession.ID
		}
		a.showSessionPicker = true
		a.sessionPicker.Show()
		a.chatView.Blur()
		return a.refreshSessionCmd()

	case "rollback":
		if a.activeSession == nil {
			return nil
		}
		a.chatView.Blur()
		return a.loadAllMessagesCmd(a.activeSession.ID)

	case "pause":
		if a.activeSession == nil {
			return nil
		}
		if a.activeSession.State != "processing" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法暂停", a.activeSession.State), true)
			return nil
		}
		return a.pauseCmd()

	case "resume":
		if a.activeSession == nil {
			return nil
		}
		if a.activeSession.State != "paused" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法恢复", a.activeSession.State), true)
			return nil
		}
		return a.resumeCmd()

	case "cancel":
		if a.activeSession == nil {
			return nil
		}
		state := a.activeSession.State
		if state != "processing" && state != "awaiting_approval" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法取消", state), true)
			return nil
		}
		return a.cancelCmd()

	case "yolo":
		return a.toggleYOLO()

	case "trust":
		if arg == "" {
			a.toast.Show("用法: /trust <level>  (low, normal, elevated)", true)
			return nil
		}
		return a.setTrustCmd(arg)

	case "help":
		a.showHelpPanel = true
		a.helpPanel.Show()
		return nil

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
