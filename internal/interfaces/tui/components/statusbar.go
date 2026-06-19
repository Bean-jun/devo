package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	AppName         string
	SessionTitle    string
	SessionState    string
	TokenUsage      string
	ServerPort      int
	ServerConnected bool
	Width           int
}

func NewStatusBar() StatusBar {
	return StatusBar{
		AppName:         "Devo",
		SessionTitle:    "",
		SessionState:    "Idle",
		TokenUsage:      "0 tok",
		ServerPort:      0,
		ServerConnected: true,
	}
}

func (s *StatusBar) View() string {
	stateColor := StateColor(s.SessionState)
	stateDisplay := lipgloss.NewStyle().Foreground(stateColor).Render(s.SessionState)

	serverIndicator := "✓"
	if !s.ServerConnected {
		serverIndicator = lipgloss.NewStyle().Foreground(ColorDanger).Render("✗")
	}

	left := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(s.AppName)
	if s.SessionTitle != "" {
		left += " · " + s.SessionTitle
	}
	left += " · " + stateDisplay
	left += " · " + s.TokenUsage

	right := fmt.Sprintf(":%d %s", s.ServerPort, serverIndicator)

	leftStyled := StatusBarStyle.Copy().Render(left)
	rightStyled := StatusBarStyle.Copy().Render(right)

	spacerWidth := s.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	spacer := StatusBarStyle.Copy().Render(lipgloss.NewStyle().Width(spacerWidth).Render(""))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, spacer, rightStyled)
}
