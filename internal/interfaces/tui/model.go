package tui

import (
	"fmt"
	"os"
	"strings"
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
	filesPanel      overlays.FilesPanel
	skillsPanel     overlays.SkillsPanel
	mcpPanel        overlays.MCPPanel
	memoryPanel     overlays.MemoryPanel
	wsPanel         overlays.WorkspacePanel
	renameModal     overlays.RenameModal
	rollback        overlays.RollbackPicker
	newSessModal    overlays.NewSessionModal
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
		viewport:     vp,
		textarea:     ta,
		renderer:     renderer.New(80),
		statusBar:    components.NewStatusBar(),
		cmdSheet:     overlays.NewCommandSheet(),
		helpPanel:    overlays.HelpPanel{},
		filesPanel:   overlays.NewFilesPanel(),
		skillsPanel:  overlays.NewSkillsPanel(),
		mcpPanel:     overlays.NewMCPPanel(),
		memoryPanel:  overlays.NewMemoryPanel(),
		wsPanel:      overlays.NewWorkspacePanel(),
		rollback:     overlays.NewRollbackPicker(nil),
		newSessModal: overlays.NewSessionModal{},
		apiClient:    api.NewClient(baseURL),
		sseClient:    api.NewSSEClient(baseURL),
		baseURL:      baseURL,
		version:      version,
		workingDir:   getWorkingDir(),
		width:        80,
		height:       24,
		ready:        true,
		loading:      make(map[overlays.OverlayType]bool),
	}

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
	return func() tea.Msg {
		sessions, err := m.apiClient.ListSessions()
		if err != nil {
			return apiResponseMsg{kind: "sessions_error", err: err}
		}
		return apiResponseMsg{kind: "sessions_loaded", data: sessions}
	}
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

func (m *Model) fetchFilesFromAPI() tea.Cmd {
	m.setLoading(overlays.OverlayFiles, true)
	return func() tea.Msg {
		files, err := m.apiClient.GetFiles("", "")
		if err != nil {
			return apiResponseMsg{kind: "files_error", err: err}
		}
		return apiResponseMsg{kind: "files_loaded", data: files}
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
		memories, err := m.apiClient.GetMemories(sessionID)
		if err != nil {
			return apiResponseMsg{kind: "memory_error", err: err}
		}
		return apiResponseMsg{kind: "memory_loaded", data: memories}
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

func (m *Model) upsertMemoryFromAPI(sessionID string, key, content string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.UpsertMemory(sessionID, key, content)
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

func (m *Model) deleteMemoryFromAPI(sessionID string, memoryID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.DeleteMemory(sessionID, memoryID)
		if err != nil {
			return apiResponseMsg{kind: "memory_delete_error", err: err}
		}
		return apiResponseMsg{kind: "memory_deleted", id: memoryID}
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

func (m *Model) applyFilesData(files []types.FileInfo) {
	var entries []overlays.FileEntry
	for _, f := range files {
		ext := ""
		if idx := strings.LastIndex(f.Name, "."); idx >= 0 {
			ext = f.Name[idx+1:]
		}
		size := fmt.Sprintf("%d", f.Size)
		if f.Size > 1024 {
			size = fmt.Sprintf("%.1fK", float64(f.Size)/1024)
		}
		entries = append(entries, overlays.FileEntry{
			Name:     f.Name,
			Size:     size,
			Type:     ext,
			Modified: "N/A",
		})
	}
	m.filesPanel.Files = entries
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
	wd := "/home/project"
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
	m.filesPanel.Width = pw
	m.skillsPanel.Width = pw
	m.mcpPanel.Width = pw
	m.memoryPanel.Width = pw
	m.wsPanel.Width = pw
	m.renameModal.Width = pw
	m.rollback.Width = pw
	m.newSessModal.Width = pw

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
	case "/files":
		m.overlay.Open(overlays.OverlayFiles)
		return m.fetchFilesFromAPI()
	case "/skills":
		m.overlay.Open(overlays.OverlaySkills)
		return m.fetchSkillsFromAPI()
	case "/mcp":
		m.overlay.Open(overlays.OverlayMCP)
		return m.fetchMCPServersFromAPI()
	case "/memory":
		m.overlay.Open(overlays.OverlayMemory)
		return m.fetchMemoriesFromAPI(m.activeSessionID)
	case "/workspace":
		m.overlay.Open(overlays.OverlayWorkspace)
		return m.fetchWorkspacesFromAPI()
	case "/help":
		m.overlay.Open(overlays.OverlayHelp)
	case "/theme":
		components.ToggleTheme()
		m.toast.Show("主题已切换", false)
	case "/yolo":
		m.statusBar.Yolo = !m.statusBar.Yolo
		msg := "YOLO 模式已启用"
		if !m.statusBar.Yolo {
			msg = "YOLO 模式已禁用"
		}
		m.toast.Show(msg, false)
	case "/pause":
		m.statusBar.Paused = !m.statusBar.Paused
		if m.activeSessionID != "" {
			if m.statusBar.Paused {
				return m.pauseSessionFromAPI(m.activeSessionID)
			}
			return m.resumeSessionFromAPI(m.activeSessionID)
		}
		msg := "已暂停"
		if !m.statusBar.Paused {
			msg = "已恢复"
		}
		m.toast.Show(msg, false)
	case "/cancel":
		if m.activeSessionID != "" {
			return m.cancelSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可取消", true)
	case "/archive":
		if m.activeSessionID != "" {
			return m.archiveSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可归档", true)
	case "/export":
		if m.activeSessionID != "" {
			return m.exportSessionFromAPI(m.activeSessionID)
		}
		m.toast.Show("没有活动会话可导出", true)
	case "/w-create":
		m.toast.Show("请输入工作区路径创建新工作区", false)
	case "/quit":
		return tea.Quit
	default:
		m.toast.Show("未知命令: "+cmd, true)
	}
	return nil
}

func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func defaultSessionTitle() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
}
