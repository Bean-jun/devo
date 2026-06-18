package tools

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
)

type Tool interface {
	Name() string
	Description() string
	RiskLevel() RiskLevel
	Execute(workingDir string, params map[string]interface{}) (string, error)
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

func (r *Registry) GetTool(name string) (Tool, bool) {
	return r.Get(name)
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Success    bool   `json:"success"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`
}
