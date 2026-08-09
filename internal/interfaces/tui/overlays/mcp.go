package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type MCPEntry struct {
	Name   string
	URL    string
	Status string
}

type MCPPanel struct {
	Width    int
	Height   int
	Selected int
	Servers  []MCPEntry
}

func NewMCPPanel() MCPPanel {
	return MCPPanel{}
}

func (mp *MCPPanel) CursorUp() {
	if mp.Selected > 0 {
		mp.Selected--
	}
}

func (mp *MCPPanel) CursorDown() {
	if mp.Selected < len(mp.Servers)-1 {
		mp.Selected++
	}
}

func (mp *MCPPanel) Render() string {
	w := mp.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f50c MCP 服务器管理"))

	for i, srv := range mp.Servers {
		statusIcon := "\u25cf"
		statusColor := components.ColorSuccess()
		switch srv.Status {
		case "disconnected":
			statusColor = components.ColorMuted()
		case "error":
			statusColor = components.ColorError()
			statusIcon = "\u2717"
		}
		status := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon + " " + srv.Status)
		name := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(srv.Name)
		url := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(srv.URL, 26))

		if i == mp.Selected {
			name = lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(srv.Name)
			url = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(truncateStr(srv.URL, 26))
			status = lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon + " " + srv.Status)
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			lines = append(lines, " "+prefix+name+"  "+url+"  "+status)
		} else {
			lines = append(lines, "  "+name+"  "+url+"  "+status)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Space] 连接/断开  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
