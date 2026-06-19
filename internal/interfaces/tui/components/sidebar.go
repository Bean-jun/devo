package components

import (
	"fmt"

	"devo/internal/interfaces/tui/types"

	"github.com/charmbracelet/lipgloss"
)

type SessionSidebar struct {
	Sessions []types.SessionInfo
	ActiveID string
	Cursor   int
	Width    int
	Height   int
	Visible  bool
}

func NewSessionSidebar() SessionSidebar {
	return SessionSidebar{
		Visible: true,
	}
}

func (s *SessionSidebar) SetSessions(sessions []types.SessionInfo) {
	s.Sessions = sessions
	if s.Cursor >= len(s.Sessions) {
		s.Cursor = len(s.Sessions) - 1
	}
	if s.Cursor < 0 {
		s.Cursor = 0
	}
}

func (s *SessionSidebar) CursorUp() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

func (s *SessionSidebar) CursorDown() {
	if s.Cursor < len(s.Sessions)-1 {
		s.Cursor++
	}
}

func (s *SessionSidebar) SelectedSession() *types.SessionInfo {
	if s.Cursor >= 0 && s.Cursor < len(s.Sessions) {
		return &s.Sessions[s.Cursor]
	}
	return nil
}

func (s *SessionSidebar) Toggle() {
	s.Visible = !s.Visible
}

func (s *SessionSidebar) View() string {
	if !s.Visible {
		return ""
	}

	content := lipgloss.NewStyle().Bold(true).Foreground(ColorMuted).Render("Sessions") + "\n\n"

	for i, sess := range s.Sessions {
		indicator := " "
		if sess.ID == s.ActiveID {
			indicator = "●"
		}

		stateColor := StateColor(sess.State)
		stateStr := lipgloss.NewStyle().Foreground(stateColor).Render(string(sess.State))

		title := sess.Title
		tokStr := ""
		if sess.TokenUsage.Total > 0 {
			tokStr = fmt.Sprintf(" %d tok", sess.TokenUsage.Total)
		}
		maxTitleLen := s.Width - 12 - len(tokStr)
		if maxTitleLen < 5 {
			maxTitleLen = 5
		}
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-1] + "…"
		}

		line := fmt.Sprintf("%s %s  %s%s", indicator, title, stateStr, tokStr)

		if i == s.Cursor {
			content += SidebarActiveStyle.Render(line) + "\n"
		} else {
			content += SidebarItemStyle.Render(line) + "\n"
		}
	}

	content += "\n" + SidebarItemStyle.Copy().Foreground(ColorInfo).Render("[+ New]")

	return SidebarStyle.Copy().Height(s.Height).Render(content)
}
