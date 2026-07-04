package tools

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"devo/internal/taskexec/sandbox"
)

var unixBlacklistPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\s+-rf\s+/`),
	regexp.MustCompile(`\brm\s+-rf\s+~/`),
	regexp.MustCompile(`\brm\s+-rf\s+\$HOME\b`),
	regexp.MustCompile(`\bmkfs\b`),
	regexp.MustCompile(`\bdd\s+if=`),
	regexp.MustCompile(`:\(\)\s*\{`),
	regexp.MustCompile(`\bcurl\b.*\|\s*(?:ba)?sh\b`),
	regexp.MustCompile(`\bwget\b.*\|\s*(?:ba)?sh\b`),
	regexp.MustCompile(`>\s*/dev/sda`),
	regexp.MustCompile(`\bchmod\s+.*777\s+/`),
}

var windowsBlacklistPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bformat\s+[A-Z]:`),
	regexp.MustCompile(`(?i)\bdel\s+/f\s+/s\s+[A-Z]:\\\*`),
	regexp.MustCompile(`(?i)\bRemove-Item\s+-Recurse\s+[A-Z]:\\`),
	regexp.MustCompile(`(?i)\brd\s+/s\s+/q\s+[A-Z]:\\`),
	regexp.MustCompile(`(?i)\bInvoke-WebRequest\b.*\|\s*Invoke-Expression\b`),
	regexp.MustCompile(`(?i)\bdiskpart\b`),
	regexp.MustCompile(`(?i)\bcleanmgr\b`),
}

type ExecuteCommandTool struct {
	executor *sandbox.NativeExecutor
}

func NewExecuteCommandTool() *ExecuteCommandTool {
	return &ExecuteCommandTool{
		executor: sandbox.NewExecutor(),
	}
}

func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (t *ExecuteCommandTool) Description() string {
	return "Execute a shell command with security filtering and timeout control"
}

func (t *ExecuteCommandTool) RiskLevel() RiskLevel {
	return RiskLevelHigh
}

func (t *ExecuteCommandTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的 shell 命令",
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "命令执行超时时间（秒），默认 30",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "执行模式：sync(等待完成)、async(启动后台任务)、auto(自动检测，默认)",
				"enum":        []string{"sync", "async", "auto"},
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecuteCommandTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "execute_command"
}

func (t *ExecuteCommandTool) GetCommandContext(workingDir string, params map[string]interface{}) map[string]any {
	timeoutSeconds := 30
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	mode := "auto"
	if m, ok := params["mode"].(string); ok && m != "" {
		mode = m
	}

	ctx := map[string]any{
		"working_directory": workingDir,
		"invocation":        getShellInvocation() + " <command> (Go native executor)",
		"timeout_seconds":   timeoutSeconds,
		"mode":              mode,
	}

	if runtime.GOOS == "windows" {
		ctx["encoding"] = "智能编码检测：UTF-8 → GBK(CP936) → UTF-8 replace"
	}

	return ctx
}

func (t *ExecuteCommandTool) PreCheck(params map[string]interface{}) error {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return fmt.Errorf("missing required parameter: command")
	}

	command = strings.TrimSpace(command)

	if isBlacklisted(command) {
		return fmt.Errorf("command rejected by security blacklist: %s", command)
	}

	return nil
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		return fmt.Errorf("missing required parameter: command")
	}

	command = strings.TrimSpace(command)

	if err := isBlacklistedDetailed(command); err != nil {
		return err
	}

	timeoutSeconds := 30
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	mode := sandbox.ExecModeAuto
	if m, ok := params["mode"].(string); ok && m != "" {
		switch m {
		case "sync":
			mode = sandbox.ExecModeSync
		case "async":
			mode = sandbox.ExecModeAsync
		case "auto":
			mode = sandbox.ExecModeAuto
		}
	}

	result, err := t.executor.ExecuteStreaming(ctx, workingDir, command, timeoutSeconds, mode, func(line string, isStderr bool) {
		if isStderr {
			w.WriteChunk("[stderr] " + line + "\n")
		} else {
			w.WriteChunk(line + "\n")
		}
	})
	if err != nil {
		return fmt.Errorf("execution failed: %v", err)
	}

	pidTag := fmt.Sprintf("\n__DEVO_CHILD_PID__=%d", result.PID)

	if result.Background {
		output := "Background process started."
		if result.Stdout != "" {
			output += fmt.Sprintf("\nInitial output:\n%s", result.Stdout)
		}
		if result.Stderr != "" {
			output += fmt.Sprintf("\nInitial stderr:\n%s", result.Stderr)
		}
		output += fmt.Sprintf("\n__DEVO_BACKGROUND__=true")
		output += pidTag
		w.WriteDone(true, output)
		return nil
	}

	if result.TimedOut {
		w.WriteDone(false, fmt.Sprintf("Command timed out after %d seconds.\nExit code: %d\nStdout:\n%s\nStderr:\n%s%s",
			timeoutSeconds, result.ExitCode, result.Stdout, result.Stderr, pidTag))
		return nil
	}

	output := fmt.Sprintf("Exit code: %d\nStdout:\n%s", result.ExitCode, result.Stdout)
	if result.Stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", result.Stderr)
	}
	output += pidTag

	w.WriteDone(true, output)
	return nil
}

func isBlacklisted(command string) bool {
	return isBlacklistedDetailed(command) != nil
}

func isBlacklistedDetailed(command string) error {
	normalized := strings.ToLower(strings.TrimSpace(command))

	var patterns []*regexp.Regexp
	if runtime.GOOS == "windows" {
		patterns = windowsBlacklistPatterns
	} else {
		patterns = unixBlacklistPatterns
	}

	for _, pattern := range patterns {
		if pattern.MatchString(command) || pattern.MatchString(normalized) {
			return fmt.Errorf("command rejected by security blacklist: matched pattern %s", pattern.String())
		}
	}

	return nil
}

func getShellInvocation() string {
	if runtime.GOOS == "windows" {
		return "cmd /c"
	}
	return "sh -c"
}
