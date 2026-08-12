package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type ModelInfo struct {
	ID     string
	Name   string
	Model  string
	Active bool
}

type ModelPicker struct {
	Width    int
	Height   int
	Models   []ModelInfo
	Selected int
}

func NewModelPicker() ModelPicker {
	return ModelPicker{
		Selected: -1,
	}
}

func (mp *ModelPicker) CursorUp() {
	if mp.Selected > 0 {
		mp.Selected--
	}
}

func (mp *ModelPicker) CursorDown() {
	if mp.Selected < len(mp.Models)-1 {
		mp.Selected++
	}
}

func (mp *ModelPicker) SelectedModel() ModelInfo {
	if mp.Selected >= 0 && mp.Selected < len(mp.Models) {
		return mp.Models[mp.Selected]
	}
	return ModelInfo{}
}

func (mp *ModelPicker) Render() string {
	w := mp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	accent := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())
	active := lipgloss.NewStyle().Foreground(components.ColorSuccess()).Bold(true)

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" 模型列表"))

	if len(mp.Models) == 0 {
		lines = append(lines, "  "+muted.Render("（无可用模型）"))
	} else {
		for i, m := range mp.Models {
			var nameStyle lipgloss.Style
			prefix := "  "
			if i == mp.Selected {
				prefix = accent.Render("\u25b8 ")
			}
			nameStyle = lipgloss.NewStyle().Foreground(components.ColorText())
			if m.Active {
				nameStyle = active
			}

			modelStr := muted.Render(m.Model)
			label := nameStyle.Render(m.Name)
			if m.Active {
				label = nameStyle.Render(m.Name + " \u2713")
			}
			pad := innerW - lipgloss.Width(prefix) - lipgloss.Width(label) - lipgloss.Width(modelStr)
			if pad < 0 {
				pad = 0
			}
			line := prefix + label + strings.Repeat(" ", pad) + modelStr
			lines = append(lines, line)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] \u5bfc\u822a  [Enter] \u6fc0\u6d3b  [Esc] \u5173\u95ed"))
	return strings.Join(lines, "\n")
}
