package tools

import (
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
	return "Create a new file or overwrite an existing file with the given content"
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
				"description": "文件路径（相对于工作目录）",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "要写入的文件内容",
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

func (t *WriteFileTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
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

	_, statErr := os.Stat(safePath)
	isNew := statErr != nil

	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	if isNew {
		return fmt.Sprintf("File created successfully: %s", path), nil
	}
	return fmt.Sprintf("File updated successfully: %s", path), nil
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
				"description": "目标文件路径（相对于工作目录）",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "编辑模式：replace（查找替换）或 patch（unified diff 补丁）",
				"enum":        []string{"replace", "patch"},
			},
			"old_str": map[string]interface{}{
				"type":        "string",
				"description": "（mode=replace 时必填）要替换的原始文本，需在文件中唯一匹配",
			},
			"new_str": map[string]interface{}{
				"type":        "string",
				"description": "（mode=replace 时选填）替换后的新文本，默认为空字符串",
			},
			"patch": map[string]interface{}{
				"type":        "string",
				"description": "（mode=patch 时必填）unified diff 格式的补丁内容",
			},
		},
		"required": []string{"path", "mode"},
	}
}

func (t *EditFileTool) OperationType(workingDir string, params map[string]interface{}) string {
	return string(OpFileEdit)
}

func (t *EditFileTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
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

	switch mode {
	case "replace":
		return t.executeReplace(safePath, params)
	case "patch":
		return t.executePatch(safePath, params)
	default:
		return "", fmt.Errorf("unknown edit mode: %s (supported: replace, patch)", mode)
	}
}

func (t *EditFileTool) executeReplace(safePath string, params map[string]interface{}) (string, error) {
	oldStr, ok := params["old_str"].(string)
	if !ok || oldStr == "" {
		return "", fmt.Errorf("missing required parameter: old_str")
	}

	newStr, ok := params["new_str"].(string)
	if !ok {
		newStr = ""
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	content := string(data)
	count := strings.Count(content, oldStr)

	if count == 0 {
		return "", fmt.Errorf("old_str not found in file")
	}

	if count > 1 {
		return "", fmt.Errorf("old_str matches %d unique locations in the file, please provide more context to make the replacement unique", count)
	}

	newContent := strings.Replace(content, oldStr, newStr, 1)
	if err := os.WriteFile(safePath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return fmt.Sprintf("File successfully edited: replaced 1 occurrence"), nil
}

func (t *EditFileTool) executePatch(safePath string, params map[string]interface{}) (string, error) {
	patchData, ok := params["patch"].(string)
	if !ok || patchData == "" {
		return "", fmt.Errorf("missing required parameter: patch")
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	patchedContent, err := applyUnifiedDiff(string(data), patchData)
	if err != nil {
		return "", fmt.Errorf("patch application failed: %v", err)
	}

	if err := os.WriteFile(safePath, []byte(patchedContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return "File successfully patched with unified diff", nil
}

func applyUnifiedDiff(original, diffText string) (string, error) {
	lines := strings.Split(original, "\n")
	diffLines := strings.Split(diffText, "\n")

	var result []string
	origIdx := 0
	diffIdx := 0
	inHunk := false
	hunkOrigStart := 0
	var hunkLines []string

	for diffIdx < len(diffLines) {
		line := diffLines[diffIdx]

		if strings.HasPrefix(line, "@@") {
			if inHunk {
				applied, newOrigIdx, err := applyHunk(lines, origIdx, hunkOrigStart, hunkLines)
				if err != nil {
					return "", fmt.Errorf("failed to apply hunk at line %d: %v", hunkOrigStart+1, err)
				}
				result = append(result, applied...)
				origIdx = newOrigIdx
			}

			inHunk = true
			hunkLines = nil
			hunkOrigStart = origIdx
			diffIdx++
			continue
		}

		if inHunk {
			hunkLines = append(hunkLines, line)
			diffIdx++
			continue
		}

		diffIdx++
	}

	if inHunk {
		applied, newOrigIdx, err := applyHunk(lines, origIdx, hunkOrigStart, hunkLines)
		if err != nil {
			return "", fmt.Errorf("failed to apply hunk at line %d: %v", hunkOrigStart+1, err)
		}
		result = append(result, applied...)
		origIdx = newOrigIdx
	}

	if origIdx < len(lines) {
		result = append(result, lines[origIdx:]...)
	}

	return strings.Join(result, "\n"), nil
}

func applyHunk(original []string, origIdx, hunkOrigStart int, hunkLines []string) ([]string, int, error) {
	for origIdx < hunkOrigStart {
		origIdx = hunkOrigStart
	}

	var result []string
	origLineIdx := hunkOrigStart

	for _, line := range hunkLines {
		if line == "" {
			continue
		}

		switch line[0] {
		case ' ':
			if origLineIdx < len(original) {
				result = append(result, original[origLineIdx])
			}
			origLineIdx++
		case '-':
			if origLineIdx < len(original) {
				if strings.TrimRight(original[origLineIdx], "\r") != strings.TrimRight(line[1:], "\r") {
					return nil, origIdx, fmt.Errorf("expected removal line %q but found %q", line[1:], original[origLineIdx])
				}
			}
			origLineIdx++
		case '+':
			result = append(result, line[1:])
		case '\\':
		default:
			return nil, origIdx, fmt.Errorf("unexpected diff line prefix: %q", string(line[0]))
		}
	}

	return result, origLineIdx, nil
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
		_ = oldStr
	case "patch":
		patchData, ok := params["patch"].(string)
		if !ok || patchData == "" {
			return fmt.Errorf("missing required parameter: patch")
		}
		_ = patchData
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
	oldContent := string(oldData)

	var newContent string

	switch mode {
	case "replace":
		oldStr, ok := params["old_str"].(string)
		if !ok || oldStr == "" {
			return "", fmt.Errorf("missing required parameter: old_str")
		}

		newStr, ok := params["new_str"].(string)
		if !ok {
			newStr = ""
		}

		count := strings.Count(oldContent, oldStr)
		if count == 0 {
			return "", fmt.Errorf("old_str not found in file")
		}
		if count > 1 {
			return "", fmt.Errorf("old_str matches %d unique locations in the file, please provide more context to make the replacement unique", count)
		}

		newContent = strings.Replace(oldContent, oldStr, newStr, 1)

	case "patch":
		patchData, ok := params["patch"].(string)
		if !ok || patchData == "" {
			return "", fmt.Errorf("missing required parameter: patch")
		}

		patched, err := applyUnifiedDiff(oldContent, patchData)
		if err != nil {
			return "", fmt.Errorf("patch application failed: %v", err)
		}
		newContent = patched

	default:
		return "", fmt.Errorf("unknown edit mode: %s", mode)
	}

	if oldContent == newContent {
		return "", nil
	}

	return generateUnifiedDiff(oldContent, newContent), nil
}
