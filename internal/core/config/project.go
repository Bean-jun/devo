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
	Skills []string `json:"skills"`
	MCP    []string `json:"mcp"`
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
	data, err := json.MarshalIndent(cfg, "", "  ")
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
