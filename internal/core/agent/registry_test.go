package agent

import (
	"testing"

	"devo/internal/config"
	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/core/skills"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/tools"
)

func newTestAgent(t *testing.T, id, name string) *Agent {
	t.Helper()
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	llm := llmclient.NewMockClient()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(llm, skillsMgr, store)

	return New(Config{
		ID:           id,
		Name:         name,
		Description:  "Test agent",
		SystemPrompt: "You are a test agent.",
		Tools:        nil,
	}, store, llm, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)
}

func TestRegistry_NewRegistry(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.DefaultAgent() == nil {
		t.Error("expected non-nil default agent")
	}
	if r.DefaultAgent().Config.ID != "devo-default" {
		t.Errorf("expected default agent ID 'devo-default', got %q", r.DefaultAgent().Config.ID)
	}
}

func TestRegistry_Get_Default(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	agent := r.Get("")
	if agent == nil {
		t.Fatal("expected non-nil agent for empty ID")
	}
	if agent.Config.ID != "devo-default" {
		t.Errorf("expected 'devo-default', got %q", agent.Config.ID)
	}
}

func TestRegistry_Get_ByID(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	otherAgent := newTestAgent(t, "python-expert", "Python Expert")
	r.Register(otherAgent)

	agent := r.Get("python-expert")
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.Config.ID != "python-expert" {
		t.Errorf("expected 'python-expert', got %q", agent.Config.ID)
	}
	if agent.Config.Name != "Python Expert" {
		t.Errorf("expected 'Python Expert', got %q", agent.Config.Name)
	}
}

func TestRegistry_Get_UnknownFallback(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	agent := r.Get("unknown-agent")
	if agent == nil {
		t.Fatal("expected non-nil agent (fallback to default)")
	}
	if agent.Config.ID != "devo-default" {
		t.Errorf("expected fallback to 'devo-default', got %q", agent.Config.ID)
	}
}

func TestRegistry_Register(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	agent1 := newTestAgent(t, "agent-1", "Agent 1")
	agent2 := newTestAgent(t, "agent-2", "Agent 2")

	r.Register(agent1)
	r.Register(agent2)

	if ag := r.Get("agent-1"); ag.Config.ID != "agent-1" {
		t.Errorf("expected 'agent-1', got %q", ag.Config.ID)
	}
	if ag := r.Get("agent-2"); ag.Config.ID != "agent-2" {
		t.Errorf("expected 'agent-2', got %q", ag.Config.ID)
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	agent1 := newTestAgent(t, "custom", "Custom V1")
	agent2 := newTestAgent(t, "custom", "Custom V2")

	r.Register(agent1)
	r.Register(agent2)

	agent := r.Get("custom")
	if agent.Config.Name != "Custom V2" {
		t.Errorf("expected 'Custom V2' after overwrite, got %q", agent.Config.Name)
	}
}

func TestRegistry_DefaultAgent(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	da := r.DefaultAgent()
	if da == nil {
		t.Fatal("expected non-nil default agent")
	}
	if da.Config.ID != "devo-default" {
		t.Errorf("expected 'devo-default', got %q", da.Config.ID)
	}
}

func TestRegistry_MultipleAgents(t *testing.T) {
	defaultAgent := newTestAgent(t, "devo-default", "Devo")
	r := NewRegistry(defaultAgent)

	agents := []string{"python-expert", "code-reviewer", "devops-helper"}
	for _, id := range agents {
		r.Register(newTestAgent(t, id, id))
	}

	for _, id := range agents {
		agent := r.Get(id)
		if agent == nil {
			t.Errorf("expected non-nil agent for %q", id)
			continue
		}
		if agent.Config.ID != id {
			t.Errorf("expected %q, got %q", id, agent.Config.ID)
		}
	}

	agent := r.Get("")
	if agent.Config.ID != "devo-default" {
		t.Errorf("expected default agent for empty ID, got %q", agent.Config.ID)
	}
}
