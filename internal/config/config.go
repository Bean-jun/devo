package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LLMConfig struct {
	APIKey       string            `json:"api_key"`
	BaseURL      string            `json:"base_url"`
	Model        string            `json:"model"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

type Config struct {
	LLM            LLMConfig         `json:"llm"`
	DBPath         string            `json:"db_path,omitempty"`
	LogPath        string            `json:"log_path,omitempty"`
	ApprovalPolicy map[string]string `json:"approval_policy,omitempty"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	candidates := []string{
		filepath.Join(".devo", "config.json"),
		filepath.Join(userHomeDir(), ".devo", "config.json"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("read config file %s: %w", p, err)
			}
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config file %s: %w", p, err)
			}
			break
		}
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
}

func applyDefaults(cfg *Config) {
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o"
	}
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
