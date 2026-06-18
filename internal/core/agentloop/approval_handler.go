package agentloop

import (
	"fmt"
	"time"

	"devo/internal/core/approval"
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

	return l.approvalManager.ResolveEffectivePolicy(sessionPolicy, opType)
}

func (l *Loop) handleApproval(sessionID, workingDir string, tc session.ToolCall, tool tools.Tool, eventBus *session.EventBus) (bool, error) {
	opType := l.determineOperationType(tool, workingDir, tc.Params)

	details, err := l.buildApprovalDetails(tool, workingDir, opType, tc.Params)
	if err != nil {
		rejectionMsg := session.Message{
			ID:         session.GenerateID("msg"),
			Role:       session.RoleTool,
			Content:    "错误: " + err.Error(),
			ToolCallID: tc.ID,
			CreatedAt:  time.Now(),
		}
		l.store.AddMessage(sessionID, rejectionMsg)
		eventBus.Publish("tool_result", map[string]any{
			"tool_name": tc.ToolName,
			"success":   false,
			"summary":   fmt.Sprintf("error: %v", err),
		})
		return false, err
	}

	riskLevel := l.determineRiskLevel(tool)

	req := l.approvalManager.CreateRequest(sessionID, tc.ID, approval.OperationType(opType), riskLevel, details)

	sess, err := l.store.Get(sessionID)
	if err == nil {
		timeoutSeconds := sess.ApprovalTimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 300
		}
		timeoutAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
		l.approvalManager.SetTimeout(req.ID, timeoutAt)
	}

	ch := make(chan ApprovalDecision, 1)
	l.mu.Lock()
	l.approvalChannels[req.ID] = ch
	l.mu.Unlock()

	defer func() {
		l.mu.Lock()
		delete(l.approvalChannels, req.ID)
		l.mu.Unlock()
	}()

	sess, err = l.store.Get(sessionID)
	if err == nil {
		sess.State = session.StateAwaitingApproval
		sess.LastActiveAt = time.Now()
		l.store.Update(sess)
	}

	eventBus.Publish("session_state_change", map[string]any{
		"old_state": string(session.StateProcessing),
		"new_state": string(session.StateAwaitingApproval),
		"reason":    "awaiting_approval",
	})

	eventBus.Publish("approval_required", map[string]any{
		"approval_id":    req.ID,
		"operation_type": string(req.OperationType),
		"risk_level":     string(req.RiskLevel),
		"details":        req.Details,
	})

	timeoutSeconds := 300
	if sess != nil && sess.ApprovalTimeoutSeconds > 0 {
		timeoutSeconds = sess.ApprovalTimeoutSeconds
	}
	timeoutCh := time.After(time.Duration(timeoutSeconds) * time.Second)

	cancelTicker := time.NewTicker(500 * time.Millisecond)
	defer cancelTicker.Stop()

	var result bool
	var resultErr error

loop:
	for {
		select {
		case decision := <-ch:
			if decision.Decision == "approve" {
				sess, err := l.store.Get(sessionID)
				if err == nil {
					sess.State = session.StateProcessing
					sess.LastActiveAt = time.Now()
					l.store.Update(sess)
				}

				eventBus.Publish("session_state_change", map[string]any{
					"old_state": string(session.StateAwaitingApproval),
					"new_state": string(session.StateProcessing),
					"reason":    "approval_granted",
				})

				result = true
				resultErr = nil
				break loop
			}

			sess, err := l.store.Get(sessionID)
			if err == nil {
				sess.State = session.StateProcessing
				sess.LastActiveAt = time.Now()
				l.store.Update(sess)
			}

			eventBus.Publish("session_state_change", map[string]any{
				"old_state": string(session.StateAwaitingApproval),
				"new_state": string(session.StateProcessing),
				"reason":    "approval_denied",
			})

			result = false
			resultErr = nil
			break loop

		case <-cancelTicker.C:
			sess, err := l.store.Get(sessionID)
			if err == nil && sess.CancelRequested {
				l.approvalManager.ResolveWithSource(req.ID, approval.StatusRejected, approval.SourceUser)

				sess.State = session.StateProcessing
				sess.LastActiveAt = time.Now()
				l.store.Update(sess)

				eventBus.Publish("approval_resolved", map[string]any{
					"approval_id": req.ID,
					"decision":    "reject",
					"source":      "cancelled",
				})

				eventBus.Publish("session_state_change", map[string]any{
					"old_state": string(session.StateAwaitingApproval),
					"new_state": string(session.StateProcessing),
					"reason":    "approval_cancelled",
				})

				result = false
				resultErr = nil
				break loop
			}

		case <-timeoutCh:
			l.approvalManager.ResolveWithSource(req.ID, approval.StatusRejected, approval.SourceTimeout)

			sess, err := l.store.Get(sessionID)
			if err == nil {
				sess.State = session.StateProcessing
				sess.LastActiveAt = time.Now()
				l.store.Update(sess)
			}

			eventBus.Publish("approval_resolved", map[string]any{
				"approval_id": req.ID,
				"decision":    "reject",
				"source":      "timeout",
			})

			eventBus.Publish("session_state_change", map[string]any{
				"old_state": string(session.StateAwaitingApproval),
				"new_state": string(session.StateProcessing),
				"reason":    "approval_timeout",
			})

			result = false
			resultErr = nil
			break loop
		}
	}

	return result, resultErr
}

func (l *Loop) determineOperationType(tool tools.Tool, workingDir string, params map[string]interface{}) string {
	if opProvider, ok := tool.(tools.OperationTypeProvider); ok {
		return opProvider.OperationType(workingDir, params)
	}

	switch tool.RiskLevel() {
	case tools.RiskLevelMedium:
		return "file_write"
	case tools.RiskLevelHigh:
		return "execute_command"
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

	case "execute_command":
		if cmd, ok := params["command"].(string); ok {
			details["command"] = cmd
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
