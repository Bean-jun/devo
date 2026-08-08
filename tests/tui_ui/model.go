package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── App Model ───

type model struct {
	viewport     viewport.Model
	textarea     textarea.Model
	renderer     *msgRenderer
	messages     []Message
	sessions     []Session
	statusBar    StatusBar
	inputArea    InputArea
	overlay      OverlayStack
	toast        Toast
	cmdSheet     CommandSheet
	sessPicker   SessionPicker
	approval     ApprovalModal
	helpPanel    HelpPanel
	filesPanel   FilesPanel
	skillsPanel  SkillsPanel
	mcpPanel     MCPPanel
	memoryPanel  MemoryPanel
	wsPanel      WorkspacePanel
	renameModal  RenameModal
	rollback     RollbackPicker
	newSessModal NewSessionModal
	width        int
	height       int
	ready        bool
}

func newModel() model {
	ta := textarea.New()
	ta.Placeholder = "输入消息..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(colorText())
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorMuted())
	ta.BlurredStyle = ta.FocusedStyle
	ta.Focus()

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	messages := mockMessages()
	sessions := mockSessions()

	return model{
		viewport: vp,
		textarea: ta,
		renderer: newMsgRenderer(),
		messages: messages,
		sessions: sessions,
		statusBar: StatusBar{
			AppName:   "Devo",
			Session:   "demo-session",
			Connected: true,
		},
		cmdSheet:   NewCommandSheet(),
		sessPicker: NewSessionPicker(sessions),
		approval: ApprovalModal{
			Operation: "write_file",
			Risk:      "HIGH",
			Diff:      "+ if x == nil { return }\n- oldFunc() {",
		},
		helpPanel:    HelpPanel{},
		filesPanel:   NewFilesPanel(),
		skillsPanel:  NewSkillsPanel(),
		mcpPanel:     NewMCPPanel(),
		memoryPanel:  NewMemoryPanel(),
		wsPanel:      NewWorkspacePanel(),
		renameModal:  RenameModal{Current: "demo-session"},
		rollback:     NewRollbackPicker(messages),
		newSessModal: NewSessionModal{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.EnterAltScreen,
	)
}

// ─── Mock 响应 ───

type mockResponseMsg struct {
	msg Message
}

func mockRespond(input string, now string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(600 * time.Millisecond)
		return mockResponseMsg{
			msg: Message{
				Role:     RoleAssistant,
				Content:  fmt.Sprintf("收到：「%s」\n\n这是一个 Mock 回复。在真实环境中，这里会调用 AI 模型生成回复。\n\n以下是 Markdown 测试：\n\n```go\nfunc main() {\n    fmt.Println(\"Hello, Devo!\")\n}\n```\n\n| 功能 | 状态 |\n|------|------|\n| 消息渲染 | ✅ |\n| 工具调用 | ✅ |\n\n> 这是一个引用块测试。", input),
				Thinking: "分析用户输入：" + truncateStr(input, 40) + "...\n检索相关上下文...\n生成回复内容...",
				ToolCalls: []ToolCall{
					{
						Name: "search_codebase", Summary: "相关代码",
						Status: "success", Duration: "0.5s",
						Diff: "找到 3 个相关文件",
					},
				},
				Time: now,
			},
		}
	}
}

// ─── Toast ───

type toastTickMsg struct{}

func tickToast() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return toastTickMsg{}
	})
}

func (m *model) showToast(msg, typ string) {
	m.toast = Toast{
		Message:  msg,
		Type:     typ,
		Duration: 3,
		Width:    m.width,
	}
}

// ─── 面板宽度计算 ───

func (m model) overlayPanelWidth() int {
	maxW := 72
	if m.width > 0 {
		candidate := m.width - 8
		if candidate < 40 {
			return 40
		}
		if candidate > maxW {
			return maxW
		}
		return candidate
	}
	return maxW
}

// ─── Viewport 刷新 ───

func (m *model) refreshViewport() {
	if m.renderer == nil {
		return
	}
	content := m.renderer.render(m.messages)
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// ─── 用户消息跳转 ───

// findUserMessageYOffsets 返回所有用户消息在渲染内容中的行偏移量
func (m *model) findUserMessageYOffsets() []int {
	if m.renderer == nil || m.renderer.cache == nil {
		return nil
	}

	var offsets []int
	lineNum := 0

	for i, msg := range m.messages {
		if msg.Role == RoleUser {
			offsets = append(offsets, lineNum)
		}
		if i < len(m.renderer.cache.cache) {
			lineNum += strings.Count(m.renderer.cache.cache[i], "\n") + 1
		}
	}
	return offsets
}

func (m *model) jumpToPrevUserMessage() {
	offsets := m.findUserMessageYOffsets()
	if len(offsets) == 0 {
		return
	}

	currentY := m.viewport.YOffset

	// 从后往前找第一个小于当前位置的用户消息
	for i := len(offsets) - 1; i >= 0; i-- {
		if offsets[i] < currentY {
			m.viewport.SetYOffset(offsets[i])
			return
		}
	}
	// 当前位置之上没有，跳到第一条
	m.viewport.SetYOffset(offsets[0])
}

func (m *model) jumpToNextUserMessage() {
	offsets := m.findUserMessageYOffsets()
	if len(offsets) == 0 {
		return
	}

	currentY := m.viewport.YOffset

	// 找第一个大于当前位置的用户消息
	for _, offset := range offsets {
		if offset > currentY {
			m.viewport.SetYOffset(offset)
			return
		}
	}
	// 当前位置之下没有，跳到底部
	m.viewport.GotoBottom()
}
