package types

type SessionInfo struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	WorkingDirectory string            `json:"working_directory"`
	State            string            `json:"state"`
	CreatedAt        string            `json:"created_at"`
	LastActiveAt     string            `json:"last_active_at"`
	TrustLevel       string            `json:"trust_level"`
	ApprovalPolicy   map[string]string `json:"approval_policy,omitempty"`
	TokenUsage       TokenUsage        `json:"token_usage"`
	MaxContextTokens int               `json:"max_context_tokens"`
}

type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

type Message struct {
	ID         string     `json:"id"`
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	CreatedAt  string     `json:"created_at"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
}

type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"is_dir"`
}

type ApprovalRequest struct {
	ApprovalID     string
	OperationType  string
	RiskLevel      string
	Summary        string
	Diff           string
	CommandPreview string
	Params         map[string]interface{}
}

type CreateSessionRequest struct {
	WorkingDirectory       string `json:"working_directory"`
	Title                  string `json:"title,omitempty"`
	ApprovalTimeoutSeconds int    `json:"approval_timeout_seconds,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type ApproveRequest struct {
	Decision string `json:"decision"`
}

type SetTrustRequest struct {
	TrustLevel string `json:"trust_level"`
}

type SetApprovalPolicyRequest struct {
	OperationType string `json:"operation_type"`
	PolicyLevel   string `json:"policy_level"`
}

type UpdateConfigRequest struct {
	ToolCallLimit int `json:"tool_call_limit,omitempty"`
}

type RollbackRequest struct {
	TargetMessageID string `json:"target_message_id"`
}

type RollbackResult struct {
	ActualRollbackMessageID string `json:"actual_rollback_message_id"`
	Adjusted                bool   `json:"adjusted"`
	AdjustmentReason        string `json:"adjustment_reason,omitempty"`
	DeletedCount            int    `json:"deleted_count"`
}

type SyncArchiveResult struct {
	ArchivePath   string `json:"archive_path"`
	LastMessageID string `json:"last_message_id"`
}

type ListSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
	Total    int           `json:"total"`
}

type GetMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int       `json:"total"`
}
