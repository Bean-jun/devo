package agentloop

import (
	"context"
	"fmt"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/config"
	"devo/internal/core/session"
	"devo/internal/taskexec/tools"
)

func (l *Loop) ResolveApproval(sessionID, approvalID, decision string) error {
	req, ok := l.approvalManager.GetRequest(approvalID)
	if !ok {
		return fmt.Errorf("approval request not found: %s", approvalID)
	}

	if req.SessionID != sessionID {
		return fmt.Errorf("approval request %s does not belong to session %s", approvalID, sessionID)
	}

	if req.Status != approval.StatusPending {
		return fmt.Errorf("approval request %s is not pending", approvalID)
	}

	if l.approvalManager.IsExpired(approvalID) {
		return fmt.Errorf("approval request %s has expired", approvalID)
	}

	var status approval.Status
	switch decision {
	case "approve":
		status = approval.StatusApproved
	case "reject":
		status = approval.StatusRejected
	default:
		return fmt.Errorf("invalid decision: %s (must be 'approve' or 'reject')", decision)
	}

	resolved, ok := l.approvalManager.Resolve(approvalID, status)
	if !ok {
		return fmt.Errorf("approval request %s is not pending", approvalID)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("approval_resolved", map[string]any{
			"approval_id": approvalID,
			"decision":    decision,
			"source":      "user",
		})
	}

	l.mu.Lock()
	ch, exists := l.approvalChannels[approvalID]
	l.mu.Unlock()

	if exists {
		ch <- ApprovalDecision{
			ApprovalID: approvalID,
			Decision:   decision,
			ResultCh:   nil,
		}
	}

	_ = resolved
	return nil
}

func (l *Loop) resolveEffectivePolicy(sess *session.Session, opType approval.OperationType) approval.PolicyLevel {
	sessionPolicy := make(map[approval.OperationType]approval.PolicyLevel)
	for k, v := range sess.ApprovalPolicy {
		sessionPolicy[approval.OperationType(k)] = approval.PolicyLevel(v)
	}

	var projectPolicy map[approval.OperationType]approval.PolicyLevel
	if sess.WorkingDirectory != "" {
		if cfg, err := config.Load(sess.WorkingDirectory); err == nil && cfg != nil && cfg.ApprovalPolicy != nil {
			projectPolicy = make(map[approval.OperationType]approval.PolicyLevel)
			for k, v := range cfg.ApprovalPolicy {
				projectPolicy[approval.OperationType(k)] = approval.PolicyLevel(v)
			}
		}
	}

	return l.approvalManager.ResolveEffectivePolicy(sessionPolicy, projectPolicy, opType)
}

func (l *Loop) determineOperationType(tool tools.Tool, workingDir string, params map[string]interface{}) string {
	if opProvider, ok := tool.(tools.OperationTypeProvider); ok {
		return opProvider.OperationType(workingDir, params)
	}

	switch tool.RiskLevel() {
	case tools.RiskLevelMedium:
		return "file_write"
	case tools.RiskLevelHigh:
		return "exec_python"
	default:
		return "unknown"
	}
}

func (l *Loop) buildApprovalDetails(tool tools.Tool, workingDir string, opType string, params map[string]interface{}) (map[string]any, error) {
	details := make(map[string]any)

	switch opType {
	case "file_write_new":
		if path, ok := params["path"]; ok {
			details["path"] = path
		}

	case "file_write_overwrite":
		if path, ok := params["path"]; ok {
			details["path"] = path
		}
		if diffPreviewer, ok := tool.(tools.DiffPreviewer); ok {
			diff, err := diffPreviewer.PreviewDiff(workingDir, params)
			if err != nil {
				return nil, err
			}
			if diff != "" {
				details["diff"] = diff
			}
		}

	case "file_edit":
		if path, ok := params["path"]; ok {
			details["path"] = path
		}
		if mode, ok := params["mode"]; ok {
			details["mode"] = mode
		}
		if diffPreviewer, ok := tool.(tools.DiffPreviewer); ok {
			diff, err := diffPreviewer.PreviewDiff(workingDir, params)
			if err != nil {
				return nil, err
			}
			if diff != "" {
				details["diff"] = diff
			}
		}

	case "exec_python":
		if code, ok := params["code"].(string); ok {
			details["code"] = code
		}
		if mode, ok := params["mode"].(string); ok {
			details["mode"] = mode
		}
		if ts, ok := params["timeout_seconds"]; ok {
			details["timeout_seconds"] = ts
		}
		if ctxProvider, ok := tool.(tools.CommandContextProvider); ok {
			details["command_context"] = ctxProvider.GetCommandContext(workingDir, params)
		}

	default:
		if path, ok := params["path"]; ok {
			details["path"] = path
		}
		if mode, ok := params["mode"]; ok {
			details["mode"] = mode
		}
	}

	return details, nil
}

func (l *Loop) determineRiskLevel(tool tools.Tool) approval.RiskLevel {
	switch tool.RiskLevel() {
	case tools.RiskLevelHigh:
		return approval.RiskHigh
	case tools.RiskLevelMedium:
		return approval.RiskMedium
	default:
		return approval.RiskMedium
	}
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
			"tool_call_id": tc.ID,
			"tool_name":    tc.ToolName,
			"success":      false,
			"summary":      fmt.Sprintf("error: %v", err),
		})
		lc.PendingToolCall = nil
		lc.PendingToolCalls = nil
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
		"old_state": session.StateToolExecuting.ToSnakeCase(),
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
				sess.State = session.StateToolExecuting
				sess.LastActiveAt = time.Now()
				l.store.Update(sess)
			}

			lc.EventBus.Publish("session_state_change", map[string]any{
				"old_state": session.StateAwaitingApproval.ToSnakeCase(),
				"new_state": session.StateToolExecuting.ToSnakeCase(),
				"reason":    "approval_granted",
			})

			lc.PendingToolCall = nil
			if len(lc.PendingToolCalls) > 0 && lc.PendingToolCalls[0].ID == tc.ID {
				lc.PendingToolCalls = lc.PendingToolCalls[1:]
			}
			nextState, err := l.executeSingleTool(ctx, lc, tc)
			if err != nil {
				return LoopStateError, err
			}
			if nextState == LoopStateIdle {
				return LoopStateIdle, nil
			}
			return LoopStateToolExecuting, nil
		}

		sess, err := l.store.Get(lc.SessionID)
		if err == nil {
			sess.State = session.StateToolExecuting
			sess.LastActiveAt = time.Now()
			l.store.Update(sess)
		}

		lc.EventBus.Publish("session_state_change", map[string]any{
			"old_state": session.StateAwaitingApproval.ToSnakeCase(),
			"new_state": session.StateToolExecuting.ToSnakeCase(),
			"reason":    "approval_denied",
		})

		rejectionMsg := l.rejectionMessage(tc)
		l.store.AddMessage(lc.SessionID, rejectionMsg)
		lc.EventBus.Publish("tool_result", map[string]any{
			"tool_call_id": tc.ID,
			"tool_name":    tc.ToolName,
			"success":      false,
			"summary":      "操作被用户拒绝",
		})

		lc.PendingToolCall = nil
		lc.PendingToolCalls = nil
		lc.PendingToolResult = nil
		return LoopStatePreparing, nil

	case <-lc.CancelCh:
		l.approvalManager.ResolveWithSource(req.ID, approval.StatusRejected, approval.SourceUser)

		sess, err := l.store.Get(lc.SessionID)
		if err == nil {
			sess.State = session.StateToolExecuting
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
			"new_state": session.StateToolExecuting.ToSnakeCase(),
			"reason":    "approval_cancelled",
		})

		lc.PendingToolCall = nil
		lc.PendingToolCalls = nil
		lc.PendingToolResult = nil
		return LoopStateCancelled, nil

	case <-timeoutCh:
		l.approvalManager.ResolveWithSource(req.ID, approval.StatusRejected, approval.SourceTimeout)

		sess, err := l.store.Get(lc.SessionID)
		if err == nil {
			sess.State = session.StateToolExecuting
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
			"new_state": session.StateToolExecuting.ToSnakeCase(),
			"reason":    "approval_timeout",
		})

		lc.PendingToolCall = nil
		lc.PendingToolCalls = nil
		lc.PendingToolResult = nil
		return LoopStatePreparing, nil
	}
}
