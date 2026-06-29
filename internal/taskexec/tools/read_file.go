package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"devo/internal/taskexec/pathsec"
)

type ReadFileTool struct{}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file at the given path"
}

func (t *ReadFileTool) RiskLevel() RiskLevel {
	return RiskLevelNone
}

func (t *ReadFileTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "文件路径（相对于工作目录或绝对路径）",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing required parameter: path")
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return "", fmt.Errorf("path security check failed")
	}

	gi := pathsec.LoadGitignore(workingDir)
	relPath, _ := filepath.Rel(workingDir, safePath)
	if gi.IsIgnored(relPath, false) {
		return "", fmt.Errorf("file is excluded by .gitignore: %s", path)
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return "", fmt.Errorf("file not found or not accessible: %s", path)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", path)
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %s", path)
	}

	return string(data), nil
}
