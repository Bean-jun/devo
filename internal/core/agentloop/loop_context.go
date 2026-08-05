package agentloop

import (
	"context"
	"strings"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

type LoopContext struct {
	SessionID string
	TraceID   string
	EventBus  *session.EventBus

	StepSeq         int
	TotalStepTokens int

	ActiveMsgs    []session.Message
	DynamicPrompt string
	LLMResult     *llmclient.CompleteResult

	PendingToolCall   *session.ToolCall
	PendingToolCalls  []session.ToolCall
	PendingToolResult *tools.ToolResult
	ApprovalCh        chan ApprovalDecision

	ExecutedToolCallIDs map[string]bool

	CancelCh chan struct{}
	PauseCh  chan struct{}
	ResumeCh chan struct{}

	PausedInState LoopState

	// TerminationReason 记录 loop 最终结束的原因
	TerminationReason string

	// ReasoningBuilder 累计本轮 LLM 流式输出的思考过程
	ReasoningBuilder strings.Builder

	// Ctx 是可取消的 context，当 Cancel() 被调用时会被取消。
	// 用于在 thinkingHandler 等长时间运行的 handler 中中断 LLM 流式请求。
	Ctx       context.Context
	CancelCtx context.CancelFunc
}
