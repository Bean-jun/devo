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
	"devo/internal/core/skills"
	"devo/internal/core/tokenmeter"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

type ToolExecutor interface {
	Execute(ctx context.Context, workingDir string, toolName string, params map[string]interface{}) (<-chan tools.StreamEvent, error)
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
	skillsManager    *skills.Manager
	solidifier       *skills.Solidifier
	bgManager        *tools.BackgroundProcessManager
	stateMachine     *StateMachine
	activeLoops      sync.Map
	loopWG           sync.WaitGroup
	mu               sync.Mutex
}

func New(store session.SessionStore, llmClient llmclient.Client) *Loop {
	pathLockManager := concurrency.NewPathLockManager()
	l := &Loop{
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
	l.stateMachine = NewStateMachine()
	l.registerHandlers(l.stateMachine)
	return l
}

func NewWithTools(store session.SessionStore, llmClient llmclient.Client, toolExecutor ToolExecutor) *Loop {
	pathLockManager := concurrency.NewPathLockManager()
	l := &Loop{
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
	l.stateMachine = NewStateMachine()
	l.registerHandlers(l.stateMachine)
	return l
}

func (l *Loop) EstimateInitialContextTokens(sess *session.Session) int {
	systemPrompt := l.promptAssembler.Assemble(sess)
	tokens := tokenmeter.EstimateTokens(systemPrompt)
	if l.toolExecutor != nil {
		tokens += tools.EstimateToolTokens(l.toolExecutor.ListTools())
	}
	return tokens
}

func (l *Loop) UpdateLLMClient(client llmclient.Client) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.llmClient = client
	l.compressor.UpdateLLMClient(client)
}

func (l *Loop) ProcessMessage(ctx context.Context, sessionID string, msg session.Message) error {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if sess.State == session.StateArchived {
		return fmt.Errorf("%w: session is archived", session.ErrSessionArchived)
	}

	if sess.State == session.StateThinking || sess.State == session.StateToolExecuting || sess.State == session.StateAwaitingApproval {
		return fmt.Errorf("%w: current state is %s", session.ErrSessionNotIdle, sess.State)
	}

	isContinuation := sess.LastLoopTerminationReason == session.LoopTerminationToolLimitReached

	oldState := string(sess.State)

	if sess.State == session.StatePaused {
		sess.State = session.StateThinking
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			return fmt.Errorf("resume session from paused: %w", err)
		}
	} else {
		sess.State = session.StateThinking
		sess.LastActiveAt = time.Now()
		if err := l.store.Update(sess); err != nil {
			return fmt.Errorf("update session state to thinking: %w", err)
		}
	}

	if eventBus, err := l.store.GetEventBus(sessionID); err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": session.State(oldState).ToSnakeCase(),
			"new_state": session.StateThinking.ToSnakeCase(),
			"reason":    "user_message",
		})
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
		ID:           msg.ID,
		Role:         session.RoleUser,
		Content:      msg.Content,
		ContentParts: msg.ContentParts,
		CreatedAt:    time.Now(),
	}
	if userMsg.ID == "" {
		userMsg.ID = session.GenerateID("msg")
	}
	if err := l.store.AddMessage(sessionID, userMsg); err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("add user message: %w", err)
	}

	l.archiveManager.AppendUserMessage(sessionID, msg)

	eventBus, err := l.store.GetEventBus(sessionID)
	if err != nil {
		sess.State = session.StateIdle
		l.store.Update(sess)
		return fmt.Errorf("get event bus: %w", err)
	}

	eventBus.Publish("thinking", map[string]string{
		"message": "开始处理用户请求...",
	})

	traceID := logging.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = logging.GenerateTraceID()
	}
	logging.WithSessionID(ctx, sessionID)

	loopCtx, loopCancel := context.WithCancel(context.Background())
	loopCtx = logging.WithTraceID(loopCtx, traceID)
	loopCtx = logging.WithSessionID(loopCtx, sessionID)

	lc := &LoopContext{
		SessionID: sessionID,
		TraceID:   traceID,
		EventBus:  eventBus,
		CancelCh:  make(chan struct{}, 1),
		PauseCh:   make(chan struct{}, 1),
		ResumeCh:  make(chan struct{}, 1),
		Ctx:       loopCtx,
		CancelCtx: loopCancel,
	}

	l.activeLoops.Store(sessionID, lc)
	l.loopWG.Add(1)
	go func() {
		defer l.loopWG.Done()
		defer l.activeLoops.Delete(sessionID)
		defer loopCancel()
		l.stateMachine.Run(loopCtx, lc)

		sess, err := l.store.Get(sessionID)
		if err == nil && sess.State != session.StatePaused {
			sess.State = session.StateIdle
			sess.LastActiveAt = time.Now()
			l.store.Update(sess)
		}

		lc.EventBus.Publish("loop.completed_with_reason", map[string]any{
			"session_id": sessionID,
			"reason":     lc.TerminationReason,
		})
	}()

	return nil
}

func (l *Loop) WaitForCompletion() {
	l.loopWG.Wait()
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

func (l *Loop) SetSkillsManager(sm *skills.Manager) {
	l.skillsManager = sm
	if l.promptAssembler != nil {
		l.promptAssembler.SetSkillsProvider(sm)
	}
}

func (l *Loop) GetSkillsManager() *skills.Manager {
	return l.skillsManager
}

func (l *Loop) SetSolidifier(sol *skills.Solidifier) {
	l.solidifier = sol
}

// SetBackgroundProcessManager injects the background process manager. Required
// for exec_python background mode to register/stream processes and for
// Cancel/Complete to stop them. May be nil in tests.
func (l *Loop) SetBackgroundProcessManager(mgr *tools.BackgroundProcessManager) {
	l.bgManager = mgr
}

// ForwardBackgroundOutput implements tools.OutputForwarder. It is called by
// the BackgroundProcessManager's pipe-reader goroutines whenever a background
// process emits output. We publish a "background_output" event on the session's
// EventBus so any SSE subscribers receive it. If the session has no bus (e.g.
// was deleted), the output is silently dropped.
func (l *Loop) ForwardBackgroundOutput(sessionID string, pid int, stream string, data []byte) {
	bus, err := l.store.GetEventBus(sessionID)
	if err != nil {
		return
	}
	bus.Publish("background_output", map[string]any{
		"pid":    pid,
		"stream": stream,
		"data":   string(data),
	})
}

func (l *Loop) SolidifySession(ctx context.Context, sessionID string) (*skills.SolidifyResult, error) {
	if l.solidifier == nil {
		return nil, fmt.Errorf("solidifier not configured")
	}
	return l.solidifier.SolidifySession(ctx, sessionID)
}
