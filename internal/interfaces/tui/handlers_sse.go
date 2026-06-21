package tui

import (
	"fmt"

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

	case "tool_result":
		toolName, _ := msg.Data["tool_name"].(string)
		success, _ := msg.Data["success"].(bool)
		summary, _ := msg.Data["summary"].(string)
		a.chatView.MessageView.UpdateToolCard(toolName, success, summary, "")

	case "message_complete":
		content, _ := msg.Data["full_text"].(string)
		if content != "" {
			assistantMsg := types.Message{
				Role:    "assistant",
				Content: content,
			}
			a.msgs = append(a.msgs, assistantMsg)
			a.chatView.MessageView.AddMessage(assistantMsg)
			if a.activeSession != nil {
				a.updateContextUsage(a.activeSession)
			}
		}
		if totalStepTokens, ok := msg.Data["total_step_tokens"].(float64); ok && totalStepTokens > 0 {
			a.chatView.MessageView.AddSystemNotice(fmt.Sprintf("本轮消耗 %d tokens", int(totalStepTokens)))
		}

	case "approval_required":
		approvalID, _ := msg.Data["approval_id"].(string)
		opType, _ := msg.Data["operation_type"].(string)
		riskLevel, _ := msg.Data["risk_level"].(string)
		summary, _ := msg.Data["summary"].(string)
		diff, _ := msg.Data["diff"].(string)
		commandPreview, _ := msg.Data["command_preview"].(string)

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
		a.statusBar.SessionState = "AwaitingApproval"

	case "approval_auto":
		summary, _ := msg.Data["summary"].(string)
		policyLevel, _ := msg.Data["policy_level"].(string)
		notice := fmt.Sprintf("已根据信任策略（%s）自动批准：%s", policyLevel, summary)
		a.chatView.MessageView.AddSystemNotice(notice)

	case "token_usage":
		inputTokens, _ := msg.Data["input_tokens"].(float64)
		outputTokens, _ := msg.Data["output_tokens"].(float64)
		in := int(inputTokens)
		out := int(outputTokens)
		total := in + out
		a.chatView.InputArea.TokenUsage = fmt.Sprintf("Tokens %s (↑%s ↓%s)", formatTokens(total), formatTokens(in), formatTokens(out))
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
			a.chatView.MessageView.AddSystemNotice("任务完成")
		case "cancelled":
			a.state = StateReady
			a.chatView.Processing = false
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

	case "error":
		errMsg, _ := msg.Data["message"].(string)
		a.toast.Show(errMsg, true)
	}

	return a.readSSEEvent()
}

func (a *App) connectSSECmd() tea.Cmd {
	return func() tea.Msg {
		a.sseClient = NewSSEClient()
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
