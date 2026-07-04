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
	viewport          viewport.Model
	Messages          []types.Message
	ToolCards         []ToolCardData
	Width             int
	Height            int
	mdRenderer        *glamour.TermRenderer
	StreamingBuffer   strings.Builder
	StreamingActive   bool
	SpinnerFrame      int
	ThinkingCollapsed bool
	ThinkingContent   string
}

type ToolCardData struct {
	ToolName     string
	Params       string
	Result       string
	Success      bool
	Duration     string
	Expanded     bool
	Stage        string
	OutputChunks []string
	GroupID      string
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
	m.ThinkingContent = ""
	m.ThinkingCollapsed = true
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddMessage(msg types.Message) {
	m.Messages = append(m.Messages, msg)
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddToolCard(card ToolCardData) {
	if len(m.ToolCards) > 0 {
		last := &m.ToolCards[len(m.ToolCards)-1]
		groupHasResult := false
		if last.GroupID != "" {
			for i := range m.ToolCards {
				if m.ToolCards[i].GroupID == last.GroupID && m.ToolCards[i].Result != "" {
					groupHasResult = true
					break
				}
			}
		}
		if last.Result == "" && !groupHasResult {
			card.GroupID = last.GroupID
			if card.GroupID == "" {
				card.GroupID = fmt.Sprintf("group_%d", len(m.ToolCards))
				last.GroupID = card.GroupID
			}
		}
	}
	m.ToolCards = append(m.ToolCards, card)
	m.renderContent()
	m.viewport.GotoBottom()
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
	m.ThinkingContent += message
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddStreamingChunk(text string) {
	m.StreamingActive = true
	m.StreamingBuffer.WriteString(text)
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) FinalizeStreaming() string {
	if !m.StreamingActive {
		return ""
	}
	m.StreamingActive = false
	content := m.StreamingBuffer.String()
	m.StreamingBuffer.Reset()

	if content != "" {
		msg := types.Message{
			Role:    "assistant",
			Content: content,
		}
		m.Messages = append(m.Messages, msg)
	}
	m.renderContent()
	m.viewport.GotoBottom()
	return content
}

func (m *MessageViewport) UpdateToolCardStage(toolName string, stage string) {
	for i := range m.ToolCards {
		if m.ToolCards[i].ToolName == toolName {
			m.ToolCards[i].Stage = stage
			break
		}
	}
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AppendToolCardChunk(toolName string, chunk string) {
	for i := range m.ToolCards {
		if m.ToolCards[i].ToolName == toolName {
			m.ToolCards[i].OutputChunks = append(m.ToolCards[i].OutputChunks, chunk)
			if m.ToolCards[i].Stage == "" {
				m.ToolCards[i].Stage = "executing"
			}
			break
		}
	}
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

	m.renderContent()
}

func (m *MessageViewport) Update(msg tea.Msg) (MessageViewport, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

func (m *MessageViewport) View() string {
	return lipgloss.NewStyle().Height(m.Height).Render(m.viewport.View())
}

func (m *MessageViewport) Refresh() {
	m.renderContent()
}

func (m *MessageViewport) renderContent() {
	if len(m.Messages) == 0 && len(m.ToolCards) == 0 && !m.StreamingActive {
		m.viewport.SetContent(m.renderEmptyState())
		return
	}

	var lines []string

	for _, msg := range m.Messages {
		switch msg.Role {
		case "user":
			lines = append(lines, m.renderUserMessage(msg))
		case "assistant":
			if msg.Content == "" {
				continue
			}
			lines = append(lines, m.renderAssistantMessage(msg))
		case "system":
			lines = append(lines, m.renderSystemMessage(msg))
		case "tool":
			lines = append(lines, m.renderToolMessage(msg))
		}
	}

	lines = append(lines, m.renderToolCards())

	if m.StreamingActive {
		streamingContent := m.StreamingBuffer.String()
		if streamingContent != "" {
			lines = append(lines, m.renderStreamingContent(streamingContent))
		}
	}

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)
}

func (m *MessageViewport) renderUserMessage(msg types.Message) string {
	prefix := UserPrefixStyle.Render("You")
	body := msg.Content

	lines := strings.Split(prefix+"\n"+body, "\n")
	maxW := m.Width - 4
	if maxW < 20 {
		maxW = 20
	}

	var result []string
	for _, line := range lines {
		lineLen := lipgloss.Width(line)
		pad := maxW - lineLen
		if pad < 0 {
			pad = 0
		}
		result = append(result, strings.Repeat(" ", pad)+line)
	}

	return strings.Join(result, "\n")
}

func (m *MessageViewport) renderAssistantMessage(msg types.Message) string {
	prefix := AssistantPrefixStyle.Render("🤖")
	body := msg.Content
	if m.mdRenderer != nil {
		rendered, err := m.mdRenderer.Render(msg.Content)
		if err == nil {
			body = rendered
		}
	}

	thinking := m.renderThinkingContent()
	if thinking == "" {
		return prefix + "\n" + body
	}

	thinkPrefix := lipgloss.NewStyle().Foreground(ColorMuted).Render("💭")
	header := prefix + "  " + thinkPrefix
	sep := lipgloss.NewStyle().Foreground(ColorBorder).Render("  ──────────────────────────────")
	return header + "\n" + thinking + "\n" + sep + "\n" + body
}

func (m *MessageViewport) renderThinkingContent() string {
	if m.ThinkingContent == "" {
		return ""
	}
	if m.ThinkingCollapsed {
		hint := lipgloss.NewStyle().Foreground(ColorMuted).Render("Thinking... (Enter 展开)")
		return "  " + hint
	}
	lines := strings.Split(m.ThinkingContent, "\n")
	var rendered []string
	for _, line := range lines {
		rendered = append(rendered, lipgloss.NewStyle().
			Foreground(ColorMuted).
			Render("  "+line))
	}
	sep := lipgloss.NewStyle().Foreground(ColorBorder).Render("  ── 思考结束 ──")
	return strings.Join(rendered, "\n") + "\n" + sep + "\n  " +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("(Enter 折叠)")
}

func (m *MessageViewport) renderSystemMessage(msg types.Message) string {
	return SystemNoticeStyle.Render("  " + msg.Content)
}

func (m *MessageViewport) renderToolMessage(msg types.Message) string {
	header := lipgloss.NewStyle().Foreground(ColorWarning).Render("[Tool Result]")
	return ToolCardStyle.Copy().Width(m.Width - 4).Render(header + "\n" + truncate(msg.Content, 200))
}

func (m *MessageViewport) renderToolCards() string {
	if len(m.ToolCards) == 0 {
		return ""
	}

	type group struct {
		id    string
		cards []int
	}

	var groups []group

	for i, card := range m.ToolCards {
		if card.GroupID == "" || i == 0 || m.ToolCards[i-1].GroupID != card.GroupID {
			g := group{id: card.GroupID}
			groups = append(groups, g)
		}
		groups[len(groups)-1].cards = append(groups[len(groups)-1].cards, i)
	}

	sepWidth := m.Width - 4
	if sepWidth < 20 {
		sepWidth = 20
	}
	sepLine := strings.Repeat("─", sepWidth)

	var lines []string
	lines = append(lines, ToolCardSeparator.Render("┌"+sepLine))

	for _, g := range groups {
		if len(g.cards) > 1 && g.id != "" {
			header := fmt.Sprintf(" 工具调用 (%d)", len(g.cards))
			groupHeader := lipgloss.NewStyle().
				Foreground(ColorInfo).
				Bold(true).
				Render(header)
			lines = append(lines, groupHeader)
		}

		for _, idx := range g.cards {
			card := m.ToolCards[idx]
			cardView := m.renderToolCard(card)
			lines = append(lines, ToolCardGroupBorder.Render(cardView))
		}
	}

	lines = append(lines, ToolCardSeparator.Render("└"+sepLine))

	return strings.Join(lines, "\n")
}

func (m *MessageViewport) ClearToolCards() {
	m.ToolCards = nil
	m.ThinkingContent = ""
	m.ThinkingCollapsed = true
	m.renderContent()
}

func (m *MessageViewport) ToggleToolCardExpanded(index int) {
	if index >= 0 && index < len(m.ToolCards) {
		m.ToolCards[index].Expanded = !m.ToolCards[index].Expanded
	}
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) ToggleThinking() {
	m.ThinkingCollapsed = !m.ThinkingCollapsed
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) renderToolCard(card ToolCardData) string {
	statusIcon := "⏳"
	statusColor := ColorMuted
	isDone := card.Result != ""
	isExecuting := !isDone

	if isExecuting {
		statusIcon = spinnerChars[m.SpinnerFrame%len(spinnerChars)]
		statusColor = ColorPrimary
	} else if isDone {
		if card.Success {
			statusIcon = "✓"
			statusColor = ColorSuccess
		} else {
			statusIcon = "✗"
			statusColor = ColorDanger
		}
	}

	oneLine := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon) +
		" " + lipgloss.NewStyle().Bold(true).Render(card.ToolName)

	if isDone && card.Duration != "" {
		oneLine += " " + lipgloss.NewStyle().Foreground(ColorMuted).Render(card.Duration)
	}
	if isExecuting {
		oneLine += " " + lipgloss.NewStyle().Foreground(ColorMuted).Render("执行中...")
	}

	if !card.Expanded {
		hint := lipgloss.NewStyle().Foreground(ColorMuted).Render("  (Enter 展开)")
		return ToolCardFoldedStyle.Render(oneLine + hint)
	}

	content := oneLine
	if card.Params != "" {
		content += "\n" + lipgloss.NewStyle().Foreground(ColorMuted).Render("  params: "+truncate(card.Params, 100))
	}
	if len(card.OutputChunks) > 0 {
		showChunks := card.OutputChunks
		if len(showChunks) > 5 {
			showChunks = showChunks[len(showChunks)-5:]
		}
		for _, chunk := range showChunks {
			content += "\n" + lipgloss.NewStyle().Foreground(ColorText).Render("    "+truncate(chunk, 200))
		}
	}
	if isDone {
		resultColor := ColorSuccess
		if !card.Success {
			resultColor = ColorDanger
		}
		content += "\n" + lipgloss.NewStyle().Foreground(resultColor).Render("  "+truncate(card.Result, 300))
	}

	return ToolCardFoldedStyle.Render(content)
}

func (m *MessageViewport) renderStreamingContent(streamingContent string) string {
	prefix := AssistantPrefixStyle.Render("🤖")
	body := streamingContent
	if m.mdRenderer != nil {
		rendered, err := m.mdRenderer.Render(streamingContent)
		if err == nil {
			body = rendered
		}
	}

	thinking := m.renderThinkingContent()
	if thinking == "" {
		return prefix + "\n" + body
	}
	thinkPrefix := lipgloss.NewStyle().Foreground(ColorMuted).Render("💭")
	header := prefix + "  " + thinkPrefix
	sep := lipgloss.NewStyle().Foreground(ColorBorder).Render("  ──────────────────────────────")
	return header + "\n" + thinking + "\n" + sep + "\n" + body
}

func (m *MessageViewport) renderEmptyState() string {
	banner := `
  ____  _____     _______ ____
  |  _ \| ____|   / /_   _/ __ \
  | | | |  _|    / /  | || |  | |
  | |_| | |___  / /___| || |__| |
  |____/|_____|/_/____|_|\____/

  Type / to see commands, Enter to send
  `
	return lipgloss.NewStyle().
		Foreground(ColorMuted).
		Width(m.Width).
		Align(lipgloss.Center).
		Render(banner)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
