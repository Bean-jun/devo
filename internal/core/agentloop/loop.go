package agentloop

import (
	"context"
	"fmt"
	"log"
	"time"

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

type Loop struct {
	store        session.SessionStore
	llmClient    llmclient.Client
	systemPrompt string
	toolExecutor ToolExecutor
}

func New(store session.SessionStore, llmClient llmclient.Client) *Loop {
	return &Loop{
		store:        store,
		llmClient:    llmClient,
		systemPrompt: defaultSystemPrompt,
	}
}

func NewWithTools(store session.SessionStore, llmClient llmclient.Client, toolExecutor ToolExecutor) *Loop {
	return &Loop{
		store:        store,
		llmClient:    llmClient,
		systemPrompt: defaultSystemPrompt,
		toolExecutor: toolExecutor,
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

				eventBus.Publish("tool_call_request", map[string]any{
					"tool_name":         tc.ToolName,
					"params":            tc.Params,
					"requires_approval": false,
					"risk_level":        "-",
				})

				toolResult, err := l.toolExecutor.Execute(sess.WorkingDirectory, tc.ToolName, tc.Params)
				if err != nil {
					eventBus.Publish("tool_result", map[string]any{
						"tool_name": tc.ToolName,
						"success":   false,
						"summary":   fmt.Sprintf("error: %v", err),
					})
					continue
				}

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
