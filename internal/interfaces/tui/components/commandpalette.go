package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type CommandItem struct {
	Label       string
	Description string
	Action      string
}

type CommandPalette struct {
	Visible  bool
	allItems []CommandItem
	Items    []CommandItem
	Cursor   int
	Width    int
	Height   int
	Query    string
}

func NewCommandPalette() CommandPalette {
	allItems := []CommandItem{
		{Label: "/new", Description: "创建新会话", Action: "new"},
		{Label: "/switch", Description: "切换会话", Action: "switch"},
		{Label: "/rename", Description: "重命名当前会话", Action: "rename"},
		{Label: "/export", Description: "导出当前会话记录", Action: "export"},
		{Label: "/rollback", Description: "回滚消息", Action: "rollback"},
		{Label: "/pause", Description: "暂停当前会话", Action: "pause"},
		{Label: "/resume", Description: "恢复当前会话", Action: "resume"},
		{Label: "/cancel", Description: "取消当前操作", Action: "cancel"},
		{Label: "/yolo", Description: "切换 YOLO 自动批准模式", Action: "yolo"},
		{Label: "/trust", Description: "设置信任级别 (low/normal/elevated)", Action: "trust"},
		{Label: "/version", Description: "显示版本信息", Action: "version"},
		{Label: "/help", Description: "显示帮助", Action: "help"},
		{Label: "/quit", Description: "退出", Action: "quit"},
	}
	return CommandPalette{
		allItems: allItems,
		Items:    allItems,
	}
}

func (c *CommandPalette) Show() {
	c.Visible = true
	c.Cursor = 0
	c.Query = ""
	c.Items = c.allItems
}

func (c *CommandPalette) Hide() {
	c.Visible = false
	c.Cursor = 0
	c.Query = ""
}

func (c *CommandPalette) Filter() {
	if c.Query == "" {
		c.Items = c.allItems
	} else {
		q := strings.ToLower(c.Query)
		var filtered []CommandItem
		for _, item := range c.allItems {
			if strings.Contains(strings.ToLower(item.Label), q) ||
				strings.Contains(strings.ToLower(item.Description), q) {
				filtered = append(filtered, item)
			}
		}
		c.Items = filtered
	}
	if c.Cursor >= len(c.Items) {
		c.Cursor = len(c.Items) - 1
	}
	if c.Cursor < 0 {
		c.Cursor = 0
	}
}

func (c *CommandPalette) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}

func (c *CommandPalette) CursorDown() {
	if c.Cursor < len(c.Items)-1 {
		c.Cursor++
	}
}

func (c *CommandPalette) SelectedAction() string {
	if c.Cursor >= 0 && c.Cursor < len(c.Items) {
		return c.Items[c.Cursor].Action
	}
	return ""
}

func (c *CommandPalette) SelectedLabel() string {
	if c.Cursor >= 0 && c.Cursor < len(c.Items) {
		return c.Items[c.Cursor].Label
	}
	return ""
}

func (c *CommandPalette) View() string {
	if !c.Visible {
		return ""
	}

	innerW := c.Width - 4

	queryDisplay := c.Query
	if c.Query == "" {
		queryDisplay = lipgloss.NewStyle().Foreground(ColorMuted).Render("输入过滤...")
	} else {
		queryDisplay += "█"
	}
	queryLine := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorPrimary).
		Background(ColorSurface).
		Width(innerW).
		Padding(0, 1).
		Render(queryDisplay)

	var lines []string
	lines = append(lines, queryLine)

	if len(c.Items) == 0 {
		emptyLine := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1).
			Width(innerW).
			Render("  无匹配命令")
		lines = append(lines, emptyLine)
	} else {
		for i, item := range c.Items {
			prefix := "  "
			line := prefix + item.Label
			if item.Description != "" {
				line += "  " + lipgloss.NewStyle().
					Foreground(ColorMuted).
					Render(item.Description)
			}

			if i == c.Cursor {
				line = lipgloss.NewStyle().
					Foreground(ColorWhite).
					Background(ColorPrimary).
					Padding(0, 1).
					Width(innerW).
					Render(line)
			} else {
				line = lipgloss.NewStyle().
					Foreground(ColorMuted).
					Background(ColorSurface).
					Padding(0, 1).
					Width(innerW).
					Render(line)
			}

			lines = append(lines, line)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Background(ColorSurface).
		Width(c.Width - 2).
		Render(content)

	return box
}