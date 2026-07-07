package components

import (
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputArea struct {
	textarea     textarea.Model
	Width        int
	ContextUsage string
	TokenUsage   string
	WorkingDir   string
}

func NewInputArea() InputArea {
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Enter 发送, Ctrl+N 换行, Ctrl+V 粘贴, / 命令面板)"
	ta.SetHeight(1)
	ta.MaxHeight = 20
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(ColorText)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
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
		if msg.Paste || len(msg.Runes) > 3 {
			s := normalizePastedText(string(msg.Runes))
			i.textarea.InsertString(s)
			return *i, nil
		}

		switch msg.String() {
		case "ctrl+n":
			i.textarea.InsertString("\n")
			return *i, nil

		case "shift+enter":
			if runtime.GOOS != "windows" {
				i.textarea.InsertString("\n")
			}
			return *i, nil

		case "ctrl+v":
			text, err := clipboard.ReadAll()
			if err == nil && text != "" {
				text = normalizePastedText(text)
				i.textarea.InsertString(text)
			}
			return *i, nil
		}
	}
	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)

	if isOSCLeak(i.textarea.Value()) {
		i.textarea.Reset()
	}

	return *i, cmd
}

func isOSCLeak(s string) bool {
	if len(s) < 5 {
		return false
	}
	if strings.Contains(s, "rgb:") {
		return true
	}
	if strings.HasPrefix(s, "]") {
		return strings.Contains(s, "11;") ||
			strings.Contains(s, "10;") ||
			strings.Contains(s, "4;")
	}
	return false
}

func normalizePastedText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimRight(s, "\n")
}

func (i *InputArea) View() string {
	borderColor := lipgloss.Color("#94A3B8")
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
