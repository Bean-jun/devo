package agentloop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/compressor"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"

	"golang.org/x/sync/errgroup"
)

func (l *Loop) registerHandlers(sm *StateMachine) {
	sm.Register(LoopStatePreparing, l.preparingHandler)
	sm.Register(LoopStateThinking, l.thinkingHandler)
	sm.Register(LoopStateEvaluatingResult, l.evaluatingResultHandler)
	sm.Register(LoopStateToolExecuting, l.toolExecutingHandler)
	sm.Register(LoopStateAwaitingApproval, l.awaitingApprovalHandler)
	sm.Register(LoopStateTextResponse, l.textResponseHandler)
}

func (l *Loop) preparingHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	msgs, _, err := l.store.GetMessages(lc.SessionID, 0, 0)
	if err != nil {
		return LoopStateError, fmt.Errorf("get messages: %w", err)
	}

	if result, err := l.compressor.Compress(ctx, lc.SessionID, lc.EventBus); err != nil {
		return LoopStateError, fmt.Errorf("compress: %w", err)
	} else if result != nil && result.CompressedCount > 0 {
		l.archiveManager.AppendSystemMessage(lc.SessionID, fmt.Sprintf("[上下文压缩摘要] %s", result.SummaryText))
	}

	msgs, _, err = l.store.GetMessages(lc.SessionID, 0, 0)
	if err != nil {
		return LoopStateError, fmt.Errorf("get messages after compress: %w", err)
	}

	sess, err := l.store.Get(lc.SessionID)
	if err != nil {
		return LoopStateError, fmt.Errorf("get session: %w", err)
	}
	activeMsgs := compressor.FilterActiveMessages(msgs, sess.CompressionState)

	currentContextTokens := compressor.EstimateContextTokens(msgs, sess.CompressionState)
	if currentContextTokens != sess.CurrentContextTokens {
		sess.CurrentContextTokens = currentContextTokens
		if err := l.store.Update(sess); err != nil {
			return LoopStateError, fmt.Errorf("update session context tokens: %w", err)
		}
	}

	dynamicPrompt := l.promptAssembler.Assemble(sess)

	lc.ActiveMsgs = activeMsgs
	lc.DynamicPrompt = dynamicPrompt

	lc.EventBus.Publish("loop.preparing_done", nil)

	return LoopStateThinking, nil
}

func (l *Loop) thinkingHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	lc.EventBus.Publish("loop.thinking_started", nil)

	var streamErr error
	err := l.llmClient.CompleteStream(ctx, lc.ActiveMsgs, lc.DynamicPrompt, func(evt llmclient.StreamEvent) {
		switch evt.Type {
		case "token":
			lc.EventBus.Publish("streaming_token", map[string]any{
				"token":            evt.Token,
				"accumulated_text": evt.FullText,
			})
		case "done":
			lc.LLMResult = &llmclient.CompleteResult{
				Text:       evt.FullText,
				ToolCalls:  evt.ToolCalls,
				TokenUsage: evt.TokenUsage,
			}
			lc.StepSeq++

			if evt.TokenUsage != nil {
				l.tokenMeter.RecordStep(lc.SessionID, lc.StepSeq, evt.TokenUsage)
				lc.TotalStepTokens += evt.TokenUsage.TotalTokens

				sess, _ := l.store.Get(lc.SessionID)
				lc.EventBus.Publish("token_usage", map[string]any{
					"step":                   lc.StepSeq,
					"input_tokens":           evt.TokenUsage.InputTokens,
					"output_tokens":          evt.TokenUsage.OutputTokens,
					"session_total_tokens":   sess.TokenUsage.Total,
					"session_input_tokens":   sess.TokenUsage.Input,
					"session_output_tokens":  sess.TokenUsage.Output,
					"current_context_tokens": sess.CurrentContextTokens,
				})
			}

			lc.EventBus.Publish("streaming_complete", map[string]any{
				"full_text":     evt.FullText,
				"tool_calls":    evt.ToolCalls,
				"finish_reason": evt.FinishReason,
			})
			lc.EventBus.Publish("loop.thinking_complete", nil)
		case "error":
			streamErr = evt.Err
		}
	})
	if err != nil {
		return LoopStateError, fmt.Errorf("llm complete: %w", err)
	}
	if streamErr != nil {
		return LoopStateError, fmt.Errorf("llm stream error: %w", streamErr)
	}

	return LoopStateEvaluatingResult, nil
}

func (l *Loop) evaluatingResultHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	if len(lc.LLMResult.ToolCalls) > 0 {
		lc.EventBus.Publish("loop.result_evaluated", map[string]string{"type": "tool_calls"})
		return LoopStateToolExecuting, nil
	}

	lc.EventBus.Publish("loop.result_evaluated", map[string]string{"type": "text"})
	return LoopStateTextResponse, nil
}

func (l *Loop) toolExecutingHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	assistantMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleAssistant,
		ToolCalls: lc.LLMResult.ToolCalls,
		CreatedAt: time.Now(),
	}
	if err := l.store.AddMessage(lc.SessionID, assistantMsg); err != nil {
		return LoopStateError, fmt.Errorf("add assistant message with tool calls: %w", err)
	}

	sess, _ := l.store.Get(lc.SessionID)
	if sess.MaxConcurrentToolCalls > 1 && len(lc.LLMResult.ToolCalls) > 1 {
		return l.executeToolsParallel(ctx, lc, sess.MaxConcurrentToolCalls)
	}

	return l.executeToolsSerial(ctx, lc)
}

func (l *Loop) executeToolsSerial(ctx context.Context, lc *LoopContext) (LoopState, error) {

	for _, tc := range lc.LLMResult.ToolCalls {
		select {
		case <-lc.CancelCh:
			return LoopStateCancelled, nil
		case <-lc.PauseCh:
			lc.PausedInState = LoopStateToolExecuting
			return LoopStatePaused, nil
		default:
		}

		l.archiveManager.AppendToolCall(lc.SessionID, tc.ToolName, tc.Params)

		if l.toolExecutor == nil {
			toolResult := &tools.ToolResult{
				ToolCallID: tc.ID,
				Success:    false,
				Content:    "",
				Error:      "no tool executor available",
			}
			toolMsg := l.toolResultToMessage(toolResult)
			l.store.AddMessage(lc.SessionID, toolMsg)
			lc.EventBus.Publish("tool_call_request", map[string]any{
				"tool_name":         tc.ToolName,
				"params":            tc.Params,
				"requires_approval": false,
				"risk_level":        "-",
			})
			lc.EventBus.Publish("tool_result", map[string]any{
				"tool_name": tc.ToolName,
				"success":   false,
				"summary":   "no tool executor available",
			})
			continue
		}

		tool, ok := l.toolExecutor.GetTool(tc.ToolName)
		if !ok {
			lc.EventBus.Publish("tool_call_request", map[string]any{
				"tool_name":         tc.ToolName,
				"params":            tc.Params,
				"requires_approval": false,
				"risk_level":        "-",
			})

			toolResult := &tools.ToolResult{
				ToolCallID: tc.ID,
				Success:    false,
				Error:      "unknown tool: " + tc.ToolName,
			}
			toolMsg := l.toolResultToMessage(toolResult)
			l.store.AddMessage(lc.SessionID, toolMsg)
			lc.EventBus.Publish("tool_result", map[string]any{
				"tool_name": tc.ToolName,
				"success":   false,
				"summary":   "unknown tool: " + tc.ToolName,
			})
			continue
		}

		riskLevel := tool.RiskLevel()
		requiresApproval := riskLevel == tools.RiskLevelMedium || riskLevel == tools.RiskLevelHigh

		lc.EventBus.Publish("tool_call_request", map[string]any{
			"tool_name":         tc.ToolName,
			"params":            tc.Params,
			"requires_approval": requiresApproval,
			"risk_level":        string(riskLevel),
		})

		if requiresApproval {
			if checker, ok := tool.(tools.PreChecker); ok {
				if err := checker.PreCheck(tc.Params); err != nil {
					rejectionMsg := session.Message{
						ID:         session.GenerateID("msg"),
						Role:       session.RoleTool,
						Content:    "安全错误: " + err.Error(),
						ToolCallID: tc.ID,
						CreatedAt:  time.Now(),
					}
					l.store.AddMessage(lc.SessionID, rejectionMsg)
					lc.EventBus.Publish("tool_result", map[string]any{
						"tool_name": tc.ToolName,
						"success":   false,
						"summary":   fmt.Sprintf("security error: %v", err),
					})
					continue
				}
			}

			sess, _ := l.store.Get(lc.SessionID)
			opType := l.determineOperationType(tool, sess.WorkingDirectory, tc.Params)
			effectivePolicy := l.resolveEffectivePolicy(sess, approval.OperationType(opType))

			if l.approvalManager.IsAutoApproved(effectivePolicy) {
				policyLevelStr := string(effectivePolicy)
				lc.EventBus.Publish("approval_auto", map[string]any{
					"operation_type": opType,
					"summary":        fmt.Sprintf("根据策略 %s 自动批准操作 %s", policyLevelStr, opType),
					"policy_level":   policyLevelStr,
				})

				systemNote := session.Message{
					ID:        session.GenerateID("msg"),
					Role:      session.RoleSystem,
					Content:   fmt.Sprintf("已根据信任策略（%s）自动批准 %s 操作", policyLevelStr, opType),
					CreatedAt: time.Now(),
				}
				l.store.AddMessage(lc.SessionID, systemNote)
			} else {
				lc.PendingToolCall = &tc
				return LoopStateAwaitingApproval, nil
			}
		}

		nextState, err := l.executeSingleTool(ctx, lc, tc)
		if err != nil {
			return LoopStateError, err
		}
		if nextState != LoopStateToolExecuting {
			return nextState, nil
		}
	}

	lc.EventBus.Publish("loop.tool_execution_done", nil)
	return LoopStatePreparing, nil
}

func (l *Loop) executeSingleTool(ctx context.Context, lc *LoopContext, tc session.ToolCall) (LoopState, error) {
	sess, err := l.store.Get(lc.SessionID)
	if err != nil {
		return LoopStateError, fmt.Errorf("get session: %w", err)
	}

	lockPath := l.getLockPath(sess.WorkingDirectory, tc.ToolName, tc.Params)
	if lockPath != "" {
		l.pathLockManager.Lock(lockPath)
	}

	toolResult, execErr := l.toolExecutor.ExecuteAsync(ctx, sess.WorkingDirectory, tc.ToolName, tc.Params, func(progress tools.ToolProgress) {
		lc.EventBus.Publish("tool_progress", map[string]any{
			"tool_name": tc.ToolName,
			"stage":     progress.Stage,
			"message":   progress.Message,
			"progress":  progress.Progress,
		})
	})

	if lockPath != "" {
		l.pathLockManager.Unlock(lockPath)
	}

	if execErr != nil {
		toolResult := &tools.ToolResult{
			ToolCallID: tc.ID,
			Success:    false,
			Error:      execErr.Error(),
		}
		toolMsg := l.toolResultToMessage(toolResult)
		l.store.AddMessage(lc.SessionID, toolMsg)

		lc.EventBus.Publish("tool_result", map[string]any{
			"tool_name": tc.ToolName,
			"success":   false,
			"summary":   fmt.Sprintf("error: %v", execErr),
		})

		if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
			return LoopStateIdle, nil
		}
		return LoopStateToolExecuting, nil
	}

	l.recordChildPID(lc.SessionID, tc.ToolName, toolResult)

	if toolResult.Success && (tc.ToolName == "write_file" || tc.ToolName == "edit_file") {
		if path, ok := tc.Params["path"].(string); ok && path != "" {
			l.store.RecordFileModification(session.FileModificationRecord{
				SessionID:         lc.SessionID,
				FilePath:          path,
				ModifiedAt:        time.Now(),
				CausedByMessageID: session.GenerateID("msg"),
			})
		}
	}
	toolMsg := l.toolResultToMessage(toolResult)
	if err := l.store.AddMessage(lc.SessionID, toolMsg); err != nil {
		return LoopStateError, fmt.Errorf("add tool result message: %w", err)
	}

	summary := toolResult.Content
	var diff string
	const diffMarker = "\n__DEVO_DIFF__\n"
	if idx := strings.Index(summary, diffMarker); idx != -1 {
		diff = summary[idx+len(diffMarker):]
		summary = summary[:idx]
	}
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}
	toolResultData := map[string]any{
		"tool_name": tc.ToolName,
		"success":   toolResult.Success,
		"summary":   summary,
	}
	if diff != "" {
		toolResultData["diff"] = diff
	}
	lc.EventBus.Publish("tool_result", toolResultData)

	l.archiveManager.AppendToolResult(lc.SessionID, tc.ToolName, toolResult.Success, summary)

	if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
		return LoopStateIdle, nil
	}

	return LoopStateToolExecuting, nil
}

func (l *Loop) executeToolsParallel(ctx context.Context, lc *LoopContext, maxConcurrent int) (LoopState, error) {
	toolCalls := lc.LLMResult.ToolCalls

	// Pre-check: if any tool needs manual approval, handle it first
	for _, tc := range toolCalls {
		select {
		case <-lc.CancelCh:
			return LoopStateCancelled, nil
		case <-lc.PauseCh:
			lc.PausedInState = LoopStateToolExecuting
			return LoopStatePaused, nil
		default:
		}

		l.archiveManager.AppendToolCall(lc.SessionID, tc.ToolName, tc.Params)

		if l.toolExecutor == nil {
			continue
		}
		tool, ok := l.toolExecutor.GetTool(tc.ToolName)
		if !ok {
			continue
		}

		riskLevel := tool.RiskLevel()
		requiresApproval := riskLevel == tools.RiskLevelMedium || riskLevel == tools.RiskLevelHigh
		if !requiresApproval {
			continue
		}

		// Check if auto-approved
		sess, _ := l.store.Get(lc.SessionID)
		opType := l.determineOperationType(tool, sess.WorkingDirectory, tc.Params)
		effectivePolicy := l.resolveEffectivePolicy(sess, approval.OperationType(opType))

		if l.approvalManager.IsAutoApproved(effectivePolicy) {
			policyLevelStr := string(effectivePolicy)
			lc.EventBus.Publish("approval_auto", map[string]any{
				"operation_type": opType,
				"summary":        fmt.Sprintf("根据策略 %s 自动批准操作 %s", policyLevelStr, opType),
				"policy_level":   policyLevelStr,
			})
			systemNote := session.Message{
				ID:        session.GenerateID("msg"),
				Role:      session.RoleSystem,
				Content:   fmt.Sprintf("已根据信任策略（%s）自动批准 %s 操作", policyLevelStr, opType),
				CreatedAt: time.Now(),
			}
			l.store.AddMessage(lc.SessionID, systemNote)
			continue
		}

		lc.PendingToolCall = &tc
		return LoopStateAwaitingApproval, nil
	}

	// Publish tool_call_request events for all tools
	for _, tc := range toolCalls {
		riskLevel := tools.RiskLevelLow
		if l.toolExecutor != nil {
			if tool, ok := l.toolExecutor.GetTool(tc.ToolName); ok {
				riskLevel = tool.RiskLevel()
			}
		}
		requiresApproval := riskLevel == tools.RiskLevelMedium || riskLevel == tools.RiskLevelHigh
		lc.EventBus.Publish("tool_call_request", map[string]any{
			"tool_name":         tc.ToolName,
			"params":            tc.Params,
			"requires_approval": requiresApproval,
			"risk_level":        string(riskLevel),
		})
	}

	// Execute all tools in parallel
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	var mu sync.Mutex
	var tooManyTools bool

	for _, tc := range toolCalls {
		tc := tc
		g.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-lc.CancelCh:
				return context.Canceled
			case <-lc.PauseCh:
				return fmt.Errorf("paused")
			default:
			}

			sess, err := l.store.Get(lc.SessionID)
			if err != nil {
				return fmt.Errorf("get session: %w", err)
			}

			lockPath := l.getLockPath(sess.WorkingDirectory, tc.ToolName, tc.Params)
			if lockPath != "" {
				l.pathLockManager.Lock(lockPath)
			}

			toolResult, execErr := l.toolExecutor.ExecuteAsync(gctx, sess.WorkingDirectory, tc.ToolName, tc.Params, func(progress tools.ToolProgress) {
				lc.EventBus.Publish("tool_progress", map[string]any{
					"tool_name": tc.ToolName,
					"stage":     progress.Stage,
					"message":   progress.Message,
					"progress":  progress.Progress,
				})
			})

			if lockPath != "" {
				l.pathLockManager.Unlock(lockPath)
			}

			mu.Lock()
			defer mu.Unlock()

			if execErr != nil {
				toolResult := &tools.ToolResult{
					ToolCallID: tc.ID,
					Success:    false,
					Error:      execErr.Error(),
				}
				toolMsg := l.toolResultToMessage(toolResult)
				l.store.AddMessage(lc.SessionID, toolMsg)

				lc.EventBus.Publish("tool_result", map[string]any{
					"tool_name": tc.ToolName,
					"success":   false,
					"summary":   fmt.Sprintf("error: %v", execErr),
				})

				if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
					tooManyTools = true
				}
				return nil
			}

			l.recordChildPID(lc.SessionID, tc.ToolName, toolResult)

			if toolResult.Success && (tc.ToolName == "write_file" || tc.ToolName == "edit_file") {
				if path, ok := tc.Params["path"].(string); ok && path != "" {
					l.store.RecordFileModification(session.FileModificationRecord{
						SessionID:         lc.SessionID,
						FilePath:          path,
						ModifiedAt:        time.Now(),
						CausedByMessageID: session.GenerateID("msg"),
					})
				}
			}

			toolResult.ToolCallID = tc.ID
			toolMsg := l.toolResultToMessage(toolResult)
			if err := l.store.AddMessage(lc.SessionID, toolMsg); err != nil {
				return fmt.Errorf("add tool result message: %w", err)
			}

			summary := toolResult.Content
			var diff string
			const diffMarker = "\n__DEVO_DIFF__\n"
			if idx := strings.Index(summary, diffMarker); idx != -1 {
				diff = summary[idx+len(diffMarker):]
				summary = summary[:idx]
			}
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}
			toolResultData := map[string]any{
				"tool_name": tc.ToolName,
				"success":   toolResult.Success,
				"summary":   summary,
			}
			if diff != "" {
				toolResultData["diff"] = diff
			}
			lc.EventBus.Publish("tool_result", toolResultData)

			l.archiveManager.AppendToolResult(lc.SessionID, tc.ToolName, toolResult.Success, summary)

			if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
				tooManyTools = true
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if err == context.Canceled || ctx.Err() == context.Canceled {
			return LoopStateCancelled, nil
		}
		select {
		case <-lc.PauseCh:
			lc.PausedInState = LoopStateToolExecuting
			return LoopStatePaused, nil
		default:
		}
		return LoopStateError, fmt.Errorf("parallel tool execution: %w", err)
	}

	if tooManyTools {
		return LoopStateIdle, nil
	}

	lc.EventBus.Publish("loop.tool_execution_done", nil)
	return LoopStatePreparing, nil
}

func (l *Loop) awaitingApprovalHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	if lc.PendingToolCall == nil {
		return LoopStateError, fmt.Errorf("awaiting approval but no pending tool call")
	}

	tc := *lc.PendingToolCall

	sess, err := l.store.Get(lc.SessionID)
	if err != nil {
		return LoopStateError, fmt.Errorf("get session: %w", err)
	}

	tool, ok := l.toolExecutor.GetTool(tc.ToolName)
	if !ok {
		return LoopStateError, fmt.Errorf("tool not found: %s", tc.ToolName)
	}

	opType := l.determineOperationType(tool, sess.WorkingDirectory, tc.Params)
	details, err := l.buildApprovalDetails(tool, sess.WorkingDirectory, opType, tc.Params)
	if err != nil {
		rejectionMsg := session.Message{
			ID:         session.GenerateID("msg"),
			Role:       session.RoleTool,
			Content:    "错误: " + err.Error(),
			ToolCallID: tc.ID,
			CreatedAt:  time.Now(),
		}
		l.store.AddMessage(lc.SessionID, rejectionMsg)
		lc.EventBus.Publish("tool_result", map[string]any{
			"tool_name": tc.ToolName,
			"success":   false,
			"summary":   fmt.Sprintf("error: %v", err),
		})
		lc.PendingToolCall = nil
		lc.PendingToolResult = nil
		return LoopStatePreparing, nil
	}

	riskLevel := l.determineRiskLevel(tool)
	req := l.approvalManager.CreateRequest(lc.SessionID, tc.ID, approval.OperationType(opType), riskLevel, details)

	timeoutSeconds := 300
	if sess.ApprovalTimeoutSeconds > 0 {
		timeoutSeconds = sess.ApprovalTimeoutSeconds
	}
	timeoutAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	l.approvalManager.SetTimeout(req.ID, timeoutAt)

	ch := make(chan ApprovalDecision, 1)
	l.mu.Lock()
	l.approvalChannels[req.ID] = ch
	l.mu.Unlock()

	defer func() {
		l.mu.Lock()
		delete(l.approvalChannels, req.ID)
		l.mu.Unlock()
	}()

	sess.State = session.StateAwaitingApproval
	sess.LastActiveAt = time.Now()
	l.store.Update(sess)

	lc.EventBus.Publish("session_state_change", map[string]any{
		"old_state": session.StateProcessing.ToSnakeCase(),
		"new_state": session.StateAwaitingApproval.ToSnakeCase(),
		"reason":    "awaiting_approval",
	})

	lc.EventBus.Publish("approval_required", map[string]any{
		"approval_id":    req.ID,
		"operation_type": string(req.OperationType),
		"risk_level":     string(req.RiskLevel),
		"details":        req.Details,
	})

	timeoutCh := time.After(time.Duration(timeoutSeconds) * time.Second)

	select {
	case decision := <-ch:
		if decision.Decision == "approve" {
			sess, err := l.store.Get(lc.SessionID)
			if err == nil {
				sess.State = session.StateProcessing
				sess.LastActiveAt = time.Now()
				l.store.Update(sess)
			}

			lc.EventBus.Publish("session_state_change", map[string]any{
				"old_state": session.StateAwaitingApproval.ToSnakeCase(),
				"new_state": session.StateProcessing.ToSnakeCase(),
				"reason":    "approval_granted",
			})

			lc.PendingToolCall = nil
			nextState, err := l.executeSingleTool(ctx, lc, tc)
			if err != nil {
				return LoopStateError, err
			}
			if nextState == LoopStateIdle {
				return LoopStateIdle, nil
			}
			return LoopStatePreparing, nil
		}

		sess, err := l.store.Get(lc.SessionID)
		if err == nil {
			sess.State = session.StateProcessing
			sess.LastActiveAt = time.Now()
			l.store.Update(sess)
		}

		lc.EventBus.Publish("session_state_change", map[string]any{
			"old_state": session.StateAwaitingApproval.ToSnakeCase(),
			"new_state": session.StateProcessing.ToSnakeCase(),
			"reason":    "approval_denied",
		})

		rejectionMsg := l.rejectionMessage(tc)
		l.store.AddMessage(lc.SessionID, rejectionMsg)
		lc.EventBus.Publish("tool_result", map[string]any{
			"tool_name": tc.ToolName,
			"success":   false,
			"summary":   "操作被用户拒绝",
		})

		lc.PendingToolCall = nil
		lc.PendingToolResult = nil
		return LoopStatePreparing, nil

	case <-lc.CancelCh:
		l.approvalManager.ResolveWithSource(req.ID, approval.StatusRejected, approval.SourceUser)

		sess, err := l.store.Get(lc.SessionID)
		if err == nil {
			sess.State = session.StateProcessing
			sess.LastActiveAt = time.Now()
			l.store.Update(sess)
		}

		lc.EventBus.Publish("approval_resolved", map[string]any{
			"approval_id": req.ID,
			"decision":    "reject",
			"source":      "cancelled",
		})

		lc.EventBus.Publish("session_state_change", map[string]any{
			"old_state": session.StateAwaitingApproval.ToSnakeCase(),
			"new_state": session.StateProcessing.ToSnakeCase(),
			"reason":    "approval_cancelled",
		})

		lc.PendingToolCall = nil
		lc.PendingToolResult = nil
		return LoopStateCancelled, nil

	case <-timeoutCh:
		l.approvalManager.ResolveWithSource(req.ID, approval.StatusRejected, approval.SourceTimeout)

		sess, err := l.store.Get(lc.SessionID)
		if err == nil {
			sess.State = session.StateProcessing
			sess.LastActiveAt = time.Now()
			l.store.Update(sess)
		}

		lc.EventBus.Publish("approval_resolved", map[string]any{
			"approval_id": req.ID,
			"decision":    "reject",
			"source":      "timeout",
		})

		lc.EventBus.Publish("session_state_change", map[string]any{
			"old_state": session.StateAwaitingApproval.ToSnakeCase(),
			"new_state": session.StateProcessing.ToSnakeCase(),
			"reason":    "approval_timeout",
		})

		lc.PendingToolCall = nil
		lc.PendingToolResult = nil
		return LoopStatePreparing, nil
	}
}

func (l *Loop) textResponseHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	assistantMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleAssistant,
		Content:   lc.LLMResult.Text,
		CreatedAt: time.Now(),
	}
	if err := l.store.AddMessage(lc.SessionID, assistantMsg); err != nil {
		return LoopStateError, fmt.Errorf("add assistant message: %w", err)
	}

	l.archiveManager.AppendAssistantMessage(lc.SessionID, lc.LLMResult.Text)

	msgCompleteData := map[string]any{
		"message_id":        assistantMsg.ID,
		"full_text":         lc.LLMResult.Text,
		"total_step_tokens": lc.TotalStepTokens,
	}
	if lc.LLMResult.TokenUsage != nil {
		msgCompleteData["input_tokens"] = lc.LLMResult.TokenUsage.InputTokens
		msgCompleteData["output_tokens"] = lc.LLMResult.TokenUsage.OutputTokens
	}
	lc.EventBus.Publish("message_complete", msgCompleteData)

	freshSess, err := l.store.Get(lc.SessionID)
	if err != nil {
		return LoopStateError, fmt.Errorf("get session for state update: %w", err)
	}

	oldState := string(freshSess.State)
	freshSess.State = session.StateIdle
	freshSess.LastActiveAt = time.Now()
	if err := l.store.Update(freshSess); err != nil {
		return LoopStateError, fmt.Errorf("update session state to idle: %w", err)
	}

	lc.EventBus.Publish("session_state_change", map[string]any{
		"old_state": session.State(oldState).ToSnakeCase(),
		"new_state": session.StateIdle.ToSnakeCase(),
		"reason":    "completed",
	})

	lc.EventBus.Publish("loop.loop_completed", nil)

	return LoopStateIdle, nil
}
