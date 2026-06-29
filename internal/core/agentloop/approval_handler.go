package agentloop

import (
	"fmt"

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
