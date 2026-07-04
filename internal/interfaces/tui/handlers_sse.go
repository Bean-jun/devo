package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/messages"
	"devo/internal/interfaces/tui/types"
)

func (a *App) handleSSEEvent(msg messages.SSEEvent) tea.Cmd {
	Log("[SSE] Event: %s", msg.Type)

	switch msg.Type {
	case "thinking":
		if text, ok := msg.Data["message"].(string); ok {
			a.chatView.MessageView.AddThinking(text)
			a.toast.Hide()
			a.statusBar.ToastActive = false
			a.statusBar.SetActivity(text)
		}

	case "streaming":
		if text, ok := msg.Data["data"].(string); ok {
			a.chatView.MessageView.AddStreamingChunk(text)
			a.toast.Hide()
			a.statusBar.ToastActive = false
			a.statusBar.SetActivity(text)
		}

	case "tool_chunk":
		toolName, _ := msg.Data["tool_name"].(string)
		chunk, _ := msg.Data["data"].(string)
		if toolName != "" && chunk != "" {
			a.chatView.MessageView.AppendToolCardChunk(toolName, chunk)
			a.toast.Hide()
			a.statusBar.ToastActive = false
			a.statusBar.SetActivity(toolName + ": " + chunk)
		}

	case "tool_progress":
		toolName, _ := msg.Data["tool_name"].(string)
		stage, _ := msg.Data["stage"].(string)
		if toolName != "" && stage != "" {
			a.chatView.MessageView.UpdateToolCardStage(toolName, stage)
			a.toast.Hide()
			a.statusBar.ToastActive = false
			a.statusBar.SetActivity(toolName + " " + stage)
		}

	case "tool_call_request":
		toolName, _ := msg.Data["tool_name"].(string)
		params, _ := msg.Data["params"]
		paramsStr := ""
		if params != nil {
			paramsStr = fmt.Sprintf("%v", params)
		}

		card := components.ToolCardData{
			ToolName: toolName,
			Params:   paramsStr,
		}
		a.chatView.MessageView.AddToolCard(card)
		a.toast.Hide()
		a.statusBar.ToastActive = false
		a.statusBar.SetActivity("调用 " + toolName + "...")

	case "tool_result":
		toolName, _ := msg.Data["tool_name"].(string)
		success, _ := msg.Data["success"].(bool)
		summary, _ := msg.Data["summary"].(string)
		a.chatView.MessageView.UpdateToolCard(toolName, success, summary, "")
		a.toast.Hide()
		a.statusBar.ToastActive = false
		a.statusBar.SetActivity(toolName + " 完成")

	case "message_complete":
		a.statusBar.SessionState = "idle"
		a.statusBar.ClearActivity()
		if a.state != StateCancelled {
			a.state = StateReady
		}
		a.chatView.Processing = false
		streamedContent := a.chatView.MessageView.FinalizeStreaming()
		if streamedContent == "" {
			content, _ := msg.Data["full_text"].(string)
			if content != "" {
				assistantMsg := types.Message{
					Role:    "assistant",
					Content: content,
				}
				a.msgs = append(a.msgs, assistantMsg)
				a.chatView.MessageView.AddMessage(assistantMsg)
			}
		} else {
			assistantMsg := types.Message{
				Role:    "assistant",
				Content: streamedContent,
			}
			a.msgs = append(a.msgs, assistantMsg)
		}
		if a.activeSession != nil {
			a.updateContextUsage(a.activeSession)
		}

	case "approval_required":
		approvalID, _ := msg.Data["approval_id"].(string)
		opType, _ := msg.Data["operation_type"].(string)
		riskLevel, _ := msg.Data["risk_level"].(string)
		summary, _ := msg.Data["summary"].(string)
		diff, _ := msg.Data["diff"].(string)
		commandPreview, _ := msg.Data["command_preview"].(string)

		if a.yoloMode {
			notice := fmt.Sprintf("YOLO: 已自动批准 %s", summary)
			a.chatView.MessageView.AddSystemNotice(notice)
			return a.sendApprovalCmd(approvalID, true)
		}

		req := &types.ApprovalRequest{
			ApprovalID:     approvalID,
			OperationType:  opType,
			RiskLevel:      riskLevel,
			Summary:        summary,
			Diff:           diff,
			CommandPreview: commandPreview,
		}
		a.approvalModal.Show(req, a.width, a.height)
		a.state = StateAwaitingApproval
		a.statusBar.SessionState = "awaiting_approval"

	case "approval_auto":
		summary, _ := msg.Data["summary"].(string)
		policyLevel, _ := msg.Data["policy_level"].(string)
		notice := fmt.Sprintf("已根据信任策略（%s）自动批准：%s", policyLevel, summary)
		a.chatView.MessageView.AddSystemNotice(notice)

	case "token_usage":
		var in, out, total int
		if si, ok := msg.Data["session_input_tokens"].(float64); ok {
			in = int(si)
		}
		if so, ok := msg.Data["session_output_tokens"].(float64); ok {
			out = int(so)
		}
		if st, ok := msg.Data["session_total_tokens"].(float64); ok {
			total = int(st)
		}
		if total == 0 {
			it, _ := msg.Data["input_tokens"].(float64)
			ot, _ := msg.Data["output_tokens"].(float64)
			in = int(it)
			out = int(ot)
			total = in + out
		}
		a.chatView.InputArea.TokenUsage = fmt.Sprintf("Tokens %s (↑%s ↓%s)",
			formatTokens(total), formatTokens(in), formatTokens(out))
		if a.activeSession != nil {
			a.updateContextUsage(a.activeSession)
		}

	case "context_compressed":
		compressedCount, _ := msg.Data["compressed_count"].(float64)
		tokensRemoved, _ := msg.Data["tokens_removed"].(float64)
		a.chatView.MessageView.AddSystemNotice(fmt.Sprintf("上下文已压缩：%d 条消息，释放约 %d tokens",
			int(compressedCount), int(tokensRemoved)))
		if a.activeSession != nil {
			a.updateContextUsage(a.activeSession)
		}
		return a.loadMessagesCmd(a.activeSession.ID)

	case "session_state_change":
		oldState, _ := msg.Data["old_state"].(string)
		newState, _ := msg.Data["new_state"].(string)
		reason, _ := msg.Data["reason"].(string)
		a.statusBar.SessionState = newState

		switch reason {
		case "completed":
			a.state = StateReady
			a.chatView.Processing = false
			a.statusBar.ClearActivity()
		case "cancelled":
			a.state = StateCancelled
			a.chatView.Processing = false
			a.statusBar.ClearActivity()
			a.chatView.MessageView.AddSystemNotice("操作已取消")
		case "tool_limit_reached":
			a.state = StateReady
			a.chatView.Processing = false
			a.chatView.MessageView.AddSystemNotice("已达到工具调用上限，输入新消息继续")
		case "error":
			a.state = StateReady
			a.chatView.Processing = false
			a.chatView.MessageView.AddSystemNotice("发生错误，请重试")
		}

		_ = oldState
		_ = newState

	case "loop.state_change":
		newState, _ := msg.Data["new_state"].(string)
		if newState != "" {
			a.statusBar.SessionState = newState
		}

	case "error":
		errMsg, _ := msg.Data["message"].(string)
		a.toast.Show(errMsg, true)

	case "reconnecting":
		attempt, _ := msg.Data["attempt"].(float64)
		maxAttempt, _ := msg.Data["max_attempt"].(float64)
		a.statusBar.ServerConnected = false
		notice := fmt.Sprintf("SSE 连接断开，正在重连... (%d/%d)", int(attempt), int(maxAttempt))
		a.chatView.MessageView.AddSystemNotice(notice)

	case "reconnected":
		a.statusBar.ServerConnected = true
		a.chatView.MessageView.AddSystemNotice("SSE 连接已恢复")
	}

	return a.readSSEEvent()
}

func (a *App) connectSSECmd() tea.Cmd {
	return func() tea.Msg {
		a.sseClient = NewSSEClient()
		a.sseClient.SetReconnectConfig(5, 1*time.Second)
		sseURL := a.apiClient.SSEEndpoint(a.activeSession.ID)
		if err := a.sseClient.Connect(sseURL); err != nil {
			return messages.APIResponse{Kind: "sse_error", Err: err}
		}
		return a.readSSEEvent()()
	}
}

func (a *App) readSSEEvent() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-a.sseClient.Events():
			if !ok {
				return nil
			}
			return messages.SSEEvent{
				Type: event.Type,
				Data: event.Data,
			}
		case err, ok := <-a.sseClient.Errors():
			if ok {
				return messages.APIResponse{Kind: "sse_error", Err: err}
			}
			return nil
		}
	}
}
