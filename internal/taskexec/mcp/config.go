package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	SourceProject = "project"
	SourceGlobal  = "global"
)

func LoadConfig(workingDir string) ([]McpServerConfig, error) {
	var allConfigs []McpServerConfig

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	configPaths := []struct {
		path   string
		source string
	}{
		{filepath.Join(workingDir, ".devo", "mcp_servers.json"), SourceProject},
	}

	if homeDir != "" {
		configPaths = append(configPaths, struct {
			path   string
			source string
		}{filepath.Join(homeDir, ".devo", "mcp_servers.json"), SourceGlobal})
	}

	seenIDs := make(map[string]bool)

	for _, cp := range configPaths {
		configs, err := loadConfigFile(cp.path, cp.source)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("load config %s: %w", cp.path, err)
		}

		for _, cfg := range configs {
			if seenIDs[cfg.ServerID] {
				continue
			}
			seenIDs[cfg.ServerID] = true
			allConfigs = append(allConfigs, cfg)
		}
	}

	return allConfigs, nil
}

func loadConfigFile(path, source string) ([]McpServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawConfigs []struct {
		ServerID  string `json:"server_id"`
		Endpoint  string `json:"endpoint"`
		Transport string `json:"transport"`
	}
	if err := json.Unmarshal(data, &rawConfigs); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	var configs []McpServerConfig
	for _, raw := range rawConfigs {
		transport := raw.Transport
		if transport == "" {
			transport = "sse"
		}
		configs = append(configs, McpServerConfig{
			ServerID:  raw.ServerID,
			Source:    source,
			Endpoint:  raw.Endpoint,
			Transport: transport,
		})
	}

	return configs, nil
}
