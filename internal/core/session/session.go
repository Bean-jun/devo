package session

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type State string

const (
	StateIdle             State = "Idle"
	StateProcessing       State = "Processing"
	StatePaused           State = "Paused"
	StateAwaitingApproval State = "AwaitingApproval"
	StateCompleted        State = "Completed"
	StateArchived         State = "Archived"
)

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
	EventBus                  *EventBus             `json:"-"`
	TrustLevel                string                `json:"trust_level"`
	ApprovalPolicy            map[string]string     `json:"approval_policy,omitempty"`
	ApprovalTimeoutSeconds    int                   `json:"approval_timeout_seconds"`
	CancelRequested           bool                  `json:"cancel_requested"`
	PauseRequested            bool                  `json:"pause_requested"`
	ToolCallLimit             int                   `json:"tool_call_limit"`
	ToolCallCount             int                   `json:"tool_call_count"`
	LastLoopTerminationReason LoopTerminationReason `json:"last_loop_termination_reason,omitempty"`
}

var (
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionNotIdle        = errors.New("session is not idle")
	ErrSessionConflict       = errors.New("session id already exists")
	ErrSessionArchived       = errors.New("session is archived")
	ErrSessionNotPaused      = errors.New("session is not paused")
	ErrSessionNotCompleted   = errors.New("session is not completed")
	ErrSessionNotProcessing  = errors.New("session is not processing")
	ErrSessionNotCancellable = errors.New("session is not in a cancellable state")
)

type SessionStore interface {
	Create(s *Session) error
	Get(id string) (*Session, error)
	Update(s *Session) error
	ListSessions(status, project string, limit, offset int) ([]Session, int, error)
	AddMessage(sessionID string, msg Message) error
	GetMessages(sessionID string, limit, offset int) ([]Message, int, error)
	GetEventBus(sessionID string) (*EventBus, error)
	AddEvent(sessionID string, event Event) error
	GetEvents(sessionID string, sinceID int64) ([]Event, error)
	IncrementSSEConnections(sessionID string) error
	DecrementSSEConnections(sessionID string) error
	Close() error
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%d-%04d", prefix, time.Now().UnixNano(), rng.Intn(10000))
}
