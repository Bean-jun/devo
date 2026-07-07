package agentloop

import (
	"context"
	"fmt"
	"log"
	"time"

	"devo/internal/core/compressor"
	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
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
			log.Printf("[DEBUG] thinkingHandler got done for session=%s, ToolCalls=%d", lc.SessionID, len(evt.ToolCalls))
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
	log.Printf("[DEBUG] evaluatingResultHandler called for session=%s, ToolCalls=%d", lc.SessionID, len(lc.LLMResult.ToolCalls))
	if len(lc.LLMResult.ToolCalls) > 0 {
		lc.ExecutedToolCallIDs = make(map[string]bool)
		lc.EventBus.Publish("loop.result_evaluated", map[string]string{"type": "tool_calls"})

		sess, err := l.store.Get(lc.SessionID)
		if err != nil {
			log.Printf("[DEBUG] evaluatingResultHandler: failed to get session: %v", err)
		} else {
			oldState := string(sess.State)
			log.Printf("[DEBUG] evaluatingResultHandler: current state=%s, setting to ToolExecuting", oldState)
			sess.State = session.StateToolExecuting
			sess.LastActiveAt = time.Now()
			if err := l.store.Update(sess); err != nil {
				log.Printf("[DEBUG] evaluatingResultHandler: failed to update state to ToolExecuting: %v", err)
			} else {
				log.Printf("[DEBUG] evaluatingResultHandler: updated state from %s to ToolExecuting", oldState)
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
	log.Printf("[DEBUG] toolExecutingHandler called for session=%s, ToolCalls=%d", lc.SessionID, len(lc.LLMResult.ToolCalls))
	if lc.ExecutedToolCallIDs == nil {
		lc.ExecutedToolCallIDs = make(map[string]bool)
	}

	if len(lc.ExecutedToolCallIDs) == 0 {
		assistantMsg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleAssistant,
			Content:   lc.LLMResult.Text,
			ToolCalls: lc.LLMResult.ToolCalls,
			CreatedAt: time.Now(),
		}
		if err := l.store.AddMessage(lc.SessionID, assistantMsg); err != nil {
			return LoopStateError, fmt.Errorf("add assistant message with tool calls: %w", err)
		}
	}

	sess, err := l.store.Get(lc.SessionID)
	if err != nil {
		log.Printf("[DEBUG] toolExecutingHandler: failed to get session: %v", err)
	} else {
		oldState := string(sess.State)
		log.Printf("[DEBUG] toolExecutingHandler: current state=%s, setting to ToolExecuting", oldState)
		sess.State = session.StateToolExecuting
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			log.Printf("[DEBUG] toolExecutingHandler: failed to update state to ToolExecuting: %v", err)
		} else {
			log.Printf("[DEBUG] toolExecutingHandler: updated state from %s to ToolExecuting", oldState)
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
