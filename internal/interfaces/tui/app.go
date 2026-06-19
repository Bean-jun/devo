package tui

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
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

	statusBar      components.StatusBar
	sidebar        components.SessionSidebar
	chatView       components.ChatView
	approvalModal  components.ApprovalModal
	commandPalette components.CommandPalette
	toast          components.Toast

	state              AppState
	showSidebar        bool
	showCommandPalette bool
	width              int
	height             int
	workingDir         string
	ready              bool
	apiBaseURL         string

	initStatus string
	initErr    error
	tickCount  int
}

func NewAppWithURL(baseURL string) (*App, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	statusBar := components.NewStatusBar()

	if u, err := url.Parse(baseURL); err == nil {
		if port, err := strconv.Atoi(u.Port()); err == nil {
			statusBar.ServerPort = port
		}
	}

	return &App{
		apiBaseURL:     baseURL,
		showSidebar:    true,
		workingDir:     wd,
		width:          80,
		height:         24,
		state:          StateReady,
		initStatus:     "Connecting to server...",
		statusBar:      statusBar,
		sidebar:        components.NewSessionSidebar(),
		chatView:       components.NewChatView(),
		approvalModal:  components.NewApprovalModal(),
		commandPalette: components.NewCommandPalette(),
		toast:          components.NewToast(),
	}, nil
}

func (a *App) Init() tea.Cmd {
	a.apiClient = NewAPIClient(a.apiBaseURL)
	Log("[TUI] Initializing, baseURL=%s, workingDir=%s", a.apiBaseURL, a.workingDir)
	a.initStatus = "Loading sessions..."
	return tea.Batch(
		a.refreshSessionCmd(),
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

		if a.showCommandPalette {
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
		switch msg.Kind {
		case "sessions_listed":
			if msg.Err != nil {
				a.initErr = msg.Err
				a.initStatus = fmt.Sprintf("Failed: %v", msg.Err)
				Log("[TUI] sessions_listed failed: %v", msg.Err)
				return a, nil
			}
			sessions := msg.Data.([]types.SessionInfo)
			a.sessions = sessions
			a.sidebar.SetSessions(sessions)
			if len(sessions) > 0 {
				Log("[TUI] Found %d existing sessions, loading most recent", len(sessions))
				a.initStatus = "Loading session..."
				return a, a.switchSessionCmd(sessions[0].ID)
			}
			Log("[TUI] No existing sessions, creating new one")
			a.initStatus = "Creating session..."
			return a, a.initSessionCmd()

		case "session_loaded":
			if msg.Err != nil {
				Log("[TUI] session_loaded failed: %v, falling back to new session", msg.Err)
				a.initStatus = "Creating session..."
				return a, a.initSessionCmd()
			}
			sess := msg.Data.(*types.SessionInfo)
			a.activeSession = sess
			a.sidebar.ActiveID = sess.ID
			a.statusBar.SessionTitle = sess.Title
			a.statusBar.SessionState = sess.State
			if sess.TokenUsage.Total > 0 {
				a.statusBar.TokenUsage = fmt.Sprintf("%d tok (↑%d ↓%d)",
					sess.TokenUsage.Total, sess.TokenUsage.Input, sess.TokenUsage.Output)
			}
			a.ready = true
			a.tickCount = 0
			a.layout()
			a.focusInput()
			Log("[TUI] Session loaded: id=%s, title=%s", sess.ID, sess.Title)
			return a, tea.Batch(a.connectSSECmd(), a.loadMessagesCmd(sess.ID))

		case "init_session":
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
			a.layout()
			a.focusInput()
			Log("[TUI] Session created: id=%s, title=%s", sess.ID, sess.Title)
			return a, tea.Batch(a.connectSSECmd(), a.refreshSessionCmd())
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

	sidebarView := ""
	if a.showSidebar {
		sidebarView = a.sidebar.View()
	}

	if a.showCommandPalette {
		chatPaletteWidth := a.width
		if a.showSidebar {
			chatPaletteWidth = a.width - 30 - 1
		}
		a.commandPalette.Width = chatPaletteWidth
		a.chatView.CommandPaletteView = a.commandPalette.View()
	} else {
		a.chatView.CommandPaletteView = ""
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

	var view string
	if toastView != "" {
		view = lipgloss.JoinVertical(lipgloss.Left,
			statusBarView,
			mainContent,
			toastView,
		)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left,
			statusBarView,
			mainContent,
		)
	}

	if a.approvalModal.Visible {
		modalView := a.approvalModal.View()
		view = lipgloss.JoinVertical(lipgloss.Left,
			statusBarView,
			modalView,
		)
	}

	// Log("[VIEW] main screen: w=%d h=%d ready=%v chatW=%d chatH=%d msgs=%d",
	// 	a.width, a.height, a.ready,
	// 	a.chatView.Width, a.chatView.Height,
	// 	len(a.chatView.MessageView.Messages))

	return view
}

func (a *App) focusInput() {
	a.chatView.Focus()
	a.statusBar.Mode = ""
}

func (a *App) layout() {
	a.statusBar.Width = a.width
	a.toast.Width = a.width

	sidebarWidth := 30
	chatWidth := a.width
	if a.showSidebar {
		chatWidth = a.width - sidebarWidth - 1
	}
	if chatWidth < 20 {
		chatWidth = 20
	}

	topHeight := 3
	chatHeight := a.height - topHeight
	if chatHeight < 10 {
		chatHeight = 10
	}

	a.sidebar.Width = sidebarWidth
	a.sidebar.Height = chatHeight
	a.chatView.SetSize(chatWidth, chatHeight)
}
