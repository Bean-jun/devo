package tui

import (
	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/types"
)

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

func (m *Model) createSessionFromAPI(workingDir, title, agentID string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.apiClient.CreateSession(workingDir, title, agentID)
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

func (m *Model) setTrustLevelFromAPI(sessionID string, level string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.SetTrustLevel(sessionID, level)
		if err != nil {
			return apiResponseMsg{kind: "trust_level_error", err: err}
		}
		return apiResponseMsg{kind: "trust_level_updated", data: level}
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

func (m *Model) sendMessageToAPI(sessionID, content string, images []string) tea.Cmd {
	return func() tea.Msg {
		req := types.SendMessageRequest{Content: content, Images: images}
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

func (m *Model) checkUpdateFromAPI() tea.Cmd {
	return func() tea.Msg {
		result, err := m.apiClient.CheckUpdate()
		if err != nil {
			return apiResponseMsg{kind: "update_check_error", err: err}
		}
		return apiResponseMsg{kind: "update_checked", data: result}
	}
}

func (m *Model) fetchModelsFromAPI() tea.Cmd {
	return func() tea.Msg {
		models, err := m.apiClient.GetModels()
		if err != nil {
			return apiResponseMsg{kind: "models_error", err: err}
		}
		return apiResponseMsg{kind: "models_loaded", data: models}
	}
}

func (m *Model) activateModelFromAPI(modelID string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.ActivateModel(modelID)
		if err != nil {
			return apiResponseMsg{kind: "model_activate_error", err: err}
		}
		return apiResponseMsg{kind: "model_activated", modelID: modelID}
	}
}

func (m *Model) fetchAgentsFromAPI() tea.Cmd {
	return func() tea.Msg {
		agents, err := m.apiClient.GetAgents()
		if err != nil {
			return apiResponseMsg{kind: "agents_error", err: err}
		}
		return apiResponseMsg{kind: "agents_loaded", data: agents}
	}
}

func (m *Model) setTeamModeFromAPI(enabled bool) tea.Cmd {
	return func() tea.Msg {
		result, err := m.apiClient.SetTeamMode(enabled)
		if err != nil {
			return apiResponseMsg{kind: "team_mode_error", err: err}
		}
		return apiResponseMsg{kind: "team_mode_updated", data: result}
	}
}
