package providers

import (
	"context"
	"devo/internal/config"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"
)

func NewClient(cfg *config.Config, registry *tools.Registry) llmclient.Client {
	active := cfg.GetActiveModel()
	if active == nil || active.APIKey == "" {
		logging.Info(context.Background(), "llm api key not configured, using mock client")
		return llmclient.NewMockClient()
	}

	return newClientFromModel(active, registry)
}

func NewClientForModel(cfg *config.Config, modelID string, registry *tools.Registry) llmclient.Client {
	for i := range cfg.LLM.Models {
		if cfg.LLM.Models[i].ID == modelID {
			return newClientFromModel(&cfg.LLM.Models[i], registry)
		}
	}
	logging.Info(context.Background(), "model not found, falling back to mock client",
		"model_id", modelID,
	)
	return llmclient.NewMockClient()
}

func newClientFromModel(active *config.ModelConfig, registry *tools.Registry) llmclient.Client {
	llmCfg := &config.LLMConfig{
		APIKey:          active.APIKey,
		BaseURL:         active.BaseURL,
		Model:           active.Model,
		ExtraHeaders:    active.ExtraHeaders,
		EnableReasoning: active.EnableReasoning,
		ReasoningEffort: active.ReasoningEffort,
		MaxTokens:       active.MaxTokens,
	}

	var reasoningEffort string
	if active.EnableReasoning {
		reasoningEffort = active.ReasoningEffort
	}

	client := openai.New(openai.Config{
		LLMConfig: llmCfg,
	}, registry)

	logging.Info(context.Background(), "using llm",
		"model_id", active.ID,
		"model_name", active.Name,
		"base_url", active.BaseURL,
		"model", active.Model,
		"reasoning_enabled", active.EnableReasoning,
		"reasoning_effort", reasoningEffort,
	)
	return client
}
