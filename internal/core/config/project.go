package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const configFileName = "config.json"

type ProjectConfig struct {
	Skills         []string          `json:"skills"`
	MCP            []string          `json:"mcp"`
	ApprovalPolicy map[string]string `json:"approval_policy,omitempty"`
}

func Load(projectDir string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectDir, ".devo", configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func Save(projectDir string, cfg *ProjectConfig) error {
	configDir := filepath.Join(projectDir, ".devo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configPath := filepath.Join(configDir, configFileName)

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

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func CreateDefault(projectDir string, availableSkills []string, availableMCP []string) (*ProjectConfig, error) {
	sort.Strings(availableSkills)
	sort.Strings(availableMCP)

	cfg := &ProjectConfig{
		Skills: availableSkills,
		MCP:    availableMCP,
	}

	if err := Save(projectDir, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
