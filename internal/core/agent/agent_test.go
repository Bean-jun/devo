package agent

import (
	"testing"

	"devo/internal/config"
	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/core/skills"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/mcp"
	"devo/internal/taskexec/tools"
)

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		ID:           "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent",
		SystemPrompt: "You are a test agent.",
		ModelID:      "gpt-4o",
		Tools:        []string{"read_file", "write_file"},
		Skills:       []string{"react", "python"},
		Builtin:      true,
		SubAgentOf:   "parent-agent",
	}

	if cfg.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got %q", cfg.ID)
	}
	if cfg.Name != "Test Agent" {
		t.Errorf("expected Name 'Test Agent', got %q", cfg.Name)
	}
	if cfg.Description != "A test agent" {
		t.Errorf("expected Description 'A test agent', got %q", cfg.Description)
	}
	if cfg.SystemPrompt != "You are a test agent." {
		t.Errorf("expected SystemPrompt 'You are a test agent.', got %q", cfg.SystemPrompt)
	}
	if cfg.ModelID != "gpt-4o" {
		t.Errorf("expected ModelID 'gpt-4o', got %q", cfg.ModelID)
	}
	if len(cfg.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(cfg.Tools))
	}
	if len(cfg.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(cfg.Skills))
	}
	if cfg.Skills[0] != "react" {
		t.Errorf("expected first skill 'react', got %q", cfg.Skills[0])
	}
	if !cfg.Builtin {
		t.Error("expected Builtin to be true")
	}
	if cfg.SubAgentOf != "parent-agent" {
		t.Errorf("expected SubAgentOf 'parent-agent', got %q", cfg.SubAgentOf)
	}
}

func TestConfig_EmptyTools(t *testing.T) {
	cfg := Config{
		ID:           "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent",
		SystemPrompt: "You are a test agent.",
		Tools:        nil,
	}
	if cfg.Tools != nil {
		t.Errorf("expected nil Tools, got %v", cfg.Tools)
	}
}

func TestNew(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent",
		SystemPrompt: "You are a test agent.",
		Tools:        nil,
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.Config.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got %q", agent.Config.ID)
	}
	if agent.Config.Name != "Test Agent" {
		t.Errorf("expected Name 'Test Agent', got %q", agent.Config.Name)
	}
	if agent.Config.SystemPrompt != "You are a test agent." {
		t.Errorf("expected SystemPrompt 'You are a test agent.', got %q", agent.Config.SystemPrompt)
	}
	if agent.loop == nil {
		t.Error("expected non-nil loop")
	}
}

func TestNew_WithTools(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "tool-agent",
		Name:         "Tool Agent",
		Description:  "Agent with specific tools",
		SystemPrompt: "You are a tool agent.",
		Tools:        []string{"read_file", "glob"},
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if len(agent.Config.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(agent.Config.Tools))
	}
	if agent.Config.Tools[0] != "read_file" {
		t.Errorf("expected first tool 'read_file', got %q", agent.Config.Tools[0])
	}
	if agent.Config.Tools[1] != "glob" {
		t.Errorf("expected second tool 'glob', got %q", agent.Config.Tools[1])
	}
}

func TestDefault(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := Default(store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.Config.ID != "devo-default" {
		t.Errorf("expected ID 'devo-default', got %q", agent.Config.ID)
	}
	if agent.Config.Name != "Devo" {
		t.Errorf("expected Name 'Devo', got %q", agent.Config.Name)
	}
	if agent.Config.Description != "通用编程助手" {
		t.Errorf("expected Description '通用编程助手', got %q", agent.Config.Description)
	}
	if agent.Config.SystemPrompt == "" {
		t.Error("expected non-empty SystemPrompt")
	}
	if agent.Config.Tools != nil {
		t.Errorf("expected Tools nil, got %v", agent.Config.Tools)
	}
}

func TestAgent_ForwardingMethods(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := Default(store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	t.Run("GetApprovalManager", func(t *testing.T) {
		mgr := agent.GetApprovalManager()
		if mgr == nil {
			t.Error("expected non-nil approval manager")
		}
	})

	t.Run("GetSkillsManager", func(t *testing.T) {
		mgr := agent.GetSkillsManager()
		if mgr == nil {
			t.Error("expected non-nil skills manager")
		}
	})

	t.Run("Pause_nonexistent", func(t *testing.T) {
		err := agent.Pause("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("Resume_nonexistent", func(t *testing.T) {
		err := agent.Resume("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("Cancel_nonexistent", func(t *testing.T) {
		err := agent.Cancel("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("Complete_nonexistent", func(t *testing.T) {
		err := agent.Complete("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("Archive_nonexistent", func(t *testing.T) {
		err := agent.Archive("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("Compact_nonexistent", func(t *testing.T) {
		_, err := agent.Compact("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("Rollback_nonexistent", func(t *testing.T) {
		_, err := agent.Rollback("nonexistent", "msg-1")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("ResolveApproval_nonexistent", func(t *testing.T) {
		err := agent.ResolveApproval("nonexistent", "approval-1", "approve")
		if err == nil {
			t.Error("expected error for nonexistent approval")
		}
	})

	t.Run("UpdateConcurrencyConfig_nonexistent", func(t *testing.T) {
		maxCalls := 3
		err := agent.UpdateConcurrencyConfig("nonexistent", &maxCalls, nil)
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("EstimateInitialContextTokens", func(t *testing.T) {
		sess := &session.Session{
			ID:               "test-sess",
			WorkingDirectory: t.TempDir(),
		}
		tokens := agent.EstimateInitialContextTokens(sess)
		if tokens <= 0 {
			t.Errorf("expected positive tokens, got %d", tokens)
		}
	})

	t.Run("GetArchiveContent_nonexistent", func(t *testing.T) {
		_, err := agent.GetArchiveContent("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("SyncArchive_nonexistent", func(t *testing.T) {
		_, err := agent.SyncArchive("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("UpdateLLMClient", func(t *testing.T) {
		newLLM := llmclient.NewMockClient()
		agent.UpdateLLMClient(newLLM)
	})

	t.Run("ForwardBackgroundOutput", func(t *testing.T) {
		agent.ForwardBackgroundOutput("sess-1", 123, "stdout", []byte("test"))
	})

	t.Run("ProcessMessage", func(t *testing.T) {
		ctx := t.Context()
		err := agent.ProcessMessage(ctx, "nonexistent", session.Message{
			Role:    session.RoleUser,
			Content: "hello",
		})
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})

	t.Run("RecoverCrashedSessions", func(t *testing.T) {
		err := agent.RecoverCrashedSessions()
		if err != nil {
			t.Errorf("expected no error for empty store, got %v", err)
		}
	})

	t.Run("SolidifySession", func(t *testing.T) {
		ctx := t.Context()
		_, err := agent.SolidifySession(ctx, "nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})
}

func TestNew_WithModelID(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "model-agent",
		Name:         "Model Agent",
		Description:  "Agent with model binding",
		SystemPrompt: "You are a model agent.",
		ModelID:      "gpt-4o",
		Tools:        nil,
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.Config.ModelID != "gpt-4o" {
		t.Errorf("expected ModelID 'gpt-4o', got %q", agent.Config.ModelID)
	}
	if agent.loop == nil {
		t.Error("expected non-nil loop")
	}
}

func TestNew_EmptyModelID(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	registry := tools.NewRegistry()
	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "default-model-agent",
		Name:         "Default Model Agent",
		Description:  "Agent without model binding",
		SystemPrompt: "You are a default model agent.",
		ModelID:      "",
		Tools:        nil,
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.Config.ModelID != "" {
		t.Errorf("expected empty ModelID, got %q", agent.Config.ModelID)
	}
}

func TestNew_TeamModeOn_DelegateToIncluded(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	cfg.TeamMode = true
	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	delegateTool := tools.NewDelegateToTool(nil, store, cfg)
	registry.Register(delegateTool)

	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "devo-default",
		Name:         "Devo",
		Description:  "Default agent",
		SystemPrompt: "You are a helpful assistant.",
		Tools:        nil,
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}

	hasDelegateTo := false
	for _, tool := range agent.ListTools() {
		if tool.Name() == "delegate_to" {
			hasDelegateTo = true
			break
		}
	}
	if !hasDelegateTo {
		t.Error("expected delegate_to to be included when TeamMode is ON")
	}
}

func TestNew_TeamModeOff_DelegateToExcluded(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	cfg.TeamMode = false
	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	delegateTool := tools.NewDelegateToTool(nil, store, cfg)
	registry.Register(delegateTool)

	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "devo-default",
		Name:         "Devo",
		Description:  "Default agent",
		SystemPrompt: "You are a helpful assistant.",
		Tools:        nil,
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}

	for _, tool := range agent.ListTools() {
		if tool.Name() == "delegate_to" {
			t.Error("expected delegate_to to be excluded when TeamMode is OFF")
		}
	}
}

func TestNew_SubAgent_DelegateToExcluded(t *testing.T) {
	store := session.NewInMemoryStore()
	cfg := config.DefaultConfig()
	cfg.TeamMode = true
	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	delegateTool := tools.NewDelegateToTool(nil, store, cfg)
	registry.Register(delegateTool)

	approvalMgr := approval.NewManager()
	memoryFileStore, _ := memory.DefaultFileStore()
	memoryMgr := memory.NewManager(memoryFileStore, concurrency.NewPathLockManager(), approvalMgr)
	skillsMgr := skills.NewManager(t.TempDir())
	bgProcManager := tools.NewBackgroundProcessManager()
	mcpMgr := mcp.NewManager(t.TempDir())
	solidifier := skills.NewSolidifier(nil, skillsMgr, store)

	agent := New(Config{
		ID:           "code-reviewer",
		Name:         "Code Reviewer",
		Description:  "Sub agent",
		SystemPrompt: "You are a code reviewer.",
		Tools:        nil,
		SubAgentOf:   "devo-default",
	}, store, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}

	for _, tool := range agent.ListTools() {
		if tool.Name() == "delegate_to" {
			t.Error("expected delegate_to to be excluded for sub-agents even when TeamMode is ON")
		}
	}
}
