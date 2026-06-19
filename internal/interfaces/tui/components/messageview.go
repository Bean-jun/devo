package components

import (
	"fmt"
	"strings"

	"devo/internal/interfaces/tui/types"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type MessageViewport struct {
	viewport   viewport.Model
	Messages   []types.Message
	ToolCards  []ToolCardData
	Width      int
	Height     int
	mdRenderer *glamour.TermRenderer
}

type ToolCardData struct {
	ToolName string
	Params   string
	Result   string
	Success  bool
	Duration string
	Expanded bool
}

func NewMessageViewport() MessageViewport {
	vp := viewport.New(80, 20)
	return MessageViewport{
		viewport: vp,
	}
}

func (m *MessageViewport) SetMessages(messages []types.Message) {
	m.Messages = messages
	m.ToolCards = nil
	m.renderContent()
}

func (m *MessageViewport) AddMessage(msg types.Message) {
	m.Messages = append(m.Messages, msg)
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddToolCard(card ToolCardData) {
	m.ToolCards = append(m.ToolCards, card)
}

func (m *MessageViewport) UpdateToolCard(toolName string, success bool, summary string, duration string) {
	for i := range m.ToolCards {
		if m.ToolCards[i].ToolName == toolName && m.ToolCards[i].Result == "" {
			m.ToolCards[i].Success = success
			m.ToolCards[i].Result = summary
			m.ToolCards[i].Duration = duration
			break
		}
	}
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddSystemNotice(content string) {
	msg := types.Message{
		Role:    "system",
		Content: content,
	}
	m.Messages = append(m.Messages, msg)
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddThinking(message string) {
	msg := types.Message{
		Role:    "thinking",
		Content: message,
	}
	m.Messages = append(m.Messages, msg)
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.viewport.Width = width
	m.viewport.Height = height

	contentWidth := width - 10
	if contentWidth < 40 {
		contentWidth = 40
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(contentWidth),
	)
	if err == nil {
		m.mdRenderer = r
	}
}

func (m *MessageViewport) Update(msg tea.Msg) (MessageViewport, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

func (m *MessageViewport) View() string {
	return m.viewport.View()
}

func (m *MessageViewport) renderContent() {
	var lines []string

	for _, msg := range m.Messages {
		switch msg.Role {
		case "user":
			lines = append(lines, m.renderUserMessage(msg))
		case "assistant":
			lines = append(lines, m.renderAssistantMessage(msg))
		case "system":
			lines = append(lines, m.renderSystemMessage(msg))
		case "tool":
			lines = append(lines, m.renderToolMessage(msg))
		case "thinking":
			lines = append(lines, m.renderThinking(msg))
		}
	}

	for _, card := range m.ToolCards {
		lines = append(lines, m.renderToolCard(card))
	}

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)
}

func (m *MessageViewport) renderUserMessage(msg types.Message) string {
	header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("[You]")
	body := msg.Content
	return UserBubbleStyle.Copy().Width(m.Width - 8).Render(header + "\n" + body)
}

func (m *MessageViewport) renderAssistantMessage(msg types.Message) string {
	header := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("[Assistant]")
	body := msg.Content
	if m.mdRenderer != nil {
		rendered, err := m.mdRenderer.Render(msg.Content)
		if err == nil {
			body = rendered
		}
	}
	return AssistantBubbleStyle.Copy().Width(m.Width - 8).Render(header + "\n" + body)
}

func (m *MessageViewport) renderSystemMessage(msg types.Message) string {
	return SystemNoticeStyle.Render("  " + msg.Content)
}

func (m *MessageViewport) renderToolMessage(msg types.Message) string {
	header := lipgloss.NewStyle().Foreground(ColorWarning).Render("[Tool Result]")
	return ToolCardStyle.Copy().Width(m.Width - 8).Render(header + "\n" + truncate(msg.Content, 200))
}

func (m *MessageViewport) renderThinking(msg types.Message) string {
	return ThinkingStyle.Render("  " + msg.Content)
}

func (m *MessageViewport) renderToolCard(card ToolCardData) string {
	statusIcon := "⏳"
	style := ToolCardStyle
	if card.Result != "" {
		if card.Success {
			statusIcon = "✓"
			style = ToolCardSuccess
		} else {
			statusIcon = "✗"
			style = ToolCardError
		}
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Tool: %s", card.ToolName))
	if card.Duration != "" {
		header += " · " + card.Duration
	}
	header += " · " + statusIcon

	content := header
	if card.Expanded || card.Result != "" {
		if card.Params != "" {
			content += "\n" + lipgloss.NewStyle().Foreground(ColorMuted).Render("  params: "+truncate(card.Params, 100))
		}
		if card.Result != "" {
			resultColor := ColorSuccess
			if !card.Success {
				resultColor = ColorDanger
			}
			content += "\n" + lipgloss.NewStyle().Foreground(resultColor).Render("  "+truncate(card.Result, 200))
		}
	}

	return style.Copy().Width(m.Width - 8).Render(content)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
