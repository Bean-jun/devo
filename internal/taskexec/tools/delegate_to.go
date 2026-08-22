package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"devo/internal/config"
	"devo/internal/core/session"
)

type SubAgentProvider interface {
	GetSubAgent(agentID string) SubAgent
	DefaultAgentID() string
}

type SubAgent interface {
	ID() string
	Name() string
	IsSubAgent() bool
	ProcessMessage(ctx context.Context, sessionID string, msg session.Message) error
}

type DelegateToTool struct {
	subAgentProvider SubAgentProvider
	sessionStore     session.SessionStore
	appCfg           *config.Config

	mu              sync.Mutex
	activeSessionID string
}

func NewDelegateToTool(
	subAgentProvider SubAgentProvider,
	sessionStore session.SessionStore,
	appCfg *config.Config,
) *DelegateToTool {
	return &DelegateToTool{
		subAgentProvider: subAgentProvider,
		sessionStore:     sessionStore,
		appCfg:           appCfg,
	}
}

func (t *DelegateToTool) SetProvider(provider SubAgentProvider) {
	t.subAgentProvider = provider
}

func (t *DelegateToTool) SetActiveSessionID(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activeSessionID = sessionID
}

func (t *DelegateToTool) Name() string {
	return "delegate_to"
}

func (t *DelegateToTool) Description() string {
	return `Delegate a subtask to a specialized sub-agent.
Use this when you need a different perspective or expertise for a specific task.
Available sub-agents: code-reviewer, architect, test-writer.
The sub-agent will work independently with its own context and return its findings to you.`
}

func (t *DelegateToTool) RiskLevel() RiskLevel {
	return RiskLevelHigh
}

func (t *DelegateToTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the sub-agent to delegate to (e.g., 'code-reviewer', 'architect', 'test-writer')",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The task description for the sub-agent. Be specific about what you want the sub-agent to do.",
			},
		},
		"required": []string{"agent_id", "task"},
	}
}

func (t *DelegateToTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	agentID, ok := params["agent_id"].(string)
	if !ok || agentID == "" {
		return fmt.Errorf("agent_id is required and must be a string")
	}

	task, ok := params["task"].(string)
	if !ok || task == "" {
		return fmt.Errorf("task is required and must be a string")
	}

	t.mu.Lock()
	parentSessionID := t.activeSessionID
	t.mu.Unlock()

	if parentSessionID == "" {
		return fmt.Errorf("no active session")
	}

	subAgent := t.subAgentProvider.GetSubAgent(agentID)
	if subAgent == nil {
		return fmt.Errorf("unknown sub-agent: %s", agentID)
	}

	if subAgent.ID() == t.subAgentProvider.DefaultAgentID() {
		return fmt.Errorf("cannot delegate to the default agent")
	}

	if subAgent.IsSubAgent() {
		return fmt.Errorf("cannot delegate to a sub-agent: %s", agentID)
	}

	subSessionID := fmt.Sprintf("%s-sub-%d", parentSessionID, time.Now().UnixNano())
	subSession := &session.Session{
		ID:               subSessionID,
		AgentID:          agentID,
		WorkingDirectory: workingDir,
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
	}
	if err := t.sessionStore.Create(subSession); err != nil {
		return fmt.Errorf("failed to create sub-session: %w", err)
	}

	w.WriteMeta(fmt.Sprintf("delegating to %s (%s)", subAgent.Name(), agentID))

	msg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleUser,
		Content:   task,
		CreatedAt: time.Now(),
	}

	if err := subAgent.ProcessMessage(ctx, subSessionID, msg); err != nil {
		t.sessionStore.DeleteSession(subSessionID)
		return fmt.Errorf("sub-agent %s failed to start: %w", agentID, err)
	}

	result, err := t.waitForCompletion(ctx, subSessionID)
	t.sessionStore.DeleteSession(subSessionID)

	if err != nil {
		return fmt.Errorf("sub-agent %s failed: %w", agentID, err)
	}

	w.WriteMeta("result")
	w.WriteChunk(result)
	return nil
}

func (t *DelegateToTool) waitForCompletion(ctx context.Context, sessionID string) (string, error) {
	timeout := 120 * time.Second
	deadline := time.Now().Add(timeout)
	pollInterval := 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("sub-agent timed out after %v", timeout)
		}

		sess, err := t.sessionStore.Get(sessionID)
		if err != nil {
			return "", fmt.Errorf("get sub-session: %w", err)
		}

		switch sess.State {
		case session.StateIdle, session.StateCompleted:
			msgs, _, err := t.sessionStore.GetMessages(sessionID, 0, 0)
			if err != nil {
				return "", fmt.Errorf("get sub-session messages: %w", err)
			}
			var result string
			for _, msg := range msgs {
				if msg.Role == session.RoleAssistant {
					result += msg.Content + "\n"
				}
			}
			if result == "" {
				return "Sub-agent completed with no output.", nil
			}
			return result, nil

		case session.StateAwaitingApproval:
			return "", fmt.Errorf("sub-agent requires approval, which is not supported in delegated mode")

		case session.StatePaused:
			return "", fmt.Errorf("sub-agent is paused")
		}

		time.Sleep(pollInterval)
	}
}
