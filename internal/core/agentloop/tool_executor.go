package agentloop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/session"
	"devo/internal/taskexec/tools"

	"golang.org/x/sync/errgroup"
)

type toolPreCheckResult struct {
	handled         bool
	pendingApproval bool
	shouldExecute   bool
}

func (l *Loop) preCheckTool(lc *LoopContext, tc session.ToolCall) toolPreCheckResult {
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
			"tool_call_id":      tc.ID,
			"tool_name":         tc.ToolName,
			"params":            tc.Params,
			"requires_approval": false,
			"risk_level":        "-",
		})
		lc.EventBus.Publish("tool_result", map[string]any{
			"tool_call_id": tc.ID,
			"tool_name":    tc.ToolName,
			"success":      false,
			"summary":      "no tool executor available",
		})
		lc.ExecutedToolCallIDs[tc.ID] = true
		return toolPreCheckResult{handled: true}
	}

	tool, ok := l.toolExecutor.GetTool(tc.ToolName)
	if !ok {
		lc.EventBus.Publish("tool_call_request", map[string]any{
			"tool_call_id":      tc.ID,
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
			"tool_call_id": tc.ID,
			"tool_name":    tc.ToolName,
			"success":      false,
			"summary":      "unknown tool: " + tc.ToolName,
		})
		lc.ExecutedToolCallIDs[tc.ID] = true
		return toolPreCheckResult{handled: true}
	}

	riskLevel := tool.RiskLevel()
	requiresApproval := riskLevel == tools.RiskLevelMedium || riskLevel == tools.RiskLevelHigh

	lc.EventBus.Publish("tool_call_request", map[string]any{
		"tool_call_id":      tc.ID,
		"tool_name":         tc.ToolName,
		"params":            tc.Params,
		"requires_approval": requiresApproval,
		"risk_level":        string(riskLevel),
	})

	if !requiresApproval {
		return toolPreCheckResult{shouldExecute: true}
	}

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
				"tool_call_id": tc.ID,
				"tool_name":    tc.ToolName,
				"success":      false,
				"summary":      fmt.Sprintf("security error: %v", err),
			})
			lc.ExecutedToolCallIDs[tc.ID] = true
			return toolPreCheckResult{handled: true}
		}
	}

	sess, _ := l.store.Get(lc.SessionID)
	opType := l.determineOperationType(tool, sess.WorkingDirectory, tc.Params)

	if sess.TrustLevel == string(approval.TrustElevated) {
		lc.EventBus.Publish("approval_auto", map[string]any{
			"operation_type": opType,
			"summary":        fmt.Sprintf("YOLO 模式：自动批准操作 %s", opType),
			"policy_level":   "yolo",
		})
		return toolPreCheckResult{shouldExecute: true}
	}

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

		return toolPreCheckResult{shouldExecute: true}
	}

	return toolPreCheckResult{pendingApproval: true}
}

func (l *Loop) executeToolsSerial(ctx context.Context, lc *LoopContext) (LoopState, error) {
	if len(lc.PendingToolCalls) > 0 {
		for len(lc.PendingToolCalls) > 0 {
			if lc.ExecutedToolCallIDs[lc.PendingToolCalls[0].ID] {
				lc.PendingToolCalls = lc.PendingToolCalls[1:]
				continue
			}
			break
		}
		if len(lc.PendingToolCalls) == 0 {
			lc.EventBus.Publish("loop.tool_execution_done", nil)
			return LoopStatePreparing, nil
		}
		lc.PendingToolCall = &lc.PendingToolCalls[0]
		return LoopStateAwaitingApproval, nil
	}

	var pendingApprovals []session.ToolCall

	for _, tc := range lc.LLMResult.ToolCalls {
		if lc.ExecutedToolCallIDs[tc.ID] {
			continue
		}

		select {
		case <-lc.CancelCh:
			return LoopStateCancelled, nil
		case <-lc.PauseCh:
			lc.PausedInState = LoopStateToolExecuting
			return LoopStatePaused, nil
		default:
		}

		l.archiveManager.AppendToolCall(lc.SessionID, tc.ToolName, tc.Params)

		result := l.preCheckTool(lc, tc)
		if result.handled {
			continue
		}
		if result.pendingApproval {
			pendingApprovals = append(pendingApprovals, tc)
			continue
		}

		nextState, err := l.executeSingleTool(ctx, lc, tc)
		if err != nil {
			return LoopStateError, err
		}
		if nextState != LoopStateToolExecuting {
			return nextState, nil
		}
	}

	if len(pendingApprovals) > 0 {
		lc.PendingToolCalls = pendingApprovals
		lc.PendingToolCall = &lc.PendingToolCalls[0]
		return LoopStateAwaitingApproval, nil
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

	toolCtx := tools.WithSessionID(ctx, lc.SessionID)
	eventCh, execErr := l.toolExecutor.Execute(toolCtx, sess.WorkingDirectory, tc.ToolName, tc.Params)

	if execErr != nil {
		if lockPath != "" {
			l.pathLockManager.Unlock(lockPath)
		}
		toolResult := &tools.ToolResult{
			ToolCallID: tc.ID,
			Success:    false,
			Error:      execErr.Error(),
		}
		toolMsg := l.toolResultToMessage(toolResult)
		l.store.AddMessage(lc.SessionID, toolMsg)

		lc.EventBus.Publish("tool_result", map[string]any{
			"tool_call_id": tc.ID,
			"tool_name":    tc.ToolName,
			"success":      false,
			"summary":      fmt.Sprintf("error: %v", execErr),
		})

		lc.ExecutedToolCallIDs[tc.ID] = true
		if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
			return LoopStateIdle, nil
		}
		return LoopStateToolExecuting, nil
	}

	toolResult := tools.CollectToolResult(eventCh, func(evt tools.StreamEvent) {
		switch evt.Type {
		case tools.StreamEventMeta:
			lc.EventBus.Publish("tool_progress", map[string]any{
				"tool_call_id": tc.ID,
				"tool_name":    tc.ToolName,
				"stage":        evt.Stage,
				"message":      evt.Message,
			})
		case tools.StreamEventChunk:
			lc.EventBus.Publish("tool_chunk", map[string]any{
				"tool_call_id": tc.ID,
				"tool_name":    tc.ToolName,
				"data":         evt.Data,
			})
		}
	})

	if lockPath != "" {
		l.pathLockManager.Unlock(lockPath)
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
		"tool_call_id": tc.ID,
		"tool_name":    tc.ToolName,
		"success":      toolResult.Success,
		"summary":      summary,
	}
	if diff != "" {
		toolResultData["diff"] = diff
	}
	lc.EventBus.Publish("tool_result", toolResultData)

	l.archiveManager.AppendToolResult(lc.SessionID, tc.ToolName, toolResult.Success, summary)

	lc.ExecutedToolCallIDs[tc.ID] = true
	if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
		return LoopStateIdle, nil
	}

	return LoopStateToolExecuting, nil
}

func (l *Loop) executeToolsParallel(ctx context.Context, lc *LoopContext, maxConcurrent int) (LoopState, error) {
	if len(lc.PendingToolCalls) > 0 {
		for len(lc.PendingToolCalls) > 0 {
			if lc.ExecutedToolCallIDs[lc.PendingToolCalls[0].ID] {
				lc.PendingToolCalls = lc.PendingToolCalls[1:]
				continue
			}
			break
		}
		if len(lc.PendingToolCalls) == 0 {
			lc.EventBus.Publish("loop.tool_execution_done", nil)
			return LoopStatePreparing, nil
		}
		lc.PendingToolCall = &lc.PendingToolCalls[0]
		return LoopStateAwaitingApproval, nil
	}

	toolCalls := lc.LLMResult.ToolCalls
	var pendingApprovals []session.ToolCall

	for _, tc := range toolCalls {
		if lc.ExecutedToolCallIDs[tc.ID] {
			continue
		}

		select {
		case <-lc.CancelCh:
			return LoopStateCancelled, nil
		case <-lc.PauseCh:
			lc.PausedInState = LoopStateToolExecuting
			return LoopStatePaused, nil
		default:
		}

		l.archiveManager.AppendToolCall(lc.SessionID, tc.ToolName, tc.Params)

		result := l.preCheckTool(lc, tc)
		if result.handled {
			continue
		}
		if result.pendingApproval {
			pendingApprovals = append(pendingApprovals, tc)
			continue
		}
	}

	pendingIDs := make(map[string]bool)
	for _, tc := range pendingApprovals {
		pendingIDs[tc.ID] = true
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	var mu sync.Mutex
	var tooManyTools bool

	for _, tc := range toolCalls {
		if lc.ExecutedToolCallIDs[tc.ID] || pendingIDs[tc.ID] {
			continue
		}

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

			toolGctx := tools.WithSessionID(gctx, lc.SessionID)
			eventCh, execErr := l.toolExecutor.Execute(toolGctx, sess.WorkingDirectory, tc.ToolName, tc.Params)

			mu.Lock()
			defer mu.Unlock()

			if execErr != nil {
				if lockPath != "" {
					l.pathLockManager.Unlock(lockPath)
				}
				toolResult := &tools.ToolResult{
					ToolCallID: tc.ID,
					Success:    false,
					Error:      execErr.Error(),
				}
				toolMsg := l.toolResultToMessage(toolResult)
				l.store.AddMessage(lc.SessionID, toolMsg)

				lc.EventBus.Publish("tool_result", map[string]any{
					"tool_call_id": tc.ID,
					"tool_name":    tc.ToolName,
					"success":      false,
					"summary":      fmt.Sprintf("error: %v", execErr),
				})

				lc.ExecutedToolCallIDs[tc.ID] = true
				if l.incrementToolCallCount(lc.SessionID, lc.EventBus) {
					tooManyTools = true
				}
				return nil
			}

			toolResult := tools.CollectToolResult(eventCh, func(evt tools.StreamEvent) {
				switch evt.Type {
				case tools.StreamEventMeta:
					lc.EventBus.Publish("tool_progress", map[string]any{
						"tool_call_id": tc.ID,
						"tool_name":    tc.ToolName,
						"stage":        evt.Stage,
						"message":      evt.Message,
					})
				case tools.StreamEventChunk:
					lc.EventBus.Publish("tool_chunk", map[string]any{
						"tool_call_id": tc.ID,
						"tool_name":    tc.ToolName,
						"data":         evt.Data,
					})
				}
			})

			if lockPath != "" {
				l.pathLockManager.Unlock(lockPath)
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
				"tool_call_id": tc.ID,
				"tool_name":    tc.ToolName,
				"success":      toolResult.Success,
				"summary":      summary,
			}
			if diff != "" {
				toolResultData["diff"] = diff
			}
			lc.EventBus.Publish("tool_result", toolResultData)

			l.archiveManager.AppendToolResult(lc.SessionID, tc.ToolName, toolResult.Success, summary)

			lc.ExecutedToolCallIDs[tc.ID] = true
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

	if len(pendingApprovals) > 0 {
		lc.PendingToolCalls = pendingApprovals
		lc.PendingToolCall = &lc.PendingToolCalls[0]
		return LoopStateAwaitingApproval, nil
	}

	if tooManyTools {
		return LoopStateIdle, nil
	}

	lc.EventBus.Publish("loop.tool_execution_done", nil)
	return LoopStatePreparing, nil
}
