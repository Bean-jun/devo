package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type MemoryEntry struct {
	ID      string
	Key     string
	Content string
}

type MemoryPanel struct {
	Width    int
	Height   int
	Selected int
	Memories []MemoryEntry
}

func NewMemoryPanel() MemoryPanel {
	return MemoryPanel{}
}

func (mp *MemoryPanel) CursorUp() {
	if mp.Selected > 0 {
		mp.Selected--
	}
}

func (mp *MemoryPanel) CursorDown() {
	if mp.Selected < len(mp.Memories)-1 {
		mp.Selected++
	}
}

func (mp *MemoryPanel) Render() string {
	w := mp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f9e0 记忆管理"))

	for i, mem := range mp.Memories {
		key := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(mem.Key)
		content := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(mem.Content, 36))

		if i == mp.Selected {
			key = lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(mem.Key)
			content = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(mem.Content, 36))
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			lines = append(lines, " "+prefix+key+"  "+content)
		} else {
			lines = append(lines, "  "+key+"  "+content)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Del] 删除  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
