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
	return `A powerful search tool built on ripgrep.

Usage:
- ALWAYS use search_codebase for search tasks. NEVER invoke grep or rg as a shell command.
- Supports full regex syntax (e.g., "log.*Error", "function\s+\w+")
- Filter files with file_pattern parameter (e.g., "*.go", "*.py") or path parameter.
  file_pattern uses filepath.Match semantics — simple glob only, no brace expansion.
- Output modes: "content" shows matching lines, "files_with_matches" shows only file paths (default), "count" shows match counts
- Use glob tool for open-ended file discovery before narrowing with search_codebase
- Pattern syntax: Uses Go regexp (not grep) - literal braces need escaping (use interface\{\} to match interface{} in Go code)`
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
				"description": "Regex pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Search path relative to working directory, defaults to working directory",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string",
				"description": "File name pattern to filter results (e.g. '*.go', '*.py'), defaults to all files. Uses filepath.Match semantics.",
			},
			"output_mode": map[string]interface{}{
				"type":        "string",
				"description": "Output mode: 'content' (show matching lines), 'files_with_matches' (show file paths only), 'count' (show match counts). Defaults to 'content'.",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the search is case-sensitive. Defaults to false (case-insensitive).",
			},
			"context_lines": map[string]interface{}{
				"type":        "integer",
				"description": "Number of context lines to show before and after each match. Defaults to 0.",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of matching results. Defaults to 50.",
			},
			"head_limit": map[string]interface{}{
				"type":        "integer",
				"description": "Limit total output lines (including context_lines). 0 means no limit.",
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

	outputMode := "content"
	if m, ok := params["output_mode"].(string); ok && m != "" {
		outputMode = m
	}

	filePattern, _ := params["file_pattern"].(string)

	caseSensitive := false
	if cs, ok := params["case_sensitive"].(bool); ok {
		caseSensitive = cs
	}

	contextLines := 0
	if cl, ok := params["context_lines"].(float64); ok {
		contextLines = int(cl)
	}
	if cl, ok := params["context_lines"].(int); ok {
		contextLines = cl
	}

	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	if mr, ok := params["max_results"].(int); ok {
		maxResults = mr
	}

	headLimit := 0
	if hl, ok := params["head_limit"].(float64); ok {
		headLimit = int(hl)
	}
	if hl, ok := params["head_limit"].(int); ok {
		headLimit = hl
	}

	effectivePattern := pattern
	if !caseSensitive {
		effectivePattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(effectivePattern)
	if err != nil {
		re, err = regexp.Compile(regexp.QuoteMeta(pattern))
		if err != nil {
			return fmt.Errorf("invalid search pattern: %s", pattern)
		}
		w.WriteMeta("Pattern treated as literal string (regex compilation failed)")
	}

	gi := pathsec.LoadGitignore(workingDir)

	var results []string
	var outputLines []string
	absWorkDir, _ := filepath.Abs(workingDir)
	fileMatchCounts := make(map[string]int)

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

		if filePattern != "" {
			matched, err := filepath.Match(filePattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		if outputMode != "files_with_matches" && len(results) >= maxResults {
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
		fileHasMatch := false

		for i, line := range lines {
			if re.MatchString(line) {
				fileHasMatch = true
				fileMatchCounts[relPath]++

				if outputMode == "count" {
					continue
				}

				if outputMode == "files_with_matches" {
					if fileMatchCounts[relPath] == 1 {
						results = append(results, relPath)
					}
					continue
				}

				if len(results) >= maxResults {
					return filepath.SkipAll
				}

				if contextLines > 0 {
					start := i - contextLines
					if start < 0 {
						start = 0
					}
					end := i + contextLines + 1
					if end > len(lines) {
						end = len(lines)
					}

					for j := start; j < end; j++ {
						marker := "  "
						if j == i {
							marker = "> "
						}
						entry := fmt.Sprintf("%s:%d:%s%s", relPath, j+1, marker, strings.TrimSpace(lines[j]))
						outputLines = append(outputLines, entry)
					}
				} else {
					match := fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line))
					results = append(results, match)
					outputLines = append(outputLines, match)
				}
			}
		}

		if outputMode == "files_with_matches" && fileHasMatch {
			return nil
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

	if outputMode == "count" {
		var countResults []string
		for file, count := range fileMatchCounts {
			if count > 0 {
				countResults = append(countResults, fmt.Sprintf("%s: %d matches", file, count))
			}
		}
		results = countResults
		outputLines = countResults
	}

	if outputMode == "content" && contextLines > 0 {
		results = outputLines
	}

	if headLimit > 0 && len(outputLines) > headLimit {
		outputLines = outputLines[:headLimit]
		outputLines = append(outputLines, fmt.Sprintf("... (output truncated at %d lines)", headLimit))
		results = outputLines
	}

	if len(results) == 0 {
		w.WriteDone(true, "No matches found.")
		return nil
	}

	if len(results) >= maxResults && outputMode == "content" && contextLines == 0 {
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
