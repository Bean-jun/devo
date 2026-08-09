package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type SkillEntry struct {
	Name        string
	Description string
	Enabled     bool
}

type SkillsPanel struct {
	Width    int
	Height   int
	Selected int
	Skills   []SkillEntry
}

func NewSkillsPanel() SkillsPanel {
	return SkillsPanel{}
}

func (sp *SkillsPanel) CursorUp() {
	if sp.Selected > 0 {
		sp.Selected--
	}
}

func (sp *SkillsPanel) CursorDown() {
	if sp.Selected < len(sp.Skills)-1 {
		sp.Selected++
	}
}

func (sp *SkillsPanel) Toggle() {
	sp.Skills[sp.Selected].Enabled = !sp.Skills[sp.Selected].Enabled
}

func (sp *SkillsPanel) Render() string {
	w := sp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f6e0 技能管理"))

	for i, sk := range sp.Skills {
		toggle := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("\u25cb")
		if sk.Enabled {
			toggle = lipgloss.NewStyle().Foreground(components.ColorSuccess()).Render("\u25cf")
		}
		name := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(sk.Name)
		desc := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(sk.Description, 28))

		if i == sp.Selected {
			if sk.Enabled {
				toggle = lipgloss.NewStyle().Foreground(components.ColorSuccess()).Render("\u25cf")
			} else {
				toggle = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("\u25cb")
			}
			name = lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(sk.Name)
			desc = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(sk.Description, 28))
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			lines = append(lines, " "+prefix+" "+toggle+" "+name+"  "+desc)
		} else {
			lines = append(lines, "   "+toggle+" "+name+"  "+desc)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Space] 启用/停用  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
