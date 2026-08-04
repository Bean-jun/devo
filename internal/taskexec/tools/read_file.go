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
	return `Reads a file from the local filesystem. You can access any file directly by using this tool.

Usage:
- The file_path parameter must be an absolute path, not a relative path.
- If the file does not exist, an error will be returned.
- Reads the entire file content at once.
- Use glob or list_files first if you're unsure of the file path.`
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
				"description": "File path relative to working directory or absolute path",
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
