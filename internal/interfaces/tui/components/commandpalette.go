package components

import (
	"github.com/charmbracelet/lipgloss"
)

type CommandItem struct {
	Label       string
	Description string
	Action      string
}

type CommandPalette struct {
	Visible bool
	Items   []CommandItem
	Cursor  int
	Width   int
	Height  int
}

func NewCommandPalette() CommandPalette {
	return CommandPalette{
		Items: []CommandItem{
			{Label: "/new", Description: "新建会话", Action: "new"},
			{Label: "/rollback", Description: "回滚对话到指定消息", Action: "rollback"},
			{Label: "/cancel", Description: "取消当前操作", Action: "cancel"},
			{Label: "/usage", Description: "查看 Token 用量", Action: "usage"},
			{Label: "/pause", Description: "暂停/恢复", Action: "pause"},
			{Label: "/export", Description: "导出会话存档 (Markdown)", Action: "export"},
			{Label: "/clear", Description: "清屏", Action: "clear"},
			{Label: "/quit", Description: "退出", Action: "quit"},
		},
	}
}

func (c *CommandPalette) Show() {
	c.Visible = true
	c.Cursor = 0
}

func (c *CommandPalette) Hide() {
	c.Visible = false
	c.Cursor = 0
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

func (c *CommandPalette) View() string {
	if !c.Visible {
		return ""
	}

	var lines []string
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
				Width(c.Width - 4).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Padding(0, 1).
				Width(c.Width - 4).
				Render(line)
		}

		lines = append(lines, line)
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
