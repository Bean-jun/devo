package agentloop

import (
	"context"
	"fmt"
	"log"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/compressor"
	"devo/internal/core/session"
	"devo/internal/taskexec/tools"
)

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

	sess.ToolCallCount = 0
	sess.LastLoopTerminationReason = ""
	if err := l.store.Update(sess); err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("update session: %w", err))
		return
	}

	var stepSeq int
	var totalStepTokens int

	for {
		msgs, _, err := l.store.GetMessages(sessionID, 0, 0)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get messages: %w", err))
			return
		}

		if l.checkControlFlags(sessionID, eventBus) {
			return
		}

		if _, err := l.compressor.Compress(ctx, sessionID, eventBus); err != nil {
			log.Printf("context compression warning for session %s: %v", sessionID, err)
		}

		msgs, _, err = l.store.GetMessages(sessionID, 0, 0)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get messages after compress: %w", err))
			return
		}

		sess, err := l.store.Get(sessionID)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get session: %w", err))
			return
		}
		activeMsgs := compressor.FilterActiveMessages(msgs, sess.CompressionState)

		result, err := l.llmClient.Complete(ctx, activeMsgs, l.systemPrompt)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("llm complete: %w", err))
			return
		}

		stepSeq++
		if result.TokenUsage != nil {
			l.tokenMeter.RecordStep(sessionID, stepSeq, result.TokenUsage)
			totalStepTokens += result.TokenUsage.TotalTokens

			sess, _ := l.store.Get(sessionID)
			eventBus.Publish("token_usage", map[string]any{
				"step":                 stepSeq,
				"input_tokens":         result.TokenUsage.InputTokens,
				"output_tokens":        result.TokenUsage.OutputTokens,
				"session_total_tokens": sess.TokenUsage.Total,
			})
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

					opType := l.determineOperationType(tool, sess.WorkingDirectory, tc.Params)
					effectivePolicy := l.resolveEffectivePolicy(sess, approval.OperationType(opType))

					if l.approvalManager.IsAutoApproved(effectivePolicy) {
						policyLevelStr := string(effectivePolicy)
						eventBus.Publish("approval_auto", map[string]any{
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
						l.store.AddMessage(sessionID, systemNote)
					} else {
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
				}

				toolResult, err := l.toolExecutor.Execute(sess.WorkingDirectory, tc.ToolName, tc.Params)
				if err != nil {
					eventBus.Publish("tool_result", map[string]any{
						"tool_name": tc.ToolName,
						"success":   false,
						"summary":   fmt.Sprintf("error: %v", err),
					})

					if l.incrementToolCallCount(sessionID, eventBus) {
						return
					}

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

				if l.incrementToolCallCount(sessionID, eventBus) {
					return
				}

				if l.checkControlFlags(sessionID, eventBus) {
					return
				}
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
			"total_step_tokens": totalStepTokens,
		})

		freshSess, err := l.store.Get(sessionID)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get session for state update: %w", err))
			return
		}

		oldState := string(freshSess.State)
		freshSess.State = session.StateIdle
		freshSess.LastActiveAt = time.Now()
		if err := l.store.Update(freshSess); err != nil {
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
