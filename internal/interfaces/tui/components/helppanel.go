package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpPanel struct {
	Visible bool
	Width   int
	Height  int
	content string
}

func NewHelpPanel() HelpPanel {
	return HelpPanel{}
}

func (h *HelpPanel) Show() {
	h.Visible = true
	h.content = h.buildContent()
}

func (h *HelpPanel) Hide() {
	h.Visible = false
}

func (h *HelpPanel) SetSize(width, height int) {
	h.Width = width
	h.Height = height
}

func (h *HelpPanel) View() string {
	if !h.Visible {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 0, 1, 0).
		Render("帮助 - 快捷键与命令")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		h.content,
		lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(1, 0, 0, 0).
			Render("按 Esc 关闭"),
	)

	return ModalBoxStyle.Copy().
		Width(h.Width - 2).
		Render(body)
}

func (h *HelpPanel) buildContent() string {
	var sections []string

	shortcutSection := lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true).
		Render("快捷键")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"Esc", "工具执行中 → 暂停, 暂停/思考/处理中 → 取消"},
		{"Alt+Y", "切换 YOLO 自动批准模式"},
		{"F2", "重命名当前会话"},
		{"Ctrl+C", "取消当前操作"},
		{"Ctrl+Q", "退出"},
		{"↑↓", "滚动消息列表"},
		{"Shift+↑↓", "输入历史"},
		{"Enter", "发送消息"},
		{"Shift+Enter", "换行"},
	}

	var shortcutLines []string
	shortcutLines = append(shortcutLines, shortcutSection)
	for _, s := range shortcuts {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Width(12).
			Render(s.key)
		descStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Render(s.desc)
		shortcutLines = append(shortcutLines, fmt.Sprintf("  %s %s", keyStyle, descStyle))
	}
	sections = append(sections, strings.Join(shortcutLines, "\n"))

	cmdSection := lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true).
		Render("命令")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/new [title]", "创建新会话"},
		{"/switch", "切换会话"},
		{"/rename [name]", "重命名当前会话"},
		{"/export", "导出当前会话记录"},
		{"/rollback", "回滚到指定消息"},
		{"/pause", "暂停当前会话"},
		{"/resume", "恢复当前会话"},
		{"/cancel", "取消当前操作"},
		{"/yolo", "切换 YOLO 模式"},
		{"/trust <level>", "设置信任级别 (low/normal/elevated)"},
		{"/help", "显示此帮助面板"},
		{"/quit", "退出"},
	}

	var cmdLines []string
	cmdLines = append(cmdLines, cmdSection)
	for _, c := range commands {
		cmdStyle := lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Width(18).
			Render(c.cmd)
		descStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Render(c.desc)
		cmdLines = append(cmdLines, fmt.Sprintf("  %s %s", cmdStyle, descStyle))
	}
	sections = append(sections, strings.Join(cmdLines, "\n"))

	return strings.Join(sections, "\n\n")
}
