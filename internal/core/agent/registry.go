package agent

import (
	"devo/internal/taskexec/tools"
	"fmt"
)

type Registry struct {
	agents       map[string]*Agent
	defaultAgent *Agent
	builtinIDs   map[string]bool
}

func NewRegistry(defaultAgent *Agent) *Registry {
	r := &Registry{
		agents:       make(map[string]*Agent),
		defaultAgent: defaultAgent,
		builtinIDs:   make(map[string]bool),
	}
	r.Register(defaultAgent)
	return r
}

func (r *Registry) Register(agent *Agent) {
	r.agents[agent.Config.ID] = agent
	if agent.Config.Builtin {
		r.builtinIDs[agent.Config.ID] = true
	}
}

func (r *Registry) Get(agentID string) *Agent {
	if agentID == "" {
		return r.defaultAgent
	}
	if agent, ok := r.agents[agentID]; ok {
		return agent
	}
	return r.defaultAgent
}

func (r *Registry) Exists(agentID string) bool {
	if agentID == "" {
		return true
	}
	_, ok := r.agents[agentID]
	return ok
}

func (r *Registry) DefaultAgent() *Agent {
	return r.defaultAgent
}

func (r *Registry) GetSubAgent(agentID string) tools.SubAgent {
	return r.Get(agentID)
}

func (r *Registry) DefaultAgentID() string {
	if r.defaultAgent != nil {
		return r.defaultAgent.Config.ID
	}
	return ""
}

func (r *Registry) List() []*Agent {
	result := make([]*Agent, 0, len(r.agents))
	for _, a := range r.agents {
		result = append(result, a)
	}
	return result
}

func (r *Registry) Unregister(id string) error {
	if id == "" {
		return fmt.Errorf("agent ID is required")
	}
	if r.defaultAgent != nil && r.defaultAgent.Config.ID == id {
		return fmt.Errorf("cannot unregister the default agent")
	}
	if r.builtinIDs[id] {
		return fmt.Errorf("cannot unregister builtin agent %q", id)
	}
	if _, ok := r.agents[id]; !ok {
		return fmt.Errorf("agent %q not found", id)
	}
	delete(r.agents, id)
	return nil
}
