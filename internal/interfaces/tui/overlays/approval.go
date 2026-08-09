package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type ApprovalModal struct {
	Width      int
	Height     int
	ApprovalID string
	Operation  string
	Risk       string
	Diff       string
}

func (am *ApprovalModal) Render() string {
	w := am.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	riskColor := components.ColorSuccess()
	if am.Risk == "HIGH" {
		riskColor = components.ColorError()
	} else if am.Risk == "MEDIUM" {
		riskColor = components.ColorWarning()
	}

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \u26a0 Approval Required"))

	lines = append(lines, " Operation: "+lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(am.Operation))
	lines = append(lines, " Risk:      "+lipgloss.NewStyle().Foreground(riskColor).Bold(true).Render(am.Risk))
	lines = append(lines, " ")

	lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorBorder()).Render("\u250c Diff "+strings.Repeat("\u2500", innerW-9)))
	for _, dl := range strings.Split(am.Diff, "\n") {
		dl = strings.TrimSpace(dl)
		if dl == "" {
			continue
		}
		color := components.ColorText()
		if strings.HasPrefix(dl, "+") {
			color = components.ColorSuccess()
		} else if strings.HasPrefix(dl, "-") {
			color = components.ColorError()
		}
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(color).Render("\u2502 "+truncateStr(dl, innerW-3)))
	}
	lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorBorder()).Render("\u2514"+strings.Repeat("\u2500", innerW-3)))

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Y] Approve  [N] Reject  [D] Diff"))
	return strings.Join(lines, "\n")
}
