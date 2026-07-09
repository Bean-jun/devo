package tools

import (
	"bufio"
	"bytes"
	"context"
	"devo/internal/pkg/process"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type ExecPythonTool struct {
	pythonBin string
}

var pythonSearchOrder = []string{"python3", "python", "python3.12", "python3.11", "python3.10"}

var pythonBlacklistPatterns = []*regexp.Regexp{
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']rm["']\s*,\s*["']-rf["']\s*,\s*["']/["']`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']rm["']\s*,\s*["']-rf["']\s*,\s*["']\$HOME`),
	regexp.MustCompile(`os\.system\s*\(\s*["'].*rm\s+-rf\s+/`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']mkfs`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']dd\s+if=`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["'](?:curl|wget)["'].*\|.*sh`),
	regexp.MustCompile(`os\.system\s*\(\s*["'].*:\s*\(\s*\)\s*\{`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']chmod["'].*777\s+/`),
	regexp.MustCompile(`(?i)subprocess\.(?:run|call|Popen)\s*\(\s*\[["'](?:format|diskpart)["']`),
	regexp.MustCompile(`(?i)subprocess\.(?:run|call|Popen)\s*\(\s*\[["'](?:del|rd|Remove-Item)["']`),
}

func NewExecPythonTool() *ExecPythonTool {
	return &ExecPythonTool{
		pythonBin: detectPython(),
	}
}

func detectPython() string {
	for _, name := range pythonSearchOrder {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

func (t *ExecPythonTool) Name() string {
	return "exec_python"
}

func (t *ExecPythonTool) Description() string {
	return `Execute Python code and return the output. This is the ONLY runtime tool — use it for ALL tasks.

In sync mode (default): Python runs to completion, Go waits for exit code and output.
  Use subprocess.run() to call shell commands: subprocess.run(["go", "build", "./..."], capture_output=True, text=True)

In background mode: Python starts a long-running process, prints __DEVO_BG_PID__=<pid>, then exits.
  Use subprocess.Popen with start_new_session=True: p = subprocess.Popen(["npm", "run", "dev"], start_new_session=True)
  Python MUST print the PID marker and exit. Do NOT use subprocess.run for background processes.

Security: Python code is pre-checked for dangerous patterns.`
}

func (t *ExecPythonTool) RiskLevel() RiskLevel {
	return RiskLevelHigh
}

func (t *ExecPythonTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code": map[string]interface{}{
				"type":        "string",
				"description": "Python 代码片段，通过 python -c 执行",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "执行模式：sync（默认，等待任务完成）| background（启动后台进程后立即返回）",
				"enum":        []string{"sync", "background"},
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "执行超时时间（秒），sync 默认 30，background 默认 10",
			},
		},
		"required": []string{"code"},
	}
}

func (t *ExecPythonTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "exec_python"
}

func (t *ExecPythonTool) GetCommandContext(workingDir string, params map[string]interface{}) map[string]any {
	mode := "sync"
	if m, ok := params["mode"].(string); ok && m == "background" {
		mode = "background"
	}

	timeoutSeconds := 30
	if mode == "background" {
		timeoutSeconds = 10
	}
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	return map[string]any{
		"working_directory": workingDir,
		"invocation":        "python -c <code>",
		"mode":              mode,
		"timeout_seconds":   timeoutSeconds,
	}
}

func (t *ExecPythonTool) PreCheck(params map[string]interface{}) error {
	code, ok := params["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("missing required parameter: code")
	}

	if isPythonCodeBlacklisted(code) {
		return fmt.Errorf("code rejected by security blacklist")
	}

	return nil
}

func (t *ExecPythonTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	code, ok := params["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("missing required parameter: code")
	}

	code = strings.TrimSpace(code)

	mode := "sync"
	if m, ok := params["mode"].(string); ok && m == "background" {
		mode = "background"
	}

	timeoutSeconds := 30
	if mode == "background" {
		timeoutSeconds = 10
	}
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if t.pythonBin == "" {
		return fmt.Errorf("python not found: none of %v are available in PATH", pythonSearchOrder)
	}

	cmd := exec.CommandContext(execCtx, t.pythonBin, "-u", "-c", code)
	cmd.Dir = workingDir
	cmd.SysProcAttr = getPythonSysProcAttr()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start python: %w", err)
	}

	pythonPID := cmd.Process.Pid

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutBuf.WriteString(line + "\n")
			w.WriteChunk(line + "\n")
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuf.WriteString(line + "\n")
			w.WriteChunk("[stderr] " + line + "\n")
		}
	}()

	wg.Wait()
	err = cmd.Wait()

	timedOut := false
	cancelled := false
	exitCode := 0

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			timedOut = true
			exitCode = -1
			process.KillProcessGroup(pythonPID)
		} else if ctx.Err() != nil {
			cancelled = true
			exitCode = -1
			process.KillProcessGroup(pythonPID)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("python execution failed: %w", err)
		}
	}

	stdout := decodePythonOutput(stdoutBuf.Bytes())
	stderr := decodePythonOutput(stderrBuf.Bytes())

	if mode == "background" {
		return t.handleBackgroundResult(w, pythonPID, stdout, stderr, timedOut, cancelled)
	}

	return t.handleSyncResult(w, pythonPID, exitCode, stdout, stderr, timedOut, cancelled, timeoutSeconds)
}

func (t *ExecPythonTool) handleSyncResult(w StreamWriter, pythonPID, exitCode int, stdout, stderr string, timedOut, cancelled bool, timeoutSeconds int) error {
	pidTag := fmt.Sprintf("\n__DEVO_CHILD_PID__=%d", pythonPID)

	if timedOut {
		output := fmt.Sprintf("Python execution timed out after %d seconds.\nExit code: %d\nStdout:\n%s", timeoutSeconds, exitCode, stdout)
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	if cancelled {
		output := fmt.Sprintf("Python execution cancelled.\nExit code: %d\nStdout:\n%s", exitCode, stdout)
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	output := fmt.Sprintf("Exit code: %d\nStdout:\n%s", exitCode, stdout)
	if stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", stderr)
	}
	output += pidTag

	w.WriteDone(exitCode == 0, output)
	return nil
}

func (t *ExecPythonTool) handleBackgroundResult(w StreamWriter, pythonPID int, stdout, stderr string, timedOut, cancelled bool) error {
	pidTag := fmt.Sprintf("\n__DEVO_CHILD_PID__=%d", pythonPID)

	if timedOut {
		output := "Python background startup timed out.\n"
		bgPID := parseBGPID(stdout)
		if bgPID > 0 {
			output += fmt.Sprintf("但已检测到后台进程 PID: %d\n", bgPID)
			output += fmt.Sprintf("__DEVO_BG_PID__=%d", bgPID)
		} else {
			output += "未检测到 __DEVO_BG_PID__ 标记，后台进程可能未成功启动。\n"
			output += "请确保 Python 代码中打印了 __DEVO_BG_PID__=<pid> 后再退出。"
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	if cancelled {
		output := "Python background startup cancelled.\n"
		bgPID := parseBGPID(stdout)
		if bgPID > 0 {
			output += fmt.Sprintf("但后台进程已启动 (PID: %d)\n", bgPID)
			output += fmt.Sprintf("__DEVO_BG_PID__=%d", bgPID)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	bgPID := parseBGPID(stdout)
	if bgPID == 0 {
		output := "后台进程启动失败：Python 进程已退出，但未检测到 __DEVO_BG_PID__ 标记。\n"
		output += "请确保 Python 代码中打印了 __DEVO_BG_PID__=<pid> 后再退出。\n"
		if stdout != "" {
			output += fmt.Sprintf("\nStdout:\n%s", stdout)
		}
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	output := fmt.Sprintf("后台进程已启动 (PID: %d)\n", bgPID)
	if stdout != "" {
		output += fmt.Sprintf("启动输出:\n%s", stdout)
	}
	if stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", stderr)
	}
	output += fmt.Sprintf("\n__DEVO_BG_PID__=%d", bgPID)
	output += pidTag

	w.WriteDone(true, output)
	return nil
}

func parseBGPID(content string) int {
	marker := "__DEVO_BG_PID__="
	idx := findLastIndex(content, marker)
	if idx < 0 {
		return 0
	}

	start := idx + len(marker)
	end := start
	for end < len(content) && content[end] >= '0' && content[end] <= '9' {
		end++
	}

	if end == start {
		return 0
	}

	pid, err := strconv.Atoi(content[start:end])
	if err != nil {
		return 0
	}
	return pid
}

func isPythonCodeBlacklisted(code string) bool {
	for _, pattern := range pythonBlacklistPatterns {
		if pattern.MatchString(code) {
			return true
		}
	}
	return false
}

func decodePythonOutput(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if utf8.Valid(data) {
		return string(data)
	}

	if runtime.GOOS == "windows" {
		reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
		decoded, err := io.ReadAll(reader)
		if err == nil {
			result := string(decoded)
			if utf8.ValidString(result) {
				return result
			}
		}
	}

	return strings.ToValidUTF8(string(data), "\uFFFD")
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
