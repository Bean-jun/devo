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
	Mode            string
	Width           int
}

func NewStatusBar() StatusBar {
	return StatusBar{
		AppName:         "Devo",
		SessionTitle:    "",
		SessionState:    "Idle",
		TokenUsage:      "0 token",
		ServerPort:      0,
		ServerConnected: true,
	}
}

func (s *StatusBar) View() string {
	stateColor := StateColor(s.SessionState)
	stateDisplay := lipgloss.NewStyle().Foreground(stateColor).Render(s.SessionState)

	serverIndicator := lipgloss.NewStyle().Foreground(ColorSuccess).Render("[Connected]")
	if !s.ServerConnected {
		serverIndicator = lipgloss.NewStyle().Foreground(ColorDanger).Render("[Disconnected]")
	}

	left := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(s.AppName)
	if s.SessionTitle != "" {
		left += " · " + s.SessionTitle
	}
	left += " · " + stateDisplay
	left += " · " + s.TokenUsage

	if s.Mode != "" {
		modeStyle := lipgloss.NewStyle().
			Background(ColorInfo).
			Foreground(ColorWhite).
			Padding(0, 1).
			Render(s.Mode)
		left += " " + modeStyle
	}

	right := fmt.Sprintf(":%d %s", s.ServerPort, serverIndicator)

	innerWidth := s.Width - 4
	spacerWidth := innerWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)

	return StatusBarStyle.Copy().Width(s.Width).Render(content)
}
