package types

type Message struct {
	ID          string     `json:"id"`
	Role        string     `json:"role"`
	Content     string     `json:"content"`
	Thinking    string     `json:"thinking,omitempty"`
	CreatedAt   string     `json:"created_at"`
	ToolCalls   []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID  string     `json:"tool_call_id,omitempty"`
	IsStreaming bool       `json:"-"`
}

type ToolCall struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Name     string                 `json:"name"`
	Summary  string                 `json:"summary"`
	Status   string                 `json:"status"`
	Duration string                 `json:"duration"`
	Diff     string                 `json:"diff"`
	Input    string                 `json:"input,omitempty"`
	Output   string                 `json:"output,omitempty"`
	Params   map[string]interface{} `json:"params"`
	Expanded bool                   `json:"expanded"`
}

type SendMessageRequest struct {
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

type GetMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int       `json:"total"`
}

type RollbackRequest struct {
	TargetMessageID string `json:"target_message_id"`
}
