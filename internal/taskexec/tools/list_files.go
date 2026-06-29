package tools

import (
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
	return "List files and directories at the given path"
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
				"description": "目录路径（相对于工作目录或绝对路径），默认为工作目录根目录",
			},
			"max_depth": map[string]interface{}{
				"type":        "integer",
				"description": "最大遍历深度，默认 1",
			},
			"max_files": map[string]interface{}{
				"type":        "integer",
				"description": "最大返回文件数，默认 500",
			},
		},
	}
}

func (t *ListFilesTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		path = "."
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return "", fmt.Errorf("path security check failed")
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return "", fmt.Errorf("path not found or not accessible: %s", path)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
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
		return "", fmt.Errorf("failed to resolve working directory")
	}

	var result strings.Builder
	count := 0

	gi := pathsec.LoadGitignore(workingDir)
	err = walkDir(&result, absWorkDir, safePath, ".", maxDepth, 0, maxFiles, &count, gi)
	if err != nil {
		return "", err
	}

	if count >= maxFiles {
		result.WriteString(fmt.Sprintf("\n... (output truncated at %d entries)", maxFiles))
	}

	return result.String(), nil
}

func walkDir(result *strings.Builder, absWorkDir, currentPath, displayPrefix string, maxDepth, currentDepth, maxFiles int, count *int, gi *pathsec.Gitignore) error {
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
		if *count >= maxFiles {
			return nil
		}

		if e.isDir {
			result.WriteString(fmt.Sprintf("%s%s/\n", strings.Repeat("  ", currentDepth), e.name))
		} else {
			result.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", currentDepth), e.name))
		}
		*count++

		if e.isDir && currentDepth < maxDepth {
			subPath := filepath.Join(currentPath, e.name)
			subDisplay := filepath.Join(displayPrefix, e.name)
			walkDir(result, absWorkDir, subPath, subDisplay, maxDepth, currentDepth+1, maxFiles, count, gi)
		}
	}

	return nil
}
