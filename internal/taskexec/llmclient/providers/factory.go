package providers

import (
	"context"
	"devo/internal/config"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"
)

func NewClient(cfg *config.GlobalConfig, registry *tools.Registry) llmclient.Client {
	if cfg.LLM.APIKey == "" {
		logging.Info(context.Background(), "llm api key not configured, using mock client")
		return llmclient.NewMockClient()
	}

	var reasoningEffort string
	if cfg.LLM.EnableReasoning {
		reasoningEffort = cfg.LLM.ReasoningEffort
	}

	client := openai.New(openai.Config{
		LLMConfig: &cfg.LLM,
	}, registry)

	logging.Info(context.Background(), "using llm",
		"base_url", cfg.LLM.BaseURL,
		"model", cfg.LLM.Model,
		"reasoning_enabled", cfg.LLM.EnableReasoning,
		"reasoning_effort", reasoningEffort,
	)
	return client
}
