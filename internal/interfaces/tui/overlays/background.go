package overlays

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/components"
)

type BackgroundPanel struct {
	Width     int
	Height    int
	Processes []api.BackgroundProcessInfo
	Selected  int
	Expanded  map[int]bool
}

func NewBackgroundPanel() BackgroundPanel {
	return BackgroundPanel{
		Expanded: make(map[int]bool),
	}
}

func (bp *BackgroundPanel) CursorUp() {
	if bp.Selected > 0 {
		bp.Selected--
	}
}

func (bp *BackgroundPanel) CursorDown() {
	if bp.Selected < len(bp.Processes)-1 {
		bp.Selected++
	}
}

func (bp *BackgroundPanel) ToggleExpand() {
	if bp.Selected < 0 || bp.Selected >= len(bp.Processes) {
		return
	}
	pid := bp.Processes[bp.Selected].PID
	if bp.Expanded[pid] {
		delete(bp.Expanded, pid)
	} else {
		bp.Expanded[pid] = true
	}
}

func (bp *BackgroundPanel) SelectedPID() int {
	if bp.Selected < 0 || bp.Selected >= len(bp.Processes) {
		return 0
	}
	return bp.Processes[bp.Selected].PID
}

func (bp *BackgroundPanel) Render() string {
	w := bp.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	accent := lipgloss.NewStyle().Foreground(components.ColorAccent())
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Background Processes"))

	if len(bp.Processes) == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+muted.Render("没有后台进程"))
		lines = append(lines, "")
	} else {
		lines = append(lines, "  "+label.Render(fmt.Sprintf("PID  命令                               状态")))
		for i, p := range bp.Processes {
			statusStyle := lipgloss.NewStyle().Foreground(components.ColorSuccess())
			statusLabel := "运行中"
			if p.Status == "stopped" {
				statusStyle = lipgloss.NewStyle().Foreground(components.ColorMuted())
				statusLabel = "已停止"
			} else if p.Status == "failed" {
				statusStyle = lipgloss.NewStyle().Foreground(components.ColorError())
				statusLabel = "失败"
			}

			prefix := "  "
			if i == bp.Selected {
				prefix = accent.Render(" \u25b8")
			}

			cmd := p.Cmd
			cmdMaxLen := innerW - 26
			if len(cmd) > cmdMaxLen {
				cmd = cmd[:cmdMaxLen-1] + "\u2026"
			}

			line := fmt.Sprintf("%s%-5d %-"+fmt.Sprintf("%d", cmdMaxLen)+"s %s",
				prefix, p.PID, cmd, statusStyle.Render(statusLabel))
			lines = append(lines, line)

			if bp.Expanded[p.PID] {
				lines = append(lines, "    "+muted.Render("已选择进程 "+fmt.Sprintf("%d", p.PID)+" - 按 Enter 停止"))
			}
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Enter] 停止  [Tab] 展开  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
