package agentloop

import (
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

type LoopContext struct {
	SessionID string
	EventBus  *session.EventBus

	StepSeq         int
	TotalStepTokens int
	HasFileChange   bool

	ActiveMsgs    []session.Message
	DynamicPrompt string
	LLMResult     *llmclient.CompleteResult

	PendingToolCall   *session.ToolCall
	PendingToolResult *tools.ToolResult
	ApprovalCh        chan ApprovalDecision

	CancelCh chan struct{}
	PauseCh  chan struct{}
	ResumeCh chan struct{}

	PausedInState LoopState
}
