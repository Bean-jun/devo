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

	client := openai.New(openai.Config{
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
		Model:   cfg.LLM.Model,
		Headers: cfg.LLM.ExtraHeaders,
	}, registry)

	logging.Info(context.Background(), "using llm",
		"base_url", cfg.LLM.BaseURL,
		"model", cfg.LLM.Model,
	)
	return client
}
