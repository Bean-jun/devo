package agent

type Registry struct {
	agents       map[string]*Agent
	defaultAgent *Agent
}

func NewRegistry(defaultAgent *Agent) *Registry {
	r := &Registry{
		agents:       make(map[string]*Agent),
		defaultAgent: defaultAgent,
	}
	r.Register(defaultAgent)
	return r
}

func (r *Registry) Register(agent *Agent) {
	r.agents[agent.Config.ID] = agent
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

func (r *Registry) DefaultAgent() *Agent {
	return r.defaultAgent
}
