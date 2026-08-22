package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type StatusBar struct {
	AppName          string
	Session          string
	Processing       bool
	Paused           bool
	Yolo             bool
	TeamMode         bool
	Connected        bool
	Width            int
	ServerPort       string
	ReasoningEnabled bool
	ReasoningEffort  string
	Activity         string
	ActivityActive   bool
	HasUpdate        bool
	LatestVersion    string
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

	team := ""
	if s.TeamMode {
		team = lipgloss.NewStyle().
			Foreground(ColorAccent()).Bold(true).
			Render(" TEAM ")
	}

	left := session + "  " + statusDot
	if team != "" {
		left += "  " + team
	}
	if yolo != "" {
		left += "  " + yolo
	}

	var center string
	if s.ActivityActive && s.Activity != "" {
		spinner := lipgloss.NewStyle().Foreground(ColorAccent()).Render("\u23f3")
		text := s.Activity
		activity := lipgloss.NewStyle().Foreground(ColorMuted()).Render(text)
		center = spinner + " " + activity
	} else if s.Processing && !s.ActivityActive {
		spinner := lipgloss.NewStyle().Foreground(ColorAccent()).Render("\u23f3")
		activity := lipgloss.NewStyle().Foreground(ColorMuted()).Render("Processing...")
		center = spinner + " " + activity
	}

	reasoning := ""
	if s.ReasoningEnabled && s.ReasoningEffort != "" {
		reasoning = lipgloss.NewStyle().
			Foreground(ColorAccent()).
			Render("\U0001f9e0 "+s.ReasoningEffort) + "  "
	} else if !s.ReasoningEnabled {
		reasoning = lipgloss.NewStyle().
			Foreground(ColorMuted()).
			Render("\U0001f9e0 off") + "  "
	}

	conn := lipgloss.NewStyle().Foreground(ColorSuccess()).Render("\u2713")
	if !s.Connected {
		conn = lipgloss.NewStyle().Foreground(ColorError()).Render("\u2717")
	}
	port := ""
	if s.ServerPort != "" {
		port = lipgloss.NewStyle().Foreground(ColorMuted()).Render(":" + s.ServerPort + " ")
	}

	update := ""
	if s.HasUpdate && s.LatestVersion != "" {
		update = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34c759")).
			Bold(true).
			Render("\u2B06 "+s.LatestVersion) + "  "
	}

	right := update + reasoning + port + conn

	leftW := lipgloss.Width(left)
	centerW := lipgloss.Width(center)
	rightW := lipgloss.Width(right)

	var content string
	if centerW > 0 {
		sep := lipgloss.NewStyle().Foreground(ColorBorder()).Render(" \u2502 ")
		sepW := lipgloss.Width(sep)
		pad := w - 2 - leftW - sepW - centerW - rightW
		if pad < 1 {
			pad = 1
		}
		content = left + sep + center + strings.Repeat(" ", pad) + right
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
