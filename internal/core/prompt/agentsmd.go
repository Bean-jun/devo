package prompt

import (
	"os"
	"path/filepath"
)

const (
	agentsMDFile  = "agents.md"
	devoRulesFile = ".devo/rules.md"
)

func LoadAgentsMD(workingDir string) (string, bool) {
	candidates := []string{
		filepath.Join(workingDir, agentsMDFile),
		filepath.Join(workingDir, devoRulesFile),
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
