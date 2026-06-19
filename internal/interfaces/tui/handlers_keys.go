package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"devo/internal/interfaces/tui/messages"
)

func (a *App) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

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
		case "up", "k":
			a.commandPalette.CursorUp()
			return nil
		case "down", "j":
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

func (a *App) executeCommand(action string) tea.Cmd {
	switch action {
	case "new":
		return a.newSessionCmd()
	case "cancel":
		return a.cancelCmd()
	case "pause":
		if a.activeSession != nil {
			if a.activeSession.State == "Processing" {
				return a.pauseCmd()
			} else if a.activeSession.State == "Paused" {
				return a.resumeCmd()
			}
		}
		return nil
	case "clear":
		return tea.ClearScreen
	case "quit":
		a.state = StateQuitting
		return tea.Quit
	}
	return nil
}
