package providers

import (
	"encoding/json"
	"log"
	"os"

	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"
)

func NewClientFromEnv(registry *tools.Registry) llmclient.Client {
	apiKey := os.Getenv("DEVO_LLM_API_KEY")
	if apiKey == "" {
		log.Println("DEVO_LLM_API_KEY not set, using MockClient")
		return llmclient.NewMockClient()
	}

	baseURL := os.Getenv("DEVO_LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("DEVO_LLM_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	headers := parseExtraHeaders(os.Getenv("DEVO_LLM_EXTRA_HEADERS"))

	client := openai.New(openai.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Headers: headers,
	})

	client.SetTools(buildToolDefinitions(registry))

	log.Printf("Using OpenAI-compatible LLM: base_url=%s, model=%s", baseURL, model)
	return client
}

func parseExtraHeaders(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		log.Printf("failed to parse DEVO_LLM_EXTRA_HEADERS: %v", err)
		return nil
	}
	return headers
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
