package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type ProjectConfig = Config

func LoadProjectConfig(projectDir string) (*Config, error) {
	configPath := ProjectConfigPath(projectDir)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func SaveProjectConfig(projectDir string, cfg *Config) error {
	configDir := filepath.Join(projectDir, ".devo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configPath := ProjectConfigPath(projectDir)

	existing := make(map[string]json.RawMessage)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	if cfg.Skills != nil {
		skillsData, err := json.Marshal(cfg.Skills)
		if err != nil {
			return fmt.Errorf("marshal skills: %w", err)
		}
		existing["skills"] = skillsData
	}
	if cfg.MCP != nil {
		mcpData, err := json.Marshal(cfg.MCP)
		if err != nil {
			return fmt.Errorf("marshal mcp: %w", err)
		}
		existing["mcp"] = mcpData
	}

	if cfg.ApprovalPolicy != nil {
		approvalData, err := json.Marshal(cfg.ApprovalPolicy)
		if err != nil {
			return fmt.Errorf("marshal approval_policy: %w", err)
		}
		existing["approval_policy"] = approvalData
	}

	if cfg.LLM.APIKey != "" || cfg.LLM.BaseURL != "" || cfg.LLM.Model != "" ||
		cfg.LLM.EnableReasoning || cfg.LLM.ReasoningEffort != "" {
		llmData, err := json.Marshal(cfg.LLM)
		if err != nil {
			return fmt.Errorf("marshal llm: %w", err)
		}
		existing["llm"] = llmData
	}
	if cfg.DBPath != "" {
		dbData, err := json.Marshal(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("marshal db_path: %w", err)
		}
		existing["db_path"] = dbData
	}
	if cfg.LogPath != "" {
		logData, err := json.Marshal(cfg.LogPath)
		if err != nil {
			return fmt.Errorf("marshal log_path: %w", err)
		}
		existing["log_path"] = logData
	}
	if cfg.ToolCallLimit > 0 {
		toolCallLimitData, err := json.Marshal(cfg.ToolCallLimit)
		if err != nil {
			return fmt.Errorf("marshal tool_call_limit: %w", err)
		}
		existing["tool_call_limit"] = toolCallLimitData
	}
	if cfg.MaxContextTokens > 0 {
		maxCtxData, err := json.Marshal(cfg.MaxContextTokens)
		if err != nil {
			return fmt.Errorf("marshal max_context_tokens: %w", err)
		}
		existing["max_context_tokens"] = maxCtxData
	}
	if cfg.KeepRecent > 0 {
		keepRecentData, err := json.Marshal(cfg.KeepRecent)
		if err != nil {
			return fmt.Errorf("marshal keep_recent: %w", err)
		}
		existing["keep_recent"] = keepRecentData
	}
	if cfg.ContextCompressRatio > 0 {
		compressRatioData, err := json.Marshal(cfg.ContextCompressRatio)
		if err != nil {
			return fmt.Errorf("marshal context_compress_ratio: %w", err)
		}
		existing["context_compress_ratio"] = compressRatioData
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func CreateDefaultProjectConfig(projectDir string, availableSkills []string, availableMCP []string) (*Config, error) {
	sort.Strings(availableSkills)
	sort.Strings(availableMCP)

	cfg := &Config{
		Skills: availableSkills,
		MCP:    availableMCP,
	}

	if err := SaveProjectConfig(projectDir, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveGlobalConfig(cfg *Config) error {
	configDir := DevoDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configPath := GlobalConfigPath()

	existing := make(map[string]json.RawMessage)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	if cfg.LLM.APIKey != "" || cfg.LLM.BaseURL != "" || cfg.LLM.Model != "" ||
		cfg.LLM.EnableReasoning || cfg.LLM.ReasoningEffort != "" {
		llmData, err := json.Marshal(cfg.LLM)
		if err != nil {
			return fmt.Errorf("marshal llm: %w", err)
		}
		existing["llm"] = llmData
	}
	if cfg.DBPath != "" {
		dbData, err := json.Marshal(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("marshal db_path: %w", err)
		}
		existing["db_path"] = dbData
	}
	if cfg.LogPath != "" {
		logData, err := json.Marshal(cfg.LogPath)
		if err != nil {
			return fmt.Errorf("marshal log_path: %w", err)
		}
		existing["log_path"] = logData
	}
	if cfg.ApprovalPolicy != nil {
		approvalData, err := json.Marshal(cfg.ApprovalPolicy)
		if err != nil {
			return fmt.Errorf("marshal approval_policy: %w", err)
		}
		existing["approval_policy"] = approvalData
	}
	if cfg.Skills != nil {
		skillsData, err := json.Marshal(cfg.Skills)
		if err != nil {
			return fmt.Errorf("marshal skills: %w", err)
		}
		existing["skills"] = skillsData
	}
	if cfg.MCP != nil {
		mcpData, err := json.Marshal(cfg.MCP)
		if err != nil {
			return fmt.Errorf("marshal mcp: %w", err)
		}
		existing["mcp"] = mcpData
	}
	if cfg.ToolCallLimit > 0 {
		toolCallLimitData, err := json.Marshal(cfg.ToolCallLimit)
		if err != nil {
			return fmt.Errorf("marshal tool_call_limit: %w", err)
		}
		existing["tool_call_limit"] = toolCallLimitData
	}
	if cfg.MaxContextTokens > 0 {
		maxCtxData, err := json.Marshal(cfg.MaxContextTokens)
		if err != nil {
			return fmt.Errorf("marshal max_context_tokens: %w", err)
		}
		existing["max_context_tokens"] = maxCtxData
	}
	if cfg.KeepRecent > 0 {
		keepRecentData, err := json.Marshal(cfg.KeepRecent)
		if err != nil {
			return fmt.Errorf("marshal keep_recent: %w", err)
		}
		existing["keep_recent"] = keepRecentData
	}
	if cfg.ContextCompressRatio > 0 {
		compressRatioData, err := json.Marshal(cfg.ContextCompressRatio)
		if err != nil {
			return fmt.Errorf("marshal context_compress_ratio: %w", err)
		}
		existing["context_compress_ratio"] = compressRatioData
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
