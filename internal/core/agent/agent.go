package agent

import (
	"context"

	"devo/internal/config"
	"devo/internal/core/agentloop"
	"devo/internal/core/approval"
	"devo/internal/core/compressor"
	"devo/internal/core/memory"
	"devo/internal/core/prompt"
	"devo/internal/core/session"
	"devo/internal/core/skills"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/tools"
)

type Config struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	SystemPrompt string           `json:"system_prompt"`
	ModelID      string           `json:"model_id"`
	Tools        []string         `json:"tools"`
	Skills       []string         `json:"skills"`
	Builtin      bool             `json:"builtin"`
	SubAgentOf   string           `json:"-"`
	LLMClient    llmclient.Client `json:"-"`
}

type Agent struct {
	Config
	loop *agentloop.Loop
}

func New(
	cfg Config,
	store session.SessionStore,
	registry *tools.Registry,
	appCfg *config.Config,
	approvalMgr *approval.Manager,
	memoryMgr *memory.Manager,
	skillsMgr *skills.Manager,
	bgProcManager *tools.BackgroundProcessManager,
	mcpMgr *mcp.Manager,
	solidifier *skills.Solidifier,
) *Agent {
	llmClient := cfg.LLMClient
	if llmClient == nil {
		llmClient = providers.NewClient(appCfg, registry)
		if cfg.ModelID != "" {
			llmClient = providers.NewClientForModel(appCfg, cfg.ModelID, registry)
		}
	}

	var toolExecutor agentloop.ToolExecutor = newDynamicToolExecutor(registry, cfg, appCfg)

	var skillsProvider prompt.SkillsProvider
	if skillsMgr != nil {
		if cfg.Skills == nil {
			skillsProvider = skillsMgr
		} else {
			skillsProvider = skillsMgr.WithFilter(cfg.Skills)
		}
	}

	a := &Agent{Config: cfg}
	a.loop = agentloop.New(store, llmClient, toolExecutor, appCfg,
		approvalMgr, memoryMgr, skillsProvider, bgProcManager, mcpMgr, solidifier,
		cfg.SystemPrompt)
	return a
}

func Default(
	store session.SessionStore, registry *tools.Registry,
	appCfg *config.Config,
	approvalMgr *approval.Manager, memoryMgr *memory.Manager, skillsMgr *skills.Manager,
	bgProcManager *tools.BackgroundProcessManager, mcpMgr *mcp.Manager, solidifier *skills.Solidifier,
) *Agent {
	return New(Config{
		ID:           "devo-default",
		Name:         "Devo",
		Description:  "通用编程助手",
		SystemPrompt: prompt.DefaultSystemPrompt(),
		Tools:        nil,
	}, store, registry, appCfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)
}

func (a *Agent) ProcessMessage(ctx context.Context, sessionID string, msg session.Message) error {
	return a.loop.ProcessMessage(ctx, sessionID, msg)
}

func (a *Agent) ID() string       { return a.Config.ID }
func (a *Agent) Name() string     { return a.Config.Name }
func (a *Agent) IsSubAgent() bool { return a.Config.SubAgentOf != "" }

func (a *Agent) ListTools() []tools.Tool { return a.loop.ListTools() }

func (a *Agent) Pause(sessionID string) error    { return a.loop.Pause(sessionID) }
func (a *Agent) Resume(sessionID string) error   { return a.loop.Resume(sessionID) }
func (a *Agent) Cancel(sessionID string) error   { return a.loop.Cancel(sessionID) }
func (a *Agent) Complete(sessionID string) error { return a.loop.Complete(sessionID) }
func (a *Agent) Archive(sessionID string) error  { return a.loop.Archive(sessionID) }
func (a *Agent) Compact(sessionID string) (*compressor.CompressResult, error) {
	return a.loop.Compact(sessionID)
}

func (a *Agent) Rollback(sessionID string, targetMessageID string) (*agentloop.RollbackResult, error) {
	return a.loop.Rollback(sessionID, targetMessageID)
}

func (a *Agent) UpdateConcurrencyConfig(sessionID string, maxConcurrentToolCalls, maxConcurrentSubprocesses *int) error {
	return a.loop.UpdateConcurrencyConfig(sessionID, maxConcurrentToolCalls, maxConcurrentSubprocesses)
}

func (a *Agent) GetApprovalManager() *approval.Manager { return a.loop.GetApprovalManager() }
func (a *Agent) ResolveApproval(sessionID, approvalID, decision string) error {
	return a.loop.ResolveApproval(sessionID, approvalID, decision)
}
func (a *Agent) EstimateInitialContextTokens(sess *session.Session) int {
	return a.loop.EstimateInitialContextTokens(sess)
}
func (a *Agent) GetArchiveContent(sessionID string) ([]byte, error) {
	return a.loop.GetArchiveContent(sessionID)
}
func (a *Agent) SyncArchive(sessionID string) (string, error) {
	return a.loop.SyncArchive(sessionID)
}

func (a *Agent) RecoverCrashedSessions() error {
	return a.loop.RecoverCrashedSessions()
}

func (a *Agent) SolidifySession(ctx context.Context, sessionID string) (*skills.SolidifyResult, error) {
	return a.loop.SolidifySession(ctx, sessionID)
}

func (a *Agent) UpdateLLMClient(client llmclient.Client) {
	a.loop.UpdateLLMClient(client)
}

func (a *Agent) ForwardBackgroundOutput(sessionID string, pid int, stream string, data []byte) {
	a.loop.ForwardBackgroundOutput(sessionID, pid, stream, data)
}

func (a *Agent) GetSkillsManager() *skills.Manager {
	return a.loop.GetSkillsManager()
}
