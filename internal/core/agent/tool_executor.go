package agent

import (
	"context"

	"devo/internal/config"
	"devo/internal/core/agentloop"
	"devo/internal/taskexec/tools"
)

type dynamicToolExecutor struct {
	registry *tools.Registry
	agentCfg Config
	appCfg   *config.Config
}

func newDynamicToolExecutor(registry *tools.Registry, agentCfg Config, appCfg *config.Config) agentloop.ToolExecutor {
	return &dynamicToolExecutor{
		registry: registry,
		agentCfg: agentCfg,
		appCfg:   appCfg,
	}
}

func (d *dynamicToolExecutor) ListTools() []tools.Tool {
	allTools := d.registry.ListTools()

	if d.agentCfg.Tools == nil {
		if d.shouldFilterDelegateTo() {
			return d.filterOut(allTools)
		}
		return allTools
	}

	allowSet := make(map[string]bool, len(d.agentCfg.Tools))
	for _, name := range d.agentCfg.Tools {
		allowSet[name] = true
	}

	result := make([]tools.Tool, 0, len(d.agentCfg.Tools))
	for _, t := range allTools {
		if allowSet[t.Name()] {
			if d.shouldFilterDelegateTo() && t.Name() == "delegate_to" {
				continue
			}
			result = append(result, t)
		}
	}
	return result
}

func (d *dynamicToolExecutor) GetTool(name string) (tools.Tool, bool) {
	if d.agentCfg.Tools != nil {
		allowed := false
		for _, n := range d.agentCfg.Tools {
			if n == name {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, false
		}
	}

	if d.shouldFilterDelegateTo() && name == "delegate_to" {
		return nil, false
	}

	return d.registry.GetTool(name)
}

func (d *dynamicToolExecutor) Execute(ctx context.Context, workingDir string, toolName string, params map[string]interface{}) (<-chan tools.StreamEvent, error) {
	return d.registry.Execute(ctx, workingDir, toolName, params)
}

func (d *dynamicToolExecutor) shouldFilterDelegateTo() bool {
	return !d.appCfg.TeamMode || d.agentCfg.SubAgentOf != ""
}

func (d *dynamicToolExecutor) filterOut(allTools []tools.Tool) []tools.Tool {
	result := make([]tools.Tool, 0, len(allTools))
	for _, t := range allTools {
		if t.Name() != "delegate_to" {
			result = append(result, t)
		}
	}
	return result
}
