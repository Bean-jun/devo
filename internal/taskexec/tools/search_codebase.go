package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"devo/internal/taskexec/pathsec"
)

type SearchCodebaseTool struct{}

func (t *SearchCodebaseTool) Name() string {
	return "search_codebase"
}

func (t *SearchCodebaseTool) Description() string {
	return "Search for a pattern in files within the working directory"
}

func (t *SearchCodebaseTool) RiskLevel() RiskLevel {
	return RiskLevelNone
}

func (t *SearchCodebaseTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "搜索的正则表达式模式",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "搜索路径（相对于工作目录），默认为工作目录",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *SearchCodebaseTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("missing required parameter: pattern")
	}

	searchPath := "."
	if p, ok := params["path"].(string); ok && p != "" {
		searchPath = p
	}

	safePath, err := pathsec.CheckPath(workingDir, searchPath)
	if err != nil {
		return "", fmt.Errorf("path security check failed")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		re, err = regexp.Compile(regexp.QuoteMeta(pattern))
		if err != nil {
			return "", fmt.Errorf("invalid search pattern: %s", pattern)
		}
	}

	maxResults := 50
	var results []string
	absWorkDir, _ := filepath.Abs(workingDir)

	err = filepath.Walk(safePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		relPath, err := filepath.Rel(absWorkDir, path)
		if err != nil {
			relPath = path
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if len(results) >= maxResults {
				return filepath.SkipAll
			}
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line)))
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("search failed: %v", err)
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	if len(results) >= maxResults {
		results = append(results, fmt.Sprintf("... (results truncated at %d matches)", maxResults))
	}

	return strings.Join(results, "\n"), nil
}
