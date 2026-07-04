package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	AppName         string
	SessionTitle    string
	SessionState    string
	ServerPort      int
	ServerConnected bool
	Mode            string
	Width           int
	YOLOMode        bool
	SpinnerFrame    int
	ActivityStream  string
	ActivityActive  bool
	ToastMessage    string
	ToastActive     bool
	ToastIsError    bool
}

var spinnerChars = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

func NewStatusBar() StatusBar {
	return StatusBar{
		AppName:         "Devo",
		SessionTitle:    "",
		SessionState:    "Idle",
		ServerPort:      0,
		ServerConnected: true,
	}
}

func (s *StatusBar) SetActivity(stream string) {
	s.ActivityStream = stream
	s.ActivityActive = true
}

func (s *StatusBar) ClearActivity() {
	s.ActivityActive = false
	s.ActivityStream = ""
}

func (s *StatusBar) View() string {
	stateColor := StateColor(s.SessionState)
	stateDot := lipgloss.NewStyle().Foreground(stateColor).Render("●")
	stateDisplay := lipgloss.NewStyle().Foreground(stateColor).Render(s.SessionState)

	serverIndicator := lipgloss.NewStyle().Foreground(ColorSuccess).Render("[Connected]")
	if !s.ServerConnected {
		serverIndicator = lipgloss.NewStyle().Foreground(ColorDanger).Render("[Disconnected]")
	}

	left := ""
	if s.SessionTitle != "" {
		left = s.SessionTitle
		left += " · "
	}
	left += stateDot + " " + stateDisplay

	if s.YOLOMode {
		yoloBadge := YOLOSmallBadgeStyle.Render("YOLO")
		left += " " + yoloBadge
	}

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
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := innerWidth - leftWidth - rightWidth
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	var middle string

	if s.ToastActive && s.ToastMessage != "" {
		toastW := innerWidth - leftWidth
		if toastW < 4 {
			toastW = 4
		}
		toastColor := ColorSuccess
		if s.ToastIsError {
			toastColor = ColorDanger
		}
		text := lipgloss.NewStyle().Foreground(toastColor).Render(s.ToastMessage)
		if lipgloss.Width(text) > toastW {
			text = truncateToWidth(text, toastW)
		}
		middle = lipgloss.NewStyle().Width(toastW).MaxHeight(1).Padding(0, 1).Render(text)
		right = ""
	} else if s.ActivityActive && s.ActivityStream != "" {
		activityW := innerWidth - leftWidth
		if activityW < 4 {
			activityW = 4
		}
		spinner := spinnerChars[s.SpinnerFrame%len(spinnerChars)]
		clean := strings.ReplaceAll(s.ActivityStream, "\n", " ")
		clean = strings.ReplaceAll(clean, "\r", "")
		text := spinner + " " + clean
		text = lipgloss.NewStyle().Foreground(ColorMuted).Render(text)
		if lipgloss.Width(text) > activityW {
			text = truncateToWidth(text, activityW)
		}
		middle = lipgloss.NewStyle().Width(activityW).MaxHeight(1).Padding(0, 1).Render(text)
		right = ""
	} else {
		middle = lipgloss.NewStyle().Width(spacerWidth).MaxHeight(1).Render("")
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)

	baseStyle := StatusBarStyle
	if s.YOLOMode {
		baseStyle = YOLOStatusBarStyle
	}

	return baseStyle.Copy().Width(s.Width).Render(content)
}

func truncateToWidth(s string, maxWidth int) string {
	runes := []rune(s)
	w := 0
	for i := range runes {
		rw := lipgloss.Width(string(runes[i]))
		if w+rw > maxWidth {
			return string(runes[:i])
		}
		w += rw
	}
	return s
}
