package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type LLMConfig struct {
	APIKey       string            `json:"api_key"`
	BaseURL      string            `json:"base_url"`
	Model        string            `json:"model"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

type Config struct {
	LLM            LLMConfig         `json:"llm,omitempty"`
	DBPath         string            `json:"db_path,omitempty"`
	LogPath        string            `json:"log_path,omitempty"`
	LogLevel       string            `json:"log_level,omitempty"`
	ApprovalPolicy map[string]string `json:"approval_policy,omitempty"`
	Skills         []string          `json:"skills,omitempty"`
	MCP            []string          `json:"mcp,omitempty"`
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

	if cfg.LLM.APIKey == "" {
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

func applyDefaults(cfg *Config) {
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = DefaultLLMBaseURL
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = DefaultLLMModel
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

	return result
}
