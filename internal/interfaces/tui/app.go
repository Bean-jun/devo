package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/messages"
	"devo/internal/interfaces/tui/types"
)

type AppState int

const (
	StateReady AppState = iota
	StateProcessing
	StateAwaitingApproval
	StateQuitting
)

type App struct {
	apiClient *APIClient
	sseClient *SSEClient

	sessions      []types.SessionInfo
	activeSession *types.SessionInfo
	msgs          []types.Message

	statusBar     components.StatusBar
	sidebar       components.SessionSidebar
	chatView      components.ChatView
	helpBar       components.HelpBar
	approvalModal components.ApprovalModal
	toast         components.Toast

	state       AppState
	showSidebar bool
	width       int
	height      int
	workingDir  string
	ready       bool
	apiBaseURL  string

	initStatus string
	initErr    error
	tickCount  int
}

func NewAppWithURL(baseURL string) (*App, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	return &App{
		apiBaseURL:    baseURL,
		showSidebar:   true,
		workingDir:    wd,
		state:         StateReady,
		initStatus:    "Connecting to server...",
		statusBar:     components.NewStatusBar(),
		sidebar:       components.NewSessionSidebar(),
		chatView:      components.NewChatView(),
		helpBar:       components.NewHelpBar(),
		approvalModal: components.NewApprovalModal(),
		toast:         components.NewToast(),
	}, nil
}

func (a *App) Init() tea.Cmd {
	a.apiClient = NewAPIClient(a.apiBaseURL)
	Log("[TUI] Initializing, baseURL=%s, workingDir=%s", a.apiBaseURL, a.workingDir)
	a.initStatus = "Creating session..."
	return tea.Batch(
		a.initSessionCmd(),
		tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return messages.TickMsg(t)
		}),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !a.ready {
		return a.updateLoading(msg)
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case messages.TickMsg:
		a.toast.Tick()
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return messages.TickMsg(t)
		}))

	case messages.APIResponse:
		cmd := a.handleAPIResponse(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case messages.SSEEvent:
		cmd := a.handleSSEEvent(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case messages.ApprovalDecision:
		cmd := a.handleApprovalDecision(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		cmd := a.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		a.layout()

		if a.approvalModal.Visible {
			a.approvalModal, cmd = a.approvalModal.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		a.chatView, cmd = a.chatView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	a.layout()

	var cmd tea.Cmd
	a.chatView, cmd = a.chatView.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if a.approvalModal.Visible {
		a.approvalModal, cmd = a.approvalModal.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case messages.TickMsg:
		a.tickCount++
		return a, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return messages.TickMsg(t)
		})

	case messages.APIResponse:
		if msg.Kind == "init_session" {
			if msg.Err != nil {
				a.initErr = msg.Err
				a.initStatus = fmt.Sprintf("Failed: %v", msg.Err)
				Log("[TUI] init_session failed: %v", msg.Err)
				return a, nil
			}
			sess := msg.Data.(*types.SessionInfo)
			a.activeSession = sess
			a.sessions = []types.SessionInfo{*sess}
			a.sidebar.SetSessions(a.sessions)
			a.sidebar.ActiveID = sess.ID
			a.statusBar.SessionTitle = sess.Title
			a.statusBar.SessionState = sess.State
			a.ready = true
			a.tickCount = 0
			a.chatView.Focus()
			a.helpBar.FocusMode = "✎ Input"
			Log("[TUI] Session created: id=%s, title=%s", sess.ID, sess.Title)
			return a, a.connectSSECmd()
		}
		return a, nil

	case tea.KeyMsg:
		if a.initErr != nil && msg.String() == "enter" {
			a.initErr = nil
			a.initStatus = "Retrying..."
			Log("[TUI] Retrying session creation...")
			return a, a.initSessionCmd()
		}
		if msg.String() == "ctrl+c" || msg.String() == "ctrl+q" {
			return a, tea.Quit
		}
	}

	return a, nil
}

func (a *App) View() string {
	if !a.ready {
		spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := spinnerChars[a.tickCount%len(spinnerChars)]

		lines := []string{
			fmt.Sprintf("%s %s", spinner, a.initStatus),
		}

		if a.initErr != nil {
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444")).
				Render(fmt.Sprintf("  Error: %v", a.initErr)))
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Render("  Press Enter to retry, Ctrl+Q to quit"))
		} else {
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Render("  Log: ~/.devo/devo.log"))
		}

		msg := lipgloss.JoinVertical(lipgloss.Left, lines...)
		result := lipgloss.NewStyle().
			Width(a.width).
			Height(a.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(msg)
		Log("[VIEW] loading screen: w=%d h=%d ready=%v", a.width, a.height, a.ready)
		return result
	}

	statusBarView := a.statusBar.View()
	helpBarView := a.helpBar.View()

	sidebarView := ""
	if a.showSidebar {
		sidebarView = a.sidebar.View()
	}

	chatView := a.chatView.View()

	var mainContent string
	if a.showSidebar {
		sidebarWidth := lipgloss.Width(sidebarView)
		chatWidth := a.width - sidebarWidth
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarView,
			lipgloss.NewStyle().Width(chatWidth).Render(chatView),
		)
	} else {
		mainContent = chatView
	}

	toastView := a.toast.View()

	view := lipgloss.JoinVertical(lipgloss.Left,
		statusBarView,
		mainContent,
		toastView,
		helpBarView,
	)

	if a.approvalModal.Visible {
		modalView := a.approvalModal.View()
		view = lipgloss.JoinVertical(lipgloss.Left,
			statusBarView,
			modalView,
			helpBarView,
		)
	}

	Log("[VIEW] main screen: w=%d h=%d ready=%v chatW=%d chatH=%d msgs=%d",
		a.width, a.height, a.ready,
		a.chatView.Width, a.chatView.Height,
		len(a.chatView.MessageView.Messages))

	return view
}

func (a *App) layout() {
	a.statusBar.Width = a.width
	a.helpBar.Width = a.width
	a.toast.Width = a.width

	sidebarWidth := 22
	chatWidth := a.width
	if a.showSidebar {
		chatWidth = a.width - sidebarWidth - 1
	}
	if chatWidth < 20 {
		chatWidth = 20
	}

	topHeight := 1
	bottomHeight := 1
	chatHeight := a.height - topHeight - bottomHeight - 2
	if chatHeight < 10 {
		chatHeight = 10
	}

	a.sidebar.Width = sidebarWidth
	a.sidebar.Height = chatHeight
	a.chatView.SetSize(chatWidth, chatHeight)
}

func (a *App) initSessionCmd() tea.Cmd {
	return func() tea.Msg {
		dirName := a.workingDir
		if idx := strings.LastIndex(dirName, "/"); idx >= 0 {
			dirName = dirName[idx+1:]
		}
		if idx := strings.LastIndex(dirName, "\\"); idx >= 0 {
			dirName = dirName[idx+1:]
		}

		Log("[API] POST /api/v1/sessions working_directory=%s title=%s", a.workingDir, dirName)
		sess, err := a.apiClient.CreateSession(a.workingDir, dirName)
		if err != nil {
			Log("[API] CreateSession failed: %v", err)
			return messages.APIResponse{Kind: "init_session", Err: err}
		}

		Log("[API] CreateSession OK: id=%s", sess.ID)
		return messages.APIResponse{Kind: "init_session", Data: sess}
	}
}

func (a *App) handleAPIResponse(msg messages.APIResponse) tea.Cmd {
	if msg.Err != nil {
		Log("[API] Response error (kind=%s): %v", msg.Kind, msg.Err)
		a.toast.Show(msg.Err.Error(), true)
		return nil
	}

	Log("[API] Response OK: kind=%s", msg.Kind)

	switch msg.Kind {
	case "init_session":
		sess := msg.Data.(*types.SessionInfo)
		a.activeSession = sess

		found := false
		for i, s := range a.sessions {
			if s.ID == sess.ID {
				a.sessions[i] = *sess
				found = true
				break
			}
		}
		if !found {
			a.sessions = append(a.sessions, *sess)
		}

		a.sidebar.SetSessions(a.sessions)
		a.sidebar.ActiveID = sess.ID
		for i, s := range a.sessions {
			if s.ID == sess.ID {
				a.sidebar.Cursor = i
				break
			}
		}
		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		a.ready = true
		a.chatView.Focus()
		a.helpBar.FocusMode = "✎ Input"
		return a.connectSSECmd()

	case "message_sent":
		// State already set in sendMessageCmd, just clear input on success
		return nil

	case "sessions_listed":
		sessions := msg.Data.([]types.SessionInfo)
		a.sessions = sessions
		a.sidebar.SetSessions(sessions)
		return nil

	case "session_loaded":
		sess := msg.Data.(*types.SessionInfo)
		a.activeSession = sess
		a.sidebar.ActiveID = sess.ID
		a.statusBar.SessionTitle = sess.Title
		a.statusBar.SessionState = sess.State
		a.chatView.Focus()
		a.helpBar.FocusMode = "✎ Input"
		return a.connectSSECmd()

	case "messages_loaded":
		msgs := msg.Data.([]types.Message)
		a.msgs = msgs
		a.chatView.MessageView.SetMessages(msgs)
		return nil

	case "approval_done":
		a.state = StateProcessing
		a.statusBar.SessionState = "Processing"
		a.chatView.Processing = true
		return nil

	case "pause_done", "resume_done", "cancel_done":
		return a.refreshSessionCmd()

	case "trust_set", "policy_set":
		a.toast.Show("设置已更新", false)
		return nil
	}

	return nil
}

func (a *App) handleSSEEvent(msg messages.SSEEvent) tea.Cmd {
	Log("[SSE] Event: %s", msg.Type)

	switch msg.Type {
	case "thinking":
		if text, ok := msg.Data["message"].(string); ok {
			a.chatView.MessageView.AddThinking(text)
		}

	case "tool_call_request":
		toolName, _ := msg.Data["tool_name"].(string)
		params, _ := msg.Data["params"]
		paramsStr := ""
		if params != nil {
			paramsStr = fmt.Sprintf("%v", params)
		}

		card := components.ToolCardData{
			ToolName: toolName,
			Params:   paramsStr,
		}
		a.chatView.MessageView.AddToolCard(card)

	case "tool_result":
		toolName, _ := msg.Data["tool_name"].(string)
		success, _ := msg.Data["success"].(bool)
		summary, _ := msg.Data["summary"].(string)
		a.chatView.MessageView.UpdateToolCard(toolName, success, summary, "")

	case "message_complete":
		content, _ := msg.Data["full_text"].(string)
		if content != "" {
			assistantMsg := types.Message{
				Role:    "assistant",
				Content: content,
			}
			a.chatView.MessageView.AddMessage(assistantMsg)
		}

	case "approval_required":
		approvalID, _ := msg.Data["approval_id"].(string)
		opType, _ := msg.Data["operation_type"].(string)
		riskLevel, _ := msg.Data["risk_level"].(string)
		summary, _ := msg.Data["summary"].(string)
		diff, _ := msg.Data["diff"].(string)
		commandPreview, _ := msg.Data["command_preview"].(string)

		req := &types.ApprovalRequest{
			ApprovalID:     approvalID,
			OperationType:  opType,
			RiskLevel:      riskLevel,
			Summary:        summary,
			Diff:           diff,
			CommandPreview: commandPreview,
		}
		a.approvalModal.Show(req, a.width, a.height)
		a.state = StateAwaitingApproval
		a.statusBar.SessionState = "AwaitingApproval"
		a.helpBar.SetApprovalMode()

	case "approval_auto":
		summary, _ := msg.Data["summary"].(string)
		policyLevel, _ := msg.Data["policy_level"].(string)
		notice := fmt.Sprintf("已根据信任策略（%s）自动批准：%s", policyLevel, summary)
		a.chatView.MessageView.AddSystemNotice(notice)

	case "token_usage":
		inputTokens, _ := msg.Data["input_tokens"].(float64)
		outputTokens, _ := msg.Data["output_tokens"].(float64)
		total := int(inputTokens) + int(outputTokens)
		a.statusBar.TokenUsage = fmt.Sprintf("%d tok", total)

	case "session_state_change":
		oldState, _ := msg.Data["old_state"].(string)
		newState, _ := msg.Data["new_state"].(string)
		reason, _ := msg.Data["reason"].(string)
		a.statusBar.SessionState = newState

		switch reason {
		case "completed":
			a.state = StateReady
			a.chatView.Processing = false
			a.chatView.MessageView.AddSystemNotice("任务完成")
		case "cancelled":
			a.state = StateReady
			a.chatView.Processing = false
			a.chatView.MessageView.AddSystemNotice("操作已取消")
		case "tool_limit_reached":
			a.state = StateReady
			a.chatView.Processing = false
			a.chatView.MessageView.AddSystemNotice("已达到工具调用上限，输入新消息继续")
		case "error":
			a.state = StateReady
			a.chatView.Processing = false
			a.chatView.MessageView.AddSystemNotice("发生错误，请重试")
		}

		_ = oldState
		_ = newState

	case "error":
		errMsg, _ := msg.Data["message"].(string)
		a.toast.Show(errMsg, true)
	}

	return a.readSSEEvent()
}

func (a *App) handleApprovalDecision(msg messages.ApprovalDecision) tea.Cmd {
	pending := a.approvalModal.Request
	a.approvalModal.Hide()
	a.helpBar.SetDefaultMode()

	if msg.Approved {
		a.toast.Show("已批准，正在执行...", false)
		if pending != nil {
			return a.sendApprovalCmd(pending.ApprovalID, true)
		}
	} else {
		a.toast.Show("已拒绝", false)
		if pending != nil {
			return a.sendApprovalCmd(pending.ApprovalID, false)
		}
	}
	return nil
}

func (a *App) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if a.state == StateAwaitingApproval {
		switch key {
		case "y", "Y":
			return func() tea.Msg { return messages.ApprovalDecision{Approved: true} }
		case "n", "N", "esc":
			return func() tea.Msg { return messages.ApprovalDecision{Approved: false} }
		case "d", "D":
			a.approvalModal.Update(msg)
			return nil
		}
		return nil
	}

	switch key {
	case "ctrl+c":
		return a.cancelCmd()

	case "ctrl+q":
		a.state = StateQuitting
		return tea.Quit

	case "ctrl+s":
		a.showSidebar = !a.showSidebar
		return nil

	case "ctrl+n":
		return a.newSessionCmd()

	case "ctrl+p":
		if a.activeSession != nil {
			if a.activeSession.State == "Processing" {
				return a.pauseCmd()
			} else if a.activeSession.State == "Paused" {
				return a.resumeCmd()
			}
		}

	case "ctrl+l":
		return tea.ClearScreen

	case "enter":
		if a.chatView.InputArea.Focused() && a.state == StateReady {
			content := strings.TrimSpace(a.chatView.InputArea.Value())
			if content != "" {
				return a.sendMessageCmd(content)
			}
		} else if a.showSidebar {
			selected := a.sidebar.SelectedSession()
			if selected != nil && selected.ID != a.sidebar.ActiveID {
				return a.switchSessionCmd(selected.ID)
			}
		}

	case "esc":
		if a.chatView.InputArea.Focused() {
			a.chatView.Blur()
			a.helpBar.FocusMode = "☰ Navigate"
		}

	case "up":
		if a.showSidebar && !a.chatView.InputArea.Focused() {
			a.sidebar.CursorUp()
		}

	case "down":
		if a.showSidebar && !a.chatView.InputArea.Focused() {
			a.sidebar.CursorDown()
		}
	}

	return nil
}

func (a *App) connectSSECmd() tea.Cmd {
	return func() tea.Msg {
		a.sseClient = NewSSEClient()
		sseURL := a.apiClient.SSEEndpoint(a.activeSession.ID)
		if err := a.sseClient.Connect(sseURL); err != nil {
			return messages.APIResponse{Kind: "sse_error", Err: err}
		}
		return a.readSSEEvent()()
	}
}

func (a *App) readSSEEvent() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-a.sseClient.Events():
			if !ok {
				return nil
			}
			return messages.SSEEvent{
				Type: event.Type,
				Data: event.Data,
			}
		case err, ok := <-a.sseClient.Errors():
			if ok {
				return messages.APIResponse{Kind: "sse_error", Err: err}
			}
			return nil
		}
	}
}

func (a *App) sendMessageCmd(content string) tea.Cmd {
	// Add user message to chat view immediately
	a.chatView.MessageView.AddMessage(types.Message{
		Role:    "user",
		Content: content,
	})
	a.chatView.InputArea.Reset()
	a.state = StateProcessing
	a.statusBar.SessionState = "Processing"
	a.chatView.Processing = true

	return func() tea.Msg {
		err := a.apiClient.SendMessage(a.activeSession.ID, content)
		if err != nil {
			return messages.APIResponse{Kind: "message_sent", Err: err}
		}
		return messages.APIResponse{Kind: "message_sent"}
	}
}

func (a *App) sendApprovalCmd(approvalID string, approved bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if approved {
			err = a.apiClient.Approve(a.activeSession.ID, approvalID)
		} else {
			err = a.apiClient.Reject(a.activeSession.ID, approvalID)
		}
		if err != nil {
			return messages.APIResponse{Kind: "approval_done", Err: err}
		}
		return messages.APIResponse{Kind: "approval_done"}
	}
}

func (a *App) cancelCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.apiClient.Cancel(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "cancel_done", Err: err}
		}
		return messages.APIResponse{Kind: "cancel_done"}
	}
}

func (a *App) pauseCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.apiClient.Pause(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "pause_done", Err: err}
		}
		return messages.APIResponse{Kind: "pause_done"}
	}
}

func (a *App) resumeCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.apiClient.Resume(a.activeSession.ID)
		if err != nil {
			return messages.APIResponse{Kind: "resume_done", Err: err}
		}
		return messages.APIResponse{Kind: "resume_done"}
	}
}

func (a *App) switchSessionCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		if a.sseClient != nil {
			a.sseClient.Disconnect()
		}

		sess, err := a.apiClient.GetSession(sessionID)
		if err != nil {
			return messages.APIResponse{Kind: "session_loaded", Err: err}
		}

		return messages.APIResponse{Kind: "session_loaded", Data: sess}
	}
}

func (a *App) newSessionCmd() tea.Cmd {
	return func() tea.Msg {
		dirName := a.workingDir
		if idx := strings.LastIndex(dirName, "/"); idx >= 0 {
			dirName = dirName[idx+1:]
		}
		if idx := strings.LastIndex(dirName, "\\"); idx >= 0 {
			dirName = dirName[idx+1:]
		}

		sess, err := a.apiClient.CreateSession(a.workingDir, dirName)
		if err != nil {
			return messages.APIResponse{Kind: "init_session", Err: err}
		}

		return messages.APIResponse{Kind: "init_session", Data: sess}
	}
}

func (a *App) refreshSessionCmd() tea.Cmd {
	return func() tea.Msg {
		sessions, err := a.apiClient.ListSessions()
		if err != nil {
			return messages.APIResponse{Kind: "sessions_listed", Err: err}
		}

		if a.activeSession != nil {
			sess, err := a.apiClient.GetSession(a.activeSession.ID)
			if err == nil {
				a.activeSession = sess
				a.statusBar.SessionState = sess.State
			}
		}

		return messages.APIResponse{Kind: "sessions_listed", Data: sessions}
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
