package agentloop

import (
	"context"
	"fmt"
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

func (l *Loop) ProcessMessage(ctx context.Context, sessionID, content string) (*session.Message, error) {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if sess.State != session.StateIdle {
		return nil, fmt.Errorf("%w: current state is %s", session.ErrSessionNotIdle, sess.State)
	}

	sess.State = session.StateProcessing
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return nil, fmt.Errorf("update session state to processing: %w", err)
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
		return nil, fmt.Errorf("add user message: %w", err)
	}

	msgs, _, err := l.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return nil, fmt.Errorf("get messages for llm: %w", err)
	}

	reply, err := l.llmClient.Complete(ctx, msgs, l.systemPrompt)
	if err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	assistantMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
	}
	if err := l.store.AddMessage(sessionID, assistantMsg); err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	sess.State = session.StateIdle
	sess.LastActiveAt = time.Now()
	if err := l.store.Update(sess); err != nil {
		return nil, fmt.Errorf("update session state to idle: %w", err)
	}

	return &assistantMsg, nil
}
