package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ModelConfig struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Provider        string            `json:"provider,omitempty"`
	APIKey          string            `json:"api_key"`
	BaseURL         string            `json:"base_url"`
	Model           string            `json:"model"`
	ExtraHeaders    map[string]string `json:"extra_headers,omitempty"`
	EnableReasoning bool              `json:"enable_reasoning,omitempty"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	MaxTokens       int               `json:"max_tokens,omitempty"`
}

type LLMConfig struct {
	APIKey          string            `json:"api_key,omitempty"`
	BaseURL         string            `json:"base_url,omitempty"`
	Model           string            `json:"model,omitempty"`
	ExtraHeaders    map[string]string `json:"extra_headers,omitempty"`
	EnableReasoning bool              `json:"enable_reasoning,omitempty"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	MaxTokens       int               `json:"max_tokens,omitempty"`

	Models        []ModelConfig `json:"models,omitempty"`
	ActiveModelID string        `json:"active_model_id,omitempty"`
}

type Config struct {
	LLM                       LLMConfig         `json:"llm,omitempty"`
	DBPath                    string            `json:"db_path,omitempty"`
	LogPath                   string            `json:"log_path,omitempty"`
	LogLevel                  string            `json:"log_level,omitempty"`
	ApprovalPolicy            map[string]string `json:"approval_policy,omitempty"`
	Skills                    []string          `json:"skills,omitempty"`
	MCP                       []string          `json:"mcp,omitempty"`
	ToolCallLimit             int               `json:"tool_call_limit,omitempty"`
	MaxContextTokens          int               `json:"max_context_tokens,omitempty"`
	KeepRecent                int               `json:"keep_recent,omitempty"`
	MaxConcurrentToolCalls    int               `json:"max_concurrent_tool_calls,omitempty"`
	MaxConcurrentSubprocesses int               `json:"max_concurrent_subprocesses,omitempty"`
}

type GlobalConfig = Config

func LoadGlobal() (*Config, error) {
	cfg := &Config{}

	globalConfigPath := GlobalConfigPath()

	if _, err := os.Stat(globalConfigPath); err == nil {
		data, err := os.ReadFile(globalConfigPath)
		if err != nil {
			return nil, fmt.Errorf("read config file %s: %w", globalConfigPath, err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file %s: %w", globalConfigPath, err)
		}
	}

	applyEnvOverrides(cfg)
	applyDefaults(cfg)
	migrateLegacyConfig(cfg)
	return cfg, nil
}

func LoadFullConfig(projectDir string) (*Config, error) {
	cfg, err := LoadGlobal()
	if err != nil {
		return nil, err
	}

	pcfg, err := LoadProjectConfig(projectDir)
	if err == nil {
		cfg = Merge(cfg, pcfg)
	}

	applyEnvOverrides(cfg)
	applyDefaults(cfg)

	if cfg.LLM.APIKey == "" && len(cfg.LLM.Models) == 0 {
		return cfg, fmt.Errorf(
			"LLM API key not configured.\n\n" +
				"Create a config file at one of these locations:\n" +
				"  ./.devo/config.json\n" +
				"  ~/.devo/config.json\n\n" +
				"Example:\n" +
				"{\n" +
				"  \"llm\": {\n" +
				"    \"api_key\": \"sk-your-key-here\",\n" +
				"    \"base_url\": \"https://api.openai.com/v1\",\n" +
				"    \"model\": \"gpt-4o\"\n" +
				"  }\n" +
				"}\n\n" +
				"Or set the environment variable: DEVO_LLM_API_KEY",
		)
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DEVO_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("DEVO_LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("DEVO_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("DEVO_LLM_EXTRA_HEADERS"); v != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(v), &headers); err == nil {
			cfg.LLM.ExtraHeaders = headers
		}
	}
	if v := os.Getenv("DEVO_LLM_ENABLE_REASONING"); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			cfg.LLM.EnableReasoning = true
		case "0", "false", "no", "off":
			cfg.LLM.EnableReasoning = false
		}
	}
	if v := os.Getenv("DEVO_LLM_REASONING_EFFORT"); v != "" {
		cfg.LLM.ReasoningEffort = v
	}
	if v := os.Getenv("DEVO_LLM_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.LLM.MaxTokens = n
		}
	}
	if v := os.Getenv("DEVO_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("DEVO_LOG_PATH"); v != "" {
		cfg.LogPath = v
	}
	if v := os.Getenv("DEVO_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
}

func parseBool(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func applyDefaults(cfg *Config) {
	ApplyDefaults(cfg)
}

func ApplyDefaults(cfg *Config) {
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = DefaultLLMBaseURL
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = DefaultLLMModel
	}
	if cfg.LLM.EnableReasoning && cfg.LLM.ReasoningEffort == "" {
		cfg.LLM.ReasoningEffort = DefaultReasoningEffort
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = DefaultMaxTokens
	}

	if cfg.ToolCallLimit == 0 {
		cfg.ToolCallLimit = DefaultToolCallLimit
	}
	if cfg.MaxContextTokens == 0 {
		cfg.MaxContextTokens = DefaultMaxContextTokens
	}
	if cfg.KeepRecent == 0 {
		cfg.KeepRecent = DefaultKeepRecent
	}
	if cfg.MaxConcurrentToolCalls == 0 {
		cfg.MaxConcurrentToolCalls = DefaultMaxConcurrentToolCalls
	}
	if cfg.MaxConcurrentSubprocesses == 0 {
		cfg.MaxConcurrentSubprocesses = DefaultMaxConcurrentSubprocesses
	}
	if cfg.ApprovalPolicy == nil {
		cfg.ApprovalPolicy = DefaultApprovalPolicyMap()
	}
}

func Merge(global, project *Config) *Config {
	if global == nil && project == nil {
		return &Config{}
	}
	if global == nil {
		return project
	}
	if project == nil {
		return global
	}

	result := &Config{}
	*result = *global

	if project.LLM.APIKey != "" {
		result.LLM.APIKey = project.LLM.APIKey
	}
	if project.LLM.BaseURL != "" {
		result.LLM.BaseURL = project.LLM.BaseURL
	}
	if project.LLM.Model != "" {
		result.LLM.Model = project.LLM.Model
	}
	if project.LLM.ExtraHeaders != nil {
		result.LLM.ExtraHeaders = project.LLM.ExtraHeaders
	}
	if project.LLM.EnableReasoning {
		result.LLM.EnableReasoning = true
	}
	if project.LLM.ReasoningEffort != "" {
		result.LLM.ReasoningEffort = project.LLM.ReasoningEffort
	}
	if project.LLM.MaxTokens > 0 {
		result.LLM.MaxTokens = project.LLM.MaxTokens
	}
	if project.DBPath != "" {
		result.DBPath = project.DBPath
	}
	if project.LogPath != "" {
		result.LogPath = project.LogPath
	}
	if project.LogLevel != "" {
		result.LogLevel = project.LogLevel
	}
	if project.ApprovalPolicy != nil {
		result.ApprovalPolicy = project.ApprovalPolicy
	}
	if project.Skills != nil {
		result.Skills = project.Skills
	}
	if project.MCP != nil {
		result.MCP = project.MCP
	}
	if project.ToolCallLimit > 0 {
		result.ToolCallLimit = project.ToolCallLimit
	}
	if project.MaxContextTokens > 0 {
		result.MaxContextTokens = project.MaxContextTokens
	}
	if project.KeepRecent > 0 {
		result.KeepRecent = project.KeepRecent
	}
	if project.MaxConcurrentToolCalls > 0 {
		result.MaxConcurrentToolCalls = project.MaxConcurrentToolCalls
	}
	if project.MaxConcurrentSubprocesses > 0 {
		result.MaxConcurrentSubprocesses = project.MaxConcurrentSubprocesses
	}

	return result
}

func migrateLegacyConfig(cfg *Config) {
	if len(cfg.LLM.Models) > 0 {
		return
	}
	if cfg.LLM.APIKey != "" {
		cfg.LLM.Models = []ModelConfig{{
			ID:              "default",
			Name:            cfg.LLM.Model,
			APIKey:          cfg.LLM.APIKey,
			BaseURL:         cfg.LLM.BaseURL,
			Model:           cfg.LLM.Model,
			ExtraHeaders:    cfg.LLM.ExtraHeaders,
			EnableReasoning: cfg.LLM.EnableReasoning,
			ReasoningEffort: cfg.LLM.ReasoningEffort,
			MaxTokens:       cfg.LLM.MaxTokens,
		}}
		cfg.LLM.ActiveModelID = "default"
	}
}

func (c *Config) GetActiveModel() *ModelConfig {
	for i := range c.LLM.Models {
		if c.LLM.Models[i].ID == c.LLM.ActiveModelID {
			return &c.LLM.Models[i]
		}
	}
	if len(c.LLM.Models) > 0 {
		return &c.LLM.Models[0]
	}
	return nil
}

func (c *Config) IsLLMConfigured() bool {
	return len(c.LLM.Models) > 0 || c.LLM.APIKey != ""
}

func Slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, ".", "-")
	return slug
}
