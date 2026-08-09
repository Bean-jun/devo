package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type HelpPanel struct {
	Width  int
	Height int
}

func (hp *HelpPanel) Render() string {
	w := hp.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	sections := []struct {
		name  string
		items []string
	}{
		{"Navigation", []string{
			"\u2191/\u2193      行滚动",
			"PgUp/Dn  页滚动",
			"Ctrl+U   跳到上一条用户消息",
			"Ctrl+D   跳到下一条用户消息",
			"Tab      展开/折叠工具卡片",
		}},
		{"Chat", []string{
			"Enter    发送消息",
			"/        打开命令面板（输入框为空时）",
			"Ctrl+N   新建会话",
			"Ctrl+S   会话列表",
			"F2       重命名会话",
		}},
		{"Mode", []string{
			"Ctrl+T   切换主题（暗/亮）",
			"Ctrl+Y   切换 YOLO 模式",
			"Ctrl+P   暂停/恢复",
		}},
		{"Overlay", []string{
			"Esc      关闭覆盖层/面板",
			"?        打开帮助",
			"\u2191/\u2193/j/k 面板内光标移动",
			"Enter    面板内确认选择",
		}},
		{"System", []string{
			"Ctrl+C   退出",
			"Ctrl+Q   退出",
		}},
	}

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Help"))

	for _, sec := range sections {
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true).Render(sec.name))
		for _, item := range sec.items {
			parts := strings.SplitN(item, "  ", 2)
			key := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(parts[0])
			desc := ""
			if len(parts) > 1 {
				desc = lipgloss.NewStyle().Foreground(components.ColorText()).Render(parts[1])
			}
			lines = append(lines, " "+key+"  "+desc)
		}
		lines = append(lines, " ")
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Esc] 关闭"))
	return strings.Join(lines, "\n")
}
