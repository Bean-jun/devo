package tools

import (
	"context"
	"devo/internal/core/tokenmeter"
	"encoding/json"
)

type RiskLevel string

const (
	RiskLevelNone   RiskLevel = "none"
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

const (
	OpFileWriteNew       = "file_write_new"
	OpFileWriteOverwrite = "file_write_overwrite"
	OpFileEdit           = "file_edit"
	OpExecuteCommand     = "execute_command"
)

type Tool interface {
	Name() string
	Description() string
	RiskLevel() RiskLevel
	ParamsSchema() map[string]interface{}
	Execute(workingDir string, params map[string]interface{}) (string, error)
}

type PreChecker interface {
	PreCheck(params map[string]interface{}) error
}

type OperationTypeProvider interface {
	OperationType(workingDir string, params map[string]interface{}) string
}

type ToolProgress struct {
	Stage    string  `json:"stage"`
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Unregister(name string) {
	delete(r.tools, name)
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) ListTools() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func EstimateToolTokens(toolList []Tool) int {
	total := 0
	for _, t := range toolList {
		total += tokenmeter.EstimateTokens(t.Name())
		total += tokenmeter.EstimateTokens(t.Description())
		if paramsJSON, err := json.Marshal(t.ParamsSchema()); err == nil {
			total += tokenmeter.EstimateTokens(string(paramsJSON))
		}
	}
	return total
}

func (r *Registry) Execute(workingDir string, toolName string, params map[string]interface{}) (*ToolResult, error) {
	t, ok := r.Get(toolName)
	if !ok {
		return &ToolResult{
			Success: false,
			Error:   "unknown tool: " + toolName,
		}, nil
	}

	content, err := t.Execute(workingDir, params)
	if err != nil {
		return &ToolResult{
			Success: false,
			Content: "",
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Content: content,
	}, nil
}

func (r *Registry) ExecuteAsync(ctx context.Context, workingDir string, toolName string, params map[string]interface{}, onProgress func(ToolProgress)) (*ToolResult, error) {
	if onProgress != nil {
		onProgress(ToolProgress{Stage: "starting", Message: "Preparing to execute " + toolName, Progress: 0.0})
	}

	t, ok := r.Get(toolName)
	if !ok {
		return &ToolResult{
			Success: false,
			Error:   "unknown tool: " + toolName,
		}, nil
	}

	if onProgress != nil {
		onProgress(ToolProgress{Stage: "running", Message: "Executing " + toolName, Progress: 0.3})
	}

	content, err := t.Execute(workingDir, params)
	if err != nil {
		if onProgress != nil {
			onProgress(ToolProgress{Stage: "error", Message: err.Error(), Progress: 1.0})
		}
		return &ToolResult{
			Success: false,
			Content: "",
			Error:   err.Error(),
		}, nil
	}

	if onProgress != nil {
		onProgress(ToolProgress{Stage: "done", Message: "Completed " + toolName, Progress: 1.0})
	}

	return &ToolResult{
		Success: true,
		Content: content,
	}, nil
}

func (r *Registry) GetTool(name string) (Tool, bool) {
	return r.Get(name)
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Success    bool   `json:"success"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`

	ExitCode int    `json:"exit_code,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	TimedOut bool   `json:"timed_out,omitempty"`
}
