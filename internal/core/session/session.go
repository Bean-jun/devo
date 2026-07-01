package session

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type State string

const (
	StateIdle             State = "Idle"             // 等待用户输入
	StateThinking         State = "Thinking"         // LLM 流式生成中（新增）
	StateToolExecuting    State = "ToolExecuting"    // 工具执行中（新增）
	StateAwaitingApproval State = "AwaitingApproval" // 等待用户审批
	StatePaused           State = "Paused"           // 工具执行暂停
	StateCompleted        State = "Completed"        // 会话正常结束
	StateArchived         State = "Archived"         // 已归档
	// StateProcessing       State = "Processing"       // 废弃：由 StateThinking + StateToolExecuting 替代
)

// ToSnakeCase converts PascalCase state to snake_case (e.g., "AwaitingApproval" → "awaiting_approval")
func (s State) ToSnakeCase() string {
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

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

type ToolCall struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
}

type Message struct {
	ID         string     `json:"id"`
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type LoopTerminationReason string

const (
	LoopTerminationCompleted        LoopTerminationReason = "completed"
	LoopTerminationCancelled        LoopTerminationReason = "cancelled"
	LoopTerminationToolLimitReached LoopTerminationReason = "tool_limit_reached"
	LoopTerminationError            LoopTerminationReason = "error"
)

const DefaultToolCallLimit = 50

type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

type CompressedRange struct {
	StartMessageID string `json:"start_message_id"`
	EndMessageID   string `json:"end_message_id"`
}

type CompressionSummary struct {
	SummaryText string          `json:"summary_text"`
	CoversRange CompressedRange `json:"covers_range"`
	CreatedAt   time.Time       `json:"created_at"`
}

type CompressionState struct {
	CompressedRanges []CompressedRange    `json:"compressed_ranges"`
	Summaries        []CompressionSummary `json:"summaries"`
}

type Session struct {
	ID                        string                `json:"id"`
	Title                     string                `json:"title"`
	WorkingDirectory          string                `json:"working_directory"`
	State                     State                 `json:"state"`
	CreatedAt                 time.Time             `json:"created_at"`
	LastActiveAt              time.Time             `json:"last_active_at"`
	Messages                  []Message             `json:"messages,omitempty"`
	ActiveSSEConnections      int                   `json:"active_sse_connections"`
	ChildPID                  *int                  `json:"child_pid,omitempty"`
	BackgroundPIDs            []int                 `json:"background_pids,omitempty"`
	EventBus                  *EventBus             `json:"-"`
	TrustLevel                string                `json:"trust_level"`
	ApprovalPolicy            map[string]string     `json:"approval_policy,omitempty"`
	ApprovalTimeoutSeconds    int                   `json:"approval_timeout_seconds"`
	CancelRequested           bool                  `json:"cancel_requested"`
	PauseRequested            bool                  `json:"pause_requested"`
	ToolCallLimit             int                   `json:"tool_call_limit"`
	ToolCallCount             int                   `json:"tool_call_count"`
	MessageCount              int                   `json:"message_count"`
	LastLoopTerminationReason LoopTerminationReason `json:"last_loop_termination_reason,omitempty"`
	TokenUsage                TokenUsage            `json:"token_usage"`
	CompressionState          *CompressionState     `json:"compression_state,omitempty"`
	CompressionCount          int                   `json:"compression_count"`
	CompressThreshold         int                   `json:"compress_threshold"`
	KeepRecent                int                   `json:"keep_recent"`
	MaxContextTokens          int                   `json:"max_context_tokens"`
	MaxConcurrentToolCalls    int                   `json:"max_concurrent_tool_calls"`
	MaxConcurrentSubprocesses int                   `json:"max_concurrent_subprocesses"`
	ArchivePath               string                `json:"archive_path,omitempty"`
	SystemPromptOverride      string                `json:"system_prompt_override,omitempty"`
	CurrentContextTokens      int                   `json:"current_context_tokens"`
	ActiveSkills              []string              `json:"active_skills,omitempty"`
	CachedDirectorySummary    *DirectorySummary     `json:"-"`
}

type DirectorySummary struct {
	Content     string    `json:"content"`
	GeneratedAt time.Time `json:"generated_at"`
	Valid       bool      `json:"valid"`
}

var (
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionNotIdle        = errors.New("session is not idle")
	ErrSessionConflict       = errors.New("session id already exists")
	ErrSessionArchived       = errors.New("session is archived")
	ErrSessionNotPaused      = errors.New("session is not paused")
	ErrSessionNotCompleted   = errors.New("session is not completed")
	ErrSessionNotProcessing  = errors.New("session is not in tool_executing state")
	ErrSessionNotCancellable = errors.New("session is not in a cancellable state")
	ErrMessageNotFound       = errors.New("message not found")
)

type FileModificationRecord struct {
	SessionID         string    `json:"session_id"`
	FilePath          string    `json:"file_path"`
	ModifiedAt        time.Time `json:"modified_at"`
	CausedByMessageID string    `json:"caused_by_message_id"`
}

type SessionStore interface {
	Create(s *Session) error
	Get(id string) (*Session, error)
	Update(s *Session) error
	ListSessions(status, project string, limit, offset int) ([]Session, int, error)
	ListUniqueWorkspaces() ([]string, error)
	DeleteByWorkspace(path string) (int, error)
	DeleteSession(id string) error
	AddMessage(sessionID string, msg Message) error
	GetMessages(sessionID string, limit, offset int) ([]Message, int, error)
	GetEventBus(sessionID string) (*EventBus, error)
	AddEvent(sessionID string, event Event) error
	GetEvents(sessionID string, sinceID int64) ([]Event, error)
	IncrementSSEConnections(sessionID string) error
	DecrementSSEConnections(sessionID string) error
	AddUsageStep(sessionID string, stepSeq int, inputTokens, outputTokens int, source string) error
	GetUsageSteps(sessionID string) ([]UsageStepRecord, error)
	UpdateSessionUsage(sessionID string, inputTokens, outputTokens int) error
	GetUsageStats(groupBy, dateRange, project string) (*UsageStatsResult, error)
	Close() error

	DeleteMessagesAfter(sessionID string, messageID string) (int, error)
	GetMessageByID(sessionID string, messageID string) (*Message, error)
	RecordFileModification(record FileModificationRecord) error
	GetFileModifications(sessionID string) ([]FileModificationRecord, error)
	DeleteFileModificationsAfter(sessionID string, afterTime time.Time) error
}

type UsageStepRecord struct {
	SessionID    string    `json:"session_id"`
	StepSeq      int       `json:"step_seq"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
}

type UsageStatsResult struct {
	Groups  []UsageGroup `json:"groups"`
	Summary TokenUsage   `json:"summary"`
}

type UsageGroup struct {
	Key          string `json:"key"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%d-%04d", prefix, time.Now().UnixNano(), rng.Intn(10000))
}
