package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

var (
	ColorPrimary = lipgloss.Color("#A78BFA")
	ColorSuccess = lipgloss.Color("#34D399")
	ColorWarning = lipgloss.Color("#FBBF24")
	ColorDanger  = lipgloss.Color("#F87171")
	ColorInfo    = lipgloss.Color("#60A5FA")
	ColorMuted   = lipgloss.Color("#CBD5E1")
	ColorBg      = lipgloss.Color("#334155")
	ColorSurface = lipgloss.Color("#475569")
	ColorBorder  = lipgloss.Color("#64748B")
	ColorText    = lipgloss.Color("#F8FAFC")
	ColorWhite   = lipgloss.Color("#FFFFFF")

	spinnerChars = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

	InputAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorBorder).
			Background(ColorSurface).
			Padding(0, 1)

	UserPrefixStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	AssistantPrefixStyle = lipgloss.NewStyle().
				Foreground(ColorInfo).
				Bold(true)

	SystemNoticeStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true).
				Padding(0, 1).
				Margin(0, 0, 1, 0)

	ToolCardStyle = lipgloss.NewStyle().
			Padding(0, 0, 0, 2).
			Margin(0, 0, 1, 0)

	ToolCardFoldedStyle = lipgloss.NewStyle().
				Padding(0, 0, 0, 2).
				Margin(0, 0, 0, 2)

	ToolCardGroupBorder = lipgloss.NewStyle().
				Padding(0, 0, 0, 1)

	ToolCardSeparator = lipgloss.NewStyle().
				Foreground(ColorBorder)

	debugLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Bold(true).
			Padding(0, 1)

	debugValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CBD5E1")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A78BFA")).
			Padding(0, 1)
)

type Message struct {
	Role    string
	Content string
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

type MessageViewport struct {
	viewport          viewport.Model
	Messages          []Message
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

func NewMessageViewport() MessageViewport {
	vp := viewport.New(80, 20)
	return MessageViewport{
		viewport: vp,
	}
}

func (m *MessageViewport) SetMessages(messages []Message) {
	m.Messages = messages
	m.ToolCards = nil
	m.ThinkingContent = ""
	m.ThinkingCollapsed = true
	m.renderContent()
	m.viewport.GotoBottom()
}

func (m *MessageViewport) AddMessage(msg Message) {
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
		glamour.WithStandardStyle("dark"),
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

func (m *MessageViewport) renderUserMessage(msg Message) string {
	prefix := UserPrefixStyle.Render("You")
	body := msg.Content

	maxW := m.Width - 4
	if maxW < 20 {
		maxW = 20
	}

	wrappedBody := lipgloss.NewStyle().Width(maxW).Render(body)

	var result []string
	prefixLen := lipgloss.Width(prefix)
	prefixPad := maxW - prefixLen
	if prefixPad < 0 {
		prefixPad = 0
	}
	result = append(result, strings.Repeat(" ", prefixPad)+prefix)

	for _, line := range strings.Split(wrappedBody, "\n") {
		lineLen := lipgloss.Width(line)
		pad := maxW - lineLen
		if pad < 0 {
			pad = 0
		}
		result = append(result, strings.Repeat(" ", pad)+line)
	}

	return strings.Join(result, "\n")
}

func (m *MessageViewport) renderAssistantMessage(msg Message) string {
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

func (m *MessageViewport) renderSystemMessage(msg Message) string {
	maxW := m.Width - 4
	if maxW < 20 {
		maxW = 20
	}
	return SystemNoticeStyle.Copy().Width(maxW).Render("  " + msg.Content)
}

func (m *MessageViewport) renderToolMessage(msg Message) string {
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
	ta.SetHeight(3)
	ta.MaxHeight = 10
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
		if msg.Paste || len(msg.Runes) > 3 {
			s := normalizePastedText(string(msg.Runes))
			if isOSCLeak(s) {
				return *i, nil
			}
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

	inputHeight := 3
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

	var spacer string
	if c.Processing {
		spacer = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Italic(true).
			Render("  Processing...")
	}

	paletteView := c.CommandPaletteView

	parts := []string{msgView}
	if spacer != "" {
		parts = append(parts, spacer)
	}
	if paletteView != "" {
		parts = append(parts, paletteView)
	}
	parts = append(parts, inputView)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

type model struct {
	chatView  ChatView
	submitted bool
	quitting  bool
	width     int
	height    int
	debugInfo string
	lastValue string
}

func initialModel() model {
	cv := NewChatView()
	cv.Focus()
	cv.InputArea.ContextUsage = "context 12.5K"
	cv.InputArea.TokenUsage = "Tokens ↑1.2K ↓3.4K"
	cv.InputArea.WorkingDir = "/home/user/project"

	// Populate with mock messages containing CJK content
	cv.MessageView.AddMessage(Message{
		Role:    "user",
		Content: "你好，帮我分析一下这个项目的代码结构",
	})
	cv.MessageView.AddMessage(Message{
		Role:    "assistant",
		Content: "好的，这个项目是一个 Go 语言编写的 AI 编程助手，\n主要包含以下模块：\n- internal/interfaces/tui：终端用户界面\n- internal/domain：核心业务逻辑\n- internal/infra：基础设施层\n\n代码结构清晰，采用了分层架构设计。",
	})
	cv.MessageView.AddMessage(Message{
		Role:    "user",
		Content: "输入框在 Linux 下显示中文乱码怎么办？",
	})
	cv.MessageView.AddMessage(Message{
		Role:    "assistant",
		Content: "中文乱码通常与终端编码设置有关，\n请检查以下环境变量：\n\n- `LANG` 应为 `zh_CN.UTF-8` 或 `C.UTF-8`\n- `LC_ALL` 和 `LC_CTYPE` 也应设为 UTF-8\n- `TERM` 推荐使用 `xterm-256color`\n\n如果使用 SSH 连接，请确保客户端也支持 UTF-8。",
	})

	// Add tool cards with Chinese content
	cv.MessageView.AddToolCard(ToolCardData{
		ToolName: "read_file",
		Params:   `{"path": "/home/user/project/main.go"}`,
		Result:   "package main\n\n中国語のテスト",
		Success:  true,
		Duration: "0.5s",
		Expanded: true,
	})
	cv.MessageView.AddToolCard(ToolCardData{
		ToolName: "execute_command",
		Params:   `{"command": "echo 你好世界"}`,
		Result:   "你好世界",
		Success:  true,
		Duration: "0.1s",
		Expanded: false,
	})

	return model{
		chatView:  cv,
		debugInfo: buildDebugInfo(),
	}
}

func buildDebugInfo() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("OS: %s/%s", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf(" | LANG: %s", os.Getenv("LANG")))
	sb.WriteString(fmt.Sprintf(" | LC_ALL: %s", os.Getenv("LC_ALL")))
	sb.WriteString(fmt.Sprintf(" | LC_CTYPE: %s", os.Getenv("LC_CTYPE")))
	sb.WriteString(fmt.Sprintf(" | TERM: %s", os.Getenv("TERM")))
	sb.WriteString(fmt.Sprintf(" | ColorProfile: %v", termenv.DefaultOutput().ColorProfile()))
	return sb.String()
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width > 20 {
			chatWidth := msg.Width
			chatHeight := msg.Height - 2
			if chatHeight < 10 {
				chatHeight = 10
			}
			m.chatView.SetSize(chatWidth, chatHeight)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.chatView.InputArea.Focused() {
				m.lastValue = m.chatView.InputArea.Value()
				m.submitted = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.chatView.InputArea, cmd = m.chatView.InputArea.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399")).
			Padding(1, 1).
			Render("Goodbye!")
	}

	if m.submitted {
		content := m.lastValue
		if content == "" {
			content = "(empty)"
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#34D399")).
				Padding(0, 1).
				Render("Submitted Content:"),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#64748B")).
				Padding(0, 1).
				Render(strings.Repeat("─", 40)),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8FAFC")).
				Padding(0, 1).
				Render(content),
			"",
			helpStyle.Render("Press Ctrl+C to exit..."),
		)
	}

	debugSection := lipgloss.JoinVertical(
		lipgloss.Left,
		debugLabelStyle.Render("Environment Debug Info:"),
		debugValueStyle.Render(m.debugInfo),
		debugValueStyle.Render(fmt.Sprintf("Width: %d | Height: %d", m.width, m.height)),
		debugValueStyle.Render(fmt.Sprintf("Input chars: %d | Input bytes: %d",
			len([]rune(m.chatView.InputArea.Value())),
			len(m.chatView.InputArea.Value()))),
		debugValueStyle.Render(fmt.Sprintf("Messages: %d | ToolCards: %d",
			len(m.chatView.MessageView.Messages),
			len(m.chatView.MessageView.ToolCards))),
	)

	helpSection := helpStyle.Render("Enter: Submit | Ctrl+Enter/Shift+Enter: Newline | Ctrl+V: Paste | Esc/Ctrl+C: Quit")

	chatView := m.chatView.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Full ChatView + MessageViewport + InputArea Test"),
		helpSection,
		debugSection,
		"",
		chatView,
	)
}

func main() {
	fmt.Printf("=== Input Box Test Environment ===\n")
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("LANG: %s\n", os.Getenv("LANG"))
	fmt.Printf("LC_ALL: %s\n", os.Getenv("LC_ALL"))
	fmt.Printf("LC_CTYPE: %s\n", os.Getenv("LC_CTYPE"))
	fmt.Printf("TERM: %s\n", os.Getenv("TERM"))
	fmt.Printf("ColorProfile: %v\n", termenv.DefaultOutput().ColorProfile())
	fmt.Printf("==================================\n\n")

	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
