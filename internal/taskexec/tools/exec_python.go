package tools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
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
	return "Execute a Python code snippet and return the output. For simple, synchronous tasks like data processing, string manipulation, or JSON handling."
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
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "执行超时时间（秒），默认 30",
			},
		},
		"required": []string{"code"},
	}
}

func (t *ExecPythonTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "exec_python"
}

func (t *ExecPythonTool) GetCommandContext(workingDir string, params map[string]interface{}) map[string]any {
	timeoutSeconds := 30
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	return map[string]any{
		"working_directory": workingDir,
		"invocation":        "python -c <code>",
		"timeout_seconds":   timeoutSeconds,
	}
}

func (t *ExecPythonTool) PreCheck(params map[string]interface{}) error {
	code, ok := params["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("missing required parameter: code")
	}

	_ = code
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

	timeoutSeconds := 30
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

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = getPythonSysProcAttr()
	}

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
	exitCode := 0

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			timedOut = true
			exitCode = -1
		} else if ctx.Err() != nil {
			exitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("python execution failed: %w", err)
		}
	}

	stdout := decodePythonOutput(stdoutBuf.Bytes())
	stderr := decodePythonOutput(stderrBuf.Bytes())

	if timedOut {
		w.WriteDone(false, fmt.Sprintf("Python execution timed out after %d seconds.\nExit code: %d\nStdout:\n%s\nStderr:\n%s",
			timeoutSeconds, exitCode, stdout, stderr))
		return nil
	}

	output := fmt.Sprintf("Exit code: %d\nStdout:\n%s", exitCode, stdout)
	if stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", stderr)
	}

	w.WriteDone(true, output)
	return nil
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
