package tui

import (
	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/types"
)

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
		if m.overlay.Current == overlays.OverlayReasoning {
			m.reasoningPicker.Selected = (m.reasoningPicker.Selected + 1) % len(overlays.ReasoningOptions)
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
			m.statusBar.Yolo = sess.TrustLevel == types.TrustLevelElevated
			m.messages = nil
			m.backgroundPanel = overlays.NewBackgroundPanel()
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
			selectedItem := m.rollback.Messages[m.rollback.Selected]
			targetMsgID := m.messages[selectedItem.MsgIndex].ID
			m.rollbackTargetContent = selectedItem.Content
			m.rollbackTargetIndex = selectedItem.MsgIndex
			return m, m.rollbackFromAPI(m.activeSessionID, targetMsgID)
		}
		return m, nil

	case overlays.OverlayBackground:
		pid := m.backgroundPanel.SelectedPID()
		if pid > 0 && m.activeSessionID != "" {
			return m, m.stopBackgroundProcessFromAPI(m.activeSessionID, pid)
		}
		return m, nil

	case overlays.OverlayReasoning:
		opt := m.reasoningPicker.SelectedOption()
		m.applyReasoningOption(opt)
		m.overlay.Close()
		m.refreshViewport()
		return m, m.updateReasoningConfig()
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
	case overlays.OverlayReasoning:
		m.reasoningPicker.CursorUp()
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
	case overlays.OverlayReasoning:
		m.reasoningPicker.CursorDown()
	}
}
