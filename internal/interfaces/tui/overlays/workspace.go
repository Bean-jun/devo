package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type WorkspaceEntry struct {
	Name   string
	Path   string
	Active bool
}

type WorkspacePanel struct {
	Width      int
	Height     int
	Selected   int
	Workspaces []WorkspaceEntry
}

func NewWorkspacePanel() WorkspacePanel {
	return WorkspacePanel{}
}

func (wp *WorkspacePanel) CursorUp() {
	if wp.Selected > 0 {
		wp.Selected--
	}
}

func (wp *WorkspacePanel) CursorDown() {
	if wp.Selected < len(wp.Workspaces)-1 {
		wp.Selected++
	}
}

func (wp *WorkspacePanel) Render() string {
	w := wp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f4c1 工作区"))

	for i, ws := range wp.Workspaces {
		active := "  "
		if ws.Active {
			active = lipgloss.NewStyle().Foreground(components.ColorSuccess()).Render("\u25cf ")
		}
		name := lipgloss.NewStyle().Foreground(components.ColorText()).Render(ws.Name)
		path := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(ws.Path)

		if i == wp.Selected {
			name = lipgloss.NewStyle().Foreground(components.ColorText()).Render(ws.Name)
			path = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(ws.Path)
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			lines = append(lines, " "+prefix+active+name+"  "+path)
		} else {
			lines = append(lines, "  "+active+name+"  "+path)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Enter] 切换  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
