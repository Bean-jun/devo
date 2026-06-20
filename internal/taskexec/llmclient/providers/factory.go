package providers

import (
	"log"

	"devo/internal/config"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"
)

func NewClient(cfg *config.Config, registry *tools.Registry) llmclient.Client {
	if cfg.LLM.APIKey == "" {
		log.Println("[devo] LLM API key not configured, using MockClient")
		return llmclient.NewMockClient()
	}

	client := openai.New(openai.Config{
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
		Model:   cfg.LLM.Model,
		Headers: cfg.LLM.ExtraHeaders,
	})

	client.SetTools(buildToolDefinitions(registry))

	log.Printf("[devo] Using LLM: base_url=%s, model=%s", cfg.LLM.BaseURL, cfg.LLM.Model)
	return client
}

func buildToolDefinitions(registry *tools.Registry) []llmclient.ToolDefinition {
	toolList := registry.ListTools()
	defs := make([]llmclient.ToolDefinition, 0, len(toolList))
	for _, t := range toolList {
		defs = append(defs, llmclient.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Params:      t.ParamsSchema(),
		})
	}
	return defs
}
