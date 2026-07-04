package components

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pasteFlushMsg struct{}

type InputArea struct {
	textarea     textarea.Model
	Width        int
	ContextUsage string
	TokenUsage   string
	WorkingDir   string

	pasteBuf []rune
}

func NewInputArea() InputArea {
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Enter 发送, Shift+Enter 换行, / 命令面板)"
	ta.SetHeight(1)
	ta.MaxHeight = 1
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(ColorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorMuted)
	ta.BlurredStyle = ta.FocusedStyle

	return InputArea{
		textarea: ta,
	}
}

func (i *InputArea) Focus() {
	i.textarea.Focus()
}

func (i *InputArea) Blur() {
	i.textarea.Blur()
}

func (i *InputArea) Focused() bool {
	return i.textarea.Focused()
}

func (i *InputArea) Value() string {
	return i.textarea.Value()
}

func (i *InputArea) Reset() {
	i.textarea.Reset()
}

func (i *InputArea) SetValue(v string) {
	i.textarea.SetValue(v)
}

func (i *InputArea) SetWidth(w int) {
	i.Width = w
	i.textarea.SetWidth(w - 4)
}

func (i *InputArea) Update(msg tea.Msg) (InputArea, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			if len(i.pasteBuf) == 0 {
				i.pasteBuf = append(i.pasteBuf, msg.Runes...)
				return *i, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
					return pasteFlushMsg{}
				})
			}
			i.pasteBuf = append(i.pasteBuf, msg.Runes...)
			return *i, nil
		}
		if (msg.Type == tea.KeyEnter || msg.Type == tea.KeyTab) && len(i.pasteBuf) > 0 {
			i.pasteBuf = append(i.pasteBuf, '\n')
			return *i, nil
		}
	case pasteFlushMsg:
		if len(i.pasteBuf) > 0 {
			i.textarea.InsertString(string(i.pasteBuf))
			i.pasteBuf = nil
		}
		return *i, nil
	}
	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)
	return *i, cmd
}

func (i *InputArea) IsPasteActive() bool {
	return len(i.pasteBuf) > 0
}

func (i *InputArea) View() string {
	borderColor := ColorBorder
	if i.textarea.Focused() {
		borderColor = ColorPrimary
	}
	style := InputAreaStyle.Copy().
		Width(i.Width - 2).
		BorderForeground(borderColor)

	var parts []string
	parts = append(parts, style.Render(i.textarea.View()))

	footer := i.buildFooter()
	if footer != "" {
		footerStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1).
			Width(i.Width - 2)
		parts = append(parts, footerStyle.Render(footer))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (i *InputArea) buildFooter() string {
	var parts []string

	if i.ContextUsage != "" {
		parts = append(parts, i.ContextUsage)
	}
	if i.TokenUsage != "" {
		parts = append(parts, i.TokenUsage)
	}

	if len(parts) == 0 && i.WorkingDir == "" {
		return ""
	}

	result := ""
	for idx, p := range parts {
		if idx > 0 {
			result += "  ·  "
		}
		result += p
	}

	if i.WorkingDir != "" {
		footerWidth := i.Width - 4
		leftWidth := lipgloss.Width(result)
		rightStr := "  ·  " + i.WorkingDir
		rightWidth := lipgloss.Width(rightStr)
		spacer := footerWidth - leftWidth - rightWidth
		if spacer < 2 {
			spacer = 2
		}
		result += strings.Repeat(" ", spacer) + rightStr
	}

	return result
}
