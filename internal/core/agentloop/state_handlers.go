package agentloop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"devo/internal/core/compressor"
	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
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

	sess, err := l.store.Get(lc.SessionID)
	if err != nil {
		return LoopStateError, fmt.Errorf("get session: %w", err)
	}

	dynamicPrompt := l.promptAssembler.Assemble(sess)
	systemPromptTokens := tokenmeter.EstimateTokens(dynamicPrompt)

	if result, err := l.compressor.Compress(ctx, lc.SessionID, lc.EventBus, systemPromptTokens); err != nil {
		return LoopStateError, fmt.Errorf("compress: %w", err)
	} else if result != nil && result.CompressedCount > 0 {
		l.archiveManager.AppendSystemMessage(lc.SessionID, fmt.Sprintf("[上下文压缩摘要] %s", result.SummaryText))
	}

	msgs, _, err = l.store.GetMessages(lc.SessionID, 0, 0)
	if err != nil {
		return LoopStateError, fmt.Errorf("get messages after compress: %w", err)
	}

	sess, err = l.store.Get(lc.SessionID)
	if err != nil {
		return LoopStateError, fmt.Errorf("get session after compress: %w", err)
	}
	activeMsgs := compressor.FilterActiveMessages(msgs, sess.CompressionState)

	currentContextTokens := compressor.EstimateContextTokens(msgs, sess.CompressionState) + systemPromptTokens
	if l.toolExecutor != nil {
		currentContextTokens += tools.EstimateToolTokens(l.toolExecutor.ListTools())
	}
	if currentContextTokens != sess.CurrentContextTokens {
		sess.CurrentContextTokens = currentContextTokens
		if err := l.store.Update(sess); err != nil {
			return LoopStateError, fmt.Errorf("update session context tokens: %w", err)
		}
	}

	lc.ActiveMsgs = activeMsgs
	lc.DynamicPrompt = dynamicPrompt

	lc.EventBus.Publish("loop.preparing_done", nil)

	sess, err = l.store.Get(lc.SessionID)
	if err == nil {
		oldState := string(sess.State)
		sess.State = session.StateThinking
		sess.LastActiveAt = time.Now()
		l.store.Update(sess)
		lc.EventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateThinking.ToSnakeCase(),
			"reason":    "preparing_done",
		})
	}

	return LoopStateThinking, nil
}

func (l *Loop) thinkingHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	lc.EventBus.Publish("loop.thinking_started", nil)
	lc.ReasoningBuilder.Reset()

	var streamErr error
	err := l.llmClient.CompleteStream(ctx, lc.ActiveMsgs, lc.DynamicPrompt, func(evt llmclient.StreamEvent) {
		switch evt.Type {
		case "reasoning_token":
			lc.ReasoningBuilder.WriteString(evt.Reasoning)
			lc.EventBus.Publish("reasoning_token", map[string]any{
				"token": evt.Reasoning,
			})
		case "token":
			lc.EventBus.Publish("streaming_token", map[string]any{
				"token": evt.Token,
			})
		case "done":
			lc.LLMResult = &llmclient.CompleteResult{
				Text:       evt.FullText,
				Reasoning:  evt.FullReasoning,
				ToolCalls:  evt.ToolCalls,
				TokenUsage: evt.TokenUsage,
			}
			logging.Debug(ctx, "thinking handler completed",
				"session_id", lc.SessionID,
				"tool_calls", len(evt.ToolCalls),
				"reasoning_len", len(evt.FullReasoning),
			)
			lc.StepSeq++

			if evt.TokenUsage != nil {
				l.tokenMeter.RecordStep(lc.SessionID, lc.StepSeq, evt.TokenUsage)
				lc.TotalStepTokens += evt.TokenUsage.TotalTokens

				sess, err := l.store.Get(lc.SessionID)
				if err != nil {
					logging.Debug(ctx, "thinking handler: get session for token_usage failed",
						"error", err,
					)
				} else {
					usageData := map[string]any{
						"step":                   lc.StepSeq,
						"input_tokens":           evt.TokenUsage.InputTokens,
						"output_tokens":          evt.TokenUsage.OutputTokens,
						"session_total_tokens":   sess.TokenUsage.Total,
						"session_input_tokens":   sess.TokenUsage.Input,
						"session_output_tokens":  sess.TokenUsage.Output,
						"current_context_tokens": sess.CurrentContextTokens,
					}
					if evt.TokenUsage.ReasoningTokens > 0 {
						usageData["reasoning_tokens"] = evt.TokenUsage.ReasoningTokens
					}
					lc.EventBus.Publish("token_usage", usageData)
				}

				hash := sha256.Sum256([]byte(lc.DynamicPrompt))
				hashStr := hex.EncodeToString(hash[:])[:12]
				cachedTokens := evt.TokenUsage.CachedTokens
				inputTokens := evt.TokenUsage.InputTokens
				var hitRate float64
				if inputTokens > 0 {
					hitRate = float64(cachedTokens) / float64(inputTokens) * 100
				}
				logging.Info(ctx, "llm cache stats",
					"session_id", lc.SessionID,
					"step", lc.StepSeq,
					"system_hash", hashStr,
					"input_tokens", inputTokens,
					"cached_tokens", cachedTokens,
					"hit_rate", fmt.Sprintf("%.1f%%", hitRate),
					"system_len", len(lc.DynamicPrompt),
				)
			}

			if lc.ReasoningBuilder.Len() > 0 {
				lc.EventBus.Publish("reasoning_complete", map[string]any{
					"full_reasoning": lc.ReasoningBuilder.String(),
				})
			}

			lc.EventBus.Publish("streaming_complete", map[string]any{
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
	logging.Debug(ctx, "evaluating result handler called",
		"session_id", lc.SessionID,
		"tool_calls", len(lc.LLMResult.ToolCalls),
	)
	if len(lc.LLMResult.ToolCalls) > 0 {
		lc.ExecutedToolCallIDs = make(map[string]bool)
		lc.EventBus.Publish("loop.result_evaluated", map[string]string{"type": "tool_calls"})

		sess, err := l.store.Get(lc.SessionID)
		if err != nil {
			logging.Debug(ctx, "evaluating result handler: failed to get session",
				"error", err,
			)
		} else {
			oldState := string(sess.State)
			logging.Debug(ctx, "evaluating result handler: setting state to ToolExecuting",
				"session_id", lc.SessionID,
				"old_state", oldState,
			)
			sess.State = session.StateToolExecuting
			sess.LastActiveAt = time.Now()
			if err := l.store.Update(sess); err != nil {
				logging.Error(ctx, "evaluating result handler: failed to update state to ToolExecuting",
					"error", err,
				)
			} else {
				logging.Debug(ctx, "evaluating result handler: updated state",
					"old_state", oldState,
					"new_state", "ToolExecuting",
				)
			}
			lc.EventBus.Publish("session_state_change", map[string]any{
				"old_state": session.State(oldState).ToSnakeCase(),
				"new_state": session.StateToolExecuting.ToSnakeCase(),
				"reason":    "tool_calls_detected",
			})
		}

		return LoopStateToolExecuting, nil
	}

	lc.EventBus.Publish("loop.result_evaluated", map[string]string{"type": "text"})
	return LoopStateTextResponse, nil
}

func (l *Loop) toolExecutingHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	logging.Debug(ctx, "tool executing handler called",
		"session_id", lc.SessionID,
		"tool_calls", len(lc.LLMResult.ToolCalls),
	)
	if lc.ExecutedToolCallIDs == nil {
		lc.ExecutedToolCallIDs = make(map[string]bool)
	}

	if len(lc.ExecutedToolCallIDs) == 0 {
		assistantMsg := session.Message{
			ID:         session.GenerateID("msg"),
			Role:       session.RoleAssistant,
			Content:    lc.LLMResult.Text,
			Reasoning:  lc.LLMResult.Reasoning,
			ToolCalls:  lc.LLMResult.ToolCalls,
			CreatedAt:  time.Now(),
		}
		if err := l.store.AddMessage(lc.SessionID, assistantMsg); err != nil {
			return LoopStateError, fmt.Errorf("add assistant message with tool calls: %w", err)
		}
	}

	sess, err := l.store.Get(lc.SessionID)
	if err != nil {
		logging.Error(ctx, "tool executing handler: failed to get session",
			"error", err,
		)
	} else {
		oldState := string(sess.State)
		logging.Debug(ctx, "tool executing handler: setting state to ToolExecuting",
			"session_id", lc.SessionID,
			"old_state", oldState,
		)
		sess.State = session.StateToolExecuting
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			logging.Error(ctx, "tool executing handler: failed to update state to ToolExecuting",
				"error", err,
			)
		} else {
			logging.Debug(ctx, "tool executing handler: updated state",
				"old_state", oldState,
				"new_state", "ToolExecuting",
			)
		}
		lc.EventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateToolExecuting.ToSnakeCase(),
			"reason":    "tool_execution_started",
		})
	}

	if sess.MaxConcurrentToolCalls > 1 && len(lc.LLMResult.ToolCalls) > 1 {
		return l.executeToolsParallel(ctx, lc, sess.MaxConcurrentToolCalls)
	}

	return l.executeToolsSerial(ctx, lc)
}

func (l *Loop) textResponseHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
	assistantMsg := session.Message{
		ID:         session.GenerateID("msg"),
		Role:       session.RoleAssistant,
		Content:    lc.LLMResult.Text,
		Reasoning:  lc.LLMResult.Reasoning,
		CreatedAt:  time.Now(),
	}
	if err := l.store.AddMessage(lc.SessionID, assistantMsg); err != nil {
		return LoopStateError, fmt.Errorf("add assistant message: %w", err)
	}

	l.archiveManager.AppendAssistantMessageWithReasoning(lc.SessionID, lc.LLMResult.Text, lc.LLMResult.Reasoning)

	msgCompleteData := map[string]any{
		"message_id":        assistantMsg.ID,
		"full_text":         lc.LLMResult.Text,
		"total_step_tokens": lc.TotalStepTokens,
	}
	if lc.LLMResult.Reasoning != "" {
		msgCompleteData["full_reasoning"] = lc.LLMResult.Reasoning
	}
	if lc.LLMResult.TokenUsage != nil {
		msgCompleteData["input_tokens"] = lc.LLMResult.TokenUsage.InputTokens
		msgCompleteData["output_tokens"] = lc.LLMResult.TokenUsage.OutputTokens
		if lc.LLMResult.TokenUsage.ReasoningTokens > 0 {
			msgCompleteData["reasoning_tokens"] = lc.LLMResult.TokenUsage.ReasoningTokens
		}
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

	lc.TerminationReason = "completed"

	return LoopStateIdle, nil
}
