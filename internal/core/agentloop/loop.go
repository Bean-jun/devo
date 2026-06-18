package agentloop

import (
	"context"
	"fmt"
	"log"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

const defaultSystemPrompt = "You are a helpful coding assistant. Respond concisely and helpfully."

type Loop struct {
	store        session.SessionStore
	llmClient    llmclient.Client
	systemPrompt string
}

func New(store session.SessionStore, llmClient llmclient.Client) *Loop {
	return &Loop{
		store:        store,
		llmClient:    llmClient,
		systemPrompt: defaultSystemPrompt,
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

	msgs, _, err := l.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("get messages: %w", err))
		return
	}

	reply, err := l.llmClient.Complete(ctx, msgs, l.systemPrompt)
	if err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("llm complete: %w", err))
		return
	}

	assistantMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
	}
	if err := l.store.AddMessage(sessionID, assistantMsg); err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("add assistant message: %w", err))
		return
	}

	eventBus.Publish("message_complete", map[string]any{
		"message_id":        assistantMsg.ID,
		"full_text":         reply,
		"total_step_tokens": nil,
	})

	sess, err := l.store.Get(sessionID)
	if err != nil {
		l.handleLoopError(sessionID, fmt.Errorf("get session for state update: %w", err))
		return
	}

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
