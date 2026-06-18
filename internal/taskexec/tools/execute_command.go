package tools

import (
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
	executor *sandbox.Executor
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

func (t *ExecuteCommandTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "execute_command"
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

func (t *ExecuteCommandTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("missing required parameter: command")
	}

	command = strings.TrimSpace(command)

	if err := isBlacklistedDetailed(command); err != nil {
		return "", err
	}

	timeoutSeconds := 30
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	if !sandbox.IsPythonAvailable() {
		return "", fmt.Errorf("python is not available on this system")
	}

	result, pid, err := t.executor.Execute(workingDir, command, timeoutSeconds)

	pidTag := fmt.Sprintf("\n__DEVO_CHILD_PID__=%d", pid)

	if err != nil {
		return "", fmt.Errorf("execution failed: %v", err)
	}

	if result.TimedOut {
		return fmt.Sprintf("Command timed out after %d seconds.\nExit code: %d\nStdout:\n%s\nStderr:\n%s%s",
			timeoutSeconds, result.ExitCode, result.Stdout, result.Stderr, pidTag), nil
	}

	output := fmt.Sprintf("Exit code: %d\nStdout:\n%s", result.ExitCode, result.Stdout)
	if result.Stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", result.Stderr)
	}
	output += pidTag

	return output, nil
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
