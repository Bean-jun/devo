package tui

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/term"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/renderer"
	"devo/internal/interfaces/tui/types"
)

type Model struct {
	viewport              viewport.Model
	textarea              textarea.Model
	renderer              *renderer.MsgRenderer
	messages              []types.Message
	sessions              []types.SessionInfo
	statusBar             components.StatusBar
	overlay               overlays.OverlayStack
	toast                 components.Toast
	cmdSheet              overlays.CommandSheet
	sessPicker            overlays.SessionPicker
	approval              overlays.ApprovalModal
	helpPanel             overlays.HelpPanel
	skillsPanel           overlays.SkillsPanel
	mcpPanel              overlays.MCPPanel
	memoryPanel           overlays.MemoryPanel
	wsPanel               overlays.WorkspacePanel
	renameModal           overlays.RenameModal
	rollback              overlays.RollbackPicker
	newSessModal          overlays.NewSessionModal
	statusPanel           overlays.StatusPanel
	versionPanel          overlays.VersionPanel
	backgroundPanel       overlays.BackgroundPanel
	dashboardPanel        overlays.DashboardPanel
	settingsPanel         overlays.SettingsPanel
	reasoningPicker       overlays.ReasoningPicker
	apiClient             *api.Client
	sseClient             *api.SSEClient
	baseURL               string
	version               string
	updateInfo            *api.UpdateCheckResult
	workingDir            string
	width                 int
	height                int
	ready                 bool
	initialized           bool
	activeSessionID       string
	loading               map[overlays.OverlayType]bool
	enableReasoning       bool
	reasoningEffort       string
	rollbackTargetContent string
	rollbackTargetIndex   int

	keyboardEnhancements int
	pasteBuffer          string
	pasteFolded          bool
	pastedImages         []string
	inputHistory         []string
	historyIndex         int
	historyDraft         string
	lastEscAt            time.Time
	lastTextareaValue    string
}

func NewModel(baseURL string, version string) Model {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	ta := textarea.New()
	ta.Placeholder = "输入消息..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Prompt = ""
	ta.SetHeight(3)
	taStyles := ta.Styles()
	taStyles.Focused.CursorLine = lipgloss.NewStyle()
	taStyles.Focused.Text = lipgloss.NewStyle().Foreground(components.ColorText())
	taStyles.Focused.Placeholder = lipgloss.NewStyle().Foreground(components.ColorMuted())
	taStyles.Blurred = taStyles.Focused
	ta.SetStyles(taStyles)
	ta.Focus()

	m := Model{
		viewport:        vp,
		textarea:        ta,
		renderer:        renderer.New(80),
		statusBar:       components.NewStatusBar(),
		cmdSheet:        overlays.NewCommandSheet(),
		helpPanel:       overlays.HelpPanel{},
		skillsPanel:     overlays.NewSkillsPanel(),
		mcpPanel:        overlays.NewMCPPanel(),
		memoryPanel:     overlays.NewMemoryPanel(),
		wsPanel:         overlays.NewWorkspacePanel(),
		rollback:        overlays.NewRollbackPicker(nil),
		newSessModal:    overlays.NewSessionModal{},
		backgroundPanel: overlays.NewBackgroundPanel(),
		dashboardPanel:  overlays.NewDashboardPanel(),
		settingsPanel:   overlays.NewSettingsPanel(),
		apiClient:       api.NewClient(baseURL),
		sseClient:       api.NewSSEClient(baseURL),
		baseURL:         baseURL,
		version:         version,
		workingDir:      getWorkingDir(),
		width:           80,
		height:          24,
		ready:           true,
		loading:         make(map[overlays.OverlayType]bool),
	}

	m.statusBar.ServerPort = extractPort(baseURL)
	m.reasoningEffort = "medium"
	m.historyIndex = -1

	return m
}

func (m *Model) connectSSE(sessionID string) tea.Cmd {
	m.sseClient.Disconnect()
	if err := m.sseClient.Connect(sessionID); err != nil {
		return nil
	}
	return m.listenSSE()
}

func (m *Model) listenSSE() tea.Cmd {
	return func() tea.Msg {
		select {
		case evt := <-m.sseClient.Events():
			return sseEventMsg{event: evt}
		case err := <-m.sseClient.Errors():
			return sseEventMsg{err: err}
		}
	}
}

type sseEventMsg struct {
	event types.SSEEvent
	err   error
}

func (m *Model) initFromAPI() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			sessions, err := m.apiClient.ListSessions()
			if err != nil {
				return apiResponseMsg{kind: "sessions_error", err: err}
			}
			return apiResponseMsg{kind: "sessions_loaded", data: sessions}
		},
		m.fetchGlobalConfigFromAPI(),
		m.checkUpdateFromAPI(),
	)
}

func (m *Model) setLoading(t overlays.OverlayType, v bool) {
	if m.loading == nil {
		m.loading = make(map[overlays.OverlayType]bool)
	}
	m.loading[t] = v
}

func (m *Model) isLoading(t overlays.OverlayType) bool {
	if m.loading == nil {
		return false
	}
	return m.loading[t]
}

func (m *Model) isEditing() bool {
	switch m.overlay.Current {
	case overlays.OverlayRename:
		return true
	case overlays.OverlayCommand:
		return true
	case overlays.OverlaySettings:
		return m.settingsPanel.Editing
	case overlays.OverlaySkills:
		return m.skillsPanel.Editing
	case overlays.OverlayMCP:
		return m.mcpPanel.Editing
	case overlays.OverlayMemory:
		return m.memoryPanel.Editing
	}
	return false
}

func (m *Model) appendEditChar(s string) {
	switch m.overlay.Current {
	case overlays.OverlayRename:
		m.renameModal.NewName += s
	case overlays.OverlayCommand:
		m.cmdSheet.Filter += s
		m.cmdSheet.Selected = 0
		m.cmdSheet.BuildFlat()
	case overlays.OverlaySettings:
		m.settingsPanel.EditBuffer += s
	case overlays.OverlaySkills:
		m.skillsPanel.EditBuffer += s
	case overlays.OverlayMCP:
		m.mcpPanel.EditBuffer += s
	case overlays.OverlayMemory:
		m.memoryPanel.EditBuffer += s
	}
}

func (m *Model) applySessionsData(sessions []types.SessionInfo) {
	m.sessions = sessions
	m.sessPicker.Sessions = sessions
}

func (m *Model) syncYoloFromSession(sessionID string) {
	for _, s := range m.sessions {
		if s.ID == sessionID {
			m.statusBar.Yolo = s.TrustLevel == types.TrustLevelElevated
			return
		}
	}
}

func (m *Model) applyMessagesData(msgs []types.Message) {
	if len(msgs) > 0 {
		existing := make(map[string]bool)
		for _, msg := range m.messages {
			existing[msg.ID] = true
		}
		for _, msg := range msgs {
			if !existing[msg.ID] {
				m.messages = append(m.messages, msg)
			}
		}
		m.refreshViewportToBottom()
	}
}

func (m *Model) applySkillsData(skills []api.SkillInfo) {
	var entries []overlays.SkillEntry
	for _, s := range skills {
		entries = append(entries, overlays.SkillEntry{
			Name:        s.Name,
			Description: s.Description,
			Enabled:     s.Enabled,
		})
	}
	m.skillsPanel.Skills = entries
}

func (m *Model) applyMCPServersData(servers []api.MCPServerInfo) {
	var entries []overlays.MCPEntry
	for _, s := range servers {
		entries = append(entries, overlays.MCPEntry{
			Name:   s.ServerID,
			URL:    s.Endpoint,
			Status: s.Status,
		})
	}
	m.mcpPanel.Servers = entries
}

func (m *Model) applyMemoriesData(memories []api.MemoryItem) {
	var entries []overlays.MemoryEntry
	for _, mem := range memories {
		entries = append(entries, overlays.MemoryEntry{
			ID:      mem.ID,
			Type:    mem.Type,
			Key:     mem.Key,
			Content: mem.Content,
		})
	}
	m.memoryPanel.Memories = entries
}

func (m *Model) applyWorkspacesData(workspaces []api.WorkspaceInfo) {
	var entries []overlays.WorkspaceEntry
	for _, ws := range workspaces {
		entries = append(entries, overlays.WorkspaceEntry{
			Name:   ws.Name,
			Path:   ws.Path,
			Active: false,
		})
	}
	m.wsPanel.Workspaces = entries
}

type apiResponseMsg struct {
	kind      string
	data      interface{}
	err       error
	sessionID string
	title     string
	path      string
	key       string
	id        string
}

func (m *Model) overlayPanelWidth() int {
	if m.width < 60 {
		return m.width - 4
	}
	maxW := m.width - 8
	if maxW > 100 {
		maxW = 100
	}
	return maxW
}

func (m *Model) refreshViewport() {
	if m.renderer == nil {
		return
	}
	content := m.renderer.Render(m.messages)
	m.viewport.SetContent(content)
}

func (m *Model) refreshViewportToBottom() {
	if m.renderer == nil {
		return
	}
	content := m.renderer.Render(m.messages)
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *Model) buildFooterText() string {
	wd := m.workingDir
	contextTokens := 0
	var tokenUsage types.TokenUsage
	if m.activeSessionID != "" {
		for _, s := range m.sessions {
			if s.ID == m.activeSessionID {
				if s.WorkingDirectory != "" {
					wd = s.WorkingDirectory
				}
				contextTokens = s.CurrentContextTokens
				tokenUsage = s.TokenUsage
				break
			}
		}
	}
	c := formatTokenCount(contextTokens)
	total := tokenUsage.Input + tokenUsage.Output
	ti := formatTokenCount(tokenUsage.Input)
	to := formatTokenCount(tokenUsage.Output)
	return fmt.Sprintf("Context %s  ·  Tokens %s (↑%s ↓%s)  ·  %s",
		c, formatTokenCount(total), ti, to, wd) + m.buildImageHint()
}

func (m *Model) buildImageHint() string {
	if len(m.pastedImages) == 0 {
		return ""
	}
	if len(m.pastedImages) == 1 {
		return "  ·  🖼 1 张图片"
	}
	return fmt.Sprintf("  ·  🖼 %d 张图片", len(m.pastedImages))
}

func formatTokenCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (m *Model) applyTermSize() {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return
	}
	m.applySize(w, h)
}

func (m *Model) applySize(w, h int) {
	if w == m.width && h == m.height {
		return
	}
	m.width = w
	m.height = h
	m.statusBar.Width = m.width

	headerH := 3
	footerH := 6
	vpHeight := m.height - headerH - footerH
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(vpHeight)
	m.textarea.SetWidth(m.width - 4)
	m.renderer.SetWidth(m.width)
	m.toast.Width = m.width

	pw := m.overlayPanelWidth()
	m.cmdSheet.Width = pw
	m.sessPicker.Width = pw
	m.approval.Width = pw
	m.approval.Height = m.height
	m.helpPanel.Width = pw
	m.helpPanel.Height = m.height
	m.skillsPanel.Width = pw
	m.mcpPanel.Width = pw
	m.memoryPanel.Width = pw
	m.wsPanel.Width = pw
	m.renameModal.Width = pw
	m.rollback.Width = pw
	m.newSessModal.Width = pw
	m.statusPanel.Width = pw
	m.versionPanel.Width = pw
	m.backgroundPanel.Width = pw
	m.dashboardPanel.Width = pw
	m.dashboardPanel.Height = m.height
	m.settingsPanel.Width = pw
	m.settingsPanel.Height = m.height

	m.refreshViewport()
}

func (m *Model) findUserMessageYOffsets() []int {
	if m.renderer == nil {
		return nil
	}
	return m.renderer.FindUserMessageYOffsets(m.messages)
}

func (m *Model) jumpToPrevUserMessage() {
	offsets := m.findUserMessageYOffsets()
	if len(offsets) == 0 {
		return
	}

	currentY := m.viewport.YOffset()

	for i := len(offsets) - 1; i >= 0; i-- {
		if offsets[i] < currentY {
			m.viewport.SetYOffset(offsets[i])
			return
		}
	}
	m.viewport.SetYOffset(offsets[0])
}

func (m *Model) jumpToNextUserMessage() {
	offsets := m.findUserMessageYOffsets()
	if len(offsets) == 0 {
		return
	}

	currentY := m.viewport.YOffset()

	for _, offset := range offsets {
		if offset > currentY {
			m.viewport.SetYOffset(offset)
			return
		}
	}
	m.viewport.SetYOffset(offsets[len(offsets)-1])
}

func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func extractPort(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Port()
}

func defaultSessionTitle() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
}
