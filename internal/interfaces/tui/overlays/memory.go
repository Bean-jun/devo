package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type MemoryEntry struct {
	ID      string
	Type    string
	Key     string
	Content string
}

type MemoryPanel struct {
	Width      int
	Height     int
	Selected   int
	Memories   []MemoryEntry
	Editing    bool
	EditBuffer string
}

func NewMemoryPanel() MemoryPanel {
	return MemoryPanel{}
}

func (mp *MemoryPanel) StartEditing() {
	mp.Editing = true
	mp.EditBuffer = ""
}

func (mp *MemoryPanel) CancelEditing() {
	mp.Editing = false
	mp.EditBuffer = ""
}

func (mp *MemoryPanel) ConfirmEditing() (string, string) {
	mp.Editing = false
	v := mp.EditBuffer
	mp.EditBuffer = ""
	parts := strings.SplitN(v, " ", 2)
	key := strings.TrimSpace(parts[0])
	content := ""
	if len(parts) > 1 {
		content = strings.TrimSpace(parts[1])
	}
	return key, content
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
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f9e0 记忆管理"))

	for i, mem := range mp.Memories {
		typeLabel := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("[" + mem.Type + "]")
		key := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(mem.Key)
		content := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(mem.Content, 30))

		if i == mp.Selected {
			typeLabel = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("[" + mem.Type + "]")
			key = lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(mem.Key)
			content = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(mem.Content, 30))
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			lines = append(lines, " "+prefix+typeLabel+" "+key+"  "+content)
		} else {
			lines = append(lines, "   "+typeLabel+" "+key+"  "+content)
		}
	}

	footer := "[\u2191\u2193] 导航  [Del] 删除  [Esc] 关闭"
	if mp.Editing {
		footer = "[Enter] 确认添加  [Esc] 取消"
	}
	lines = append(lines, components.PanelFooterStyle(innerW).Render(footer))
	if mp.Editing {
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(components.ColorAccent()).
			Width(innerW - 2).
			Render(mp.EditBuffer + lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("\u2588"))
		lines = append(lines, " "+inputBox)
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("格式: key content"))
	}
	return strings.Join(lines, "\n")
}
