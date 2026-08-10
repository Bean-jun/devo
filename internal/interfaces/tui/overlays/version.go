package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type VersionPanel struct {
	Width   int
	Height  int
	Version string
}

func (vp *VersionPanel) Render() string {
	w := vp.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	value := lipgloss.NewStyle().Foreground(components.ColorText()).Bold(true)
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Version"))

	lines = append(lines, "")
	lines = append(lines, "  "+label.Render("Devo")+"  "+value.Render("v"+vp.Version))
	lines = append(lines, "")

	lines = append(lines, "  "+muted.Render("AI-Powered Development Orchestrator"))
	lines = append(lines, "  "+muted.Render("Terminal User Interface"))
	lines = append(lines, "")

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Esc] 关闭"))
	return strings.Join(lines, "\n")
}
