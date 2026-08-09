package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type FlatCommand struct {
	Name        string
	Description string
	GroupName   string
}

type CommandGroup struct {
	Name     string
	Commands []CommandItem
}

type CommandItem struct {
	Name        string
	Description string
}

type CommandSheet struct {
	Width        int
	Height       int
	Filter       string
	Selected     int
	Groups       []CommandGroup
	FlatCommands []FlatCommand
}

func NewCommandSheet() CommandSheet {
	cs := CommandSheet{
		Groups: []CommandGroup{
			{
				Name: "SESSION",
				Commands: []CommandItem{
					{"/new", "创建新会话"},
					{"/switch", "切换会话"},
					{"/rename", "重命名会话"},
					{"/export", "导出会话"},
					{"/archive", "归档会话"},
					{"/rollback", "回滚到消息"},
					{"/pause", "暂停/恢复"},
					{"/cancel", "取消当前操作"},
				},
			},
			{
				Name: "PANEL",
				Commands: []CommandItem{
					{"/files", "文件管理"},
					{"/skills", "技能管理"},
					{"/mcp", "MCP 管理"},
					{"/memory", "记忆管理"},
				},
			},
			{
				Name: "WORKSPACE",
				Commands: []CommandItem{
					{"/workspace", "切换工作区"},
					{"/w-create", "新建工作区"},
				},
			},
			{
				Name: "APP",
				Commands: []CommandItem{
					{"/yolo", "切换 YOLO 模式"},
					{"/theme", "切换主题"},
					{"/help", "帮助"},
					{"/quit", "退出"},
				},
			},
		},
	}
	cs.buildFlat()
	return cs
}

func (cs *CommandSheet) BuildFlat() {
	cs.FlatCommands = nil
	for _, g := range cs.Groups {
		for _, cmd := range g.Commands {
			if cs.Filter != "" && !strings.Contains(strings.ToLower(cmd.Name), strings.ToLower(cs.Filter)) && !strings.Contains(strings.ToLower(cmd.Description), strings.ToLower(cs.Filter)) {
				continue
			}
			cs.FlatCommands = append(cs.FlatCommands, FlatCommand{
				Name:        cmd.Name,
				Description: cmd.Description,
				GroupName:   g.Name,
			})
		}
	}
}

func (cs *CommandSheet) buildFlat() {
	cs.BuildFlat()
}

func (cs *CommandSheet) CursorUp() {
	if cs.Selected > 0 {
		cs.Selected--
	}
}

func (cs *CommandSheet) CursorDown() {
	if cs.Selected < len(cs.FlatCommands)-1 {
		cs.Selected++
	}
}

func (cs *CommandSheet) SelectedCommand() FlatCommand {
	if cs.Selected >= 0 && cs.Selected < len(cs.FlatCommands) {
		return cs.FlatCommands[cs.Selected]
	}
	return FlatCommand{}
}

func (cs *CommandSheet) Render() string {
	w := cs.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())
	accent := lipgloss.NewStyle().Foreground(components.ColorAccent())

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f50d 命令面板"))

	if cs.Filter != "" {
		filterLine := lipgloss.NewStyle().
			Foreground(components.ColorMuted()).
			Render("  过滤: " + cs.Filter + "\u2588")
		lines = append(lines, filterLine)
	}

	lastGroup := ""
	currentIdx := 0
	for _, g := range cs.Groups {
		for _, cmd := range g.Commands {
			if cs.Filter != "" && !strings.Contains(strings.ToLower(cmd.Name), strings.ToLower(cs.Filter)) && !strings.Contains(strings.ToLower(cmd.Description), strings.ToLower(cs.Filter)) {
				continue
			}
			if g.Name != lastGroup {
				lines = append(lines, "  "+label.Render(g.Name))
				lastGroup = g.Name
			}
			name := accent.Render(cmd.Name)
			desc := muted.Render(cmd.Description)
			if currentIdx == cs.Selected {
				name = accent.Render("\u25b8" + cmd.Name[1:])
				desc = lipgloss.NewStyle().Foreground(components.ColorText()).Render(cmd.Description)
			}
			line := name + strings.Repeat(" ", innerW-2-lipgloss.Width(name)-lipgloss.Width(desc)) + desc
			lines = append(lines, " "+line)
			currentIdx++
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Enter] 执行  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}
