package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/renderer"
	"devo/internal/interfaces/tui/types"
)

type resizeTickMsg struct{}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return types.TickMsg(t)
		}),
		tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return resizeTickMsg{}
		}),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case apiResponseMsg:
		return m.handleAPIResponse(msg)

	case sseEventMsg:
		return m.handleSSEEvent(msg)

	case resizeTickMsg:
		m.applyTermSize()
		return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return resizeTickMsg{}
		})

	case tea.WindowSizeMsg:
		if !m.initialized {
			m.initialized = true
			m.applyTermSize()
			if m.width == 80 && m.height == 24 {
				m.applySize(msg.Width, msg.Height)
			}
			return m, m.initFromAPI()
		}
		m.applySize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if m.overlay.IsOpen() {
			return m.handleOverlayKey(msg)
		}

		key := msg.String()
		keyCode := msg.Key().Code

		switch {
		case key == "ctrl+c" || key == "ctrl+q":
			return m, tea.Quit

		case key == "esc" || keyCode == tea.KeyEsc:
			m.overlay.Close()
			m.refreshViewport()
			return m, nil

		case key == "ctrl+u":
			if m.textarea.Value() == "" {
				m.jumpToPrevUserMessage()
				return m, nil
			}

		case key == "ctrl+d":
			if m.textarea.Value() == "" {
				m.jumpToNextUserMessage()
				return m, nil
			}

		case key == "?":
			m.overlay.Open(overlays.OverlayHelp)
			m.refreshViewport()
			return m, nil

		case key == "/":
			if m.textarea.Value() == "" {
				m.overlay.Open(overlays.OverlayCommand)
				m.refreshViewport()
				return m, nil
			}

		case key == "ctrl+y":
			m.statusBar.Yolo = !m.statusBar.Yolo
			yoloMsg := "YOLO 模式已开启"
			if !m.statusBar.Yolo {
				yoloMsg = "YOLO 模式已关闭"
			}
			m.toast.Show(yoloMsg, false)
			m.refreshViewport()
			return m, nil

		case key == "ctrl+t":
			components.ToggleTheme()
			m.renderer = renderer.New(m.width)
			m.refreshViewport()
			m.toast.Show("主题已切换为 "+components.CurrentTheme.Name, false)
			return m, nil

		case key == "ctrl+s":
			m.sessPicker.Sessions = m.sessions
			m.overlay.Open(overlays.OverlaySession)
			return m, m.fetchSessionsFromAPI()

		case key == "ctrl+n":
			m.overlay.Open(overlays.OverlayNewSession)
			return m, nil

		case key == "ctrl+p":
			m.statusBar.Paused = !m.statusBar.Paused
			m.statusBar.Processing = !m.statusBar.Paused
			if m.statusBar.Paused {
				m.toast.Show("已暂停", false)
			} else {
				m.toast.Show("已恢复", false)
			}
			m.refreshViewport()
			return m, nil

		case key == "f2":
			m.overlay.Open(overlays.OverlayRename)
			return m, nil

		case key == "tab":
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "assistant" && len(m.messages[i].ToolCalls) > 0 {
					for j := range m.messages[i].ToolCalls {
						m.messages[i].ToolCalls[j].Expanded = !m.messages[i].ToolCalls[j].Expanded
					}
					m.renderer.Invalidate(i)
					m.refreshViewport()
					break
				}
			}
			return m, nil

		case key == "enter":
			content := m.textarea.Value()
			if content == "" {
				return m, nil
			}

			userMsg := types.Message{
				Role:    "user",
				Content: content,
			}
			m.messages = append(m.messages, userMsg)
			m.renderer.Invalidate(len(m.messages) - 1)
			m.textarea.Reset()
			m.statusBar.Processing = true
			m.refreshViewportToBottom()

			if m.activeSessionID != "" {
				cmds = append(cmds, m.sendMessageToAPI(m.activeSessionID, content))
			} else {
				assistantMsg := types.Message{
					Role:    "assistant",
					Content: "未连接到会话。请先创建或切换会话。",
				}
				m.messages = append(m.messages, assistantMsg)
				m.statusBar.Processing = false
				m.refreshViewportToBottom()
			}
			cmds = append(cmds, m.textarea.Focus())
			return m, tea.Batch(cmds...)
		}

	case types.TickMsg:
		if m.toast.Duration > 0 {
			m.toast.Tick()
		}
		return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return types.TickMsg(t)
		})
	}

	var taCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleAPIResponse(msg apiResponseMsg) (tea.Model, tea.Cmd) {
	switch msg.kind {
	case "sessions_loaded":
		if sessions, ok := msg.data.([]types.SessionInfo); ok {
			m.applySessionsData(sessions)
		} else {
			m.toast.Show("会话数据格式错误", true)
		}
		m.setLoading(overlays.OverlaySession, false)
		if m.activeSessionID != "" {
			return m, tea.Batch(
				m.fetchMessagesFromAPI(m.activeSessionID),
				m.connectSSE(m.activeSessionID),
			)
		}
		if len(m.sessions) > 0 && m.sessions[0].MessageCount == 0 {
			first := m.sessions[0]
			m.activeSessionID = first.ID
			m.statusBar.Session = first.Title
			return m, tea.Batch(
				m.fetchMessagesFromAPI(first.ID),
				m.connectSSE(first.ID),
			)
		}
		return m, m.createSessionFromAPI(m.workingDir, defaultSessionTitle())

	case "sessions_error":
		m.toast.Show("获取会话列表失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlaySession, false)

	case "messages_loaded":
		if msgs, ok := msg.data.([]types.Message); ok {
			m.applyMessagesData(msgs)
		} else {
			m.toast.Show("消息数据格式错误", true)
		}
		if msg.sessionID != "" {
			m.activeSessionID = msg.sessionID
		}

	case "messages_error":
		m.toast.Show("获取消息失败: "+msg.err.Error(), true)

	case "skills_loaded":
		if skills, ok := msg.data.([]api.SkillInfo); ok {
			m.applySkillsData(skills)
		} else {
			m.toast.Show("技能数据格式错误", true)
		}
		m.setLoading(overlays.OverlaySkills, false)

	case "skills_error":
		m.toast.Show("获取技能列表失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlaySkills, false)

	case "mcp_loaded":
		if servers, ok := msg.data.([]api.MCPServerInfo); ok {
			m.applyMCPServersData(servers)
		} else {
			m.toast.Show("MCP 数据格式错误", true)
		}
		m.setLoading(overlays.OverlayMCP, false)

	case "mcp_error":
		m.toast.Show("获取 MCP 服务器列表失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlayMCP, false)

	case "memory_loaded":
		if memories, ok := msg.data.([]api.MemoryItem); ok {
			m.applyMemoriesData(memories)
		} else {
			m.toast.Show("记忆数据格式错误", true)
		}
		m.setLoading(overlays.OverlayMemory, false)

	case "memory_error":
		m.toast.Show("获取记忆失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlayMemory, false)

	case "workspace_loaded":
		if workspaces, ok := msg.data.([]api.WorkspaceInfo); ok {
			m.applyWorkspacesData(workspaces)
		} else {
			m.toast.Show("工作区数据格式错误", true)
		}
		m.setLoading(overlays.OverlayWorkspace, false)

	case "workspace_error":
		m.toast.Show("获取工作区列表失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlayWorkspace, false)

	case "session_created":
		if sess, ok := msg.data.(*types.SessionInfo); ok {
			m.activeSessionID = sess.ID
			m.statusBar.Session = sess.Title
			m.sessions = append(m.sessions, *sess)
			m.sessPicker.Sessions = m.sessions
			m.toast.Show("会话已创建: "+sess.Title, false)
			return m, tea.Batch(
				m.fetchMessagesFromAPI(sess.ID),
				m.connectSSE(sess.ID),
			)
		} else {
			m.toast.Show("创建会话数据格式错误", true)
		}

	case "create_session_error":
		m.toast.Show("创建会话失败: "+msg.err.Error(), true)

	case "rename_done":
		m.statusBar.Session = msg.title
		m.toast.Show("会话已重命名为: "+msg.title, false)

	case "rename_error":
		m.toast.Show("重命名失败: "+msg.err.Error(), true)

	case "session_deleted":
		var filtered []types.SessionInfo
		for _, s := range m.sessions {
			if s.ID != msg.sessionID {
				filtered = append(filtered, s)
			}
		}
		m.sessions = filtered
		m.toast.Show("会话已删除", false)

	case "delete_session_error":
		m.toast.Show("删除会话失败: "+msg.err.Error(), true)

	case "export_done":
		m.toast.Show("会话已导出到 ~/devo-exports/", false)

	case "export_error":
		m.toast.Show("导出失败: "+msg.err.Error(), true)

	case "workspace_switched":
		m.toast.Show("已切换工作区: "+msg.path, false)
		return m, m.fetchWorkspacesFromAPI()

	case "workspace_switch_error":
		m.toast.Show("切换工作区失败: "+msg.err.Error(), true)

	case "message_sent":
		if msgData, ok := msg.data.(*types.Message); ok {
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "user" {
				m.messages[len(m.messages)-1].ID = msgData.ID
				m.messages[len(m.messages)-1].CreatedAt = msgData.CreatedAt
			}
			m.refreshViewportToBottom()
		}
		m.toast.Show("消息发送成功，正在等待响应...", false)

	case "send_message_error":
		m.statusBar.Processing = false
		m.toast.Show("发送消息失败: "+msg.err.Error(), true)

	case "rollback_done":
		if resp, ok := msg.data.(*types.GetMessagesResponse); ok {
			m.messages = resp.Messages
			m.refreshViewportToBottom()
			m.toast.Show("已回滚到选定消息", false)
		} else {
			m.toast.Show("回滚数据格式错误", true)
		}

	case "rollback_error":
		m.toast.Show("回滚失败: "+msg.err.Error(), true)

	case "skill_toggled":
		m.toast.Show("技能已更新", false)

	case "skill_toggle_error":
		m.toast.Show("技能切换失败: "+msg.err.Error(), true)

	case "mcp_toggled":
		m.toast.Show("MCP 服务器状态已切换", false)
		return m, m.fetchMCPServersFromAPI()

	case "mcp_toggle_error":
		m.toast.Show("MCP 切换失败: "+msg.err.Error(), true)

	case "skill_installed":
		m.toast.Show("技能安装成功", false)
		return m, m.fetchSkillsFromAPI()

	case "skill_install_error":
		m.toast.Show("技能安装失败: "+msg.err.Error(), true)

	case "mcp_added":
		m.toast.Show("MCP 服务器添加成功", false)
		return m, m.fetchMCPServersFromAPI()

	case "mcp_add_error":
		m.toast.Show("MCP 添加失败: "+msg.err.Error(), true)

	case "memory_deleted":
		var filtered []overlays.MemoryEntry
		for _, mem := range m.memoryPanel.Memories {
			if mem.ID != msg.id {
				filtered = append(filtered, mem)
			}
		}
		m.memoryPanel.Memories = filtered
		m.toast.Show("记忆已删除", false)

	case "memory_delete_error":
		m.toast.Show("删除记忆失败: "+msg.err.Error(), true)

	case "session_paused":
		m.toast.Show("会话已暂停", false)

	case "pause_error":
		m.toast.Show("暂停失败: "+msg.err.Error(), true)

	case "session_resumed":
		m.toast.Show("会话已恢复", false)

	case "resume_error":
		m.toast.Show("恢复失败: "+msg.err.Error(), true)

	case "session_cancelled":
		m.toast.Show("会话已取消", false)
		m.statusBar.Processing = false

	case "cancel_error":
		m.toast.Show("取消失败: "+msg.err.Error(), true)

	case "session_archived":
		m.toast.Show("会话已归档", false)
		if msg.sessionID == m.activeSessionID {
			m.activeSessionID = ""
			m.statusBar.Session = ""
			m.messages = nil
		}

	case "archive_error":
		m.toast.Show("归档失败: "+msg.err.Error(), true)

	case "compact_done":
		result, ok := msg.data.(map[string]interface{})
		if ok {
			compressedCount := 0
			tokensRemoved := 0
			if v, ok := result["compressed_count"].(float64); ok {
				compressedCount = int(v)
			}
			if v, ok := result["tokens_removed"].(float64); ok {
				tokensRemoved = int(v)
			}
			if compressedCount > 0 {
				m.toast.Show(fmt.Sprintf("上下文已压缩，压缩了 %d 条消息，减少约 %d tokens", compressedCount, tokensRemoved), false)
			} else {
				m.toast.Show("当前上下文无需压缩", false)
			}
		} else {
			m.toast.Show("上下文压缩完成", false)
		}

	case "compact_error":
		m.toast.Show("压缩失败: "+msg.err.Error(), true)

	case "approve_done":
		m.toast.Show("已批准操作", false)
		m.overlay.Close()
		m.refreshViewport()

	case "approve_error":
		m.toast.Show("批准失败: "+msg.err.Error(), true)

	case "background_loaded":
		if processes, ok := msg.data.([]api.BackgroundProcessInfo); ok {
			m.backgroundPanel.Processes = processes
		}
		m.setLoading(overlays.OverlayBackground, false)

	case "background_error":
		m.toast.Show("获取后台进程失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlayBackground, false)

	case "background_stopped":
		m.toast.Show("后台进程已停止", false)
		if m.activeSessionID != "" {
			return m, m.fetchBackgroundProcessesFromAPI(m.activeSessionID)
		}

	case "background_stop_error":
		m.toast.Show("停止后台进程失败: "+msg.err.Error(), true)

	case "dashboard_loaded":
		if data, ok := msg.data.(map[string]interface{}); ok {
			if su, ok := data["session_usage"].(*api.SessionUsageInfo); ok {
				m.dashboardPanel.SessionUsage = su
			}
			if pu, ok := data["project_usage"].(*api.ProjectUsageInfo); ok {
				m.dashboardPanel.ProjectUsage = pu
			}
		}
		m.setLoading(overlays.OverlayDashboard, false)

	case "dashboard_error":
		m.toast.Show("获取仪表盘数据失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlayDashboard, false)

	case "project_config_loaded":
		if cfg, ok := msg.data.(*api.ProjectConfigInfo); ok {
			m.settingsPanel.ProjectConfig = cfg
			m.settingsPanel.BuildFields()
		}
		m.setLoading(overlays.OverlaySettings, false)

	case "project_config_error":
		m.toast.Show("获取项目配置失败: "+msg.err.Error(), true)
		m.setLoading(overlays.OverlaySettings, false)

	case "global_config_loaded":
		if cfg, ok := msg.data.(*api.GlobalConfigInfo); ok {
			m.settingsPanel.GlobalConfig = cfg
			m.settingsPanel.BuildFields()
		}

	case "global_config_error":
		m.toast.Show("获取全局配置失败: "+msg.err.Error(), true)

	case "project_config_saved":
		m.toast.Show("项目配置已保存", false)

	case "project_config_save_error":
		m.toast.Show("保存项目配置失败: "+msg.err.Error(), true)

	case "global_config_saved":
		m.toast.Show("全局配置已保存", false)

	case "global_config_save_error":
		m.toast.Show("保存全局配置失败: "+msg.err.Error(), true)

	case "memory_upserted":
		m.toast.Show("记忆已保存", false)

	case "memory_upsert_error":
		m.toast.Show("保存记忆失败: "+msg.err.Error(), true)

	default:
		if msg.err != nil {
			m.toast.Show("API 错误: "+msg.err.Error(), true)
		}
	}

	return m, nil
}

func (m *Model) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	keyCode := msg.Key().Code
	switch {
	case key == "esc" || keyCode == tea.KeyEsc:
		if m.overlay.Current == overlays.OverlaySettings && m.settingsPanel.Editing {
			m.settingsPanel.CancelEditing()
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySkills && m.skillsPanel.Editing {
			m.skillsPanel.CancelEditing()
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMCP && m.mcpPanel.Editing {
			m.mcpPanel.CancelEditing()
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMemory && m.memoryPanel.Editing {
			m.memoryPanel.CancelEditing()
			return m, nil
		}
		m.cmdSheet.Filter = ""
		m.overlay.Close()
		m.refreshViewport()
		return m, nil

	case key == "y" || key == "Y":
		if m.overlay.Current == overlays.OverlayApproval {
			approvalID := m.approval.ApprovalID
			return m, m.approveFromAPI(m.activeSessionID, approvalID)
		}
		if m.isEditing() {
			s := key
			if len(s) == 1 {
				m.appendEditChar(s)
			}
			return m, nil
		}

	case key == "n" || key == "N":
		if m.overlay.Current == overlays.OverlayApproval {
			m.overlay.Close()
			m.refreshViewport()
			m.toast.Show("已拒绝审批请求", false)
			return m, nil
		}
		if m.isEditing() {
			s := key
			if len(s) == 1 {
				m.appendEditChar(s)
			}
			return m, nil
		}

	case key == "enter":
		if m.overlay.Current == overlays.OverlaySkills && m.skillsPanel.Editing {
			value := m.skillsPanel.ConfirmEditing()
			if value != "" {
				return m, m.installSkillFromAPI(value)
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMCP && m.mcpPanel.Editing {
			serverID, endpoint := m.mcpPanel.ConfirmEditing()
			if serverID != "" && endpoint != "" {
				return m, m.addMCPServerFromAPI(serverID, endpoint)
			}
			m.toast.Show("请输入 server_id endpoint", true)
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMemory && m.memoryPanel.Editing {
			key, content := m.memoryPanel.ConfirmEditing()
			if key != "" && content != "" {
				if m.activeSessionID == "" {
					m.toast.Show("没有活动会话", true)
					return m, nil
				}
				return m, m.upsertMemoryFromAPI(m.activeSessionID, "user", key, content)
			}
			m.toast.Show("请输入 key content", true)
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySettings && m.settingsPanel.Editing {
			f, _, ok := m.settingsPanel.ConfirmEditing()
			if ok && f != nil {
				if f.Group == "project" {
					return m, m.saveProjectConfigFromAPI(m.settingsPanel.BuildProjectSaveBody())
				}
				return m, m.saveGlobalConfigFromAPI(m.settingsPanel.BuildGlobalSaveBody())
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySettings {
			f := m.settingsPanel.SelectedField()
			if f != nil && f.FieldType == "int" {
				m.settingsPanel.StartEditing()
				return m, nil
			}
		}
		m.cmdSheet.Filter = ""
		return m.handleOverlayEnter()

	case key == "up":
		m.handleOverlayCursorUp()
		return m, nil

	case key == "k":
		if m.isEditing() {
			m.appendEditChar("k")
			return m, nil
		}
		m.handleOverlayCursorUp()
		return m, nil

	case key == "down":
		m.handleOverlayCursorDown()
		return m, nil

	case key == "j":
		if m.isEditing() {
			m.appendEditChar("j")
			return m, nil
		}
		m.handleOverlayCursorDown()
		return m, nil

	case key == "backspace":
		if m.overlay.Current == overlays.OverlaySettings && m.settingsPanel.Editing {
			if len(m.settingsPanel.EditBuffer) > 0 {
				m.settingsPanel.EditBuffer = m.settingsPanel.EditBuffer[:len(m.settingsPanel.EditBuffer)-1]
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySkills && m.skillsPanel.Editing {
			if len(m.skillsPanel.EditBuffer) > 0 {
				m.skillsPanel.EditBuffer = m.skillsPanel.EditBuffer[:len(m.skillsPanel.EditBuffer)-1]
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMCP && m.mcpPanel.Editing {
			if len(m.mcpPanel.EditBuffer) > 0 {
				m.mcpPanel.EditBuffer = m.mcpPanel.EditBuffer[:len(m.mcpPanel.EditBuffer)-1]
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMemory && m.memoryPanel.Editing {
			if len(m.memoryPanel.EditBuffer) > 0 {
				m.memoryPanel.EditBuffer = m.memoryPanel.EditBuffer[:len(m.memoryPanel.EditBuffer)-1]
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayCommand {
			if len(m.cmdSheet.Filter) > 0 {
				m.cmdSheet.Filter = m.cmdSheet.Filter[:len(m.cmdSheet.Filter)-1]
				m.cmdSheet.Selected = 0
				m.cmdSheet.BuildFlat()
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayRename {
			if len(m.renameModal.NewName) > 0 {
				m.renameModal.NewName = m.renameModal.NewName[:len(m.renameModal.NewName)-1]
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySession {
			if m.sessPicker.Selected < len(m.sessPicker.Sessions) {
				sess := m.sessPicker.Sessions[m.sessPicker.Selected]
				return m, m.deleteSessionFromAPI(sess.ID)
			}
		}
		return m, nil

	case key == "tab":
		if m.overlay.Current == overlays.OverlayBackground {
			m.backgroundPanel.ToggleExpand()
			return m, nil
		}
		return m, nil

	case key == " " || key == "space":
		if m.overlay.Current == overlays.OverlaySettings {
			f := m.settingsPanel.CycleEnum()
			if f != nil {
				if f.Group == "project" {
					return m, m.saveProjectConfigFromAPI(m.settingsPanel.BuildProjectSaveBody())
				}
				return m, m.saveGlobalConfigFromAPI(m.settingsPanel.BuildGlobalSaveBody())
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySkills {
			if m.skillsPanel.Selected < len(m.skillsPanel.Skills) {
				if m.activeSessionID == "" {
					m.toast.Show("没有活动会话", true)
					return m, nil
				}
				skill := m.skillsPanel.Skills[m.skillsPanel.Selected]
				skill.Enabled = !skill.Enabled
				m.skillsPanel.Skills[m.skillsPanel.Selected] = skill
				return m, m.toggleSkillFromAPI(m.activeSessionID, skill.Name, skill.Enabled)
			}
		}
		if m.overlay.Current == overlays.OverlayMCP {
			if m.mcpPanel.Selected < len(m.mcpPanel.Servers) {
				server := m.mcpPanel.Servers[m.mcpPanel.Selected]
				return m, m.toggleMCPServerFromAPI(server.Name)
			}
		}
		return m, nil

	case key == "delete":
		if m.overlay.Current == overlays.OverlayMemory {
			if m.memoryPanel.Selected < len(m.memoryPanel.Memories) {
				if m.activeSessionID == "" {
					m.toast.Show("没有活动会话", true)
					return m, nil
				}
				mem := m.memoryPanel.Memories[m.memoryPanel.Selected]
				return m, m.deleteMemoryFromAPI(m.activeSessionID, mem.ID)
			}
		}
		if m.overlay.Current == overlays.OverlaySession {
			if m.sessPicker.Selected < len(m.sessPicker.Sessions) {
				sess := m.sessPicker.Sessions[m.sessPicker.Selected]
				return m, m.deleteSessionFromAPI(sess.ID)
			}
		}
		return m, nil

	default:
		if m.overlay.Current == overlays.OverlaySettings && m.settingsPanel.Editing {
			s := key
			if len(s) == 1 && s >= "0" && s <= "9" {
				m.settingsPanel.EditBuffer += s
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlaySkills && m.skillsPanel.Editing {
			s := key
			if len(s) == 1 {
				m.skillsPanel.EditBuffer += s
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMCP && m.mcpPanel.Editing {
			s := key
			if len(s) == 1 {
				m.mcpPanel.EditBuffer += s
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayMemory && m.memoryPanel.Editing {
			s := key
			if len(s) == 1 {
				m.memoryPanel.EditBuffer += s
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayCommand {
			s := key
			if len(s) == 1 {
				m.cmdSheet.Filter += s
				m.cmdSheet.Selected = 0
				m.cmdSheet.BuildFlat()
			}
			return m, nil
		}
		if m.overlay.Current == overlays.OverlayRename {
			s := key
			if len(s) == 1 {
				m.renameModal.NewName += s
			}
			return m, nil
		}
		if key == "a" {
			if m.overlay.Current == overlays.OverlaySkills {
				m.skillsPanel.StartEditing()
				return m, nil
			}
			if m.overlay.Current == overlays.OverlayMCP {
				m.mcpPanel.StartEditing()
				return m, nil
			}
			if m.overlay.Current == overlays.OverlayMemory {
				m.memoryPanel.StartEditing()
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *Model) handleOverlayEnter() (tea.Model, tea.Cmd) {
	switch m.overlay.Current {
	case overlays.OverlayCommand:
		cmd := m.cmdSheet.SelectedCommand()
		m.overlay.Close()
		if cmd.Name != "" {
			return m, m.routeCommand(cmd.Name)
		}
		return m, nil

	case overlays.OverlaySession:
		if m.sessPicker.Selected < len(m.sessPicker.Sessions) {
			sess := m.sessPicker.Sessions[m.sessPicker.Selected]
			m.activeSessionID = sess.ID
			m.statusBar.Session = sess.Title
			m.messages = nil
			m.renderer.Invalidate(0)
			m.toast.Show("已切换到会话: "+sess.Title, false)
			m.overlay.Close()
			m.refreshViewportToBottom()
			return m, tea.Batch(
				m.fetchMessagesFromAPI(sess.ID),
				m.connectSSE(sess.ID),
			)
		}
		return m, nil

	case overlays.OverlaySkills:
		m.toast.Show("技能已更新", false)
		return m, nil

	case overlays.OverlayMCP:
		if m.mcpPanel.Selected < len(m.mcpPanel.Servers) {
			s := m.mcpPanel.Servers[m.mcpPanel.Selected]
			m.toast.Show("MCP 服务器: "+s.Name+" ("+s.Status+")", false)
		}
		return m, nil

	case overlays.OverlayMemory:
		if m.memoryPanel.Selected < len(m.memoryPanel.Memories) {
			mem := m.memoryPanel.Memories[m.memoryPanel.Selected]
			m.toast.Show("记忆: "+mem.Key+" = "+mem.Content, false)
		}
		return m, nil

	case overlays.OverlayWorkspace:
		if m.wsPanel.Selected < len(m.wsPanel.Workspaces) {
			ws := m.wsPanel.Workspaces[m.wsPanel.Selected]
			m.overlay.Close()
			return m, m.switchWorkspaceFromAPI(ws.Path)
		}
		return m, nil

	case overlays.OverlayNewSession:
		m.overlay.Close()
		return m, m.createSessionFromAPI(m.workingDir, defaultSessionTitle())

	case overlays.OverlayRename:
		newName := m.renameModal.NewName
		if newName == "" {
			newName = m.renameModal.Current
		}
		m.overlay.Close()
		return m, m.renameSessionFromAPI(m.activeSessionID, newName)

	case overlays.OverlayRollback:
		if m.rollback.Selected < len(m.rollback.Messages) {
			m.overlay.Close()
			targetMsgID := ""
			for i, msg := range m.messages {
				if i == m.rollback.Selected {
					targetMsgID = msg.ID
					break
				}
			}
			return m, m.rollbackFromAPI(m.activeSessionID, targetMsgID)
		}
		return m, nil

	case overlays.OverlayBackground:
		pid := m.backgroundPanel.SelectedPID()
		if pid > 0 && m.activeSessionID != "" {
			return m, m.stopBackgroundProcessFromAPI(m.activeSessionID, pid)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleOverlayCursorUp() {
	switch m.overlay.Current {
	case overlays.OverlayCommand:
		m.cmdSheet.CursorUp()
	case overlays.OverlaySession:
		m.sessPicker.CursorUp()
	case overlays.OverlaySkills:
		m.skillsPanel.CursorUp()
	case overlays.OverlayMCP:
		m.mcpPanel.CursorUp()
	case overlays.OverlayMemory:
		m.memoryPanel.CursorUp()
	case overlays.OverlayWorkspace:
		m.wsPanel.CursorUp()
	case overlays.OverlayRollback:
		m.rollback.CursorUp()
	case overlays.OverlayBackground:
		m.backgroundPanel.CursorUp()
	case overlays.OverlaySettings:
		if !m.settingsPanel.Editing {
			m.settingsPanel.CursorUp()
		}
	}
}

func (m *Model) handleOverlayCursorDown() {
	switch m.overlay.Current {
	case overlays.OverlayCommand:
		m.cmdSheet.CursorDown()
	case overlays.OverlaySession:
		m.sessPicker.CursorDown()
	case overlays.OverlaySkills:
		m.skillsPanel.CursorDown()
	case overlays.OverlayMCP:
		m.mcpPanel.CursorDown()
	case overlays.OverlayMemory:
		m.memoryPanel.CursorDown()
	case overlays.OverlayWorkspace:
		m.wsPanel.CursorDown()
	case overlays.OverlayRollback:
		m.rollback.CursorDown()
	case overlays.OverlayBackground:
		m.backgroundPanel.CursorDown()
	case overlays.OverlaySettings:
		if !m.settingsPanel.Editing {
			m.settingsPanel.CursorDown()
		}
	}
}

func (m *Model) handleSSEEvent(msg sseEventMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		if m.activeSessionID != "" {
			return m, m.connectSSE(m.activeSessionID)
		}
		return m, nil
	}

	evt := msg.event
	eventType := evt.Type
	if t, ok := evt.Data["type"].(string); ok {
		eventType = t
	}

	switch eventType {
	case "thinking":
		m.statusBar.Processing = true
		m.refreshViewport()

	case "reasoning_token":
		chunk := ""
		if token, ok := evt.Data["token"].(string); ok {
			chunk = token
		} else if reasoning, ok := evt.Data["reasoning"].(string); ok {
			chunk = reasoning
		}
		if chunk != "" {
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
				last := &m.messages[len(m.messages)-1]
				last.Thinking += chunk
				last.IsStreaming = true
			} else {
				m.messages = append(m.messages, types.Message{
					Role:        "assistant",
					Thinking:    chunk,
					IsStreaming: true,
				})
			}
			m.renderer.Invalidate(len(m.messages) - 1)
			m.refreshViewportToBottom()
		}

	case "reasoning_complete":
		if full, ok := evt.Data["full_reasoning"].(string); ok && full != "" {
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
				m.messages[len(m.messages)-1].Thinking = full
			}
		}

	case "streaming_token":
		content := ""
		if c, ok := evt.Data["content"].(string); ok {
			content = c
		} else if c, ok := evt.Data["token"].(string); ok {
			content = c
		}
		if content != "" {
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
				last := &m.messages[len(m.messages)-1]
				last.Content += content
				last.IsStreaming = true
			} else {
				m.messages = append(m.messages, types.Message{
					Role:        "assistant",
					Content:     content,
					IsStreaming: true,
				})
			}
			m.renderer.Invalidate(len(m.messages) - 1)
			m.refreshViewportToBottom()
		}

	case "streaming_complete":
		toolCalls, _ := evt.Data["tool_calls"].([]interface{})
		if len(toolCalls) > 0 && len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			last := &m.messages[len(m.messages)-1]
			last.IsStreaming = false
		}

	case "tool_call_request":
		toolCallID := ""
		toolName := ""
		if id, ok := evt.Data["tool_call_id"].(string); ok {
			toolCallID = id
		}
		if name, ok := evt.Data["tool_name"].(string); ok {
			toolName = name
		}
		if toolName != "" {
			tc := types.ToolCall{
				ID:       toolCallID,
				ToolName: toolName,
				Name:     toolName,
				Status:   "pending",
				Expanded: false,
			}
			if params, ok := evt.Data["params"].(map[string]interface{}); ok {
				tc.Params = params
			}
			m.messages = append(m.messages, types.Message{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    "",
				ToolCalls:  []types.ToolCall{tc},
			})
			m.refreshViewportToBottom()
		}

	case "tool_result":
		toolCallID := ""
		if id, ok := evt.Data["tool_call_id"].(string); ok {
			toolCallID = id
		}
		status := "success"
		if s, ok := evt.Data["status"].(string); ok {
			status = s
		}
		output := ""
		if o, ok := evt.Data["output"].(string); ok {
			output = o
		} else if result, ok := evt.Data["result"].(map[string]interface{}); ok {
			if stdout, ok := result["stdout"].(string); ok {
				output = stdout
			}
		}
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].ToolCallID == toolCallID {
				m.messages[i].Content = output
				if len(m.messages[i].ToolCalls) > 0 {
					m.messages[i].ToolCalls[0].Status = status
					m.messages[i].ToolCalls[0].Output = output
				}
				m.refreshViewportToBottom()
				break
			}
		}

	case "tool_progress":
		toolCallID := ""
		if id, ok := evt.Data["tool_call_id"].(string); ok {
			toolCallID = id
		}
		stage := ""
		if s, ok := evt.Data["stage"].(string); ok {
			stage = s
		}
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].ToolCallID == toolCallID && len(m.messages[i].ToolCalls) > 0 {
				m.messages[i].ToolCalls[0].Status = "executing"
				m.messages[i].ToolCalls[0].Summary = stage
				m.refreshViewportToBottom()
				break
			}
		}

	case "tool_chunk":
		toolCallID := ""
		if id, ok := evt.Data["tool_call_id"].(string); ok {
			toolCallID = id
		}
		chunk := ""
		if c, ok := evt.Data["data"].(string); ok {
			chunk = c
		} else if c, ok := evt.Data["content"].(string); ok {
			chunk = c
		}
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].ToolCallID == toolCallID {
				m.messages[i].Content += chunk
				m.refreshViewportToBottom()
				break
			}
		}

	case "message_complete":
		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1].IsStreaming = false
		}
		m.statusBar.Processing = false
		if m.activeSessionID != "" && evt.Data["input_tokens"] != nil {
			for i := range m.sessions {
				if m.sessions[i].ID == m.activeSessionID {
					if t, ok := evt.Data["input_tokens"].(float64); ok {
						m.sessions[i].TokenUsage.Input = int(t)
					}
					if t, ok := evt.Data["output_tokens"].(float64); ok {
						m.sessions[i].TokenUsage.Output = int(t)
					}
					m.sessions[i].TokenUsage.Total = m.sessions[i].TokenUsage.Input + m.sessions[i].TokenUsage.Output
					if ctx, ok := evt.Data["current_context_tokens"].(float64); ok {
						m.sessions[i].CurrentContextTokens = int(ctx)
					}
					break
				}
			}
		}
		m.refreshViewportToBottom()

	case "token_usage":
		if m.activeSessionID != "" {
			inputTokens := 0
			outputTokens := 0
			if t, ok := evt.Data["input_tokens"].(float64); ok {
				inputTokens = int(t)
			}
			if t, ok := evt.Data["output_tokens"].(float64); ok {
				outputTokens = int(t)
			}
			for i := range m.sessions {
				if m.sessions[i].ID == m.activeSessionID {
					m.sessions[i].TokenUsage = types.TokenUsage{
						Input: inputTokens, Output: outputTokens,
						Total: inputTokens + outputTokens,
					}
					if ctxTokens, ok := evt.Data["current_context_tokens"].(float64); ok {
						m.sessions[i].CurrentContextTokens = int(ctxTokens)
					}
					break
				}
			}
		}

	case "session_state_change":
		newState := "idle"
		if s, ok := evt.Data["new_state"].(string); ok {
			newState = s
		}
		reason := ""
		if r, ok := evt.Data["reason"].(string); ok {
			reason = r
		}
		switch newState {
		case "idle", "completed":
			m.statusBar.Processing = false
		case "cancelled":
			m.statusBar.Processing = false
			m.toast.Show("会话已取消", false)
		case "paused":
			m.statusBar.Paused = true
			m.statusBar.Processing = false
		case "tool_executing":
			m.statusBar.Processing = true
		case "thinking":
			m.statusBar.Processing = true
		}
		if reason == "tool_limit_reached" {
			m.toast.Show("已达到工具调用上限，输入新消息继续", false)
		} else if reason == "error" {
			m.toast.Show("发生错误，请重试", true)
		}
		m.refreshViewport()

	case "context_compressed":
		count := 0
		if c, ok := evt.Data["compressed_count"].(float64); ok {
			count = int(c)
		}
		tokens := 0
		if t, ok := evt.Data["tokens_removed"].(float64); ok {
			tokens = int(t)
		}
		sysMsg := types.Message{
			Role:    "system",
			Content: fmt.Sprintf("上下文已压缩：%d 条消息，释放约 %d tokens", count, tokens),
		}
		m.messages = append(m.messages, sysMsg)
		m.refreshViewportToBottom()

	case "file_state_warning":
		warnMsg := ""
		if msg, ok := evt.Data["message"].(string); ok {
			warnMsg = msg
		}
		if warnMsg != "" {
			sysMsg := types.Message{
				Role:    "system",
				Content: "文件状态警告：" + warnMsg,
			}
			m.messages = append(m.messages, sysMsg)
			m.refreshViewportToBottom()
		}

	case "skill_solidified":
		skillName := ""
		if n, ok := evt.Data["skill_name"].(string); ok {
			skillName = n
		}
		if skillName != "" {
			m.toast.Show("技能已固化: "+skillName, false)
		}

	case "memory_updated":
		memoryKey := ""
		if k, ok := evt.Data["key"].(string); ok {
			memoryKey = k
		}
		if memoryKey != "" {
			m.toast.Show("记忆已更新: "+memoryKey, false)
		}

	case "mcp_tool_discovered":
		m.toast.Show("发现新的 MCP 工具", false)

	case "background_output":
		pid := 0
		if p, ok := evt.Data["pid"].(float64); ok {
			pid = int(p)
		}
		if pid > 0 {
			chunk := ""
			if c, ok := evt.Data["data"].(string); ok {
				chunk = c
			}
			stream := "stdout"
			if s, ok := evt.Data["stream"].(string); ok {
				stream = s
			}
			prefix := fmt.Sprintf("[bg:%d]", pid)
			if stream == "stderr" {
				prefix = fmt.Sprintf("[bg:%d:err]", pid)
			}
			m.messages = append(m.messages, types.Message{
				Role:    "system",
				Content: prefix + " " + chunk,
			})
			m.refreshViewportToBottom()
		}

	case "error":
		errMsg := ""
		if msg, ok := evt.Data["message"].(string); ok {
			errMsg = msg
		} else if msg, ok := evt.Data["error"].(string); ok {
			errMsg = msg
		}
		if errMsg != "" {
			m.toast.Show("SSE 错误: "+errMsg, true)
		}

	case "done", "complete":
		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1].IsStreaming = false
		}
		m.statusBar.Processing = false
		m.refreshViewportToBottom()

	case "delta", "message":
		if content, ok := evt.Data["content"].(string); ok {
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
				last := &m.messages[len(m.messages)-1]
				if last.IsStreaming {
					last.Content += content
				} else {
					newMsg := types.Message{
						Role:        "assistant",
						Content:     content,
						IsStreaming: true,
					}
					m.messages = append(m.messages, newMsg)
				}
			} else {
				newMsg := types.Message{
					Role:        "assistant",
					Content:     content,
					IsStreaming: true,
				}
				m.messages = append(m.messages, newMsg)
			}
			m.renderer.Invalidate(len(m.messages) - 1)
			m.refreshViewportToBottom()
		}

	case "tool_use":
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			if toolName, ok := evt.Data["tool_name"].(string); ok {
				tc := types.ToolCall{
					Name:     toolName,
					Expanded: false,
				}
				if toolInput, ok := evt.Data["tool_input"].(string); ok {
					tc.Input = toolInput
				}
				m.messages[len(m.messages)-1].ToolCalls = append(
					m.messages[len(m.messages)-1].ToolCalls, tc,
				)
				m.refreshViewportToBottom()
			}
		}

	case "approval_required":
		operation := ""
		risk := "medium"
		diff := ""
		approvalID := ""
		if id, ok := evt.Data["approval_id"].(string); ok {
			approvalID = id
		}
		if op, ok := evt.Data["operation"].(string); ok {
			operation = op
		} else if toolName, ok := evt.Data["tool_name"].(string); ok {
			operation = toolName
		}
		if r, ok := evt.Data["risk_level"].(string); ok {
			risk = r
		}
		if d, ok := evt.Data["diff"].(string); ok {
			diff = d
		}
		m.approval.ApprovalID = approvalID
		m.approval.Operation = operation
		m.approval.Risk = risk
		m.approval.Diff = diff
		m.overlay.Open(overlays.OverlayApproval)
		m.refreshViewport()

	case "approval_auto":
		summary := ""
		policy := ""
		if s, ok := evt.Data["summary"].(string); ok {
			summary = s
		}
		if p, ok := evt.Data["policy_level"].(string); ok {
			policy = p
		}
		if policy != "yolo" {
			m.toast.Show("已自动批准 "+summary+"（策略："+policy+"）", false)
		}

	case "approval_resolved":
		if m.overlay.Current == overlays.OverlayApproval {
			m.overlay.Close()
		}
		m.refreshViewport()

	case "loop.completed_with_reason":
		reason := ""
		if r, ok := evt.Data["reason"].(string); ok {
			reason = r
		}
		if reason == "completed" {
			m.toast.Show("任务完成", false)
		}
	}

	return m, m.listenSSE()
}
