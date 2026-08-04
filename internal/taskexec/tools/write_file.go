package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"devo/internal/taskexec/pathsec"
)

func (t *WriteFileTool) PreviewDiff(workingDir string, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing required parameter: path")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing required parameter: content")
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return "", fmt.Errorf("path security check failed")
	}

	oldData, err := os.ReadFile(safePath)
	if err != nil {
		return "", nil
	}

	oldContent := string(oldData)
	if oldContent == content {
		return "", nil
	}

	return generateUnifiedDiff(oldContent, content), nil
}

type WriteFileTool struct{}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return `Writes a file to the local filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path.
- If this is an existing file, you MUST use the ReadFile tool first to read the file's contents. This tool will fail if you did not read the file first.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
- Only use emojis if the user explicitly requests it. Avoid writing emojis to files unless asked.`
}

func (t *WriteFileTool) RiskLevel() RiskLevel {
	return RiskLevelMedium
}

func (t *WriteFileTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path relative to working directory",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) OperationType(workingDir string, params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return string(OpFileWriteNew)
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return string(OpFileWriteNew)
	}

	if _, err := os.Stat(safePath); err == nil {
		return string(OpFileWriteOverwrite)
	}
	return string(OpFileWriteNew)
}

func (t *WriteFileTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("missing required parameter: path")
	}

	content, ok := params["content"].(string)
	if !ok {
		return fmt.Errorf("missing required parameter: content")
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return fmt.Errorf("path security check failed")
	}

	gi := pathsec.LoadGitignore(workingDir)
	relPath, _ := filepath.Rel(workingDir, safePath)
	if gi.IsIgnored(relPath, false) {
		return fmt.Errorf("file is excluded by .gitignore: %s", path)
	}

	_, statErr := os.Stat(safePath)
	isNew := statErr != nil

	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	if isNew {
		w.WriteDone(true, fmt.Sprintf("File created successfully: %s", path))
	} else {
		w.WriteDone(true, fmt.Sprintf("File updated successfully: %s", path))
	}
	return nil
}

type EditFileTool struct{}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file using replace or patch mode"
}

func (t *EditFileTool) RiskLevel() RiskLevel {
	return RiskLevelMedium
}

func (t *EditFileTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Target file path relative to working directory",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Edit mode: 'replace' (find and replace) or 'patch' (unified diff patch)",
				"enum":        []string{"replace", "patch"},
			},
			"old_str": map[string]interface{}{
				"type":        "string",
				"description": "(Required when mode=replace) The original text to replace, must be unique in the file",
			},
			"new_str": map[string]interface{}{
				"type":        "string",
				"description": "(Optional when mode=replace) The replacement text, defaults to empty string",
			},
			"patch": map[string]interface{}{
				"type":        "string",
				"description": "(Required when mode=patch) Unified diff patch content",
			},
		},
		"required": []string{"path", "mode"},
	}
}

func (t *EditFileTool) OperationType(workingDir string, params map[string]interface{}) string {
	return string(OpFileEdit)
}

func (t *EditFileTool) PreCheck(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("missing required parameter: path")
	}

	mode, ok := params["mode"].(string)
	if !ok {
		return fmt.Errorf("missing required parameter: mode")
	}

	switch mode {
	case "replace":
		oldStr, ok := params["old_str"].(string)
		if !ok || oldStr == "" {
			return fmt.Errorf("missing required parameter: old_str")
		}
	case "patch":
		patch, ok := params["patch"].(string)
		if !ok || patch == "" {
			return fmt.Errorf("missing required parameter: patch")
		}
	default:
		return fmt.Errorf("unknown edit mode: %s (supported: replace, patch)", mode)
	}

	return nil
}

func (t *EditFileTool) PreviewDiff(workingDir string, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing required parameter: path")
	}

	mode, ok := params["mode"].(string)
	if !ok {
		return "", fmt.Errorf("missing required parameter: mode")
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return "", fmt.Errorf("path security check failed")
	}

	oldData, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	originalContent := string(oldData)

	switch mode {
	case "replace":
		oldStr, ok := params["old_str"].(string)
		if !ok || oldStr == "" {
			return "", fmt.Errorf("missing required parameter: old_str")
		}

		newStr, _ := params["new_str"].(string)

		count := strings.Count(originalContent, oldStr)
		if count == 0 {
			return "", fmt.Errorf("old_str not found in file")
		}
		if count > 1 {
			return "", fmt.Errorf("old_str matches %d unique locations in the file, please provide more context to make the replacement unique", count)
		}

		newContent := strings.Replace(originalContent, oldStr, newStr, 1)
		return generateUnifiedDiff(originalContent, newContent), nil

	case "patch":
		patchContent, ok := params["patch"].(string)
		if !ok || patchContent == "" {
			return "", fmt.Errorf("missing required parameter: patch")
		}

		patchedContent, err := applyPatch(originalContent, patchContent)
		if err != nil {
			return "", fmt.Errorf("failed to apply patch: %v", err)
		}

		return generateUnifiedDiff(originalContent, patchedContent), nil

	default:
		return "", fmt.Errorf("unknown edit mode: %s", mode)
	}
}

func (t *EditFileTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("missing required parameter: path")
	}

	mode, ok := params["mode"].(string)
	if !ok {
		return fmt.Errorf("missing required parameter: mode")
	}

	safePath, err := pathsec.CheckPath(workingDir, path)
	if err != nil {
		return fmt.Errorf("path security check failed")
	}

	gi := pathsec.LoadGitignore(workingDir)
	relPath, _ := filepath.Rel(workingDir, safePath)
	if gi.IsIgnored(relPath, false) {
		return fmt.Errorf("file is excluded by .gitignore: %s", path)
	}

	switch mode {
	case "replace":
		return t.executeReplace(ctx, safePath, params, w)
	case "patch":
		return t.executePatch(ctx, safePath, params, w)
	default:
		return fmt.Errorf("unknown edit mode: %s (supported: replace, patch)", mode)
	}
}

func (t *EditFileTool) executeReplace(ctx context.Context, safePath string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	oldStr, ok := params["old_str"].(string)
	if !ok || oldStr == "" {
		return fmt.Errorf("missing required parameter: old_str")
	}

	newStr, ok := params["new_str"].(string)
	if !ok {
		newStr = ""
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	content := string(data)
	count := strings.Count(content, oldStr)

	if count == 0 {
		return fmt.Errorf("old_str not found in file")
	}

	if count > 1 {
		return fmt.Errorf("old_str matches %d unique locations in the file, please provide more context to make the replacement unique", count)
	}

	newContent := strings.Replace(content, oldStr, newStr, 1)

	diff := generateUnifiedDiff(content, newContent)

	if err := os.WriteFile(safePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	result := fmt.Sprintf("File successfully edited: replaced 1 occurrence")
	if diff != "" {
		result += "\n__DEVO_DIFF__\n" + diff
	}
	w.WriteDone(true, result)
	return nil
}

func (t *EditFileTool) executePatch(ctx context.Context, safePath string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	patchContent, ok := params["patch"].(string)
	if !ok || patchContent == "" {
		return fmt.Errorf("missing required parameter: patch for patch mode")
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	originalContent := string(data)
	patchedContent, err := applyPatch(originalContent, patchContent)
	if err != nil {
		return fmt.Errorf("failed to apply patch: %v", err)
	}

	diff := generateUnifiedDiff(originalContent, patchedContent)

	if err := os.WriteFile(safePath, []byte(patchedContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	result := fmt.Sprintf("File successfully patched")
	if diff != "" {
		result += "\n__DEVO_DIFF__\n" + diff
	}
	w.WriteDone(true, result)
	return nil
}

func applyPatch(originalContent, patchContent string) (string, error) {
	origLines := strings.Split(originalContent, "\n")
	patchedLines := make([]string, 0, len(origLines))

	lines := strings.Split(patchContent, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimRight(lines[i], "\r")
		i++

		if strings.HasPrefix(line, "@@") {
			oldStart, oldCount, newStart, newCount := parseHunkHeader(line)
			if oldStart < 0 || newStart < 0 {
				continue
			}

			origIdx := oldStart - 1
			newIdx := newStart - 1

			if origIdx < 0 {
				origIdx = 0
			}
			if newIdx < 0 {
				newIdx = 0
			}

			for origIdx > len(patchedLines) && len(patchedLines) < len(origLines) {
				patchedLines = append(patchedLines, origLines[len(patchedLines)])
			}

			_ = oldCount
			_ = newCount

			tempResult := patchedLines[:newIdx]

			for i < len(lines) {
				line = strings.TrimRight(lines[i], "\r")
				i++

				if strings.HasPrefix(line, "@@") {
					i--
					break
				}

				if strings.HasPrefix(line, " ") {
					if origIdx < len(origLines) {
						expected := origLines[origIdx]
						actual := line[1:]
						if expected != actual {
							return "", fmt.Errorf("patch hunk context mismatch at line %d: expected %q, got %q", origIdx+1, expected, actual)
						}
						tempResult = append(tempResult, origLines[origIdx])
					} else {
						tempResult = append(tempResult, line[1:])
					}
					origIdx++
				} else if strings.HasPrefix(line, "-") {
					if origIdx < len(origLines) {
						expected := origLines[origIdx]
						actual := line[1:]
						if expected != actual {
							return "", fmt.Errorf("patch hunk deletion mismatch at line %d: expected to delete %q, but original has %q", origIdx+1, actual, expected)
						}
					}
					origIdx++
				} else if strings.HasPrefix(line, "+") {
					tempResult = append(tempResult, line[1:])
				}
			}

			for origIdx < len(origLines) {
				tempResult = append(tempResult, origLines[origIdx])
				origIdx++
			}

			patchedLines = tempResult
		}
	}

	result := strings.Join(patchedLines, "\n")
	return result, nil
}

func parseHunkHeader(line string) (oldStart, oldCount, newStart, newCount int) {
	oldStart = -1
	newStart = -1

	parts := strings.Split(line, " ")

	for _, part := range parts {
		isOld := strings.HasPrefix(part, "-")
		isNew := strings.HasPrefix(part, "+")

		clean := strings.TrimPrefix(part, "-")
		clean = strings.TrimPrefix(clean, "+")

		if idx := strings.Index(clean, ","); idx != -1 {
			start := 0
			count := 0
			fmt.Sscanf(clean, "%d,%d", &start, &count)
			if isOld {
				oldStart = start
				oldCount = count
			} else if isNew {
				newStart = start
				newCount = count
			}
		} else {
			start := 0
			fmt.Sscanf(clean, "%d", &start)
			if isOld {
				oldStart = start
				oldCount = 1
			} else if isNew {
				newStart = start
				newCount = 1
			}
		}
	}

	return
}
