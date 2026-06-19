package components

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputArea struct {
	textarea textarea.Model
	Width    int
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

func (i *InputArea) SetWidth(w int) {
	i.Width = w
	// textarea width = outer width - border(2) - padding(2) - style overhead(2)
	i.textarea.SetWidth(w - 6)
}

func (i *InputArea) Update(msg tea.Msg) (InputArea, tea.Cmd) {
	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)
	return *i, cmd
}

func (i *InputArea) View() string {
	borderColor := ColorBorder
	if i.textarea.Focused() {
		borderColor = ColorPrimary
	}
	style := InputAreaStyle.Copy().
		Width(i.Width - 2).
		BorderForeground(borderColor)
	return style.Render(i.textarea.View())
}
