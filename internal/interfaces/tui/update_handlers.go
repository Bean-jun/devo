package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/types"
)

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
			m.syncYoloFromSession(m.activeSessionID)
			return m, tea.Batch(
				m.fetchMessagesFromAPI(m.activeSessionID),
				m.connectSSE(m.activeSessionID),
			)
		}
		if len(m.sessions) > 0 && m.sessions[0].MessageCount == 0 {
			first := m.sessions[0]
			m.activeSessionID = first.ID
			m.statusBar.Session = first.Title
			m.statusBar.Yolo = first.TrustLevel == types.TrustLevelElevated
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
			m.messages = nil
			m.backgroundPanel = overlays.NewBackgroundPanel()
			m.renderer.Invalidate(0)
			m.refreshViewportToBottom()
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

	case "trust_level_updated":
		m.toast.Show("信任级别已更新", false)

	case "trust_level_error":
		m.toast.Show("更新信任级别失败: "+msg.err.Error(), true)

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
		if m.rollbackTargetIndex >= 0 && m.rollbackTargetIndex < len(m.messages) {
			m.messages = m.messages[:m.rollbackTargetIndex]
		}
		m.refreshViewportToBottom()
		m.renderer.Invalidate(m.rollbackTargetIndex)
		if m.rollbackTargetContent != "" {
			m.textarea.SetValue(m.rollbackTargetContent)
			m.textarea.CursorEnd()
			m.rollbackTargetContent = ""
		}
		m.toast.Show("已回滚到选定消息", false)

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
			m.syncReasoningFromConfig(cfg)
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

	case "update_checked":
		if result, ok := msg.data.(*api.UpdateCheckResult); ok {
			m.updateInfo = result
			if result.HasUpdate {
				m.statusBar.HasUpdate = true
				m.statusBar.LatestVersion = result.LatestVersion
			}
		}

	case "update_check_error":
		// Silently ignore - update check is best-effort

	case "models_loaded":
		if models, ok := msg.data.([]api.ModelInfo); ok {
			m.models = models
			m.modelPicker.Models = make([]overlays.ModelInfo, 0, len(models))
			for _, mod := range models {
				m.modelPicker.Models = append(m.modelPicker.Models, overlays.ModelInfo{
					ID:     mod.ID,
					Name:   mod.Name,
					Model:  mod.Model,
					Active: mod.Active,
				})
				if mod.Active {
					m.activeModelName = mod.Name
					m.modelPicker.Selected = len(m.modelPicker.Models) - 1
				}
			}
		}

	case "models_error":
		m.toast.Show("获取模型列表失败: "+msg.err.Error(), true)

	case "model_activated":
		for i := range m.modelPicker.Models {
			m.modelPicker.Models[i].Active = m.modelPicker.Models[i].ID == msg.modelID
		}
		for _, mod := range m.modelPicker.Models {
			if mod.ID == msg.modelID {
				m.activeModelName = mod.Name
				break
			}
		}
		m.toast.Show("模型已切换", false)
		m.overlay.Close()
		m.refreshViewport()
		return m, nil

	case "model_activate_error":
		m.toast.Show("切换模型失败: "+msg.err.Error(), true)
		m.overlay.Close()
		m.refreshViewport()

	default:
		if msg.err != nil {
			m.toast.Show("API 错误: "+msg.err.Error(), true)
		}
	}

	return m, nil
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
		m.statusBar.ActivityActive = true
		if msg, ok := evt.Data["message"].(string); ok && msg != "" {
			m.statusBar.Activity = msg
		} else {
			m.statusBar.Activity = "思考中..."
		}
		m.refreshViewport()

	case "reasoning_token":
		chunk := ""
		if token, ok := evt.Data["token"].(string); ok {
			chunk = token
		} else if reasoning, ok := evt.Data["reasoning"].(string); ok {
			chunk = reasoning
		}
		if chunk != "" {
			m.statusBar.ActivityActive = true
			m.statusBar.Activity = "思考中: " + truncateActivity(chunk)
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
			m.statusBar.ActivityActive = true
			m.statusBar.Activity = truncateActivity(content)
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
		m.statusBar.ActivityActive = false
		m.statusBar.Activity = ""
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
			m.backgroundPanel.AppendOutput(pid, prefix+" "+chunk+"\n")
			if m.overlay.Current == overlays.OverlayBackground {
				m.refreshViewport()
			}
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
		m.statusBar.ActivityActive = false
		m.statusBar.Activity = ""
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
		if m.statusBar.Yolo {
			return m, m.approveFromAPI(m.activeSessionID, approvalID)
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
