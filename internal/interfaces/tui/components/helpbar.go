package components

import (
	"github.com/charmbracelet/lipgloss"
)

type HelpBar struct {
	Width     int
	Items     []string
	FocusMode string
}

func NewHelpBar() HelpBar {
	return HelpBar{
		Items: []string{
			"^S sessions",
			"^N new",
			"^C cancel",
			"^P pause",
			"^Q quit",
		},
		FocusMode: "✎ Input",
	}
}

func (h *HelpBar) SetApprovalMode() {
	h.Items = []string{
		"[Y] Approve",
		"[N] Reject",
		"[D] Full Diff",
		"[Esc] Reject",
	}
}

func (h *HelpBar) SetDefaultMode() {
	h.Items = []string{
		"^S sessions",
		"^N new",
		"^C cancel",
		"^P pause",
		"^Q quit",
	}
}

func (h *HelpBar) View() string {
	var parts []string
	for _, item := range h.Items {
		parts = append(parts, item)
	}
	shortcuts := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	mode := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(h.FocusMode)

	spacerWidth := h.Width - lipgloss.Width(shortcuts) - lipgloss.Width(mode) - 2
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	content := lipgloss.JoinHorizontal(lipgloss.Top, shortcuts, spacer, mode)
	return HelpBarStyle.Copy().Width(h.Width).Render(content)
}
