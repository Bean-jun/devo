package types

type SessionInfo struct {
	ID                   string            `json:"id"`
	Title                string            `json:"title"`
	WorkingDirectory     string            `json:"working_directory"`
	State                string            `json:"state"`
	CreatedAt            string            `json:"created_at"`
	LastActiveAt         string            `json:"last_active_at"`
	MessageCount         int               `json:"message_count"`
	TrustLevel           string            `json:"trust_level"`
	ApprovalPolicy       map[string]string `json:"approval_policy,omitempty"`
	TokenUsage           TokenUsage        `json:"token_usage"`
	CurrentContextTokens int               `json:"current_context_tokens"`
	MaxContextTokens     int               `json:"max_context_tokens"`
}

// NormalizeState converts the State field from PascalCase to snake_case.
func (s *SessionInfo) NormalizeState() {
	s.State = ToSnakeCase(s.State)
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

// ToSnakeCase converts PascalCase to snake_case (e.g., "AwaitingApproval" -> "awaiting_approval")
func ToSnakeCase(s string) string {
	var result []byte
	for i, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(ch+32))
		} else {
			result = append(result, byte(ch))
		}
	}
	return string(result)
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
