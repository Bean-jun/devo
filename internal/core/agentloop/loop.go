package agentloop

import (
	"context"
	"fmt"
	"sync"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/archive"
	"devo/internal/core/compressor"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/prompt"
	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

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
	promptAssembler  *prompt.Assembler
	toolExecutor     ToolExecutor
	approvalManager  *approval.Manager
	approvalChannels map[string]chan ApprovalDecision
	tokenMeter       *tokenmeter.Meter
	compressor       *compressor.Compressor
	pathLockManager  *concurrency.PathLockManager
	archiveManager   *archive.ArchiveManager
	memoryManager    *memory.Manager
	mu               sync.Mutex
}

func New(store session.SessionStore, llmClient llmclient.Client) *Loop {
	pathLockManager := concurrency.NewPathLockManager()
	return &Loop{
		store:            store,
		llmClient:        llmClient,
		promptAssembler:  prompt.NewAssembler(),
		approvalManager:  approval.NewManager(),
		approvalChannels: make(map[string]chan ApprovalDecision),
		tokenMeter:       tokenmeter.NewMeter(store),
		compressor:       compressor.New(llmClient, store),
		pathLockManager:  pathLockManager,
		archiveManager:   archive.NewArchiveManager(store, pathLockManager),
	}
}

func NewWithTools(store session.SessionStore, llmClient llmclient.Client, toolExecutor ToolExecutor) *Loop {
	pathLockManager := concurrency.NewPathLockManager()
	return &Loop{
		store:            store,
		llmClient:        llmClient,
		promptAssembler:  prompt.NewAssembler(),
		toolExecutor:     toolExecutor,
		approvalManager:  approval.NewManager(),
		approvalChannels: make(map[string]chan ApprovalDecision),
		tokenMeter:       tokenmeter.NewMeter(store),
		compressor:       compressor.New(llmClient, store),
		pathLockManager:  pathLockManager,
		archiveManager:   archive.NewArchiveManager(store, pathLockManager),
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
		l.archiveManager.AppendSystemMessage(sessionID, pauseMsg.Content)
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

	l.archiveManager.AppendUserMessage(sessionID, content)

	eventBus, err := l.store.GetEventBus(sessionID)
	if err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("get event bus: %w", err)
	}

	go l.runAgentLoop(context.Background(), sessionID, eventBus)

	return nil
}

func (l *Loop) GetArchiveContent(sessionID string) ([]byte, error) {
	return l.archiveManager.GetArchiveContent(sessionID)
}

func (l *Loop) SyncArchive(sessionID string) (string, error) {
	return l.archiveManager.SyncArchive(sessionID)
}

func (l *Loop) SetPromptAssembler(pa *prompt.Assembler) {
	l.promptAssembler = pa
}

func (l *Loop) SetMemoryManager(mm *memory.Manager) {
	l.memoryManager = mm
	if l.promptAssembler != nil {
		l.promptAssembler.SetMemoryProvider(mm)
	}
}

func (l *Loop) GetApprovalManager() *approval.Manager {
	return l.approvalManager
}
