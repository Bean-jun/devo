package prompt

import (
	"os"
	"path/filepath"

	"devo/internal/config"
)

const (
	agentsMDFile = "agents.md"
)

func LoadAgentsMD(workingDir string) (string, bool) {
	candidates := []string{
		filepath.Join(workingDir, agentsMDFile),
		config.ProjectRulesFilePath(workingDir),
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		return string(data), true
	}

	return "", false
}
