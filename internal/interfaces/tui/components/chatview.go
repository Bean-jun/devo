package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ChatView struct {
	MessageView        MessageViewport
	InputArea          InputArea
	Width              int
	Height             int
	Processing         bool
	CommandPaletteView string
}

func NewChatView() ChatView {
	return ChatView{
		MessageView: NewMessageViewport(),
		InputArea:   NewInputArea(),
	}
}

func (c *ChatView) SetSize(width, height int) {
	c.Width = width
	c.Height = height

	inputHeight := 4
	msgHeight := height - inputHeight
	if msgHeight < 5 {
		msgHeight = 5
	}

	c.MessageView.SetSize(width, msgHeight)
	c.InputArea.SetWidth(width)
}

func (c *ChatView) Focus() {
	c.InputArea.Focus()
}

func (c *ChatView) Blur() {
	c.InputArea.Blur()
}

func (c *ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd
	c.MessageView, cmd = c.MessageView.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	c.InputArea, cmd = c.InputArea.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return *c, tea.Batch(cmds...)
}

func (c *ChatView) View() string {
	msgView := c.MessageView.View()
	inputView := c.InputArea.View()

	spacer := ""
	if c.Processing {
		spacer = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Italic(true).
			Render("  Processing...")
	}

	paletteView := c.CommandPaletteView

	if paletteView != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			msgView,
			spacer,
			paletteView,
			inputView,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		msgView,
		spacer,
		inputView,
	)
}
