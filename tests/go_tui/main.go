package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	textarea  textarea.Model
	submitted bool
	quitting  bool
	width     int
	height    int
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "输入内容... (Enter 提交, Ctrl+Enter 换行, Ctrl+V 粘贴, Esc 退出)"
	ta.Focus()
	ta.SetHeight(8)
	ta.MaxHeight = 20
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ta.BlurredStyle = ta.FocusedStyle

	return model{
		textarea: ta,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width > 4 {
			m.textarea.SetWidth(msg.Width - 4)
		}

	case tea.KeyMsg:
		if msg.Paste || len(msg.Runes) > 3 {
			s := normalizePastedText(string(msg.Runes))
			m.textarea.InsertString(s)
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+v":
			text, err := clipboard.ReadAll()
			if err == nil && text != "" {
				text = normalizePastedText(text)
				m.textarea.InsertString(text)
			}
			return m, nil

		case "enter":
			m.submitted = true
			return m, tea.Quit

		case "ctrl+n":
			m.textarea.InsertString("\n")
			return m, nil

		case "shift+enter":
			if runtime.GOOS != "windows" {
				m.textarea.InsertString("\n")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return quitStyle.Render("Goodbye!")
	}

	if m.submitted {
		content := m.textarea.Value()
		if content == "" {
			content = "(empty)"
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			submittedHeaderStyle.Render("Submitted Content:"),
			dividerStyle.Render(strings.Repeat("─", 40)),
			submittedContentStyle.Render(content),
			"",
			hintStyle.Render("Press Ctrl+C to exit..."),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render("Terminal Input Box"),
		helpStyle.Render("Enter: Submit | Ctrl+Enter: Newline | Ctrl+V: Paste | Esc: Quit"),
		"",
		m.textarea.View(),
		"",
		hintStyle.Render(fmt.Sprintf("Characters: %d", len(m.textarea.Value()))),
	)
}

func normalizePastedText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimRight(s, "\n")
}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 1)

	quitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Padding(1, 1)

	submittedHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("86")).
				Padding(0, 1)

	submittedContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
)

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
