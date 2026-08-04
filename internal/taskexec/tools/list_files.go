package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"devo/internal/taskexec/pathsec"
)

type ListFilesTool struct{}

func (t *ListFilesTool) Name() string {
	return "list_files"
}

func (t *ListFilesTool) Description() string {
	return `List files and directories in a given path.

Usage:
- The path parameter must be an absolute path, not a relative path.
- You can optionally provide an array of glob patterns to ignore with the ignore parameter.
- Use max_depth to control recursion depth (default 1, only lists direct children).
- Use max_files to limit the number of entries returned (default 500).
- Results are sorted by modification time (newest first) for quick project overview.`
}

func (t *ListFilesTool) RiskLevel() RiskLevel {
	return RiskLevelNone
}

func (t *ListFilesTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path relative to working directory or absolute path, defaults to working directory root",
			},
			"max_depth": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum recursion depth. Defaults to 1.",
			},
			"max_files": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of files to return. Defaults to 500.",
			},
		},
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path, _ := params["path"].(string)
	if path == "" {
		path = "."
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return fmt.Errorf("path security check failed")
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return fmt.Errorf("path not found or not accessible: %s", path)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	maxDepth := 1
	if d, ok := params["max_depth"].(float64); ok {
		maxDepth = int(d)
	}

	maxFiles := 500
	if m, ok := params["max_files"].(float64); ok {
		maxFiles = int(m)
	}

	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("failed to resolve working directory")
	}

	var result strings.Builder
	count := 0

	gi := pathsec.LoadGitignore(workingDir)
	err = walkDirStream(ctx, &result, absWorkDir, safePath, ".", maxDepth, 0, maxFiles, &count, gi, w)
	if err != nil {
		return err
	}

	if count >= maxFiles {
		result.WriteString(fmt.Sprintf("\n... (output truncated at %d entries)", maxFiles))
	}

	w.WriteDone(true, result.String())
	return nil
}

func walkDirStream(ctx context.Context, result *strings.Builder, absWorkDir, currentPath, displayPrefix string, maxDepth, currentDepth, maxFiles int, count *int, gi *pathsec.Gitignore, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if currentDepth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %s", displayPrefix)
	}

	type entry struct {
		name  string
		isDir bool
	}

	var sorted []entry
	for _, e := range entries {
		name := e.Name()
		if name == ".git" || strings.HasPrefix(name, ".") {
			continue
		}
		relPath, _ := filepath.Rel(absWorkDir, filepath.Join(currentPath, name))
		if gi.IsIgnored(relPath, e.IsDir()) {
			continue
		}
		sorted = append(sorted, entry{name: name, isDir: e.IsDir()})
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].isDir != sorted[j].isDir {
			return sorted[i].isDir
		}
		return sorted[i].name < sorted[j].name
	})

	for _, e := range sorted {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if *count >= maxFiles {
			return nil
		}

		line := ""
		if e.isDir {
			line = fmt.Sprintf("%s%s/\n", strings.Repeat("  ", currentDepth), e.name)
		} else {
			line = fmt.Sprintf("%s%s\n", strings.Repeat("  ", currentDepth), e.name)
		}
		result.WriteString(line)
		*count++

		if e.isDir && currentDepth < maxDepth {
			subPath := filepath.Join(currentPath, e.name)
			subDisplay := filepath.Join(displayPrefix, e.name)
			if err := walkDirStream(ctx, result, absWorkDir, subPath, subDisplay, maxDepth, currentDepth+1, maxFiles, count, gi, w); err != nil {
				return err
			}
		}
	}

	return nil
}
