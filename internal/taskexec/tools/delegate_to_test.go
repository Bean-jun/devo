package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"devo/internal/config"
	"devo/internal/core/session"
)

type mockSubAgent struct {
	id        string
	name      string
	subAgent  bool
	processFn func(ctx context.Context, sessionID string, msg session.Message) error
}

func (m *mockSubAgent) ID() string       { return m.id }
func (m *mockSubAgent) Name() string     { return m.name }
func (m *mockSubAgent) IsSubAgent() bool { return m.subAgent }
func (m *mockSubAgent) ProcessMessage(ctx context.Context, sessionID string, msg session.Message) error {
	if m.processFn != nil {
		return m.processFn(ctx, sessionID, msg)
	}
	return nil
}

type mockSubAgentProvider struct {
	agents    map[string]SubAgent
	defaultID string
}

func (m *mockSubAgentProvider) GetSubAgent(agentID string) SubAgent {
	return m.agents[agentID]
}

func (m *mockSubAgentProvider) DefaultAgentID() string {
	return m.defaultID
}

type mockStreamWriter struct {
	chunks []string
	metas  []string
}

func (m *mockStreamWriter) WriteChunk(data string) {
	m.chunks = append(m.chunks, data)
}

func (m *mockStreamWriter) WriteMeta(stage string) {
	m.metas = append(m.metas, stage)
}

func (m *mockStreamWriter) WriteDone(success bool, summary string) {}
func (m *mockStreamWriter) WriteError(err error)                   {}

func TestDelegateToTool_MissingAgentID(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents:    make(map[string]SubAgent),
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"task": "review code",
	}, w)

	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id is required") {
		t.Errorf("expected 'agent_id is required', got: %v", err)
	}
}

func TestDelegateToTool_MissingTask(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents:    make(map[string]SubAgent),
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"agent_id": "code-reviewer",
	}, w)

	if err == nil {
		t.Fatal("expected error for missing task")
	}
	if !strings.Contains(err.Error(), "task is required") {
		t.Errorf("expected 'task is required', got: %v", err)
	}
}

func TestDelegateToTool_UnknownSubAgent(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents:    make(map[string]SubAgent),
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"agent_id": "nonexistent",
		"task":     "review code",
	}, w)

	if err == nil {
		t.Fatal("expected error for unknown sub-agent")
	}
	if !strings.Contains(err.Error(), "unknown sub-agent") {
		t.Errorf("expected 'unknown sub-agent', got: %v", err)
	}
}

func TestDelegateToTool_CannotDelegateToDefault(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents: map[string]SubAgent{
			"devo-default": &mockSubAgent{id: "devo-default", name: "Devo"},
		},
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"agent_id": "devo-default",
		"task":     "review code",
	}, w)

	if err == nil {
		t.Fatal("expected error for delegating to default agent")
	}
	if !strings.Contains(err.Error(), "cannot delegate to the default agent") {
		t.Errorf("expected 'cannot delegate to the default agent', got: %v", err)
	}
}

func TestDelegateToTool_CannotDelegateToSubAgent(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents: map[string]SubAgent{
			"devo-default": &mockSubAgent{id: "devo-default", name: "Devo"},
			"nested":       &mockSubAgent{id: "nested", name: "Nested", subAgent: true},
		},
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"agent_id": "nested",
		"task":     "review code",
	}, w)

	if err == nil {
		t.Fatal("expected error for delegating to a sub-agent")
	}
	if !strings.Contains(err.Error(), "cannot delegate to a sub-agent") {
		t.Errorf("expected 'cannot delegate to a sub-agent', got: %v", err)
	}
}

func TestDelegateToTool_NoActiveSession(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents: map[string]SubAgent{
			"devo-default":  &mockSubAgent{id: "devo-default", name: "Devo"},
			"code-reviewer": &mockSubAgent{id: "code-reviewer", name: "Reviewer"},
		},
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"agent_id": "code-reviewer",
		"task":     "review code",
	}, w)

	if err == nil {
		t.Fatal("expected error for no active session")
	}
	if !strings.Contains(err.Error(), "no active session") {
		t.Errorf("expected 'no active session', got: %v", err)
	}
}

func TestDelegateToTool_SuccessfulDelegation(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents: map[string]SubAgent{
			"devo-default": &mockSubAgent{id: "devo-default", name: "Devo"},
			"code-reviewer": &mockSubAgent{
				id:   "code-reviewer",
				name: "Code Reviewer",
				processFn: func(ctx context.Context, sessionID string, msg session.Message) error {
					store.AddMessage(sessionID, session.Message{
						ID:        session.GenerateID("msg"),
						Role:      session.RoleAssistant,
						Content:   "Code review complete: looks good!",
						CreatedAt: time.Now(),
					})
					sess, _ := store.Get(sessionID)
					sess.State = session.StateIdle
					store.Update(sess)
					return nil
				},
			},
		},
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	w := &mockStreamWriter{}
	err := tool.Execute(context.Background(), "/tmp", map[string]interface{}{
		"agent_id": "code-reviewer",
		"task":     "review auth.go",
	}, w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(w.metas) < 2 {
		t.Errorf("expected at least 2 metas, got %d: %v", len(w.metas), w.metas)
	}

	if len(w.chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	if !strings.Contains(w.chunks[0], "Code review complete") {
		t.Errorf("expected 'Code review complete', got: %v", w.chunks[0])
	}
}

func TestDelegateToTool_SubAgentTimeout(t *testing.T) {
	store := session.NewInMemoryStore()
	provider := &mockSubAgentProvider{
		agents: map[string]SubAgent{
			"devo-default": &mockSubAgent{id: "devo-default", name: "Devo"},
			"code-reviewer": &mockSubAgent{
				id:   "code-reviewer",
				name: "Code Reviewer",
				processFn: func(ctx context.Context, sessionID string, msg session.Message) error {
					sess, _ := store.Get(sessionID)
					sess.State = session.StateThinking
					store.Update(sess)
					return nil
				},
			},
		},
		defaultID: "devo-default",
	}

	tool := NewDelegateToTool(provider, store, config.DefaultConfig())
	tool.SetActiveSessionID("sess-1")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	w := &mockStreamWriter{}
	err := tool.Execute(ctx, "/tmp", map[string]interface{}{
		"agent_id": "code-reviewer",
		"task":     "review auth.go",
	}, w)

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDelegateToTool_Name(t *testing.T) {
	tool := NewDelegateToTool(nil, nil, nil)
	if tool.Name() != "delegate_to" {
		t.Errorf("expected 'delegate_to', got %s", tool.Name())
	}
}

func TestDelegateToTool_Description(t *testing.T) {
	tool := NewDelegateToTool(nil, nil, nil)
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestDelegateToTool_RiskLevel(t *testing.T) {
	tool := NewDelegateToTool(nil, nil, nil)
	if tool.RiskLevel() != RiskLevelHigh {
		t.Errorf("expected RiskLevelHigh, got %v", tool.RiskLevel())
	}
}

func TestDelegateToTool_ParamsSchema(t *testing.T) {
	tool := NewDelegateToTool(nil, nil, nil)
	schema := tool.ParamsSchema()
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties")
	}
	if props["agent_id"] == nil {
		t.Error("expected agent_id in properties")
	}
	if props["task"] == nil {
		t.Error("expected task in properties")
	}
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("expected required to be []string")
	}
	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(required))
	}
}
