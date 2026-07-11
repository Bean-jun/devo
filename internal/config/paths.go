package config

import (
	"os"
	"path/filepath"
)

func DevoDir() string {
	if v := os.Getenv("DEVO_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".devo")
}

func GlobalConfigPath() string {
	return filepath.Join(DevoDir(), "config.json")
}

func GlobalMcpServersPath() string {
	return filepath.Join(DevoDir(), "mcp_servers.json")
}

func GlobalSkillsDir() string {
	return filepath.Join(DevoDir(), "skills")
}

func MemoryDir() string {
	return filepath.Join(DevoDir(), "memory")
}

func DBPath() string {
	return filepath.Join(DevoDir(), "devo.db")
}

func LogPath() string {
	return filepath.Join(DevoDir(), "devo.log")
}

func ProjectConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".devo", "config.json")
}

func ProjectMcpServersPath(projectDir string) string {
	return filepath.Join(projectDir, ".devo", "mcp_servers.json")
}

func ProjectSkillsDir(projectDir string) string {
	return filepath.Join(projectDir, ".devo", "skills")
}

func ProjectSessionsDir(projectDir string) string {
	return filepath.Join(projectDir, ".devo", "sessions")
}

func ProjectRulesFilePath(projectDir string) string {
	return filepath.Join(projectDir, ".devo", "rules.md")
}
