package tools

import (
	"context"
	"devo/internal/core/tokenmeter"
	"encoding/json"
	"fmt"
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
	OpExecPython         = "exec_python"
)

type StreamEventType string

const (
	StreamEventChunk StreamEventType = "chunk"
	StreamEventMeta  StreamEventType = "meta"
	StreamEventDone  StreamEventType = "done"
	StreamEventError StreamEventType = "error"
)

type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Stage   string          `json:"stage,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    string          `json:"data,omitempty"`
	Success bool            `json:"success,omitempty"`
	Summary string          `json:"summary,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type StreamWriter interface {
	WriteChunk(data string)
	WriteMeta(stage string)
	WriteDone(success bool, summary string)
	WriteError(err error)
}

type Tool interface {
	Name() string
	Description() string
	RiskLevel() RiskLevel
	ParamsSchema() map[string]interface{}
	Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error
}

type PreChecker interface {
	PreCheck(params map[string]interface{}) error
}

type OperationTypeProvider interface {
	OperationType(workingDir string, params map[string]interface{}) string
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
	if r == nil {
		return
	}
	r.tools[t.Name()] = t
}

func (r *Registry) Unregister(name string) {
	if r == nil {
		return
	}
	delete(r.tools, name)
}

func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) ListTools() []Tool {
	if r == nil {
		return nil
	}
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *Registry) Filter(toolNames []string) *Registry {
	filtered := NewRegistry()
	for _, name := range toolNames {
		if tool, ok := r.Get(name); ok {
			filtered.Register(tool)
		}
	}
	return filtered
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

func (r *Registry) Execute(ctx context.Context, workingDir string, toolName string, params map[string]interface{}) (<-chan StreamEvent, error) {
	if r == nil {
		ch := make(chan StreamEvent, 1)
		ch <- StreamEvent{
			Type:  StreamEventError,
			Error: "tool registry is nil",
		}
		close(ch)
		return ch, nil
	}
	t, ok := r.Get(toolName)
	if !ok {
		ch := make(chan StreamEvent, 1)
		ch <- StreamEvent{
			Type:  StreamEventError,
			Error: "unknown tool: " + toolName,
		}
		close(ch)
		return ch, nil
	}

	ch := make(chan StreamEvent, 256)
	sw := NewChannelStreamWriter(ch)

	sw.WriteMeta("starting")

	go func() {
		defer close(ch)

		sw.WriteMeta("running")

		if err := t.Execute(ctx, workingDir, params, sw); err != nil {
			sw.WriteMeta("error")
			sw.WriteError(err)
			return
		}

		sw.WriteMeta("done")
	}()

	return ch, nil
}

func (r *Registry) GetTool(name string) (Tool, bool) {
	return r.Get(name)
}

type ChannelStreamWriter struct {
	ch      chan<- StreamEvent
	done    bool
	success bool
	summary string
	err     error
}

func NewChannelStreamWriter(ch chan<- StreamEvent) *ChannelStreamWriter {
	return &ChannelStreamWriter{ch: ch}
}

func (w *ChannelStreamWriter) WriteChunk(data string) {
	w.ch <- StreamEvent{
		Type: StreamEventChunk,
		Data: data,
	}
}

func (w *ChannelStreamWriter) WriteMeta(stage string) {
	w.ch <- StreamEvent{
		Type:    StreamEventMeta,
		Stage:   stage,
		Message: stage,
	}
}

func (w *ChannelStreamWriter) WriteDone(success bool, summary string) {
	w.done = true
	w.success = success
	w.summary = summary
	w.ch <- StreamEvent{
		Type:    StreamEventDone,
		Success: success,
		Summary: summary,
	}
}

func (w *ChannelStreamWriter) WriteError(err error) {
	w.err = err
	w.ch <- StreamEvent{
		Type:  StreamEventError,
		Error: err.Error(),
	}
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

func CollectToolResult(events <-chan StreamEvent, onEvent ...func(evt StreamEvent)) *ToolResult {
	result := &ToolResult{Success: true}
	var contentParts []string

	for evt := range events {
		if len(onEvent) > 0 && onEvent[0] != nil {
			onEvent[0](evt)
		}
		switch evt.Type {
		case StreamEventChunk:
			contentParts = append(contentParts, evt.Data)
		case StreamEventDone:
			result.Success = evt.Success
			if evt.Summary != "" {
				contentParts = append(contentParts, evt.Summary)
			}
		case StreamEventError:
			result.Success = false
			result.Error = evt.Error
		}
	}

	if len(contentParts) > 0 {
		result.Content = fmt.Sprintf("%s", joinStrings(contentParts))
	}

	if !result.Success && result.Error != "" {
		result.Content = result.Error
	}

	return result
}

func joinStrings(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += parts[i]
	}
	return result
}
