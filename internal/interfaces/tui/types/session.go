package types

type SessionState string

const (
	SessionStateIdle             SessionState = "idle"
	SessionStateThinking         SessionState = "thinking"
	SessionStateToolExecuting    SessionState = "tool_executing"
	SessionStateProcessing       SessionState = "processing"
	SessionStateAwaitingApproval SessionState = "awaiting_approval"
	SessionStatePaused           SessionState = "paused"
	SessionStateCancelled        SessionState = "cancelled"
	SessionStateCompleted        SessionState = "completed"
	SessionStateArchived         SessionState = "archived"
)

type TrustLevel string

const (
	TrustLevelNormal   TrustLevel = "normal"
	TrustLevelElevated TrustLevel = "elevated"
)

type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

type SessionInfo struct {
	ID                   string       `json:"id"`
	Title                string       `json:"title"`
	State                SessionState `json:"state"`
	WorkingDirectory     string       `json:"working_directory"`
	CreatedAt            string       `json:"created_at"`
	LastActiveAt         string       `json:"last_active_at"`
	MessageCount         int          `json:"message_count"`
	TokenUsage           TokenUsage   `json:"token_usage"`
	TrustLevel           TrustLevel   `json:"trust_level"`
	CurrentContextTokens int          `json:"current_context_tokens"`
	LastMessageContent   string       `json:"last_message_content"`
	LastMessageTime      string       `json:"last_message_time"`
}

type CreateSessionRequest struct {
	WorkingDirectory       string `json:"working_directory"`
	Title                  string `json:"title,omitempty"`
	ApprovalTimeoutSeconds int    `json:"approval_timeout_seconds,omitempty"`
}

type ListSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
	Total    int           `json:"total"`
}
