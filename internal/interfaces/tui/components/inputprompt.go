package components

import (
	"github.com/charmbracelet/lipgloss"
)

type InputPrompt struct {
	Visible      bool
	Title        string
	Placeholder  string
	Value        string
	ConfirmLabel string
	Width        int
}

func NewInputPrompt() InputPrompt {
	return InputPrompt{
		ConfirmLabel: "[确认  Enter]",
	}
}

func (p *InputPrompt) Show(title, placeholder, defaultValue, confirmLabel string) {
	p.Visible = true
	p.Title = title
	p.Placeholder = placeholder
	p.Value = defaultValue
	if confirmLabel != "" {
		p.ConfirmLabel = confirmLabel
	}
}

func (p *InputPrompt) Hide() {
	p.Visible = false
	p.Value = ""
}

func (p *InputPrompt) View() string {
	if !p.Visible {
		return ""
	}

	w := 50
	if p.Width > 0 && p.Width < w {
		w = p.Width - 4
	}
	innerW := w - 4

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Padding(0, 1)
	titleLine := titleStyle.Render(p.Title)

	inputDisplay := p.Value
	if p.Value == "" && p.Placeholder != "" {
		inputDisplay = lipgloss.NewStyle().Foreground(ColorMuted).Render(p.Placeholder)
	}
	inputDisplay = inputDisplay + "█"

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorPrimary).
		Width(innerW).
		Padding(0, 1)
	inputLine := inputStyle.Render(inputDisplay)

	confirmBtn := lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorWhite).
		Padding(0, 2).
		Render(p.ConfirmLabel)
	cancelBtn := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 2).
		Render("[取消  Esc]")
	footer := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Right).Render(cancelBtn + "  " + confirmBtn)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		"",
		inputLine,
		"",
		footer,
	)

	return lipgloss.NewStyle().
		Width(w).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Render(content)
}
