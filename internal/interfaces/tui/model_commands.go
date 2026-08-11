package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/overlays"
)

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
		for i := len(m.messages) - 1; i >= 0; i-- {
			msg := m.messages[i]
			if msg.Role == "user" {
				items = append(items, overlays.RollbackItem{
					Content:  msg.Content,
					Role:     msg.Role,
					Time:     formatRollbackTime(msg.CreatedAt),
					MsgIndex: i,
				})
			}
		}
		picker := overlays.NewRollbackPicker(items)
		picker.TotalMessages = len(m.messages)
		m.rollback = picker
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

func formatRollbackTime(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return createdAt
	}
	return t.Format("2006-01-02 15:04:00")
}
