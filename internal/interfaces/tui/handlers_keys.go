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

	if a.showInputPrompt {
		switch key {
		case "esc":
			a.showInputPrompt = false
			a.inputPrompt.Hide()
			a.focusInput()
			return nil
		case "enter":
			value := a.inputPrompt.Value
			a.showInputPrompt = false
			a.inputPrompt.Hide()
			a.focusInput()
			return a.executeInputPromptAction(value)
		case "backspace":
			if len(a.inputPrompt.Value) > 0 {
				a.inputPrompt.Value = a.inputPrompt.Value[:len(a.inputPrompt.Value)-1]
			}
			return nil
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= ' ' {
				a.inputPrompt.Value += string(msg.Runes[0])
			}
			return nil
		}
	}

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

	case "ctrl+p":
		if a.activeSession == nil {
			return nil
		}
		state := a.activeSession.State
		if state == "Processing" {
			return a.pauseCmd()
		}
		if state == "Paused" {
			return a.resumeCmd()
		}
		a.toast.Show(fmt.Sprintf("当前状态为 %s，无法切换", state), true)
		return nil

	case "ctrl+r":
		if a.activeSession == nil {
			return nil
		}
		state := a.activeSession.State
		if state != "Paused" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法恢复", state), true)
			return nil
		}
		return a.resumeCmd()

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
		} else {
			a.chatView.Focus()
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

func (a *App) executeCommand(action string) tea.Cmd {
	switch action {
	case "new":
		a.pendingInputAction = "new"
		a.showInputPrompt = true
		a.inputPrompt.Show("创建新会话", "会话名称（可选，留空则自动生成）", "", "[创建  Enter]")
		return nil
	case "rollback":
		return a.loadAllMessagesCmd(a.activeSession.ID)
	case "switch":
		a.sessionPicker.Sessions = a.sessions
		if a.activeSession != nil {
			a.sessionPicker.ActiveID = a.activeSession.ID
		}
		a.showSessionPicker = true
		a.sessionPicker.Show()
		return a.refreshSessionCmd()
	case "rename":
		if a.activeSession == nil {
			return nil
		}
		a.pendingInputAction = "rename"
		a.showInputPrompt = true
		a.inputPrompt.Show("重命名当前会话", "输入新名称", a.activeSession.Title, "[重命名  Enter]")
		return nil
	case "cancel":
		if a.activeSession == nil {
			return nil
		}
		state := a.activeSession.State
		if state != "Processing" && state != "AwaitingApproval" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法取消", state), true)
			return nil
		}
		return a.cancelCmd()
	case "pause":
		if a.activeSession == nil {
			return nil
		}
		state := a.activeSession.State
		if state != "Processing" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法暂停", state), true)
			return nil
		}
		return a.pauseCmd()
	case "resume":
		if a.activeSession == nil {
			return nil
		}
		state := a.activeSession.State
		if state != "Paused" {
			a.toast.Show(fmt.Sprintf("当前状态为 %s，无法恢复", state), true)
			return nil
		}
		return a.resumeCmd()
	case "help":
		a.toast.Show("快捷键: Ctrl+P 暂停/恢复  Ctrl+R 恢复  Ctrl+C 取消  Ctrl+Q 退出", false)
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

func (a *App) executeInputPromptAction(value string) tea.Cmd {
	switch a.pendingInputAction {
	case "new":
		return a.newSessionCmd(value)
	case "rename":
		if a.activeSession != nil {
			return a.renameSessionCmd(value)
		}
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
