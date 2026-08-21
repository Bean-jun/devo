# Devo 多 Agent 架构设计文档

**版本**：2.0.0

**状态**：设计阶段

**关联文档**：[2-architecture.md](./2-architecture.md)、[14-agent-abstraction-design.md](./14-agent-abstraction-design.md)

---

## 目录

1. [背景与动机](#1-背景与动机)
2. [核心设计决策](#2-核心设计决策)
3. [Agent 模型设计](#3-agent-模型设计)
4. [LLM 模型绑定](#4-llm-模型绑定)
5. [工具隔离](#5-工具隔离)
6. [Skills 隔离](#6-skills-隔离)
7. [Session 与 Agent 的关系](#7-session-与-agent-的关系)
8. [Agent 注册与路由](#8-agent-注册与路由)
9. [Agent 生命周期](#9-agent-生命周期)
10. [配置体系](#10-配置体系)
11. [REST API 设计](#11-rest-api-设计)
12. [Team Mode：子 Agent 委托](#12-team-mode子-agent-委托)
13. [数据结构变更](#13-数据结构变更)
14. [实施路径](#14-实施路径)
15. [不做的事情](#15-不做的事情)
16. [风险与应对](#16-风险与应对)

---

## 1. 背景与动机

### 1.1 行业背景：多 Agent 之争

2025 年，AI Coding 领域围绕多 Agent 爆发了一场公开争论：

| 阵营 | 代表 | 核心主张 |
|------|------|---------|
| **多 Agent 派** | Anthropic（Claude Code） | "猛虎难敌群狼"——复杂任务需要专业化分工，主 Agent 调度子 Agent 各司其职 |
| **单 Agent 派** | Cognition（Devin） | "Don't Build Multi-Agents"——子 Agent 不共享完整上下文会做出互相冲突的隐式决策 |

两派的**共识**是：上下文要么完整共享，要么彻底隔离，最忌讳部分共享。

Claude Code 的实践表明，**内置的 `delegate_to` 模式**（主 Agent 将子任务委托给子 Agent，子 Agent 拥有独立上下文，只看任务描述不看完整对话历史）是有效的。而让多个 Agent 自主协商、自动决策的模式则容易失控——"烧了 1000 美金才知道的血泪经验"。

**Devo 的立场**：采用 `delegate_to` 模式，子 Agent 上下文隔离，用户通过对话自然触发，AI 不自动决策。

### 1.2 当前状态

项目已通过 [14-agent-abstraction-design.md](./14-agent-abstraction-design.md) 完成了 Agent 抽象层的基础建设：

- `agent.Config` 定义了 Agent 的身份（ID、Name、Description、SystemPrompt、Tools）
- `agent.Registry` 提供了 Agent 注册和路由能力
- `Session.AgentID` 建立了 Session 到 Agent 的关联
- `Handler.getAgent()` 实现了按 Session 路由到对应 Agent

目前只有一个 `devo-default` Agent，所有对话都在同一个 Agent 下进行。

**当前实现的关键缺陷**（本次设计必须修复）：

| 缺陷 | 现状 | 影响 |
|------|------|------|
| **SystemPrompt 未传递** | `agent.Config.SystemPrompt` 已定义，但 `agent.New` 创建 Loop 时从未传入；Loop 内部硬编码了 `prompt.DefaultSystemPrompt()` | 不同 Agent 的"人格"完全无法生效，多 Agent 的核心价值丧失 |
| **LLM Client 全局唯一** | `app.go` 只有一个 `a.llm`，所有 Agent 共用；`providers.NewClient` 只创建激活模型的客户端 | Agent 无法绑定独立模型 |
| **Session 列表无 agent_id** | `listSessionsItem` 不包含 `agent_id` 字段，`SessionStore.ListSessions` 不支持按 Agent 过滤 | 前端无法展示 Session 使用的 Agent |
| **use_skill 无 Agent 感知** | `UseSkillTool` 直接调用 `skills.Manager`，不感知 Agent 的 Skills 白名单 | Agent 的 Skills 限制可被绕过 |
| **默认 Agent ID 硬编码** | `CreateSession` 中 `agentID = "devo-default"` 是魔术字符串 | 与 Registry 脱耦，重构风险 |

### 1.3 为什么需要多 Agent

不同的任务需要不同的"人格"和能力：

| 场景 | 当前体验 | 多 Agent 体验 |
|------|---------|--------------|
| 代码审查 | 同一个 Agent 做所有事，审查风格和写代码风格混在一起 | 创建 Session 时选 `code-reviewer`，专注审查，只读工具，更安全 |
| 写测试 | 通用 Agent 写测试，可能过度设计 | 创建 Session 时选 `test-writer`，专注测试，风格一致 |
| 项目分析 | 通用 Agent 分析代码，可能忍不住动手改 | 创建 Session 时选 `architect`，只分析不修改 |
| 日常开发 + 自动审查 | 写完代码自己检查，容易遗漏 | 开启 Team Mode，主 Agent 写完代码自动委托 `code-reviewer` 审查，发现并修复问题 |

### 1.4 目标

构建一个**对话驱动的多 Agent 系统**，分两层：

**基础层——多 Persona**：用户可以在创建 Session 时选择 Agent，每个 Agent 有独立的 SystemPrompt、工具集、Skills 和 LLM 模型。Session 创建后 Agent 不可切换。通过 API 查询可用的 Agent 列表。

**协作层——Team Mode**：用户可开启 Team Mode（类似 YOLO 模式），开启后主 Agent 获得 `delegate_to` 工具，可以将子任务委托给内置子 Agent。默认关闭，用户主动开启。

---

## 2. 核心设计决策

### 2.1 设计哲学

```
默认：单 Agent，用户选择角色
开关：Team Mode，主 Agent 可委托子 Agent
未来：用户自定义工作流（不在本期范围）
```

**核心原则**：

1. **用户主导，AI 辅助**：Agent 的定义由用户创建，AI 可以在对话中建议但必须用户确认。AI 不自动创建 Agent。
2. **对话驱动，按需委托**：子 Agent 的调用由主 Agent 在对话中通过 `delegate_to` 工具触发，用户通过自然语言间接控制。不存在"AI 自主调度"。
3. **上下文隔离**：子 Agent 拥有独立上下文，只接收委托任务描述，不共享主 Agent 的完整对话历史。这是 Anthropic 和 Cognition 两派共识的实践。
4. **Team Mode 是开关，不是默认**：默认行为不变（单 Agent），用户需要显式开启才能使用多 Agent 协作。

### 2.2 决策表

| 决策 | 结论 | 理由 |
|------|------|------|
| **Agent 的 LLM 模型** | Agent 可绑定自己的模型，未绑定时降级到系统激活模型 | 灵活性 + 零配置可用 |
| **工具集** | 每个 Agent 拥有独立的工具集，不共享 | 安全和职责隔离，code-reviewer 不该有 write_file |
| **Session 切换 Agent** | **不允许** | Session 的消息历史与 Agent 的 SystemPrompt 强绑定，切换会导致上下文断裂 |
| **Skills 隔离** | 每个 Agent 激活的 Skills 独立 | 不同 Agent 需要不同的技能组合 |
| **子 Agent 委托** | 主 Agent 通过 `delegate_to` 工具调用子 Agent；子 Agent 上下文隔离 | 取 Claude Code 的委托模式，避 Cognition 警告的上下文共享陷阱 |
| **Team Mode 开关** | 默认关闭，用户显式开启；开启后主 Agent 获得 `delegate_to` 工具 | 不让 AI 自动决策，用户控制协作范围 |
| **Agent 间自主协商** | **不做** | 容易失控，Token 消耗不可控，用户反馈"烧钱" |
| **用户自定义工作流** | **不在本期范围**，后续单独设计 | 当前聚焦委托模式，工作流编排是更高阶需求 |

---

## 3. Agent 模型设计

### 3.1 Agent Config（扩展后）

```go
// internal/core/agent/agent.go

type Config struct {
    ID           string   `json:"id"`            // 唯一标识，如 "devo-default"、"code-reviewer"
    Name         string   `json:"name"`          // 显示名称，如 "Devo"、"Code Reviewer"
    Description  string   `json:"description"`   // 描述，展示给用户
    SystemPrompt string   `json:"system_prompt"` // 系统提示词，定义 Agent 的行为风格
    ModelID      string   `json:"model_id"`      // 绑定的 LLM 模型 ID，为空则使用系统激活模型
    Tools        []string `json:"tools"`          // 可用工具列表，nil = 全部工具，[] = 无工具
    Skills       []string `json:"skills"`         // 激活的 Skills 列表，nil = 使用全局 Skills，[] = 无 Skills
    Builtin      bool     `json:"builtin"`        // 是否为内置 Agent（内置不可删除）
    SubAgentOf   string   `json:"-"`              // 非持久化：标记此 Agent 是哪个主 Agent 的子 Agent（为空 = 顶级 Agent）
}
```

**字段语义**：

| 字段 | nil | 空切片 `[]` | 有值 |
|------|-----|------------|------|
| `Tools` | 全部工具可用 | 无工具可用 | 仅指定工具可用 |
| `Skills` | 使用全局 Skills 配置 | 无 Skills | 仅指定 Skills 激活 |
| `ModelID` | 降级到系统激活模型 | — | 使用指定模型 |
| `SubAgentOf` | 顶级 Agent | — | 标记为某 Agent 的子 Agent |

### 3.2 Agent 与基础设施的关系

```
                    ┌──────────────────────────────┐
                    │         Agent                 │
                    │                              │
                    │  Config:                     │
                    │    ID, Name, Description      │
                    │    SystemPrompt (身份)  ──────┼──────────┐
                    │    ModelID    (大脑选择) ─────┼──────┐   │
                    │    Tools      (手脚选择) ─────┼───┐  │   │
                    │    Skills     (技能选择) ─────┼─┐ │  │   │
                    │                              │ │ │  │   │
                    │  ┌────────────────────────┐  │ │ │  │   │
                    │  │   agentloop.Loop       │  │ │ │  │   │
                    │  │   (执行引擎)            │  │ │ │  │   │
                    │  │                        │  │ │ │  │   │
                    │  │  - promptAssembler ◄───┼──┼─┼──┼───┘
                    │  │  - toolExecutor    ◄───┼──┼─┼──┘
                    │  │  - llmClient       ◄───┼──┼─┘
                    │  │  - skillsProvider  ◄───┼──┘
                    │  │  - stateMachine        │  │
                    │  │  - approvalManager     │  │
                    │  │  - ...                 │  │
                    │  └────────────────────────┘  │
                    └──────────────────────────────┘
```

Agent 是顶层实体，Loop 是内部实现细节。**关键数据流**：

| 数据 | 流向 | 说明 |
|------|------|------|
| `SystemPrompt` | `Config` → `promptAssembler.SetSystemPrompt()` | 定义 Agent 的"人格"，注入到每次 LLM 调用的上下文中 |
| `ModelID` | `Config` → `providers.NewClientForModel()` → `llmClient` | 决定 Agent 使用哪个 LLM 模型 |
| `Tools` | `Config` → `registry.Filter()` → `toolExecutor` | 决定 Agent 可以调用哪些工具 |
| `Skills` | `Config` → `skillsMgr.WithFilter()` → `skillsProvider` | 决定 Agent 可以访问哪些 Skills |

**当前代码的断裂点**：`agent.New` 接收了 `Config.SystemPrompt` 但完全不传给 Loop。`Loop.New` 内部硬编码 `prompt.DefaultSystemPrompt()`。本次重构必须打通这条链路。

### 3.3 预定义的内置 Agent

系统启动时注册以下内置 Agent：

```go
var BuiltinAgents = []Config{
    {
        ID:           "devo-default",
        Name:         "Devo",
        Description:  "通用编程助手，具备完整的代码读写和执行能力",
        SystemPrompt: prompt.DefaultSystemPrompt(),
        ModelID:      "",    // 使用系统激活模型
        Tools:        nil,   // 全部工具
        Skills:       nil,   // 使用全局 Skills
        Builtin:      true,
    },
    {
        ID:           "code-reviewer",
        Name:         "Code Reviewer",
        Description:  "代码审查专家，只读分析，不修改代码",
        SystemPrompt: prompt.CodeReviewerPrompt(),
        ModelID:      "",    // 使用系统激活模型
        Tools:        []string{"read_file", "glob", "list_files", "search_codebase"},
        Skills:       nil,
        Builtin:      true,
    },
    {
        ID:           "architect",
        Name:         "Architect",
        Description:  "架构分析与技术方案设计，不执行代码操作",
        SystemPrompt: prompt.ArchitectPrompt(),
        ModelID:      "",    // 使用系统激活模型，建议配合 reasoning 模型
        Tools:        []string{"read_file", "glob", "list_files", "search_codebase"},
        Skills:       nil,
        Builtin:      true,
    },
    {
        ID:           "test-writer",
        Name:         "Test Writer",
        Description:  "测试用例编写专家，专注代码质量和边界条件",
        SystemPrompt: prompt.TestWriterPrompt(),
        ModelID:      "",
        Tools:        []string{"read_file", "write_file", "edit_file", "glob", "list_files", "search_codebase", "exec_python"},
        Skills:       nil,
        Builtin:      true,
    },
}
```

---

## 4. LLM 模型绑定

### 4.1 模型选择优先级

```
Agent.Config.ModelID（Agent 指定）
    ↓ 为空
系统激活模型（config.LLM.ActiveModelID）
    ↓ 未设置
config.LLM.Models 的第一个模型
    ↓ 都没有
config.LLM.Model（旧版单模型字段）
```

### 4.2 实现方式

**核心变更**：`agent.New` 不再接收 `llmclient.Client` 参数，改为接收 `*config.Config`，由 Agent 内部根据 `ModelID` 自行创建 LLM Client。

**第一步：在 `providers` 包新增按 ModelID 创建 Client 的函数**：

```go
// internal/taskexec/llmclient/providers/factory.go

// NewClientForModel 根据 ModelID 从配置中查找模型并创建专属 LLM Client
// 若 modelID 为空或找不到，返回 nil
func NewClientForModel(cfg *config.Config, modelID string, registry *tools.Registry) llmclient.Client {
    if modelID == "" {
        return nil
    }
    for i := range cfg.LLM.Models {
        if cfg.LLM.Models[i].ID == modelID {
            target := &cfg.LLM.Models[i]
            return openai.New(openai.Config{
                LLMConfig: &config.LLMConfig{
                    APIKey:          target.APIKey,
                    BaseURL:         target.BaseURL,
                    Model:           target.Model,
                    ExtraHeaders:    target.ExtraHeaders,
                    EnableReasoning: target.EnableReasoning,
                    ReasoningEffort: target.ReasoningEffort,
                    MaxTokens:       target.MaxTokens,
                },
            }, registry)
        }
    }
    return nil
}
```

**第二步：重构 `agent.New` 签名**：

```go
// internal/core/agent/agent.go

// 旧签名（废弃）
// func New(cfg Config, store session.SessionStore, llm llmclient.Client, ...) *Agent

// 新签名
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
    a := &Agent{Config: cfg}

    // 1. 决定 LLM Client
    var llmForAgent llmclient.Client
    if cfg.ModelID != "" {
        llmForAgent = providers.NewClientForModel(appCfg, cfg.ModelID, registry)
    }
    if llmForAgent == nil {
        llmForAgent = providers.NewClient(appCfg, registry)
    }

    // 2. 决定工具集
    var filteredExecutor agentloop.ToolExecutor
    if cfg.Tools == nil {
        filteredExecutor = registry
    } else {
        filteredExecutor = registry.Filter(cfg.Tools)
    }

    // 3. 决定 Skills 视图
    var skillsForAgent agentloop.SkillsProvider
    if cfg.Skills == nil {
        skillsForAgent = skillsMgr
    } else {
        skillsForAgent = skillsMgr.WithFilter(cfg.Skills)
    }

    // 4. 创建 Loop，将 SystemPrompt 注入到 promptAssembler
    a.loop = agentloop.New(store, llmForAgent, filteredExecutor, appCfg,
        approvalMgr, memoryMgr, skillsForAgent, bgProcManager, mcpMgr, solidifier,
        cfg.SystemPrompt,  // ← 关键：Agent 的 SystemPrompt 传入 Loop
    )

    return a
}
```

**关键点**：
- `agent.New` **不再接收** `llmclient.Client`，改为内部根据 `ModelID` 和 `appCfg` 决定
- 如果 Agent 指定了 `ModelID`，调用 `providers.NewClientForModel` 创建专属 Client
- 如果 `ModelID` 为空或找不到，降级到 `providers.NewClient`（系统激活模型）
- **`cfg.SystemPrompt` 必须传入 `agentloop.New`，由 Loop 注入到 `promptAssembler`**

### 4.3 对 Loop.New 的修改

`Loop.New` 增加 `systemPrompt` 参数，不再硬编码：

```go
// internal/core/agentloop/loop.go

func New(
    store session.SessionStore,
    llmClient llmclient.Client,
    toolExecutor ToolExecutor,
    cfg *config.Config,
    approvalMgr *approval.Manager,
    memoryMgr *memory.Manager,
    skillsMgr *skills.Manager,
    bgProcManager *tools.BackgroundProcessManager,
    mcpMgr *mcp.Manager,
    solidifier *skills.Solidifier,
    systemPrompt string,  // ← 新增参数
) *Loop {
    pathLockManager := concurrency.NewPathLockManager()
    if systemPrompt == "" {
        systemPrompt = prompt.DefaultSystemPrompt()
    }
    assembler := prompt.NewAssembler(systemPrompt)
    // ... 其余不变
}
```

### 4.4 对 app.go 的影响

```go
// 旧代码（删除）
func (a *App) initLLM() {
    a.llm = providers.NewClient(a.cfg, a.toolRegistry)
}

func (a *App) initAgent() {
    // ...
    defaultAgent := agent.New(cfg, a.store, a.llm, a.toolRegistry, ...)
    a.agentRegistry = agent.NewRegistry(defaultAgent)
}

// 新代码
func (a *App) initAgent() {
    approvalMgr := approval.NewManager()
    solidifier := skills.NewSolidifier(a.llm, a.skillsMgr, a.store)

    // 注册所有内置 Agent，每个独立创建 LLM Client
    for _, cfg := range agent.BuiltinAgents {
        ag := agent.New(cfg, a.store, a.toolRegistry, a.cfg,
            approvalMgr, a.memoryMgr, a.skillsMgr, a.bgProcManager, a.mcpMgr, solidifier)
        a.agentRegistry.Register(ag)
    }

    // 若要做 background process 输出转发，只对 default agent 做
    a.bgProcManager.SetOutputForwarder(a.agentRegistry.DefaultAgent())

    // 删除 a.llm 字段，不再需要全局 LLM Client
}
```

**注意**：`a.llm` 字段从 `App` 结构体中删除。`a.solidifier` 之前依赖 `a.llm`，需要改为使用 `a.agentRegistry.DefaultAgent()` 内部的 LLM Client，或直接传入 `providers.NewClient(a.cfg, a.toolRegistry)`。

---

## 5. 工具隔离

### 5.1 核心原则

**每个 Agent 拥有独立的工具集。** 工具隔离在 Agent 级别，不在 Loop 级别。

### 5.2 实现方式

工具过滤已在第 4.2 节 `agent.New` 中实现，核心逻辑：

```go
// 在 agent.New 内部
var filteredExecutor agentloop.ToolExecutor
if cfg.Tools == nil {
    filteredExecutor = registry       // nil = 全部工具
} else {
    filteredExecutor = registry.Filter(cfg.Tools)  // 指定工具列表
}
```

### 5.3 tools.Registry 增加 Filter 方法

```go
// internal/taskexec/tools/registry.go

func (r *Registry) Filter(toolNames []string) *Registry {
    filtered := NewRegistry()
    for _, name := range toolNames {
        if tool, ok := r.GetTool(name); ok {
            filtered.Register(tool)
        }
    }
    return filtered
}
```

`Filter` 返回一个新的 `Registry`，只包含指定的工具。新 Registry 实现 `agentloop.ToolExecutor` 接口，可以直接传给 Loop。

### 5.4 工具集设计原则

| Agent 类型 | 工具策略 | 示例工具 |
|-----------|---------|---------|
| 通用 Agent | 全部工具 | read_file, write_file, edit_file, exec_python, glob, search_codebase, list_files, use_skill |
| 只读 Agent | 仅读取类 | read_file, glob, list_files, search_codebase |
| 执行 Agent | 读取 + 执行 | read_file, write_file, edit_file, exec_python, glob |

**安全约束**：
- `write_file` 和 `edit_file` 永远不给只读 Agent
- `exec_python` 永远不给只读 Agent
- `use_skill` 由 Skills 隔离进一步控制（见第 6 节）

---

## 6. Skills 隔离

### 6.1 核心原则

**每个 Agent 激活的 Skills 独立。** 不同 Agent 看到的 Skills 列表不同。

### 6.2 Agent 与 Session 的 Skills 优先级

`Session` 有 `ActiveSkills []string` 字段，Agent 也有 `Skills []string` 配置。两者共同决定最终可用的 Skills：

```
Agent.Skills（硬限制，白名单上限）
    ↓ 交集
Session.ActiveSkills（用户选择，在 Agent 白名单内二次筛选）
    ↓
最终可用 Skills
```

**规则**：
- `Agent.Skills = nil`：无 Agent 限制，使用 Session.ActiveSkills + 全局配置
- `Agent.Skills = []`：Agent 禁用所有 Skills，无论 Session 配置如何
- `Agent.Skills = ["react", "python"]`：Agent 白名单，Session 只能在此范围内选择
- 最终生效的 Skills = `Agent.Skills ∩ Session.ActiveSkills`（nil 视为全集）

### 6.3 实现方式

Skills 过滤已在第 4.2 节 `agent.New` 中实现，核心逻辑：

```go
// 在 agent.New 内部
var skillsForAgent agentloop.SkillsProvider
if cfg.Skills == nil {
    skillsForAgent = skillsMgr     // nil = 使用全局 Skills
} else {
    skillsForAgent = skillsMgr.WithFilter(cfg.Skills)  // 指定 Skills 白名单
}
```

### 6.4 SkillsManager 增加 Filter 支持

```go
// internal/core/skills/manager.go

func (m *Manager) WithFilter(skillNames []string) *FilteredSkillsView {
    return &FilteredSkillsView{
        manager:      m,
        allowedNames: skillNames,
    }
}

type FilteredSkillsView struct {
    manager      *Manager
    allowedNames []string
}

func (v *FilteredSkillsView) GetActiveSkillsPrompt() string {
    allSkills := v.manager.GetAllSkills()
    var parts []string
    for _, skill := range allSkills {
        if !skill.Enabled {
            continue
        }
        if !v.isAllowed(skill.Name) {
            continue
        }
        desc := skill.Description
        if desc == "" {
            desc = "No description"
        }
        parts = append(parts, fmt.Sprintf("- **%s**: %s", skill.Name, desc))
    }
    if len(parts) == 0 {
        return ""
    }
    return "## Available Skills\nYou have access to the following skills. Use the `use_skill` tool to load full instructions for any skill.\n\n" +
        strings.Join(parts, "\n") + "\n"
}

func (v *FilteredSkillsView) isAllowed(name string) bool {
    if v.allowedNames == nil {
        return true
    }
    for _, allowed := range v.allowedNames {
        if strings.EqualFold(allowed, name) {
            return true
        }
    }
    return false
}
```

### 6.5 use_skill 工具增加 Agent Skills 白名单检查

**当前问题**：`UseSkillTool` 直接调用 `skills.Manager.GetSkill()`，不感知 Agent 的 Skills 限制。Agent 即使配置了 `Skills: ["react"]`，用户仍可通过 `use_skill` 调用 `python` skill。

**修改方案**：`UseSkillTool` 不再直接持有 `*skills.Manager`，改为持有 `SkillsProvider` 接口（扩展后的）：

```go
// internal/core/prompt/types.go —— 扩展 SkillsProvider 接口
type SkillsProvider interface {
    GetActiveSkillsPrompt() string
    IsSkillAllowed(name string) bool   // 新增：检查 skill 是否在白名单内
    GetSkill(name string) (*skills.Skill, error)  // 新增：获取单个 skill
}

// internal/taskexec/tools/use_skill.go —— 修改
type UseSkillTool struct {
    skillsProvider SkillsProvider  // 改为接口，不再直接持有 *skills.Manager
    loaded         map[string]bool
}

func (t *UseSkillTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
    skillName, ok := params["skill_name"].(string)
    if !ok || skillName == "" {
        return fmt.Errorf("missing required parameter: skill_name")
    }

    // 检查 Agent Skills 白名单
    if !t.skillsProvider.IsSkillAllowed(skillName) {
        return fmt.Errorf("skill '%s' is not available for this agent", skillName)
    }

    // ... 其余逻辑不变
}
```

### 6.6 Skills 隔离影响的范围

| 组件 | 影响 |
|------|------|
| `PromptAssembler` | 只注入 Agent 允许的 Skills prompt |
| `use_skill` 工具 | 执行时检查 Agent 的 Skills 白名单，不允许的 skill 拒绝加载 |
| `Solidifier` | 固化时只考虑 Agent 允许的 Skills |
| TUI/前端 Skills 面板 | 展示当前 Agent 可用的 Skills |

---

## 7. Session 与 Agent 的关系

### 7.1 核心约束

**Session 创建时绑定 Agent，之后不可切换。**

```
创建 Session:
  POST /api/v1/sessions
  {
    "working_directory": "/path/to/project",
    "agent_id": "code-reviewer"    // 可选，默认使用 Registry 的 DefaultAgent 的 ID
  }

创建后:
  Session.AgentID = "code-reviewer"  // 不可变
```

### 7.2 为什么不允许切换

1. **SystemPrompt 绑定**：Session 的消息历史中，assistant 的回复风格与 Agent 的 SystemPrompt 强相关。切换 Agent 意味着切换 SystemPrompt，历史消息的上下文连贯性会被破坏。

2. **工具集不一致**：Agent A 用了 `write_file`，Agent B 没有这个工具。切换后，LLM 看到历史中有 `write_file` 的调用记录，但当前工具列表中没有它，会产生混淆。

3. **Skills 不一致**：Agent A 激活了 `react` skill，Agent B 没有。切换后 Skill 相关的上下文会断裂。

4. **语义清晰**：一个 Session = 一个对话 = 一个 Agent 身份。用户如果需要不同 Agent，应该创建新 Session。

### 7.3 Session 的 AgentID 字段

```go
type Session struct {
    AgentID string `json:"agent_id"` // 创建时设置，不可变
    // ... 其余字段
}
```

- 创建 Session 时：`AgentID` 来自请求参数，若为空则使用 `agentRegistry.DefaultAgent().Config.ID`
- 查询 Session 时：返回 `AgentID`，前端可展示当前使用的 Agent
- Session 列表：返回 `AgentID`，可按 `AgentID` 过滤
- `PUT /api/v1/sessions/{id}` 重命名接口：不接受 `agent_id` 字段，传入则忽略（不报错）

**创建 Session 时的默认值逻辑**：

```go
// 旧代码（硬编码，删除）
// agentID := req.AgentID
// if agentID == "" {
//     agentID = "devo-default"
// }

// 新代码
agentID := req.AgentID
if agentID == "" {
    agentID = h.agentRegistry.DefaultAgent().Config.ID
}
```

同时需要验证 `agentID` 是否存在于 Registry 中，若不存在则返回 400 错误：

```go
if agentID != "" {
    if ag := h.agentRegistry.Get(agentID); ag == nil {
        writeError(w, http.StatusBadRequest, "unknown agent_id: "+agentID)
        return
    }
}
```

---

## 8. Agent 注册与路由

### 8.1 Agent 来源

```
内置 Agent（代码硬编码）
  ├── devo-default
  ├── code-reviewer
  ├── architect
  └── test-writer
```

当前只有内置 Agent，用户不能创建自定义 Agent。未来如需支持自定义 Agent，可通过 SDK 外部实现，不在 Devo 内核范围内。

### 8.2 Registry 设计

```go
// internal/core/agent/registry.go

type Registry struct {
    agents       map[string]*Agent
    defaultAgent *Agent
    builtinIDs   map[string]bool  // 标记内置 Agent，不可删除
}

func NewRegistry(defaultAgent *Agent) *Registry {
    r := &Registry{
        agents:       make(map[string]*Agent),
        defaultAgent: defaultAgent,
        builtinIDs:   make(map[string]bool),
    }
    r.Register(defaultAgent)
    return r
}

func (r *Registry) Register(agent *Agent) {
    if agent.Config.Builtin {
        r.builtinIDs[agent.Config.ID] = true
    }
    r.agents[agent.Config.ID] = agent
}

func (r *Registry) Unregister(agentID string) error {
    if r.builtinIDs[agentID] {
        return fmt.Errorf("cannot delete builtin agent: %s", agentID)
    }
    delete(r.agents, agentID)
    return nil
}

func (r *Registry) Get(agentID string) *Agent {
    if agentID == "" {
        return r.defaultAgent
    }
    if agent, ok := r.agents[agentID]; ok {
        return agent
    }
    return r.defaultAgent
}

func (r *Registry) List() []*Agent {
    result := make([]*Agent, 0, len(r.agents))
    for _, ag := range r.agents {
        result = append(result, ag)
    }
    return result
}

func (r *Registry) DefaultAgent() *Agent {
    return r.defaultAgent
}
```

### 8.3 路由流程

```
HTTP Request → Handler
  → sessionID 从 URL 提取
  → sess, _ := store.Get(sessionID)
  → agent := h.agentRegistry.Get(sess.AgentID)
      → 若 sess.AgentID 对应的 Agent 存在，返回该 Agent
      → 若 sess.AgentID 对应的 Agent 已被删除，回退到 defaultAgent
  → agent.ProcessMessage(ctx, sessionID, msg)
```

回退到默认 Agent 时，前端应收到明确的事件通知，提示 Agent 已不可用。

---

## 9. Agent 生命周期

### 9.1 状态机

```
                    ┌──────────┐
                    │  Created  │  (编译时定义)
                    └────┬─────┘
                         │
                         ▼
                    ┌──────────┐
                    │  Active   │  (已注册到 Registry，可被 Session 使用)
                    └──────────┘
```

- **Created**：Agent 定义产生（内置 Agent 编译时定义）
- **Active**：已注册到 Registry，Session 可以使用

---

## 10. 配置体系

### 10.1 配置层次

```
config.Config (全局配置)
  ├── LLM.Models[]        ← Agent 的 ModelID 引用这里
  ├── LLM.ActiveModelID   ← Agent 未指定 ModelID 时的降级
  ├── ApprovalPolicy      ← 全局审批策略
  ├── ToolCallLimit       ← 全局工具调用上限
  ├── TeamMode            ← Team Mode 开关（新增）
  └── ...

Agent.Config (Agent 配置)
  ├── ID, Name, Description
  ├── SystemPrompt
  ├── ModelID             ← 引用 config.LLM.Models[].ID
  ├── Tools               ← 工具列表，引用 tools.Registry 中的工具名
  ├── Skills              ← Skills 列表，引用 skills.Manager 中的 skill 名
  └── Builtin
```

### 10.2 配置加载流程

```
1. 加载内置 Agent 列表（代码硬编码）
2. 注册到 agent.Registry
3. 第一个注册的 Agent 为 defaultAgent（devo-default）
```

---

## 11. REST API 设计

### 11.1 新增 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/agents` | 列出所有可用 Agent（内置 Agent 的只读列表） |
| `GET` | `/api/v1/agents/{id}` | 获取单个 Agent 详情 |
| `PUT` | `/api/v1/config/team-mode` | 开启/关闭 Team Mode |

### 11.2 修改现有 API

**`POST /api/v1/sessions`** —— 创建 Session 时增加 `agent_id` 字段，并验证合法性：

```json
// 请求
{
  "working_directory": "/path/to/project",
  "agent_id": "code-reviewer"    // 可选，默认使用 Registry 的 DefaultAgent 的 ID
}

// 响应
{
  "id": "sess-xxx",
  "agent_id": "code-reviewer",
  ...
}

// 若 agent_id 不存在，返回 400
{
  "error": "unknown agent_id: nonexistent-agent"
}
```

**`GET /api/v1/sessions`** —— 列表增加 `agent_id` 字段和过滤参数：

```
GET /api/v1/sessions?agent_id=code-reviewer
```

响应中 `listSessionsItem` 增加 `agent_id` 字段。

**`GET /api/v1/sessions/{id}`** —— 返回 `agent_id` 字段（已实现）。

**`PUT /api/v1/sessions/{id}`** —— 不接受 `agent_id`，传入则忽略（不报错，保持向后兼容）。

**`SessionStore` 接口变更**：

```go
// internal/core/session/session.go

type SessionStore interface {
    // ... 现有方法 ...

    // ListSessions 增加 agentID 参数（空字符串表示不过滤）
    ListSessions(status, project, agentID string, limit, offset int) ([]Session, int, error)

    // ...
}
```

SQLite 实现需要同步修改，增加 `WHERE agent_id = ?` 条件。

### 11.3 API 响应格式

**`GET /api/v1/agents`**：

```json
{
  "agents": [
    {
      "id": "devo-default",
      "name": "Devo",
      "description": "通用编程助手，具备完整的代码读写和执行能力",
      "model_id": "",
      "tools": null,
      "skills": null,
      "builtin": true
    },
    {
      "id": "code-reviewer",
      "name": "Code Reviewer",
      "description": "代码审查专家，只读分析，不修改代码",
      "model_id": "",
      "tools": ["read_file", "glob", "list_files", "search_codebase"],
      "skills": null,
      "builtin": true
    },
    {
      "id": "architect",
      "name": "Architect",
      "description": "架构设计专家，专注于系统设计和架构评审",
      "model_id": "",
      "tools": ["read_file", "glob", "list_files", "search_codebase"],
      "skills": null,
      "builtin": true
    },
    {
      "id": "test-writer",
      "name": "Test Writer",
      "description": "测试编写专家，专注于生成和维护测试代码",
      "model_id": "",
      "tools": ["read_file", "write_file", "glob", "list_files", "search_codebase", "exec"],
      "skills": null,
      "builtin": true
    }
  ]
}
```

**`PUT /api/v1/config/team-mode`**：

```json
// 请求
{
  "enabled": true
}

// 响应 200
{
  "team_mode": true,
  "available_sub_agents": ["code-reviewer", "architect", "test-writer"]
}
```

---

## 12. Team Mode：子 Agent 委托

### 12.1 概念

Team Mode 是 Devo 多 Agent 协作的核心机制。它借鉴了 Claude Code 的 `delegate_to` 模式，但做了关键简化：**它是一个用户可控的开关，不是 AI 自动决策的黑盒**。

```
默认（Team Mode OFF）：
  用户 → 主 Agent → 执行任务 → 返回结果

Team Mode ON：
  用户 → 主 Agent → 执行任务
                    ↓
              delegate_to("code-reviewer", "审查这段代码")
                    ↓
              子 Agent 独立上下文，执行审查
                    ↓
              结果返回主 Agent
                    ↓
              主 Agent 根据结果继续
```

### 12.2 开关控制

```
用户侧：
  前端 UI → Team Mode 开关（类似 YOLO 模式）
  配置项：config.TeamMode bool（持久化）
  
API：
  PUT /api/v1/config/team-mode  { "enabled": true/false }
  GET /api/v1/config            → 返回 team_mode 状态

后端行为：
  Team Mode OFF → 主 Agent 的工具集不包含 delegate_to
  Team Mode ON  → 主 Agent 的工具集包含 delegate_to（动态注入）
```

**开关的影响范围**：Team Mode 是全局开关，影响所有 Session。开启后，所有 Session 的主 Agent 都能使用 `delegate_to`。

### 12.3 delegate_to 工具

```go
// internal/taskexec/tools/delegate_to.go

type DelegateToTool struct {
    agentRegistry *agent.Registry   // 用于查找子 Agent
    sessionStore  session.SessionStore
    activeSession string            // 当前 Session ID
    toolRegistry  *tools.Registry   // 用于为子 Agent 创建过滤后的工具集
    skillsMgr     *skills.Manager
    appCfg        *config.Config
}

func (t *DelegateToTool) Name() string {
    return "delegate_to"
}

func (t *DelegateToTool) Description() string {
    return `Delegate a subtask to a specialized sub-agent. 
Use this when you need a different perspective or expertise.
Available sub-agents: code-reviewer, architect, test-writer.
The sub-agent will work independently and return its findings to you.`
}

func (t *DelegateToTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "agent_id": map[string]interface{}{
                "type":        "string",
                "description": "The ID of the sub-agent to delegate to (e.g., 'code-reviewer', 'architect', 'test-writer')",
            },
            "task": map[string]interface{}{
                "type":        "string",
                "description": "The task description for the sub-agent. Be specific about what you want.",
            },
        },
        "required": []string{"agent_id", "task"},
    }
}

func (t *DelegateToTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
    agentID := params["agent_id"].(string)
    task := params["task"].(string)

    // 1. 查找子 Agent
    subAgent := t.agentRegistry.Get(agentID)
    if subAgent == nil {
        return fmt.Errorf("unknown sub-agent: %s", agentID)
    }

    // 2. 验证子 Agent 是否允许被委托（不能委托给自己，不能委托给非子 Agent）
    if subAgent.Config.ID == t.activeSessionAgentID() {
        return fmt.Errorf("cannot delegate to self")
    }

    // 3. 创建临时子 Session
    subSessionID := fmt.Sprintf("%s-sub-%d", t.activeSession, time.Now().UnixNano())
    subSession := &session.Session{
        ID:               subSessionID,
        AgentID:          agentID,
        WorkingDirectory: workingDir,
        Status:           session.StatusIdle,
        ParentSessionID:  t.activeSession,  // 标记父 Session
    }
    if err := t.sessionStore.Create(subSession); err != nil {
        return fmt.Errorf("failed to create sub-session: %w", err)
    }

    // 4. 向子 Agent 发送任务
    w.Write("stream", fmt.Sprintf("Delegating to %s...", subAgent.Config.Name))
    result, err := subAgent.ProcessMessage(ctx, subSessionID, task)
    if err != nil {
        return fmt.Errorf("sub-agent %s failed: %w", agentID, err)
    }

    // 5. 清理子 Session
    _ = t.sessionStore.Delete(subSessionID)

    // 6. 返回子 Agent 的结果
    w.Write("result", result)
    return nil
}
```

### 12.4 子 Agent 上下文隔离

**这是最关键的设计决策**。子 Agent **不共享**主 Agent 的对话历史。

```
主 Agent 上下文：
  SystemPrompt: "You are a general coding assistant..."
  Messages: [用户消息, assistant 回复, 工具调用, ...]
  ↓ delegate_to("code-reviewer", "审查 auth.go 的安全性")

子 Agent 上下文（独立）：
  SystemPrompt: "You are a code review expert..."
  Messages: [
    {
      role: "user",
      content: "审查 auth.go 的安全性\n\nContext: The main agent is working on implementing user authentication.\n\nTask: 审查 auth.go 的安全性"
    }
  ]
  Tools: [read_file, glob, search_codebase, list_files]  ← 只读工具
```

**为什么隔离**：
1. **避免上下文冲突**（Cognition 的警告）：子 Agent 的 SystemPrompt 和主 Agent 完全不同，混入主 Agent 的对话历史会导致子 Agent 行为异常
2. **节省 Token**：子 Agent 不需要看到主 Agent 的完整对话历史
3. **安全边界**：子 Agent 只看它需要看的内容，无法访问主 Agent 的敏感对话

**主 Agent 传递给子 Agent 的信息**：
- `task` 参数（必须）
- 当前工作目录（自动继承）
- 主 Agent 可以在 `task` 中包含必要的上下文（如文件路径、代码片段）

### 12.5 子 Agent 的工具集

子 Agent 使用自己的工具集配置，**不受主 Agent 工具集影响**。

```
主 Agent (devo-default): Tools = nil（全部工具）
  → delegate_to("code-reviewer", ...)
  → code-reviewer: Tools = [read_file, glob, list_files, search_codebase]（只读）

主 Agent 有 write_file，子 Agent 没有。
即使主 Agent 被限制为只读工具，子 Agent 仍使用自己的工具集。
```

### 12.6 子 Agent 的可用范围

Team Mode 开启后，主 Agent 可委托的子 Agent 为当前 Registry 中所有 `SubAgentOf` 为空的顶级 Agent（排除主 Agent 自身）。

```
Registry 中的 Agent：
  devo-default    ← 主 Agent，可委托给 code-reviewer, architect, test-writer
  code-reviewer   ← 子 Agent，可被委托
  architect       ← 子 Agent，可被委托
  test-writer     ← 子 Agent，可被委托
```

### 12.7 委托调用链

**不支持嵌套委托**。子 Agent 不能继续委托给其他 Agent。防止无限递归和 Token 爆炸。

```
主 Agent → delegate_to("code-reviewer", ...)
  code-reviewer → delegate_to("architect", ...)  ← 不允许，返回错误
```

### 12.8 用户可见的体验

**前端 UI**：
- 聊天面板增加 Team Mode 开关（类似 YOLO 模式的开关）
- 开启后，主 Agent 的工具调用日志中显示 `delegate_to` 的调用
- 子 Agent 的执行过程可折叠展开查看

**用户对话示例**：

```
用户: "帮我实现一个 LRU Cache，要线程安全，然后审查一下"

Devo: 好的，我来实现。

[调用 write_file: lru.go]
  ✓ 已创建 lru.go，实现了线程安全的 LRU Cache

[Team Mode] 委托 code-reviewer 审查 lru.go...
  [code-reviewer 执行中]
    [调用 read_file: lru.go]
    [调用 search_codebase: "sync.Mutex"]
  [code-reviewer 完成]
  发现 3 个问题：
    1. Get 方法在锁内调用 evict，可能死锁
    2. 缺少容量为 0 的边界检查
    3. Put 方法未处理 key 已存在的情况

根据审查意见修复...
[调用 edit_file: lru.go]
  ✓ 已修复 3 个问题

✅ 完成。LRU Cache 已实现并通过 code-reviewer 审查。
```

### 12.9 注意事项

1. **子 Agent 调用是同步的**：主 Agent 等待子 Agent 完成后才继续。当前不支持并行委托。
2. **子 Agent 会话是临时的**：子 Session 在执行完成后立即清理，不保留在 Session 列表中。
3. **审批策略**：子 Agent 的工具调用仍受全局审批策略控制。如果开启了审批，子 Agent 的 `write_file` 等操作也需要审批。
4. **Token 消耗**：每次 `delegate_to` 会产生额外的 LLM 调用，用户应知晓 Team Mode 会增加 Token 消耗。
5. **子 Agent 也使用自己的 LLM 模型**：如果 `code-reviewer` 配置了 `ModelID`，子 Agent 使用自己的模型，否则使用主 Agent 的模型。

---

## 13. 数据结构变更

### 13.1 agent.Config 新增字段

```go
type Config struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    Description  string   `json:"description"`
    SystemPrompt string   `json:"system_prompt"`
    ModelID      string   `json:"model_id"`     // 新增：绑定的 LLM 模型
    Tools        []string `json:"tools"`
    Skills       []string `json:"skills"`        // 新增：激活的 Skills
    Builtin      bool     `json:"builtin"`       // 新增：是否内置
    SubAgentOf   string   `json:"-"`             // 新增：标记子 Agent 归属
}
```

### 13.2 config.Config 新增字段

```go
type Config struct {
    // ... 现有字段 ...
    TeamMode bool `json:"team_mode,omitempty"`  // 新增：Team Mode 开关
}
```

### 13.3 session.Session 现有字段

```go
type Session struct {
    AgentID string `json:"agent_id"`  // 已有，无需修改
    // ... 其余字段不变
}
```

### 13.4 agent.Registry 新增方法和字段

```go
// 新增字段
builtinIDs map[string]bool

// 新增方法
func (r *Registry) Unregister(agentID string) error
func (r *Registry) List() []*Agent
```

### 13.5 tools.Registry 新增方法

```go
func (r *Registry) Filter(toolNames []string) *Registry
```

### 13.6 skills.Manager 新增类型

```go
type FilteredSkillsView struct { ... }
func (m *Manager) WithFilter(skillNames []string) *FilteredSkillsView
```

### 13.7 prompt.SkillsProvider 接口扩展

```go
// 旧接口（仅用于 PromptAssembler）
type SkillsProvider interface {
    GetActiveSkillsPrompt() string
}

// 新接口（同时用于 PromptAssembler 和 UseSkillTool）
type SkillsProvider interface {
    GetActiveSkillsPrompt() string
    IsSkillAllowed(name string) bool            // 新增
    GetSkill(name string) (*skills.Skill, error) // 新增
}
```

### 13.8 providers 包新增函数

```go
// internal/taskexec/llmclient/providers/factory.go
func NewClientForModel(cfg *config.Config, modelID string, registry *tools.Registry) llmclient.Client
```

### 13.9 agentloop.Loop.New 签名变更

```go
// 新增参数
systemPrompt string  // Agent 的 SystemPrompt，为空时降级到 DefaultSystemPrompt
```

### 13.10 session.SessionStore.ListSessions 签名变更

```go
// 旧签名
ListSessions(status, project string, limit, offset int) ([]Session, int, error)

// 新签名
ListSessions(status, project, agentID string, limit, offset int) ([]Session, int, error)
```

### 13.11 rest.listSessionsItem 新增字段

```go
type listSessionsItem struct {
    // ... 现有字段 ...
    AgentID string `json:"agent_id"`  // 新增
}
```

### 13.12 App 结构体变更

```go
type App struct {
    // 删除
    // llm llmclient.Client  ← 不再需要全局 LLM Client

    // 新增
    // （无需新增字段，llmClient 由各 Agent 内部管理）
}
```

### 13.13 UseSkillTool 结构体变更

```go
// 旧
type UseSkillTool struct {
    manager *skills.Manager
    loaded  map[string]bool
}

// 新
type UseSkillTool struct {
    skillsProvider SkillsProvider  // 改为接口，支持 Agent 级别的 Skills 过滤
    loaded         map[string]bool
}
```

### 13.14 新增 DelegateToTool

```go
// internal/taskexec/tools/delegate_to.go（新文件）
type DelegateToTool struct {
    agentRegistry *agent.Registry
    sessionStore  session.SessionStore
    activeSession string
    toolRegistry  *tools.Registry
    skillsMgr     *skills.Manager
    appCfg        *config.Config
}
```

---

## 14. 实施路径

### Phase 1：静态多 Agent + Team Mode（本期）

**目标**：
1. 用户可以在创建 Session 时选择预定义的 Agent，每个 Agent 有不同的 SystemPrompt 和工具集
2. 修复所有当前实现的关键缺陷
3. 实现 Team Mode 开关和 `delegate_to` 工具

**任务清单**：

| # | 任务 | 涉及文件 | 预估 |
|---|------|---------|------|
| **1** | `agent.Config` 增加 `ModelID`、`Skills`、`Builtin`、`SubAgentOf` 字段 | `internal/core/agent/agent.go` | 0.5h |
| **2** | `tools.Registry` 增加 `Filter` 方法 | `internal/taskexec/tools/registry.go` | 0.5h |
| **3** | `skills.Manager` 增加 `WithFilter` + `FilteredSkillsView`，实现 `SkillsProvider` 新接口 | `internal/core/skills/manager.go` | 1h |
| **4** | `prompt.SkillsProvider` 接口扩展，增加 `IsSkillAllowed` 和 `GetSkill` 方法 | `internal/core/prompt/types.go` | 0.5h |
| **5** | `UseSkillTool` 改为持有 `SkillsProvider` 接口，增加白名单检查 | `internal/taskexec/tools/use_skill.go` | 0.5h |
| **6** | `providers` 包新增 `NewClientForModel` 函数 | `internal/taskexec/llmclient/providers/factory.go` | 0.5h |
| **7** | `agentloop.Loop.New` 增加 `systemPrompt` 参数，不再硬编码 | `internal/core/agentloop/loop.go` | 0.5h |
| **8** | **重构 `agent.New`**：不再接收 `llmclient.Client`，内部根据 `ModelID` 创建 LLM Client、过滤工具、过滤 Skills、传入 SystemPrompt | `internal/core/agent/agent.go` | 1.5h |
| **9** | 编写内置 Agent 的 SystemPrompt（`CodeReviewerPrompt`、`ArchitectPrompt`、`TestWriterPrompt`） | `internal/core/prompt/` 新增文件 | 1h |
| **10** | `agent.Registry` 增加 `List`、`Unregister` 方法，`builtinIDs` 字段 | `internal/core/agent/registry.go` | 0.5h |
| **11** | **重构 `app.initAgent()`**：删除 `a.llm` 字段，循环注册 `BuiltinAgents`，每个 Agent 独立创建 LLM Client | `internal/cli/app.go` | 1h |
| **12** | `GET /api/v1/agents` 接口 | `internal/interfaces/rest/` 新增 handler | 1h |
| **13** | `POST /api/v1/sessions` 验证 `agent_id` 合法性，默认值从 Registry 获取 | `internal/interfaces/rest/session_handler.go` | 0.5h |
| **14** | `SessionStore.ListSessions` 增加 `agentID` 参数，SQLite 实现同步修改 | `internal/core/session/`、`internal/store/sqlite/` | 0.5h |
| **15** | `GET /api/v1/sessions` 和 `GET /api/v1/sessions/{id}` 返回 `agent_id`，列表支持过滤 | `internal/interfaces/rest/session_handler.go` | 0.5h |
| **16** | `config.Config` 增加 `TeamMode` 字段 | `internal/config/config.go` | 0.5h |
| **17** | `PUT /api/v1/config/team-mode` 接口 | `internal/interfaces/rest/` 新增 handler | 0.5h |
| **18** | 实现 `DelegateToTool` | `internal/taskexec/tools/delegate_to.go`（新文件） | 2h |
| **19** | `agent.New` 中根据 `TeamMode` 动态注入 `delegate_to` 工具到主 Agent | `internal/core/agent/agent.go` | 1h |
| **20** | 前端/TUI 增加 Agent 选择器 + Team Mode 开关 | `web/`、`internal/interfaces/tui/` | 2h |

**总预估**：约 16 小时（2-2.5 天）。

**关键重构点**（需重点测试）：

| 重构项 | 影响范围 | 测试要点 |
|--------|---------|---------|
| `agent.New` 不再接收 `llmclient.Client` | 所有调用 `agent.New` 的地方 | 确保 LLM Client 正确创建，ModelID 降级逻辑正确 |
| `Loop.New` 增加 `systemPrompt` 参数 | 所有调用 `Loop.New` 的地方 | 确保 SystemPrompt 正确注入到 promptAssembler |
| `UseSkillTool` 改为接口 | 所有创建 `UseSkillTool` 的地方 | 确保 Skills 白名单检查生效 |
| `App.llm` 字段删除 | `app.go` 中所有引用 `a.llm` 的地方 | 确保 solidifier 等依赖方改用新的 LLM Client 来源 |
| `SessionStore.ListSessions` 签名变更 | 所有调用方和 SQLite 实现 | 确保 agent_id 过滤正确 |
| `DelegateToTool` 集成 | 主 Agent 的工具调用链 | 确保子 Agent 上下文隔离，委托链正确，Token 消耗可控 |

### Phase 2：用户自定义工作流（远期）

**目标**：用户可定义 Agent 协作工作流（如 YAML 配置），编排多个 Agent 的协作流程。单独设计文档，不在本文档范围内。

---

## 15. 不做的事情（以及设计预留）

| 项目 | 说明 | 预留 |
|------|------|------|
| Agent 间自主协商 | 不做。AI 不自动决策调度哪个 Agent，容易失控 | — |
| Session 切换 Agent | 明确禁止，理由见第 7 节 | — |
| 按 Agent 的 Token 统计 | 暂不区分，待用量统计重构时统一考虑 | `session.UsageStepRecord` 预留 `agent_id` 字段（可为空），避免后续改表 |
| 嵌套委托 | 子 Agent 不能再委托给其他 Agent，防止无限递归 | — |
| 并行委托 | 当前只支持同步委托，并行委托增加复杂度且收益有限 | 可在 `delegate_to` 参数中预留 `parallel` 字段 |
| 用户自定义工作流 | 不在本期范围，Phase 2 单独设计 | — |
| 外部 SDK/CLI 编排 | 不做。Devo 的定位是 IDE 内对话驱动，不是脚本编排工具 | — |

---

## 16. 风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| 工具过滤后 Loop 行为异常 | LLM 可能调用不存在的工具导致错误 | `Filter` 时做校验，工具名不存在时打 warning 并跳过 |
| 不同 Agent 不同 LLM Provider 的兼容性 | 不同 Provider 的 tool call 格式不同 | 已验证 `providers.Factory` 支持多 Provider，`NewClientForModel` 内部可根据 Provider 类型创建客户端 |
| Skills 过滤后 `use_skill` 工具仍可用 | 用户可以绕过 Agent 限制调用 Skill | `use_skill` 工具在执行时检查 Agent 的 Skills 白名单（第 6.5 节） |
| `agent.New` 签名变更导致编译错误 | 所有调用 `agent.New` 的地方需要修改 | 测试文件中同步修改；`agent.New` 内部降级逻辑确保零配置也能正常工作 |
| `SessionStore.ListSessions` 签名变更导致编译错误 | 所有实现 `SessionStore` 接口的地方需要修改 | SQLite 实现和 mock 实现同步修改 |
| FilteredSkillsView 与原 Manager 共享底层数据 | 并发修改 Skills 时可能不一致 | `FilteredSkillsView` 只读，不修改底层数据；Manager 的写操作通过 `sync.RWMutex` 保护 |
| 多个 Agent 各自创建 LLM Client 导致连接数膨胀 | 资源消耗增大 | 内置 Agent 仅 4 个，连接数可控 |
| Team Mode 导致 Token 消耗激增 | 每次 `delegate_to` 产生额外 LLM 调用 | 默认关闭；前端展示 Token 消耗预估；子 Agent 上下文精简（只传 task，不传完整历史） |
| 子 Agent 执行失败阻塞主 Agent | 主 Agent 等待超时或收到错误 | `delegate_to` 有超时机制（默认 120s）；失败时返回错误给主 Agent，主 Agent 可决定重试或跳过 |
| 子 Agent 与主 Agent 工具调用冲突 | 文件锁冲突 | 子 Agent 只读工具（code-reviewer, architect）无冲突；test-writer 可能写文件，通过 `pathLockManager` 管理 |