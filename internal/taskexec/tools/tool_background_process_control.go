package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type ListBackgroundProcessesTool struct {
	manager *BackgroundProcessManager
}

func NewListBackgroundProcessesTool(manager *BackgroundProcessManager) *ListBackgroundProcessesTool {
	return &ListBackgroundProcessesTool{manager: manager}
}

func (t *ListBackgroundProcessesTool) Name() string { return "list_background_processes" }

func (t *ListBackgroundProcessesTool) Description() string {
	return `List background processes started in this session via exec_python(mode="background").
Shows PID, the command used to start it, whether it's still alive, when it started, and its log
file path (if any). Use this before starting a new background process on a port/service you may
have already started earlier, and before guessing a PID to stop — don't rely on remembering PIDs
from earlier in the conversation.`
}

func (t *ListBackgroundProcessesTool) RiskLevel() RiskLevel { return RiskLevelLow }

func (t *ListBackgroundProcessesTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListBackgroundProcessesTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "list_background_processes"
}

func (t *ListBackgroundProcessesTool) GetCommandContext(workingDir string, params map[string]interface{}) map[string]any {
	return map[string]any{}
}

func (t *ListBackgroundProcessesTool) PreCheck(params map[string]interface{}) error { return nil }

func (t *ListBackgroundProcessesTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	sessionID := SessionIDFromContext(ctx)
	procs := t.manager.List(sessionID)
	if len(procs) == 0 {
		w.WriteDone(true, "No background processes are currently registered for this session.")
		return nil
	}

	data, err := json.MarshalIndent(procs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal background process list: %w", err)
	}
	w.WriteDone(true, string(data))
	return nil
}

type StopBackgroundProcessTool struct {
	manager *BackgroundProcessManager
}

func NewStopBackgroundProcessTool(manager *BackgroundProcessManager) *StopBackgroundProcessTool {
	return &StopBackgroundProcessTool{manager: manager}
}

func (t *StopBackgroundProcessTool) Name() string { return "stop_background_process" }

func (t *StopBackgroundProcessTool) Description() string {
	return `Stop a background process previously started via exec_python(mode="background").
Pass the pid printed as __DEVO_BG_PID__=<pid> when it was started. This sends a graceful
termination signal first, waits briefly, then force-kills the entire process tree if it
hasn't exited. Do NOT try to kill background processes yourself with os.killpg/taskkill from
exec_python — use this tool instead, so PID-reuse safety and cross-platform kill behavior are
handled correctly. If you don't remember the PID, call list_background_processes first.`
}

func (t *StopBackgroundProcessTool) RiskLevel() RiskLevel { return RiskLevelMedium }

func (t *StopBackgroundProcessTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pid": map[string]interface{}{
				"type":        "integer",
				"description": "PID of the background process to stop, as printed in __DEVO_BG_PID__=<pid>",
			},
		},
		"required": []string{"pid"},
	}
}

func (t *StopBackgroundProcessTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "stop_background_process"
}

func (t *StopBackgroundProcessTool) GetCommandContext(workingDir string, params map[string]interface{}) map[string]any {
	pid, _ := params["pid"].(float64)
	return map[string]any{"pid": int(pid)}
}

func (t *StopBackgroundProcessTool) PreCheck(params map[string]interface{}) error {
	if _, ok := params["pid"].(float64); !ok {
		return fmt.Errorf("missing required parameter: pid")
	}
	return nil
}

func (t *StopBackgroundProcessTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	pidFloat, ok := params["pid"].(float64)
	if !ok {
		return fmt.Errorf("missing required parameter: pid")
	}
	pid := int(pidFloat)

	if err := t.manager.Stop(pid); err != nil {
		w.WriteDone(false, fmt.Sprintf("Failed to stop process %d: %v", pid, err))
		return nil
	}
	w.WriteDone(true, fmt.Sprintf("Process %d stopped.", pid))
	return nil
}
