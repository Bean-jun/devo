package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type RollbackItem struct {
	MessageID   string
	Content     string
	DisplayText string
	CreatedAt   string
}

type RollbackPicker struct {
	Visible bool
	Items   []RollbackItem
	Cursor  int
	Width   int
	Height  int
}

func NewRollbackPicker() RollbackPicker {
	return RollbackPicker{}
}

func (r *RollbackPicker) Show(items []RollbackItem) {
	r.Visible = true
	r.Items = items
	r.Cursor = 0
}

func (r *RollbackPicker) Hide() {
	r.Visible = false
	r.Cursor = 0
	r.Items = nil
}

func (r *RollbackPicker) CursorUp() {
	if r.Cursor > 0 {
		r.Cursor--
	}
}

func (r *RollbackPicker) CursorDown() {
	if r.Cursor < len(r.Items)-1 {
		r.Cursor++
	}
}

func (r *RollbackPicker) SelectedMessageID() string {
	if r.Cursor >= 0 && r.Cursor < len(r.Items) {
		return r.Items[r.Cursor].MessageID
	}
	return ""
}

func (r *RollbackPicker) View() string {
	if !r.Visible {
		return ""
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("选择要回滚到的消息 (↑↓ 选择, Enter 确认, Esc 取消)")

	var lines []string
	lines = append(lines, header)

	for i, item := range r.Items {
		displayText := item.DisplayText
		if displayText == "" {
			displayText = truncateText(item.Content, 60)
		}

		line := lipgloss.NewStyle().
			Padding(0, 1).
			Width(r.Width - 6).
			Render(displayText)

		if i == r.Cursor {
			line = lipgloss.NewStyle().
				Foreground(ColorWhite).
				Background(ColorPrimary).
				Padding(0, 1).
				Width(r.Width - 6).
				Render(displayText)
		}

		lines = append(lines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Background(ColorSurface).
		Width(r.Width - 2).
		Render(content)
}

func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	runes := []rune(text)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return text
}
