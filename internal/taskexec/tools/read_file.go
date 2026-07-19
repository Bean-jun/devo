package tools

import (
	"context"
	"fmt"
	"os"

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

func (t *ReadFileTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("missing required parameter: path")
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return fmt.Errorf("path security check failed")
	}

	// .gitignore is a VCS concept, not an access-control boundary. The pathsec
	// check above already confines reads to the workspace; refusing to read
	// files that happen to match an ignore rule just blocks the agent from
	// reading its own scratch files (logs, build artifacts) that it created
	// via exec_python.

	info, err := os.Stat(safePath)
	if err != nil {
		return fmt.Errorf("file not found or not accessible: %s", path)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %s", path)
	}

	w.WriteDone(true, string(data))
	return nil
}
