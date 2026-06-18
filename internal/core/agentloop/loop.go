package agentloop

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

const defaultSystemPrompt = "You are a helpful coding assistant. Respond concisely and helpfully."

type ToolExecutor interface {
	Execute(workingDir string, toolName string, params map[string]interface{}) (*tools.ToolResult, error)
	GetTool(name string) (tools.Tool, bool)
	ListTools() []tools.Tool
}

type ApprovalDecision struct {
	ApprovalID string
	Decision   string
	ResultCh   chan error
}

type Loop struct {
	store            session.SessionStore
	llmClient        llmclient.Client
	systemPrompt     string
	toolExecutor     ToolExecutor
	approvalManager  *approval.Manager
	approvalChannels map[string]chan ApprovalDecision
	mu               sync.Mutex
}

func New(store session.SessionStore, llmClient llmclient.Client) *Loop {
	return &Loop{
		store:            store,
		llmClient:        llmClient,
		systemPrompt:     defaultSystemPrompt,
		approvalManager:  approval.NewManager(),
		approvalChannels: make(map[string]chan ApprovalDecision),
	}
}

func NewWithTools(store session.SessionStore, llmClient llmclient.Client, toolExecutor ToolExecutor) *Loop {
	return &Loop{
		store:            store,
		llmClient:        llmClient,
		systemPrompt:     defaultSystemPrompt,
		toolExecutor:     toolExecutor,
		approvalManager:  approval.NewManager(),
		approvalChannels: make(map[string]chan ApprovalDecision),
	}
}

func (l *Loop) ProcessMessage(ctx context.Context, sessionID, content string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StateIdle {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotIdle, sess.State)
	}

	sess.State = session.StateProcessing
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return fmt.Errorf("update session state to processing: %w", err)
	}

	userMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleUser,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if err := l.store.AddMessage(sessionID, userMsg); err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("add user message: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("get event bus: %w", err)
	}

	go l.runAgentLoop(context.Background(), sessionID, eventBus)

	return nil
}

func (l *Loop) ResolveApproval(sessionID, approvalID, decision string) error {
	req, ok := l.approvalManager.GetRequest(approvalID)
	if !ok {
		return fmt.Errorf("approval request not found: %s", approvalID)
	}

	if req.SessionID != sessionID {
		return fmt.Errorf("approval request %s does not belong to session %s", approvalID, sessionID)
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

func (l *Loop) runAgentLoop(ctx context.Context, sessionID string, eventBus *session.EventBus) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("agent loop panic for session %s: %v", sessionID, r)
			l.handleLoopError(sessionID, fmt.Errorf("panic: %v", r))
		}
	}()

	eventBus.Publish("thinking", map[string]string{
		"message": "开始处理用户请求...",
	})

	sess, err := l.store.Get(sessionID)
	if err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("get session: %w", err))
		return
	}

	for {
		msgs, _, err := l.store.GetMessages(sessionID, 0, 0)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get messages: %w", err))
			return
		}

		result, err := l.llmClient.Complete(ctx, msgs, l.systemPrompt)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("llm complete: %w", err))
			return
		}

		if len(result.ToolCalls) > 0 {
			assistantMsg := session.Message{
				ID:        session.GenerateID("msg"),
				Role:      session.RoleAssistant,
				ToolCalls: result.ToolCalls,
				CreatedAt: time.Now(),
			}
			if err := l.store.AddMessage(sessionID, assistantMsg); err != nil {
				l.handleLoopError(sessionID, fmt.Errorf("add assistant message with tool calls: %w", err))
				return
			}

			for _, tc := range result.ToolCalls {
				if l.toolExecutor == nil {
					toolResult := &tools.ToolResult{
						ToolCallID: tc.ID,
						Success:    false,
						Content:    "",
						Error:      "no tool executor available",
					}
					toolMsg := l.toolResultToMessage(toolResult)
					l.store.AddMessage(sessionID, toolMsg)
					eventBus.Publish("tool_call_request", map[string]any{
						"tool_name":         tc.ToolName,
						"params":            tc.Params,
						"requires_approval": false,
						"risk_level":        "-",
					})
					eventBus.Publish("tool_result", map[string]any{
						"tool_name": tc.ToolName,
						"success":   false,
						"summary":   "no tool executor available",
					})
					continue
				}

				tool, ok := l.toolExecutor.GetTool(tc.ToolName)
				if !ok {
					eventBus.Publish("tool_call_request", map[string]any{
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
					l.store.AddMessage(sessionID, toolMsg)
					eventBus.Publish("tool_result", map[string]any{
						"tool_name": tc.ToolName,
						"success":   false,
						"summary":   "unknown tool: " + tc.ToolName,
					})
					continue
				}

				riskLevel := tool.RiskLevel()

				requiresApproval := riskLevel == tools.RiskLevelMedium || riskLevel == tools.RiskLevelHigh

				eventBus.Publish("tool_call_request", map[string]any{
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
							l.store.AddMessage(sessionID, rejectionMsg)
							eventBus.Publish("tool_result", map[string]any{
								"tool_name": tc.ToolName,
								"success":   false,
								"summary":   fmt.Sprintf("security error: %v", err),
							})
							continue
						}
					}

					approved, err := l.handleApproval(sessionID, sess.WorkingDirectory, tc, tool, eventBus)
					if err != nil {
						eventBus.Publish("tool_result", map[string]any{
							"tool_name": tc.ToolName,
							"success":   false,
							"summary":   fmt.Sprintf("error: %v", err),
						})
						continue
					}

					if !approved {
						rejectionMsg := l.rejectionMessage(tc)
						l.store.AddMessage(sessionID, rejectionMsg)
						eventBus.Publish("tool_result", map[string]any{
							"tool_name": tc.ToolName,
							"success":   false,
							"summary":   "操作被用户拒绝",
						})
						continue
					}
				}

				toolResult, err := l.toolExecutor.Execute(sess.WorkingDirectory, tc.ToolName, tc.Params)
				if err != nil {
					eventBus.Publish("tool_result", map[string]any{
						"tool_name": tc.ToolName,
						"success":   false,
						"summary":   fmt.Sprintf("error: %v", err),
					})
					continue
				}

				l.recordChildPID(sessionID, tc.ToolName, toolResult)

				toolResult.ToolCallID = tc.ID
				toolMsg := l.toolResultToMessage(toolResult)
				if err := l.store.AddMessage(sessionID, toolMsg); err != nil {
					l.handleLoopError(sessionID, fmt.Errorf("add tool result message: %w", err))
					return
				}

				summary := toolResult.Content
				if len(summary) > 200 {
					summary = summary[:200] + "..."
				}
				eventBus.Publish("tool_result", map[string]any{
					"tool_name": tc.ToolName,
					"success":   toolResult.Success,
					"summary":   summary,
				})
			}

			continue
		}

		assistantMsg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleAssistant,
			Content:   result.Text,
			CreatedAt: time.Now(),
		}
		if err := l.store.AddMessage(sessionID, assistantMsg); err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("add assistant message: %w", err))
			return
		}

		eventBus.Publish("message_complete", map[string]any{
			"message_id":        assistantMsg.ID,
			"full_text":         result.Text,
			"total_step_tokens": nil,
		})

		oldState := string(sess.State)
		sess.State = session.StateIdle
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("update session state to idle: %w", err))
			return
		}

		eventBus.Publish("session_state_change", map[string]any{
			"old_state": oldState,
			"new_state": string(session.StateIdle),
			"reason":    "completed",
		})

		return
	}
}

func (l *Loop) handleApproval(sessionID, workingDir string, tc session.ToolCall, tool tools.Tool, eventBus *session.EventBus) (bool, error) {
	opType := l.determineOperationType(tool, workingDir, tc.Params)

	details := l.buildApprovalDetails(opType, tc.Params)

	riskLevel := l.determineRiskLevel(tool)

	req := l.approvalManager.CreateRequest(sessionID, tc.ID, approval.OperationType(opType), riskLevel, details)

	ch := make(chan ApprovalDecision, 1)
	l.mu.Lock()
	l.approvalChannels[req.ID] = ch
	l.mu.Unlock()

	defer func() {
		l.mu.Lock()
		delete(l.approvalChannels, req.ID)
		l.mu.Unlock()
	}()

	sess, err := l.store.Get(sessionID)
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

			return true, nil
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

		return false, nil
	}
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

func (l *Loop) buildApprovalDetails(opType string, params map[string]interface{}) map[string]any {
	details := make(map[string]any)

	switch opType {
	case "execute_command":
		if cmd, ok := params["command"].(string); ok {
			details["command"] = cmd
		}
		if ts, ok := params["timeout_seconds"]; ok {
			details["timeout_seconds"] = ts
		}
	default:
		if path, ok := params["path"]; ok {
			details["path"] = path
		}
		if mode, ok := params["mode"]; ok {
			details["mode"] = mode
		}
	}

	return details
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

func (l *Loop) rejectionMessage(tc session.ToolCall) session.Message {
	return session.Message{
		ID:         session.GenerateID("msg"),
		Role:       session.RoleTool,
		Content:    "操作被用户拒绝",
		ToolCallID: tc.ID,
		CreatedAt:  time.Now(),
	}
}

func (l *Loop) toolResultToMessage(tr *tools.ToolResult) session.Message {
	content := tr.Content
	if !tr.Success {
		content = "错误: " + tr.Error
	}
	return session.Message{
		ID:         session.GenerateID("msg"),
		Role:       session.RoleTool,
		Content:    content,
		ToolCallID: tr.ToolCallID,
		CreatedAt:  time.Now(),
	}
}

func (l *Loop) handleLoopError(sessionID string, err error) {
	log.Printf("agent loop error for session %s: %v", sessionID, err)

	sess, getErr := l.store.Get(sessionID)
	if getErr != nil {
		return
	}

	sess.State = session.StateIdle
	l.store.Update(sess)
}

func (l *Loop) recordChildPID(sessionID, toolName string, tr *tools.ToolResult) {
	if toolName != "execute_command" {
		return
	}

	pid := extractPIDFromContent(tr.Content)
	if pid <= 0 {
		return
	}

	sess, err := l.store.Get(sessionID)
	if err != nil {
		return
	}

	sess.ChildPID = &pid
	l.store.Update(sess)
}

func extractPIDFromContent(content string) int {
	marker := "__DEVO_CHILD_PID__="
	idx := findLastIndex(content, marker)
	if idx < 0 {
		return 0
	}

	start := idx + len(marker)
	end := start
	for end < len(content) && content[end] >= '0' && content[end] <= '9' {
		end++
	}

	if end == start {
		return 0
	}

	pid := 0
	for i := start; i < end; i++ {
		pid = pid*10 + int(content[i]-'0')
	}
	return pid
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
