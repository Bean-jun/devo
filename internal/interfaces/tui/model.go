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
	viewport        viewport.Model
	textarea        textarea.Model
	renderer        *renderer.MsgRenderer
	messages        []types.Message
	sessions        []types.SessionInfo
	statusBar       components.StatusBar
	overlay         overlays.OverlayStack
	toast           components.Toast
	cmdSheet        overlays.CommandSheet
	sessPicker      overlays.SessionPicker
	approval        overlays.ApprovalModal
	helpPanel       overlays.HelpPanel
	skillsPanel     overlays.SkillsPanel
	mcpPanel        overlays.MCPPanel
	memoryPanel     overlays.MemoryPanel
	wsPanel         overlays.WorkspacePanel
	renameModal     overlays.RenameModal
	rollback        overlays.RollbackPicker
	newSessModal    overlays.NewSessionModal
	statusPanel     overlays.StatusPanel
	versionPanel    overlays.VersionPanel
	backgroundPanel overlays.BackgroundPanel
	dashboardPanel  overlays.DashboardPanel
	settingsPanel   overlays.SettingsPanel
	reasoningPicker overlays.ReasoningPicker
	apiClient       *api.Client
	sseClient       *api.SSEClient
	baseURL         string
	version         string
	workingDir      string
	width           int
	height          int
	ready           bool
	initialized     bool
	activeSessionID string
	loading         map[overlays.OverlayType]bool
	enableReasoning bool
	reasoningEffort string
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

func (m *Model) fetchSessionsFromAPI() tea.Cmd {
	m.setLoading(overlays.OverlaySession, true)
	return func() tea.Msg {
		sessions, err := m.apiClient.ListSessions()
		if err != nil {
			return apiResponseMsg{kind: "sessions_error", err: err}
		}
		return apiResponseMsg{kind: "sessions_loaded", data: sessions}
	}
}

func (m *Model) fetchMessagesFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := m.apiClient.GetMessages(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "messages_error", err: err}
		}
		return apiResponseMsg{kind: "messages_loaded", data: msgs, sessionID: sessionID}
	}
}

func (m *Model) fetchSkillsFromAPI() tea.Cmd {
	m.setLoading(overlays.OverlaySkills, true)
	return func() tea.Msg {
		skills, err := m.apiClient.GetSkills()
		if err != nil {
			return apiResponseMsg{kind: "skills_error", err: err}
		}
		return apiResponseMsg{kind: "skills_loaded", data: skills}
	}
}

func (m *Model) fetchMCPServersFromAPI() tea.Cmd {
	m.setLoading(overlays.OverlayMCP, true)
	return func() tea.Msg {
		servers, err := m.apiClient.GetMCPServers()
		if err != nil {
			return apiResponseMsg{kind: "mcp_error", err: err}
		}
		return apiResponseMsg{kind: "mcp_loaded", data: servers}
	}
}

func (m *Model) fetchMemoriesFromAPI(sessionID string) tea.Cmd {
	m.setLoading(overlays.OverlayMemory, true)
	return func() tea.Msg {
		userMemories, userErr := m.apiClient.GetMemories(sessionID, "user")
		projMemories, projErr := m.apiClient.GetMemories(sessionID, "project")
		if userErr != nil && projErr != nil {
			return apiResponseMsg{kind: "memory_error", err: userErr}
		}
		all := append(userMemories, projMemories...)
		return apiResponseMsg{kind: "memory_loaded", data: all}
	}
}

func (m *Model) fetchWorkspacesFromAPI() tea.Cmd {
	m.setLoading(overlays.OverlayWorkspace, true)
	return func() tea.Msg {
		workspaces, err := m.apiClient.GetWorkspaces()
		if err != nil {
			return apiResponseMsg{kind: "workspace_error", err: err}
		}
		return apiResponseMsg{kind: "workspace_loaded", data: workspaces}
	}
}

func (m *Model) createSessionFromAPI(workingDir, title string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.apiClient.CreateSession(workingDir, title)
		if err != nil {
			return apiResponseMsg{kind: "create_session_error", err: err}
		}
		return apiResponseMsg{kind: "session_created", data: session}
	}
}

func (m *Model) renameSessionFromAPI(sessionID, title string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.RenameSession(sessionID, title)
		if err != nil {
			return apiResponseMsg{kind: "rename_error", err: err}
		}
		return apiResponseMsg{kind: "rename_done", title: title}
	}
}

func (m *Model) deleteSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.DeleteSession(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "delete_session_error", err: err}
		}
		return apiResponseMsg{kind: "session_deleted", sessionID: sessionID}
	}
}

func (m *Model) exportSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.ExportSession(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "export_error", err: err}
		}
		return apiResponseMsg{kind: "export_done", sessionID: sessionID}
	}
}

func (m *Model) pauseSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.Pause(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "pause_error", err: err}
		}
		return apiResponseMsg{kind: "session_paused", sessionID: sessionID}
	}
}

func (m *Model) resumeSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.Resume(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "resume_error", err: err}
		}
		return apiResponseMsg{kind: "session_resumed", sessionID: sessionID}
	}
}

func (m *Model) cancelSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.Cancel(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "cancel_error", err: err}
		}
		return apiResponseMsg{kind: "session_cancelled", sessionID: sessionID}
	}
}

func (m *Model) archiveSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.ArchiveSession(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "archive_error", err: err}
		}
		return apiResponseMsg{kind: "session_archived", sessionID: sessionID}
	}
}

func (m *Model) compactSessionFromAPI(sessionID string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.apiClient.CompactSession(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "compact_error", err: err}
		}
		return apiResponseMsg{kind: "compact_done", data: result}
	}
}

func (m *Model) approveFromAPI(sessionID string, approvalID string) tea.Cmd {
	return func() tea.Msg {
		req := types.ApproveRequest{Decision: "approve", ApprovalID: approvalID}
		err := m.apiClient.Approve(sessionID, req)
		if err != nil {
			return apiResponseMsg{kind: "approve_error", err: err}
		}
		return apiResponseMsg{kind: "approve_done"}
	}
}

func (m *Model) upsertMemoryFromAPI(sessionID string, memoryType string, key, content string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.UpsertMemory(sessionID, memoryType, key, content)
		if err != nil {
			return apiResponseMsg{kind: "memory_upsert_error", err: err}
		}
		return apiResponseMsg{kind: "memory_upserted"}
	}
}

func (m *Model) switchWorkspaceFromAPI(path string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.SetWorkspace(path)
		if err != nil {
			return apiResponseMsg{kind: "workspace_switch_error", err: err}
		}
		return apiResponseMsg{kind: "workspace_switched", path: path}
	}
}

func (m *Model) sendMessageToAPI(sessionID, content string) tea.Cmd {
	return func() tea.Msg {
		req := types.SendMessageRequest{Content: content}
		msg, err := m.apiClient.SendMessage(sessionID, req)
		if err != nil {
			return apiResponseMsg{kind: "send_message_error", err: err}
		}
		return apiResponseMsg{kind: "message_sent", data: msg}
	}
}

func (m *Model) rollbackFromAPI(sessionID string, targetMsgID string) tea.Cmd {
	return func() tea.Msg {
		req := types.RollbackRequest{TargetMessageID: targetMsgID}
		resp, err := m.apiClient.Rollback(sessionID, req)
		if err != nil {
			return apiResponseMsg{kind: "rollback_error", err: err}
		}
		return apiResponseMsg{kind: "rollback_done", data: resp}
	}
}

func (m *Model) toggleSkillFromAPI(sessionID string, skillName string, enabled bool) tea.Cmd {
	return func() tea.Msg {
		if enabled {
			err := m.apiClient.SetSessionSkills(sessionID, []string{skillName})
			if err != nil {
				return apiResponseMsg{kind: "skill_toggle_error", err: err}
			}
		} else {
			err := m.apiClient.RemoveSessionSkill(sessionID, skillName)
			if err != nil {
				return apiResponseMsg{kind: "skill_toggle_error", err: err}
			}
		}
		return apiResponseMsg{kind: "skill_toggled"}
	}
}

func (m *Model) toggleMCPServerFromAPI(serverID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.ToggleMcpServer(serverID)
		if err != nil {
			return apiResponseMsg{kind: "mcp_toggle_error", err: err}
		}
		return apiResponseMsg{kind: "mcp_toggled"}
	}
}

func (m *Model) installSkillFromAPI(value string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.InstallSkill(value)
		if err != nil {
			return apiResponseMsg{kind: "skill_install_error", err: err}
		}
		return apiResponseMsg{kind: "skill_installed"}
	}
}

func (m *Model) addMCPServerFromAPI(serverID, endpoint string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.AddMCPServer(serverID, endpoint, "sse", "project")
		if err != nil {
			return apiResponseMsg{kind: "mcp_add_error", err: err}
		}
		return apiResponseMsg{kind: "mcp_added"}
	}
}

func (m *Model) deleteMemoryFromAPI(sessionID string, memoryID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.DeleteMemory(sessionID, memoryID)
		if err != nil {
			return apiResponseMsg{kind: "memory_delete_error", err: err}
		}
		return apiResponseMsg{kind: "memory_deleted", id: memoryID}
	}
}

func (m *Model) fetchBackgroundProcessesFromAPI(sessionID string) tea.Cmd {
	m.setLoading(overlays.OverlayBackground, true)
	return func() tea.Msg {
		processes, err := m.apiClient.GetBackgroundProcesses(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "background_error", err: err}
		}
		return apiResponseMsg{kind: "background_loaded", data: processes}
	}
}

func (m *Model) stopBackgroundProcessFromAPI(sessionID string, pid int) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.StopBackgroundProcess(sessionID, pid)
		if err != nil {
			return apiResponseMsg{kind: "background_stop_error", err: err}
		}
		return apiResponseMsg{kind: "background_stopped"}
	}
}

func (m *Model) fetchDashboardDataFromAPI(sessionID string) tea.Cmd {
	m.setLoading(overlays.OverlayDashboard, true)
	return func() tea.Msg {
		sessionUsage, err := m.apiClient.GetSessionUsage(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "dashboard_error", err: err}
		}
		projectUsage, err := m.apiClient.GetProjectUsage(m.workingDir, "date")
		if err != nil {
			projectUsage = nil
		}
		return apiResponseMsg{kind: "dashboard_loaded", data: map[string]interface{}{
			"session_usage": sessionUsage,
			"project_usage": projectUsage,
		}}
	}
}

func (m *Model) fetchProjectConfigFromAPI() tea.Cmd {
	m.setLoading(overlays.OverlaySettings, true)
	return func() tea.Msg {
		cfg, err := m.apiClient.GetProjectConfig()
		if err != nil {
			return apiResponseMsg{kind: "project_config_error", err: err}
		}
		return apiResponseMsg{kind: "project_config_loaded", data: cfg}
	}
}

func (m *Model) fetchGlobalConfigFromAPI() tea.Cmd {
	return func() tea.Msg {
		cfg, err := m.apiClient.GetGlobalConfig()
		if err != nil {
			return apiResponseMsg{kind: "global_config_error", err: err}
		}
		return apiResponseMsg{kind: "global_config_loaded", data: cfg}
	}
}

func (m *Model) saveProjectConfigFromAPI(body map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.UpdateProjectConfig(body)
		if err != nil {
			return apiResponseMsg{kind: "project_config_save_error", err: err}
		}
		return apiResponseMsg{kind: "project_config_saved"}
	}
}

func (m *Model) saveGlobalConfigFromAPI(body map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.UpdateGlobalConfig(body)
		if err != nil {
			return apiResponseMsg{kind: "global_config_save_error", err: err}
		}
		return apiResponseMsg{kind: "global_config_saved"}
	}
}

func (m *Model) applySessionsData(sessions []types.SessionInfo) {
	m.sessions = sessions
	m.sessPicker.Sessions = sessions
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
	if maxW > 80 {
		maxW = 80
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
		c, formatTokenCount(total), ti, to, wd)
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

func (m *Model) routeCommand(cmd string) tea.Cmd {
	switch cmd {
	case "/new":
		m.overlay.Open(overlays.OverlayNewSession)
	case "/switch":
		m.sessPicker.Sessions = m.sessions
		m.overlay.Open(overlays.OverlaySession)
		return m.fetchSessionsFromAPI()
	case "/rename":
		m.renameModal.Current = m.statusBar.Session
		m.renameModal.NewName = ""
		m.overlay.Open(overlays.OverlayRename)
	case "/rollback":
		var items []overlays.RollbackItem
		for _, msg := range m.messages {
			items = append(items, overlays.RollbackItem{
				Content: msg.Content,
				Role:    msg.Role,
				Time:    msg.CreatedAt,
			})
		}
		m.rollback = overlays.NewRollbackPicker(items)
		m.overlay.Open(overlays.OverlayRollback)
	case "/skills":
		m.overlay.Open(overlays.OverlaySkills)
		return m.fetchSkillsFromAPI()
	case "/mcp":
		m.overlay.Open(overlays.OverlayMCP)
		return m.fetchMCPServersFromAPI()
	case "/memory":
		m.overlay.Open(overlays.OverlayMemory)
		if m.activeSessionID != "" {
			return m.fetchMemoriesFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话", true)
	case "/workspace-switch":
		m.overlay.Open(overlays.OverlayWorkspace)
		return m.fetchWorkspacesFromAPI()
	case "/help":
		m.overlay.Open(overlays.OverlayHelp)
	case "/toggle-theme":
		components.ToggleTheme()
		m.toast.Show("主题已切换", false)
	case "/reasoning":
		m.reasoningPicker = overlays.NewReasoningPicker(m.enableReasoning, m.reasoningEffort)
		m.overlay.Open(overlays.OverlayReasoning)
	case "/pause":
		if m.activeSessionID != "" {
			return m.pauseSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可暂停", true)
	case "/resume":
		if m.activeSessionID != "" {
			return m.resumeSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可恢复", true)
	case "/cancel":
		if m.activeSessionID != "" {
			return m.cancelSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可取消", true)
	case "/export":
		if m.activeSessionID != "" {
			return m.exportSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可导出", true)
	case "/compact":
		if m.activeSessionID != "" {
			return m.compactSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可压缩", true)
	case "/background":
		m.backgroundPanel = overlays.NewBackgroundPanel()
		m.overlay.Open(overlays.OverlayBackground)
		if m.activeSessionID != "" {
			return m.fetchBackgroundProcessesFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话", true)
	case "/dashboard":
		m.overlay.Open(overlays.OverlayDashboard)
		if m.activeSessionID != "" {
			return m.fetchDashboardDataFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话", true)
	case "/settings":
		m.overlay.Open(overlays.OverlaySettings)
		return tea.Batch(
			m.fetchProjectConfigFromAPI(),
			m.fetchGlobalConfigFromAPI(),
		)
	case "/status":
		m.updateStatusInfo()
		m.overlay.Open(overlays.OverlayStatus)
	case "/version":
		m.versionPanel.Version = m.version
		m.overlay.Open(overlays.OverlayVersion)
	default:
		m.toast.Show("未知命令: "+cmd, true)
	}
	return nil
}

func (m *Model) updateStatusInfo() {
	sessionName := m.statusBar.Session
	if sessionName == "" {
		sessionName = "无"
	}
	status := "空闲"
	if m.statusBar.Paused {
		status = "已暂停"
	} else if m.statusBar.Processing {
		status = "处理中"
	}

	contextTokens := 0
	var inputTokens, outputTokens int
	if m.activeSessionID != "" {
		for _, s := range m.sessions {
			if s.ID == m.activeSessionID {
				contextTokens = s.CurrentContextTokens
				inputTokens = s.TokenUsage.Input
				outputTokens = s.TokenUsage.Output
				break
			}
		}
	}

	m.statusPanel.Info = overlays.StatusInfo{
		SessionName:      sessionName,
		SessionStatus:    status,
		Yolo:             m.statusBar.Yolo,
		WorkingDir:       m.workingDir,
		Version:          m.version,
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		ContextTokens:    contextTokens,
		Processing:       m.statusBar.Processing,
		ReasoningEnabled: m.enableReasoning,
		ReasoningEffort:  m.reasoningEffort,
	}
}

func (m *Model) applyReasoningOption(opt overlays.ReasoningOption) {
	if opt.Value == "off" {
		m.enableReasoning = false
		m.reasoningEffort = "medium"
		m.toast.Show("思维链: 关闭", false)
	} else {
		m.enableReasoning = true
		m.reasoningEffort = opt.Effort
		m.toast.Show("思维链: "+opt.Label, false)
	}
	m.statusBar.ReasoningEnabled = m.enableReasoning
	m.statusBar.ReasoningEffort = m.reasoningEffort
}

func (m *Model) updateReasoningConfig() tea.Cmd {
	body := map[string]interface{}{
		"llm": map[string]interface{}{
			"enable_reasoning": m.enableReasoning,
			"reasoning_effort": m.reasoningEffort,
		},
	}
	return m.saveGlobalConfigFromAPI(body)
}

func (m *Model) syncReasoningFromConfig(cfg *api.GlobalConfigInfo) {
	if cfg.LLM != nil {
		m.enableReasoning = cfg.LLM.EnableReasoning
		if cfg.LLM.ReasoningEffort != "" {
			m.reasoningEffort = cfg.LLM.ReasoningEffort
		}
	} else {
		m.enableReasoning = false
		m.reasoningEffort = ""
	}
	m.statusBar.ReasoningEnabled = m.enableReasoning
	m.statusBar.ReasoningEffort = m.reasoningEffort
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
