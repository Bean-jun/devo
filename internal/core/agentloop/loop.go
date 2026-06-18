package agentloop

import (
	"context"
	"fmt"
	"sync"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

const defaultSystemPrompt = "You are a helpful coding assistant. Respond concisely and helpfully."

type ToolExecutor interface {
	Execute(workingDir string, toolName string, params map[string]interface{}) (*tools.ToolResult, error)
	GetTool(name string) (tools.Tool, bool)
	ListTools() []tools.Tool
}

type ApprovalDecision struct {
	ApprovalID string
	Decision   string
	ResultCh   chan error
}

type Loop struct {
	store            session.SessionStore
	llmClient        llmclient.Client
	systemPrompt     string
	toolExecutor     ToolExecutor
	approvalManager  *approval.Manager
	approvalChannels map[string]chan ApprovalDecision
	mu               sync.Mutex
}

func New(store session.SessionStore, llmClient llmclient.Client) *Loop {
	return &Loop{
		store:            store,
		llmClient:        llmClient,
		systemPrompt:     defaultSystemPrompt,
		approvalManager:  approval.NewManager(),
		approvalChannels: make(map[string]chan ApprovalDecision),
	}
}

func NewWithTools(store session.SessionStore, llmClient llmclient.Client, toolExecutor ToolExecutor) *Loop {
	return &Loop{
		store:            store,
		llmClient:        llmClient,
		systemPrompt:     defaultSystemPrompt,
		toolExecutor:     toolExecutor,
		approvalManager:  approval.NewManager(),
		approvalChannels: make(map[string]chan ApprovalDecision),
	}
}

func (l *Loop) ProcessMessage(ctx context.Context, sessionID, content string) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State == session.StateArchived {
		return fmt.Errorf("%w: session is archived", session.ErrSessionArchived)
	}

	if sess.State == session.StateProcessing || sess.State == session.StateAwaitingApproval {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotIdle, sess.State)
	}

	isContinuation := sess.LastLoopTerminationReason == session.LoopTerminationToolLimitReached

	if sess.State == session.StatePaused {
		sess.State = session.StateProcessing
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			return fmt.Errorf("resume session from paused: %w", err)
		}
	} else {
		sess.State = session.StateProcessing
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			return fmt.Errorf("update session state to processing: %w", err)
		}
	}

	if isContinuation {
		pauseMsg := session.Message{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleSystem,
			Content:   fmt.Sprintf("上一次任务因达到工具调用上限（%d 次）而暂停。当前任务尚未完成，请从中断点继续。已完成的工作保留在文件系统中，无需重复执行。", sess.ToolCallCount),
			CreatedAt: time.Now(),
		}
		if err := l.store.AddMessage(sessionID, pauseMsg); err != nil {
			sess.State = session.StateIdle
			l.store.Update(sess)
			return fmt.Errorf("add continuation system message: %w", err)
		}
		sess.ToolCallCount = 0
		sess.LastLoopTerminationReason = ""
		l.store.Update(sess)
	}

	userMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleUser,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if err := l.store.AddMessage(sessionID, userMsg); err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("add user message: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("get event bus: %w", err)
	}

	go l.runAgentLoop(context.Background(), sessionID, eventBus)

	return nil
}
