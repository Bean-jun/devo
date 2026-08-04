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

type GlobTool struct{}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return `Fast file pattern matching tool that works with any codebase size.

Usage:
- Supports glob patterns like "**/*.go", "src/**/*.ts", "**/*_test.go".
- Combine with list_files to explore unknown directories before narrowing with glob.
- Returns matching file paths sorted by modification time.
- Use path parameter to restrict search to a specific subdirectory.`
}

func (t *GlobTool) RiskLevel() RiskLevel {
	return RiskLevelNone
}

func (t *GlobTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to match files (e.g., '**/*.go', 'src/**/*_test.go')",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Search root directory relative to working directory, defaults to working directory",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return fmt.Errorf("missing required parameter: pattern")
	}

	searchPath := "."
	if p, ok := params["path"].(string); ok && p != "" {
		searchPath = p
	}

	safePath, err := pathsec.CheckPath(workingDir, searchPath)
	if err != nil {
		return fmt.Errorf("path security check failed")
	}

	gi := pathsec.LoadGitignore(workingDir)
	absWorkDir, _ := filepath.Abs(workingDir)

	maxResults := 500
	var results []string

	err = filepath.Walk(safePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			name := info.Name()
			if name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			relPath, _ := filepath.Rel(absWorkDir, path)
			if gi.IsIgnored(relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}

		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		relPath, _ := filepath.Rel(absWorkDir, path)
		if gi.IsIgnored(relPath, false) {
			return nil
		}

		matched, err := matchGlob(pattern, relPath)
		if err != nil {
			return err
		}
		if matched {
			results = append(results, relPath)
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return fmt.Errorf("glob failed: %v", err)
	}

	if len(results) == 0 {
		w.WriteDone(true, "No files found matching pattern: "+pattern)
		return nil
	}

	sort.Strings(results)

	if len(results) >= maxResults {
		results = append(results, fmt.Sprintf("... (output truncated at %d results)", maxResults))
	}

	w.WriteDone(true, strings.Join(results, "\n"))
	return nil
}

func matchGlob(pattern, name string) (bool, error) {
	pattern = filepath.ToSlash(pattern)
	name = filepath.ToSlash(name)

	if !strings.Contains(pattern, "**") {
		return filepath.Match(pattern, name)
	}

	parts := splitGlobPattern(pattern)
	return matchGlobParts(parts, name)
}

func splitGlobPattern(pattern string) []string {
	var parts []string
	for _, part := range strings.Split(pattern, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func matchGlobParts(parts []string, name string) (bool, error) {
	nameParts := strings.Split(name, "/")
	return matchRecursive(parts, nameParts, 0, 0)
}

func matchRecursive(parts, nameParts []string, pi, ni int) (bool, error) {
	if pi == len(parts) {
		return ni == len(nameParts), nil
	}

	if parts[pi] == "**" {
		if pi == len(parts)-1 {
			return true, nil
		}
		nextPattern := parts[pi+1]
		for i := ni; i <= len(nameParts); i++ {
			if i == len(nameParts) {
				return false, nil
			}
			matched, err := filepath.Match(nextPattern, nameParts[i])
			if err != nil {
				return false, err
			}
			if matched {
				ok, err := matchRecursive(parts, nameParts, pi+2, i+1)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
		}
		return false, nil
	}

	if ni >= len(nameParts) {
		return false, nil
	}

	matched, err := filepath.Match(parts[pi], nameParts[ni])
	if err != nil {
		return false, err
	}
	if !matched {
		return false, nil
	}

	return matchRecursive(parts, nameParts, pi+1, ni+1)
}
