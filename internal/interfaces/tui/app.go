package tui

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
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

	statusBar      components.StatusBar
	chatView       components.ChatView
	approvalModal  components.ApprovalModal
	commandPalette components.CommandPalette
	rollbackPicker components.RollbackPicker
	sessionPicker  components.SessionPicker
	toast          components.Toast

	state              AppState
	showCommandPalette bool
	showRollbackPicker bool
	showSessionPicker  bool
	width              int
	height             int
	workingDir         string
	ready              bool
	apiBaseURL         string

	inputHistory   []string
	historyIndex   int
	pastedFullText string
	pasteLabel     string
	lastKeyTime    time.Time
	isPasting      bool
	pasteBuffer    string
	preKeyValue    string
	prePreKeyValue string

	initStatus string
	initErr    error
	tickCount  int
}

func NewAppWithURL(baseURL string, version string) (*App, error) {
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

	chatView := components.NewChatView()
	chatView.InputArea.Version = "v" + version

	return &App{
		apiBaseURL:     baseURL,
		workingDir:     wd,
		width:          80,
		height:         24,
		state:          StateReady,
		initStatus:     "Connecting to server...",
		statusBar:      statusBar,
		chatView:       chatView,
		approvalModal:  components.NewApprovalModal(),
		commandPalette: components.NewCommandPalette(),
		rollbackPicker: components.NewRollbackPicker(),
		sessionPicker:  components.NewSessionPicker(),
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
		if a.isPasting && time.Since(a.lastKeyTime) > 120*time.Millisecond {
			a.finishPaste()
		}
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
		wasModalShowing := a.showCommandPalette || a.showRollbackPicker || a.showSessionPicker
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

		if a.showRollbackPicker {
			return a, tea.Batch(cmds...)
		}

		if a.showSessionPicker {
			return a, tea.Batch(cmds...)
		}

		if wasModalShowing {
			return a, tea.Batch(cmds...)
		}

		if msg.String() != "esc" {
			now := time.Now()
			elapsed := now.Sub(a.lastKeyTime)
			a.lastKeyTime = now

			if a.isPasting {
				if elapsed > 120*time.Millisecond {
					a.finishPaste()
				} else {
					a.appendToPasteBuffer(msg)
					return a, tea.Batch(cmds...)
				}
			} else if elapsed < 40*time.Millisecond && a.isPasteableKey(msg) {
				a.isPasting = true
				a.pasteBuffer = a.chatView.InputArea.Value()
				a.chatView.InputArea.SetValue(a.preKeyValue)
				a.appendToPasteBuffer(msg)
				return a, tea.Batch(cmds...)
			}

			a.prePreKeyValue = a.preKeyValue
			a.preKeyValue = a.chatView.InputArea.Value()
			a.chatView, cmd = a.chatView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return a, tea.Batch(cmds...)
	}

	a.layout()

	var cmd tea.Cmd
	a.chatView, cmd = a.chatView.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	a.compressPasteIfNeeded()

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
			sess.NormalizeState()
			a.activeSession = sess
			a.statusBar.SessionTitle = sess.Title
			a.statusBar.SessionState = sess.State
			a.chatView.InputArea.TokenUsage = fmt.Sprintf("Tokens %d (↑%d ↓%d)",
				sess.TokenUsage.Total, sess.TokenUsage.Input, sess.TokenUsage.Output)
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
			sess.NormalizeState()
			a.activeSession = sess
			a.sessions = []types.SessionInfo{*sess}
			a.statusBar.SessionTitle = sess.Title
			a.statusBar.SessionState = sess.State
			a.chatView.InputArea.TokenUsage = fmt.Sprintf("Tokens %d (↑%d ↓%d)",
				sess.TokenUsage.Total, sess.TokenUsage.Input, sess.TokenUsage.Output)
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

	if a.showCommandPalette {
		a.commandPalette.Width = a.width
		a.chatView.CommandPaletteView = a.commandPalette.View()
	} else if a.showRollbackPicker {
		a.rollbackPicker.Width = a.width
		a.chatView.CommandPaletteView = a.rollbackPicker.View()
	} else {
		a.chatView.CommandPaletteView = ""
	}

	chatView := a.chatView.View()

	var mainContent string
	if a.showSessionPicker {
		a.sessionPicker.Width = a.width
		a.sessionPicker.Height = a.height
		mainContent = lipgloss.Place(
			a.width,
			a.height-3,
			lipgloss.Center,
			lipgloss.Center,
			a.sessionPicker.View(),
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

func (a *App) pushInputHistory(text string) {
	if text == "" {
		return
	}
	if len(a.inputHistory) > 0 && a.inputHistory[len(a.inputHistory)-1] == text {
		return
	}
	a.inputHistory = append(a.inputHistory, text)
	a.historyIndex = len(a.inputHistory)
}

func (a *App) historyPrev() {
	if len(a.inputHistory) == 0 {
		return
	}
	if a.historyIndex > 0 {
		a.historyIndex--
		a.chatView.InputArea.SetValue(a.inputHistory[a.historyIndex])
	}
}

func (a *App) historyNext() {
	if len(a.inputHistory) == 0 {
		return
	}
	if a.historyIndex < len(a.inputHistory)-1 {
		a.historyIndex++
		a.chatView.InputArea.SetValue(a.inputHistory[a.historyIndex])
	} else {
		a.historyIndex = len(a.inputHistory)
		a.chatView.InputArea.Reset()
	}
}

func (a *App) isPasteableKey(msg tea.KeyMsg) bool {
	key := msg.String()
	if key == "enter" {
		return true
	}
	return len(msg.Runes) == 1 && msg.Runes[0] >= ' '
}

func (a *App) appendToPasteBuffer(msg tea.KeyMsg) {
	key := msg.String()
	if key == "enter" {
		a.pasteBuffer += "\n"
	} else if len(msg.Runes) == 1 {
		a.pasteBuffer += string(msg.Runes[0])
	}
}

func (a *App) finishPaste() {
	a.isPasting = false
	val := a.pasteBuffer
	a.pasteBuffer = ""

	lines := strings.Split(val, "\n")
	if len(val) > 200 || len(lines) > 3 {
		a.pastedFullText = val
		prefix := a.preKeyValue
		if len(prefix) > 60 {
			prefix = prefix[:60] + "..."
		}
		if prefix != "" {
			prefix += " "
		}
		a.pasteLabel = prefix + fmt.Sprintf("[已粘贴 %d 行文本]", len(lines))
		a.chatView.InputArea.SetValue(a.pasteLabel)
	} else {
		a.chatView.InputArea.SetValue(val)
	}
}

func (a *App) compressPasteIfNeeded() {
	// During rapid input (paste), skip compression to avoid flashing
	if time.Since(a.lastKeyTime) < 100*time.Millisecond {
		return
	}

	val := a.chatView.InputArea.Value()
	if val == "" {
		return
	}

	if a.pasteLabel != "" && val == a.pasteLabel {
		return
	}

	if a.pasteLabel != "" && val != a.pasteLabel {
		a.pastedFullText = ""
		a.pasteLabel = ""
		return
	}

	lines := strings.Split(val, "\n")
	if len(val) > 200 || len(lines) > 3 {
		a.pastedFullText = val
		// Keep first line of existing text as visual context
		firstLine := lines[0]
		if len(firstLine) > 60 {
			firstLine = firstLine[:60] + "..."
		}
		a.pasteLabel = firstLine + fmt.Sprintf(" [已粘贴 %d 行文本]", len(lines))
		a.chatView.InputArea.SetValue(a.pasteLabel)
	}
}

func (a *App) layout() {
	a.statusBar.Width = a.width
	a.toast.Width = a.width

	chatWidth := a.width
	if chatWidth < 20 {
		chatWidth = 20
	}

	topHeight := 3
	chatHeight := a.height - topHeight
	if chatHeight < 10 {
		chatHeight = 10
	}

	a.chatView.SetSize(chatWidth, chatHeight)
}
