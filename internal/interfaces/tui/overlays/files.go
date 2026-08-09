package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type FileEntry struct {
	Name     string
	Size     string
	Type     string
	Modified string
}

type FilesPanel struct {
	Width    int
	Height   int
	Selected int
	Files    []FileEntry
}

func NewFilesPanel() FilesPanel {
	return FilesPanel{}
}

func (fp *FilesPanel) CursorUp() {
	if fp.Selected > 0 {
		fp.Selected--
	}
}

func (fp *FilesPanel) CursorDown() {
	if fp.Selected < len(fp.Files)-1 {
		fp.Selected++
	}
}

func (fp *FilesPanel) Render() string {
	w := fp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f4c2 文件管理"))

	for i, f := range fp.Files {
		name := lipgloss.NewStyle().Foreground(components.ColorText()).Render(f.Name)
		meta := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(f.Size + "  " + f.Modified)
		if i == fp.Selected {
			name = lipgloss.NewStyle().Foreground(components.ColorText()).Render(f.Name)
			meta = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(f.Size + "  " + f.Modified)
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			line := prefix + name
			pad := innerW - 2 - lipgloss.Width(line) - lipgloss.Width(meta)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, " "+line+strings.Repeat(" ", pad)+meta)
		} else {
			line := " " + name
			pad := innerW - 2 - lipgloss.Width(line) - lipgloss.Width(meta)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, " "+line+strings.Repeat(" ", pad)+meta)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Enter] 打开  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
