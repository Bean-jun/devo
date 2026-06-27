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

func AddServerConfig(workingDir, scope, serverID, endpoint, transport string) error {
	if transport == "" {
		transport = "sse"
	}

	var configPath string
	if scope == SourceGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}
		configPath = filepath.Join(homeDir, ".devo", "mcp_servers.json")
	} else {
		configPath = filepath.Join(workingDir, ".devo", "mcp_servers.json")
	}

	var configs []struct {
		ServerID  string `json:"server_id"`
		Endpoint  string `json:"endpoint"`
		Transport string `json:"transport"`
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &configs); err != nil {
			return fmt.Errorf("parse existing config: %w", err)
		}
	}

	for _, c := range configs {
		if c.ServerID == serverID {
			return fmt.Errorf("server_id %q already exists", serverID)
		}
	}

	configs = append(configs, struct {
		ServerID  string `json:"server_id"`
		Endpoint  string `json:"endpoint"`
		Transport string `json:"transport"`
	}{
		ServerID:  serverID,
		Endpoint:  endpoint,
		Transport: transport,
	})

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func RemoveServerConfig(workingDir, scope, serverID string) error {
	var configPath string
	if scope == SourceGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}
		configPath = filepath.Join(homeDir, ".devo", "mcp_servers.json")
	} else {
		configPath = filepath.Join(workingDir, ".devo", "mcp_servers.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("server %q not found", serverID)
		}
		return err
	}

	var configs []struct {
		ServerID  string `json:"server_id"`
		Endpoint  string `json:"endpoint"`
		Transport string `json:"transport"`
	}
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	found := false
	filtered := make([]struct {
		ServerID  string `json:"server_id"`
		Endpoint  string `json:"endpoint"`
		Transport string `json:"transport"`
	}, 0, len(configs))
	for _, c := range configs {
		if c.ServerID == serverID {
			found = true
			continue
		}
		filtered = append(filtered, c)
	}
	if !found {
		return fmt.Errorf("server %q not found", serverID)
	}

	newData, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
