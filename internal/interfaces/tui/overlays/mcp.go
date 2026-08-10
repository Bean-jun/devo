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
	Width      int
	Height     int
	Selected   int
	Servers    []MCPEntry
	Editing    bool
	EditBuffer string
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

func (mp *MCPPanel) StartEditing() {
	mp.Editing = true
	mp.EditBuffer = ""
}

func (mp *MCPPanel) CancelEditing() {
	mp.Editing = false
	mp.EditBuffer = ""
}

func (mp *MCPPanel) ConfirmEditing() (string, string) {
	mp.Editing = false
	v := mp.EditBuffer
	mp.EditBuffer = ""
	parts := strings.SplitN(v, " ", 2)
	serverID := strings.TrimSpace(parts[0])
	endpoint := ""
	if len(parts) > 1 {
		endpoint = strings.TrimSpace(parts[1])
	}
	return serverID, endpoint
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

	footer := "[\u2191\u2193] 导航  [Space] 连接/断开  [a] 添加  [Esc] 关闭"
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
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("格式: server_id endpoint"))
	}
	return strings.Join(lines, "\n")
}
