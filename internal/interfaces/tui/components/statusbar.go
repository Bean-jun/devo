package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type StatusBar struct {
	AppName    string
	Session    string
	Processing bool
	Paused     bool
	Yolo       bool
	Connected  bool
	Width      int
}

func NewStatusBar() StatusBar {
	return StatusBar{
		AppName:   "Devo",
		Connected: true,
	}
}

func (s *StatusBar) Render() string {
	w := s.Width
	if w < 10 {
		w = 10
	}

	app := lipgloss.NewStyle().Foreground(ColorAccent()).Bold(true).Render("\U0001f680 " + s.AppName)
	divider := lipgloss.NewStyle().Foreground(ColorBorder()).Render(" · ")
	session := lipgloss.NewStyle().Foreground(ColorText()).Render(s.Session)

	statusColor := ColorSuccess()
	statusText := "\u25cf idle"
	if s.Processing {
		statusColor = ColorAccent()
		statusText = "\u25cf Processing"
	}
	if s.Paused {
		statusColor = ColorMuted()
		statusText = "\u25cf Paused"
	}
	statusDot := lipgloss.NewStyle().Foreground(statusColor).Render(statusText)

	yolo := ""
	if s.Yolo {
		yolo = lipgloss.NewStyle().
			Foreground(ColorWarning()).Bold(true).
			Render(" YOLO ")
	}

	left := app + divider + session + "  " + statusDot
	if yolo != "" {
		left += "  " + yolo
	}

	var center string
	if s.Processing {
		spinner := lipgloss.NewStyle().Foreground(ColorAccent()).Render("\u23f3")
		activity := lipgloss.NewStyle().Foreground(ColorMuted()).Render("Processing...")
		center = spinner + " " + activity
	}

	conn := lipgloss.NewStyle().Foreground(ColorSuccess()).Render("\u2713")
	if !s.Connected {
		conn = lipgloss.NewStyle().Foreground(ColorError()).Render("\u2717")
	}
	themeIconStr := "\U0001f319"
	if !CurrentTheme.IsDark {
		themeIconStr = "\u2600\ufe0f"
	}
	themeIcon := lipgloss.NewStyle().Foreground(ColorMuted()).Render(themeIconStr)

	right := conn + "  " + themeIcon

	leftW := lipgloss.Width(left)
	centerW := lipgloss.Width(center)
	rightW := lipgloss.Width(right)

	var content string
	if centerW > 0 {
		midPad := w - 2 - leftW - centerW - rightW
		if midPad < 2 {
			midPad = 2
		}
		mid := midPad / 2
		content = left + strings.Repeat(" ", mid) + center + strings.Repeat(" ", midPad-mid) + right
	} else {
		pad := w - 2 - leftW - rightW
		if pad < 1 {
			pad = 1
		}
		content = left + strings.Repeat(" ", pad) + right
	}

	headerLine := StatusBarBg().Width(w).Render(content)

	sep := lipgloss.NewStyle().
		Foreground(ColorBorder()).
		Render(strings.Repeat("\u2500", w))

	return headerLine + "\n" + sep
}
