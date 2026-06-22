package agentloop

import (
	"context"
	"fmt"
	"log"
	"strings"
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
			l.handleLoopError(sessionID, fmt.Errorf("panic: %v", r), eventBus)
		}
	}()

	eventBus.Publish("thinking", map[string]string{
		"message": "开始处理用户请求...",
	})

	sess, err := l.store.Get(sessionID)
	if err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("get session: %w", err), eventBus)
		return
	}

	sess.ToolCallCount = 0
	sess.LastLoopTerminationReason = ""
	if err := l.store.Update(sess); err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("update session: %w", err), eventBus)
		return
	}

	var stepSeq int
	var totalStepTokens int

	for {
		msgs, _, err := l.store.GetMessages(sessionID, 0, 0)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get messages: %w", err), eventBus)
			return
		}

		if l.checkControlFlags(sessionID, eventBus) {
			return
		}

		if result, err := l.compressor.Compress(ctx, sessionID, eventBus); err != nil {
			log.Printf("context compression warning for session %s: %v", sessionID, err)
		} else if result != nil && result.CompressedCount > 0 {
			l.archiveManager.AppendSystemMessage(sessionID, fmt.Sprintf("[上下文压缩摘要] %s", result.SummaryText))
		}

		msgs, _, err = l.store.GetMessages(sessionID, 0, 0)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get messages after compress: %w", err), eventBus)
			return
		}

		sess, err := l.store.Get(sessionID)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get session: %w", err), eventBus)
			return
		}
		activeMsgs := compressor.FilterActiveMessages(msgs, sess.CompressionState)

		result, err := l.llmClient.Complete(ctx, activeMsgs, l.systemPrompt)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("llm complete: %w", err), eventBus)
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
				l.handleLoopError(sessionID, fmt.Errorf("add assistant message with tool calls: %w", err), eventBus)
				return
			}

			for _, tc := range result.ToolCalls {
				l.archiveManager.AppendToolCall(sessionID, tc.ToolName, tc.Params)
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

				lockPath := l.getLockPath(sess.WorkingDirectory, tc.ToolName, tc.Params)
				if lockPath != "" {
					l.pathLockManager.Lock(lockPath)
				}

				toolResult, err := l.toolExecutor.Execute(sess.WorkingDirectory, tc.ToolName, tc.Params)

				if lockPath != "" {
					l.pathLockManager.Unlock(lockPath)
				}

				if err != nil {
					toolResult := &tools.ToolResult{
						ToolCallID: tc.ID,
						Success:    false,
						Error:      err.Error(),
					}
					toolMsg := l.toolResultToMessage(toolResult)
					l.store.AddMessage(sessionID, toolMsg)

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

				if toolResult.Success && (tc.ToolName == "write_file" || tc.ToolName == "edit_file") {
					if path, ok := tc.Params["path"].(string); ok && path != "" {
						l.store.RecordFileModification(session.FileModificationRecord{
							SessionID:         sessionID,
							FilePath:          path,
							ModifiedAt:        time.Now(),
							CausedByMessageID: assistantMsg.ID,
						})
					}
				}

				toolResult.ToolCallID = tc.ID
				toolMsg := l.toolResultToMessage(toolResult)
				if err := l.store.AddMessage(sessionID, toolMsg); err != nil {
					l.handleLoopError(sessionID, fmt.Errorf("add tool result message: %w", err), eventBus)
					return
				}

				summary := toolResult.Content
				// 提取 diff（如果存在）
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
				eventBus.Publish("tool_result", toolResultData)

				l.archiveManager.AppendToolResult(sessionID, tc.ToolName, toolResult.Success, summary)

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
			l.handleLoopError(sessionID, fmt.Errorf("add assistant message: %w", err), eventBus)
			return
		}

		l.archiveManager.AppendAssistantMessage(sessionID, result.Text)

		msgCompleteData := map[string]any{
			"message_id":        assistantMsg.ID,
			"full_text":         result.Text,
			"total_step_tokens": totalStepTokens,
		}
		if result.TokenUsage != nil {
			msgCompleteData["input_tokens"] = result.TokenUsage.InputTokens
			msgCompleteData["output_tokens"] = result.TokenUsage.OutputTokens
		}
		eventBus.Publish("message_complete", msgCompleteData)

		freshSess, err := l.store.Get(sessionID)
		if err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("get session for state update: %w", err), eventBus)
			return
		}

		oldState := string(freshSess.State)
		freshSess.State = session.StateIdle
		freshSess.LastActiveAt = time.Now()
		if err := l.store.Update(freshSess); err != nil {
			l.handleLoopError(sessionID, fmt.Errorf("update session state to idle: %w", err), eventBus)
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
