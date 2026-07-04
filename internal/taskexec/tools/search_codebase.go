package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

func (t *SearchCodebaseTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
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

	re, err := regexp.Compile(pattern)
	if err != nil {
		re, err = regexp.Compile(regexp.QuoteMeta(pattern))
		if err != nil {
			return fmt.Errorf("invalid search pattern: %s", pattern)
		}
	}

	gi := pathsec.LoadGitignore(workingDir)

	maxResults := 50
	var results []string
	absWorkDir, _ := filepath.Abs(workingDir)

	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = filepath.Walk(safePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-searchCtx.Done():
			return searchCtx.Err()
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

		if isBinaryFile(info) {
			return nil
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
				match := fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line))
				results = append(results, match)
				w.WriteChunk(match + "\n")
			}
		}

		return nil
	})

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		if err == filepath.SkipAll {
			err = nil
		}
		if err != nil {
			return fmt.Errorf("search failed: %v", err)
		}
	}

	if len(results) == 0 {
		w.WriteDone(true, "No matches found.")
		return nil
	}

	if len(results) >= maxResults {
		results = append(results, fmt.Sprintf("... (results truncated at %d matches)", maxResults))
	}

	w.WriteDone(true, strings.Join(results, "\n"))
	return nil
}

func isBinaryFile(info os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(info.Name()))
	switch ext {
	case ".exe", ".dll", ".so", ".dylib", ".bin", ".dat",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".vsix", ".vsixmanifest",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wav", ".ogg",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".ttf", ".otf", ".woff", ".woff2", ".eot",
		".pyc", ".class", ".jar", ".war",
		".wasm", ".o", ".a", ".lib", ".obj":
		return true
	}
	return false
}
