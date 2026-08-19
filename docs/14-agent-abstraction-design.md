# Devo Agent 抽象层设计文档

**版本**：2.0.0

**状态**：设计阶段

**关联文档**：[2-architecture.md](./2-architecture.md)、[6-agent-loop-event-driven-refactor.md](./6-agent-loop-event-driven-refactor.md)

---

## 目录

1. [背景与动机](#1-背景与动机)
2. [Agent 与 Session 的关系模型](#2-agent-与-session-的关系模型)
3. [现状分析](#3-现状分析)
4. [设计目标](#4-设计目标)
5. [Agent 模型设计](#5-agent-模型设计)
6. [Session 瘦身](#6-session-瘦身)
7. [Loop 重构](#7-loop-重构)
8. [文件结构变化](#8-文件结构变化)
9. [实施路径](#9-实施路径)
10. [Agent 初始化细节](#10-agent-初始化细节)
11. [Agent 执行流程细节](#11-agent-执行流程细节)
12. [配置体系设计](#12-配置体系设计)
13. [REST API 变更影响](#13-rest-api-变更影响)
14. [前端与 TUI 变更](#14-前端与-tui-变更)
15. [不做的事情](#15-不做的事情)
16. [风险与应对](#16-风险与应对)
17. [补充细节](#17-补充细节v150-新增)
18. [深度分析与重构建议](#18-深度分析与重构建议v150-新增)
- [附录 A：Session 字段迁移对照表](#附录-asession-字段迁移对照表)
- [附录 B：Loop 配置读取对照表](#附录-bloop-配置读取对照表)

---

## 1. 背景与动机

### 1.1 什么是 Agent

在 AI 编码助手领域，**Agent（智能体）** 是一个具备以下能力的自治系统：

```
Agent = LLM（大脑） + 工具（手脚） + 循环（心跳）
```

Agent 的核心职责是：接收用户输入 → 自主推理 → 调用工具 → 观察结果 → 迭代直到完成任务。

Devo 已经实现了完整的 Agent 能力（状态机驱动的 Agent Loop、工具系统、审批门控、记忆系统等），但代码中缺少一个显式的 **Agent 概念**——没有一个结构体、类型或抽象层来表达"这是一个 Agent"。

### 1.2 当前问题

`internal/core/agentloop/loop.go` 中的 `Loop` 结构体承担了所有 Agent 职责，但它本质上是一个**执行引擎**，而非 Agent 本身。这导致以下问题：

| 问题 | 表现 | 影响 |
|------|------|------|
| **配置散落** | Agent 行为参数分散在 `Session`、`PromptAssembler`、`Loop` 等多个位置 | 修改行为需要改多处代码 |
| **Session 职责过重** | `Session` 包含了 `ToolCallLimit`、`ApprovalPolicy` 等不属于会话的属性 | 语义混乱，Session 不内聚 |
| **System Prompt 硬编码** | `prompt/assembler.go` 中 `defaultBasePrompt` 是常量 | 无法在不改代码的情况下调整 Agent 行为 |
| **Loop 上帝对象** | `Loop` 构造函数需要注入 10+ 个依赖 | 难以测试、难以理解 |
| **Agent 无身份** | 没有名称、描述等标识 | 用户感知不到"在和谁协作" |

---

## 2. Agent 与 Session 的关系模型

### 2.1 核心关系：Agent 1 : N Session

Agent 是顶层实体，每个 Agent 拥有多个 Session。每次对话都在某个 Agent 的某个 Session 中进行：

```
Agent "devo-default"
  ├── Session A（项目 /home/bean/project1）
  │     ├── 消息历史
  │     └── ...
  ├── Session B（项目 /home/bean/project2）
  │     ├── 消息历史
  │     └── ...
  └── Session C（项目 /home/bean/project3）
        └── ...

未来多 Agent 时，每个 Agent 独立拥有自己的 Session：

Agent "devo-default"          Agent "code-reviewer"
  ├── Session A                 ├── Session D
  ├── Session B                 └── Session E
  └── Session C

Agent "devops-helper"
  ├── Session F
  └── Session G
```

`create_agent()` 创建一个 Agent，之后的对话都在该 Agent 的某个 Session 下进行。

### 2.2 职责划分

```
┌─────────────────────────────────────────────┐
│ Agent                                       │
│ 定义"我是谁"和"我能做什么"                    │
│                                             │
│  - System Prompt（定义全部行为风格）           │
│  - 名称、描述                                │
│  - 工具集                                    │
│  - 运行时约束（MaxToolCalls、审批策略等）        │
│                                             │
│  Agent 内部通过 Loop（执行引擎）来驱动         │
│  Loop 是 Agent 的实现细节，不暴露给上层        │
└──────────────┬──────────────────────────────┘
               │ 1
               │ 拥有
               ▼ N
┌─────────────────────────────────────────────┐
│ Session                                     │
│ 定义"我们在哪"和"聊了什么"                    │
│                                             │
│  - 工作目录                                  │
│  - 消息历史                                  │
│  - 对话状态                                  │
│  - 每个 Session 独立，互不干扰                 │
└─────────────────────────────────────────────┘
```

### 2.3 Loop 的位置

Loop 是 Agent 内部的执行引擎，**不是关系模型中的独立一层**：

```go
// Agent 创建时内部持有 Loop 实例
type Agent struct {
    ID           string
    Name         string
    SystemPrompt string
    // ... 配置 ...

    loop *agentloop.Loop  // 内部执行引擎，不对外暴露
}
```

对外暴露的 API 是：

```
create_agent(...)          → Agent
agent.create_session(...)  → Session
agent.send_message(...)    → 在当前 Session 中对话
```

Agent 内部的 Loop 如何工作，是 Agent 自己的事。

### 2.4 用户视角

```
"我在和 Devo 聊天，
 当前在 project1 这个会话中。
 未来如果有 code-reviewer，
 我可以切换到它，在它的会话中审查代码。"
```

---

## 3. 现状分析

### 3.1 当前 Agent 相关配置的分布

```
Session（不应该有的字段）
  ├── ToolCallLimit          int                   ← 应迁移到 Agent
  ├── MaxContextTokens       int                   ← 应迁移到 Agent
  ├── ApprovalPolicy         map[string]string     ← 应迁移到 Agent
  └── SystemPromptOverride   string                ← 暂不支持，删除

PromptAssembler（assembler.go）
  └── defaultBasePrompt      string（常量）        ← Agent 身份/行为，应迁移到 Agent

Loop（loop.go）
  ├── llmClient              llmclient.Client      ← 全局绑定，无法按 Agent 切换
  └── toolExecutor           ToolExecutor          ← 全局绑定，无法按 Agent 过滤

App（app.go）
  └── 负责把所有组件手动组装到一起
```

### 3.2 当前 Session 结构体全字段分析

`Session` 当前包含 25 个字段，按职责可分为三类：

```go
// session.go - Session 结构体（完整字段列表）
type Session struct {
    // === 第一类：会话核心字段（保留在 Session） ===
    ID               string    // 会话唯一标识
    Title            string    // 会话标题
    WorkingDirectory string    // 工作目录
    State            State     // 当前状态（Idle/Thinking/...）
    CreatedAt        time.Time // 创建时间
    LastActiveAt     time.Time // 最后活跃时间
    Messages         []Message // 消息历史

    // === 第二类：行为配置默认值（全部迁移到 config.Config，Session 不再存储） ===
    ToolCallLimit          int               // → config.Config.ToolCallLimit（Session 不再存储）
    MaxContextTokens       int               // → config.Config.MaxContextTokens（Session 不再存储）
    KeepRecent             int               // → config.Config.KeepRecent（Session 不再存储）
    SystemPromptOverride   string            // → **删除**（Agent.Config.SystemPrompt 替代）
    MaxConcurrentToolCalls int               // → config.Config.MaxConcurrentToolCalls（Session 保留字段用于覆盖）
    MaxConcurrentSubprocesses int            // → config.Config.MaxConcurrentSubprocesses（Session 保留字段用于覆盖）
    ApprovalPolicy         map[string]string // → config.Config.ApprovalPolicy（Session 保留字段仅存覆盖值）

    // === 第三类：运行时状态（保留在 Session） ===
    ToolCallCount             int                    // 当前 Loop 中已执行的工具调用次数
    MessageCount              int                    // 消息总数
    LastLoopTerminationReason LoopTerminationReason  // 上次 Loop 终止原因
    TokenUsage                TokenUsage             // Token 用量统计（累计）
    CompressionCount          int                    // 压缩次数
    CurrentContextTokens       int                    // 当前上下文 Token 估算值
    ActiveSSEConnections      int                    // 活跃 SSE 连接数
    CancelRequested           bool                   // 取消请求标志
    PauseRequested            bool                   // 暂停请求标志
    ChildPID                  *int                   // 子进程 PID
    BackgroundPIDs            []int                  // 后台进程 PID 列表
    ArchivePath               string                 // 归档路径
    TrustLevel                string                 // 信任级别
    ActiveSkills              []string               // 当前激活的技能列表
    CachedDirectorySummary    *DirectorySummary      // 目录摘要缓存
    EventBus                  *EventBus              // 事件总线（仅内存）
    ApprovalTimeoutSeconds    int                    // 审批超时秒数（保留在 Session）
}
```

**分类说明**：

- **第一类（会话核心）**：定义"我们在哪"和"聊了什么"，是 Session 的本质属性，保持不变
- **第二类（Agent 行为配置）**：默认值来源迁移到 Agent。其中 `ToolCallLimit`、`MaxContextTokens`、`KeepRecent` 从 Session 中完全移除；`MaxConcurrentToolCalls`、`MaxConcurrentSubprocesses`、`ApprovalPolicy` 保留在 Session 中用于运行时覆盖（默认值从 Agent 读取）
- **第三类（运行时状态）**：Session 执行过程中的瞬时状态和累计数据，保留在 Session
- **第四类（Session 级配置）**：`ApprovalTimeoutSeconds` 纯粹是 Session 级配置，不在 Agent 中定义

### 3.3 当前 Loop 的依赖注入

```go
// app.go
func (a *App) initTools() {
    a.loop.SetBackgroundProcessManager(bgProcManager)
    bgProcManager.SetOutputForwarder(a.loop)
    // ...
}

func (a *App) initMemory() error {
    a.loop.SetMemoryManager(a.memoryMgr)
    // ...
}

func (a *App) initSkills() {
    a.loop.SetSkillsManager(a.skillsMgr)
    a.loop.SetSolidifier(solidifier)
    // ...
}
```

Loop 需要逐个注入依赖，且这些依赖并非 Agent 级别的配置，而是基础设施。

---

## 4. 设计目标与重构原则

### 4.1 设计目标

1. **身份与行为彻底分离**：Agent 只管"我是谁"（ID、Name、SystemPrompt、Tools），行为参数全部归 `config.Config`
2. **配置单源**：`config.Config`（Global + Project 合并 + `ApplyDefaults`）是唯一的行为配置来源，Loop 直接 `l.cfg.XXX`，零兜底
3. **Session 不再做配置中转站**：Session 只存核心数据 + 运行时覆盖，`ToolCallLimit` 等行为默认值全部踢掉
4. **依赖注入一步到位**：构造即完整，消灭所有 `SetXxx` 方法
5. **审批链 3 层**：Session 覆盖 > config.ApprovalPolicy > ApplyDefaults 兜底
6. **Agent 注册表路由**：`agent.Registry` 管理所有 Agent，Session 通过 `AgentID` 路由到对应 Agent，Handler 不再绑定单一 Agent

### 4.2 重构原则（必须遵守）

> **禁止为了少写代码而打补丁。要有重构系统的勇气。**

| 原则 | 说明 |
|------|------|
| **不留兼容层** | 旧字段直接删除，不保留 "JSON tag 但不赋值" 之类的过渡代码 |
| **不写兜底分支** | `l.cfg` 已经过 `ApplyDefaults`，所有值保证非零，禁止再写 `if <= 0` 兜底 |
| **不搞"暂不处理"** | 设计上认定要改的，就一次改到位，不写 "TODO" "暂缓" "后续优化" |
| **不迁就旧测试** | 测试用例跟着代码一起改，不为通过旧测试而在新代码里留兼容逻辑 |
| **删比留好** | 不确定的字段/方法/接口，先删除；将来需要时从 git 历史找回，比留着烂代码强 |

---

## 5. Agent 模型设计

### 5.1 核心原则

**Agent 只管"身份"，不管"行为"。**

- **身份**：我是谁、我有什么能力、我怎么说话 —— Agent Config 的职责
- **行为**：调用上限、上下文窗口、并发数、审批策略 —— `config.Config` 的职责

两个 Config **不重复任何字段**。

### 5.2 Agent Config

```go
// internal/core/agent/agent.go
package agent

type Config struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    Description  string   `json:"description"`
    SystemPrompt string   `json:"system_prompt"`
    Tools        []string `json:"tools"` // nil = 全部工具可用
}

type Agent struct {
    Config
    loop *agentloop.Loop
}
```

**只有这 5 个字段。** `ToolCallLimit`、`MaxContextTokens`、`KeepRecent`、`Approval` 等行为参数全部走 `config.Config`（见第 12 节）。

### 5.3 Agent Registry（关键）

**Agent 不绑定到 Handler。** Handler 持有 `*agent.Registry`，Session 通过 `AgentID` 路由到正确的 Agent。

```go
// internal/core/agent/registry.go
package agent

type Registry struct {
    agents  map[string]*Agent
    default *Agent
}

func NewRegistry(defaultAgent *Agent) *Registry {
    r := &Registry{
        agents:  make(map[string]*Agent),
        default: defaultAgent,
    }
    r.Register(defaultAgent)
    return r
}

func (r *Registry) Register(agent *Agent) {
    r.agents[agent.Config.ID] = agent
}

func (r *Registry) Get(agentID string) *Agent {
    if agentID == "" {
        return r.default
    }
    if agent, ok := r.agents[agentID]; ok {
        return agent
    }
    return r.default  // 未知 AgentID 回退到默认
}
```

**路由逻辑**：

```
HTTP Request → Handler.PostMessage()
  → sess, _ := store.Get(sessionID)
  → agent := h.agentRegistry.Get(sess.AgentID)  ← 按 Session 的 AgentID 路由
  → agent.ProcessMessage(ctx, sessionID, msg)
```

这样 Session 里的 `AgentID` 字段就有了真正的意义——不同 Session 可以使用不同 Agent。

### 5.4 Default 工厂

```go
func Default(store session.SessionStore, llm llmclient.Client, registry *tools.Registry,
    cfg *config.Config,
    approvalMgr *approval.Manager, memoryMgr *memory.Manager, skillsMgr *skills.Manager,
    bgProcManager *tools.BackgroundProcessManager, mcpMgr *mcp.Manager, solidifier *skills.Solidifier,
) *Agent {
    return New(Config{
        ID:           "devo-default",
        Name:         "Devo",
        Description:  "通用编程助手",
        SystemPrompt: defaultSystemPrompt(),  // 从 prompt/assembler.go 迁移
        Tools:        nil,
    }, store, llm, registry, cfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)
}
```

### 5.5 Agent 的定位

Agent 是顶层实体，Loop 是内部执行引擎，对外不暴露。**Session 通过 AgentID 选择 Agent，不同 Agent 可以有不同 SystemPrompt、不同 Tools。**

```go
// 创建 Registry
registry := agent.NewRegistry(defaultAgent)

// 可选：注册更多 Agent
registry.Register(agent.New(agent.Config{
    ID:           "python-expert",
    Name:         "Python 专家",
    Description:  "专注于 Python 开发的助手",
    SystemPrompt: pythonExpertPrompt(),
    Tools:        []string{"read_file", "write_file", "edit_file", "exec_python"},
}, store, llm, toolRegistry, cfg, ...))

// Handler 持有 Registry
handler := rest.NewHandler(store, registry, ...)

// 创建 Session 时指定 Agent
POST /api/v1/sessions  { "agent_id": "python-expert", "working_directory": "..." }

// 消息处理时按 Session.AgentID 路由
agent := registry.Get(sess.AgentID)
agent.ProcessMessage(ctx, sessionID, msg)
```

- **Agent 是操作入口**：上层代码只和 Agent 交互
- **Registry 是路由器**：Handler 不绑定单一 Agent，按 Session.AgentID 分发
- **Loop 是内部实现**：Agent 持有 Loop 引用，`ProcessMessage` 转发给 Loop
- **`cfg` 是合并后的配置**：`*config.Config`（Global + Project 合并），Agent 传给 Loop 用于运行时读取行为参数

---

## 6. Session 瘦身

### 6.1 核心思路

Session 不再存储任何**行为默认值**。行为参数全部由 `config.Config`（Global + Project 合并）提供，`config.ApplyDefaults()` 保证所有字段都有值。

Session 只保留两类内容：
- **会话核心数据**：ID、消息、状态、统计
- **运行时覆盖**：并发数、审批策略（用户在 Session 运行期间覆盖的值）

### 6.2 从 Session 移除的字段

以下字段从 Session 结构体中**彻底移除**，不再存储：

| 字段 | 移除理由 | 运行时从哪读 |
|------|---------|-------------|
| `ToolCallLimit` | 行为配置，不是会话数据 | `cfg.ToolCallLimit`（`config.ApplyDefaults()` 保证非零） |
| `MaxContextTokens` | 行为配置，不是会话数据 | `cfg.MaxContextTokens` |
| `KeepRecent` | 行为配置，不是会话数据 | `cfg.KeepRecent` |
| `SystemPromptOverride` | 不再支持，SystemPrompt 是 Agent 身份的一部分 | 走 Agent 切换 |

### 6.3 Session 保留字段（运行时覆盖）

以下字段保留在 Session 中，但**创建时不写值（零值）**，运行时由用户通过 API 设置覆盖：

| 字段 | 默认值来源 | 运行时覆盖 |
|------|-----------|-----------|
| `MaxConcurrentToolCalls` | `cfg.MaxConcurrentToolCalls`（`config.ApplyDefaults()` 设为 1） | ✅ `UpdateConfig` API |
| `MaxConcurrentSubprocesses` | `cfg.MaxConcurrentSubprocesses`（`config.ApplyDefaults()` 设为 5） | ✅ `UpdateConfig` API |
| `ApprovalPolicy` | `cfg.ApprovalPolicy`（`config.ApplyDefaults()` 调用 `DefaultApprovalPolicy()`） | ✅ `SetApprovalPolicy` API |

> **运行时逻辑**：`sess.MaxConcurrentToolCalls > 0 ? sess.MaxConcurrentToolCalls : cfg.MaxConcurrentToolCalls`

### 6.4 Session 核心字段（不变）

| 分类 | 字段 |
|------|------|
| 标识 | `ID`, `Title`, `WorkingDirectory`, `AgentID`（新增） |
| 状态 | `State`, `TrustLevel` |
| 时间 | `CreatedAt`, `LastActiveAt` |
| 数据 | `Messages`, `ArchivePath` |
| 统计 | `MessageCount`, `ToolCallCount`, `TokenUsage`, `CompressionCount`, `CurrentContextTokens` |
| 控制 | `CancelRequested`, `PauseRequested` |
| 进程 | `ChildPID`, `BackgroundPIDs` |
| 基础设施 | `EventBus`, `CachedDirectorySummary`, `ActiveSSEConnections`, `ActiveSkills` |
| Session 配置 | `ApprovalPolicy`（覆盖值）, `ApprovalTimeoutSeconds`, `MaxConcurrentToolCalls`, `MaxConcurrentSubprocesses` |

### 6.5 Session 新增字段

```go
type Session struct {
    AgentID string `json:"agent_id"` // 默认 "devo-default"
    // ... 其余字段
}
```

---

## 7. Loop 重构

### 7.1 角色关系

Loop 持有 `cfg *config.Config`（Global + Project 合并后的最终配置），运行时直接从 `cfg` 读取所有行为参数：

```go
type Loop struct {
    cfg             *config.Config        // 合并后的最终配置，所有行为参数从这里读
    store           session.SessionStore
    llmClient       llmclient.Client
    promptAssembler *prompt.Assembler
    toolExecutor    ToolExecutor
    approvalManager *approval.Manager
    // ...
}
```

> **不再有 `agent *agent.Agent` 字段。** Loop 不需要从 Agent 读任何东西。Agent 的 Config 是身份（ID/Name/SystemPrompt），Loop 需要的 `ToolCallLimit` 等行为参数全部在 `cfg` 里。

### 7.2 构造函数变化

```go
// 当前：Loop 自己决定一切
loop := agentloop.NewWithTools(store, llm, toolRegistry)

// 之后：Agent 创建时内部创建 Loop，cfg 一次性传入
agent := agent.New(agentCfg, store, llm, toolRegistry, cfg,
    approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

// agent.New 内部：
//   a.loop = agentloop.New(store, llm, toolRegistry, cfg,
//       approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)
```

Loop 运行时从 `l.cfg` 读取行为参数，不再从 Session 或 Agent 读取。

### 7.3 App 初始化变化

#### 7.3.1 App 结构体

```diff
// internal/cli/app.go
type App struct {
    cfg           *config.Config   // Global + Project 合并后的配置
    store         session.SessionStore
-   loop          *agentloop.Loop
+   agent         *agent.Agent
    handler       *rest.Handler
    // ...
}
```

#### 7.3.2 初始化顺序

```go
func NewApp(...) (*App, error) {
    app.initDB()        // 1. SQLite
    app.initRegistry()  // 2. 工具注册表
    app.initMCP()       // 3. MCP
    app.initSkills()    // 4. 技能

    // 5. 加载配置（Global + Project 合并，ApplyDefaults 已调用）
    cfg, _ := config.LoadFullConfig(projectDir)
    app.cfg = cfg

    // 6. 独立创建 approval.Manager
    approvalMgr := approval.NewManager(cfg.ApprovalPolicy, ...)

    // 7. Memory（需要 approvalMgr）
    app.initMemory(approvalMgr)

    app.initLLM()       // 8. LLM 客户端

    // 9. 后台进程管理器
    app.bgProcManager = tools.NewBackgroundProcessManager()

    // 10. 创建 Agent，cfg 一并传入
    app.agent = agent.New(agent.Default(), app.store, app.llm, app.toolRegistry,
        app.cfg,  // ← 合并后的配置
        approvalMgr, app.memoryMgr, app.skillsMgr,
        app.bgProcManager, app.mcpMgr, app.solidifier)

    // 11. Handler
    app.initHandler()

    return app, nil
}

func (a *App) initLLM() {
    a.llm = providers.NewClient(a.cfg, a.toolRegistry)
}

func (a *App) initMemory(approvalMgr *approval.Manager) error {
    pathLockMgr := concurrency.NewPathLockManager()
    memoryStore, _ := memory.DefaultFileStore()
    a.memoryMgr = memory.NewManager(memoryStore, pathLockMgr, approvalMgr)
    return nil
}

#### 7.3.3 Run() 与 bgProcManager

```diff
func (a *App) Run() {
-   if err := a.loop.RecoverCrashedSessions(); err != nil {
+   if err := a.agent.RecoverCrashedSessions(); err != nil {
```

`bgProcManager` 在 `NewApp()` 中创建，Agent 作为 `OutputForwarder`：

```go
// initTools() 中
a.bgProcManager.SetOutputForwarder(a.agent)
```

### 7.4 不再需要的 SetXxx 调用

以下调用全部消除，依赖在 Agent 构造时一次性注入：

| 消除的调用 | 所在位置 | 说明 |
|-----------|---------|------|
| `a.loop.SetBackgroundProcessManager()` | `initTools()` | `bgProcManager` 在 `NewApp()` 中创建，Agent 构造时传入 |
| `a.loop.SetMemoryManager()` | `initMemory()` | Memory 构造时传入 |
| `a.loop.SetSkillsManager()` | `initSkills()` | Skills 构造时传入 |
| `a.loop.SetSolidifier()` | `initSkills()` | Solidifier 构造时传入 |
| `a.loop.SetMCPManager()` | `initMCP()` | MCP 构造时传入 |

### 7.5 配置读取方式变更

**之前**：Loop 从 `sess.ToolCallLimit` 等 Session 字段读取，然后兜底

```go
toolCallLimit := sess.ToolCallLimit
if toolCallLimit <= 0 { toolCallLimit = 50 }
```

**之后**：Loop 从 `l.cfg` 读取，`cfg` 已经过 `Merge()` + `ApplyDefaults()`，保证所有字段非零

```go
toolCallLimit := l.cfg.ToolCallLimit  // 一定有值，无需兜底
```

### 7.6 PromptAssembler 变化

`PromptAssembler` 不再持有硬编码的 `defaultBasePrompt`，改为从 Agent 注入：

```go
// 当前
type Assembler struct {
    basePrompt     string  // 硬编码常量
    skillsProvider SkillsProvider
    memoryProvider MemoryProvider
}

// 之后
type Assembler struct {
    systemPrompt   string  // 从 Agent.Config.SystemPrompt 注入
    skillsProvider SkillsProvider
    memoryProvider MemoryProvider
}

func (a *Assembler) Assemble(sess *session.Session) string {
    var parts []string
    parts = append(parts, a.systemPrompt)  // 不再用硬编码常量
    // ...
}
---

## 8. 文件结构变化

### 8.1 新增文件

```
internal/core/agent/
  ├── agent.go          # Agent 结构体定义 + Default() 工厂函数
  └── registry.go       # Agent Registry：管理多个 Agent，按 AgentID 路由
```

### 8.2 修改文件

| 文件 | 改动 |
|------|------|
| `internal/config/config.go` | `Config` 新增 `MaxConcurrentToolCalls`、`MaxConcurrentSubprocesses`；`ApplyDefaults()` 设置所有行为默认值 |
| `internal/core/agent/agent.go` | **新增**：Agent 结构体 + `New()` / `Default()`（仅身份字段） |
| `internal/core/agent/registry.go` | **新增**：Agent Registry，管理多个 Agent，按 AgentID 路由 |
| `internal/core/session/session.go` | Session 移除 `ToolCallLimit`/`MaxContextTokens`/`KeepRecent`/`SystemPromptOverride`，新增 `AgentID` |
| `internal/core/agentloop/loop.go` | Loop 构造函数接收 `*config.Config`；运行时从 `l.cfg` 读取行为参数 |
| `internal/core/prompt/assembler.go` | 移除硬编码 `defaultBasePrompt`，改为从 Agent 注入 |
| `internal/cli/app.go` | `App.loop` → `App.agentRegistry`；初始化顺序调整；消除所有 `SetXxx` |
| `internal/interfaces/rest/handler.go` | `Handler.loop` → `Handler.agentRegistry`；`NewHandler` 接收 `*agent.Registry` |
| `internal/interfaces/rest/session_handler.go` | `CreateSession` 接受 `agent_id` 参数；不再存行为字段；响应字段裁剪 |
| `internal/interfaces/rest/approval_handler.go` | `SetApprovalPolicy` 存储方式调整 |
| `internal/interfaces/rest/*_handler.go` | 所有 `h.loop` 调用改为 `h.agentRegistry.Get(sess.AgentID).XXX()` |
| `internal/storage/sqlite/models.go` | Session 表新增 `agent_id` 列 |
| `internal/storage/sqlite/store_session.go` | 创建 Session 时默认 `agent_id = "devo-default"` |
| 所有引用 `sess.ToolCallLimit` 等字段的代码 | 改为从 `l.cfg` 读取 |

---

## 9. 实施路径

### 第一阶段：Agent 结构体 + 配置收拢

1. 新建 `internal/core/agent/agent.go`，定义 `Agent` 结构体和 `New()` / `Default()`
2. 将 `approval.Manager` 从 Loop 内部剥离，独立创建
3. 将 `defaultBasePrompt` 从 `assembler.go` 迁移到 `agent.go`
4. Loop 构造函数扩展，接受全部依赖参数
5. `App` 初始化顺序调整，构造 Agent 时一次性传入所有依赖

### 第二阶段：Session 瘦身

1. Session 新增 `AgentID` 字段
2. Session 移除 4 个 Agent 行为参数字段
3. 所有引用这些字段的代码改为从 Agent 读取
4. SQLite Session 表新增 `agent_id` 列，做数据迁移

### 第三阶段：清理

1. 删除 `PromptAssembler` 中的 `basePrompt` 字段和 `SetBasePrompt` 方法
2. 更新相关测试
3. 回归测试

---

## 10. Agent 初始化细节

### 10.1 引入 Agent Registry 后的初始化流程

```go
// internal/core/agent/agent.go
package agent

type Config struct {
    ID           string
    Name         string
    Description  string
    SystemPrompt string
    Tools        []string // nil = 全部工具可用
}

type Agent struct {
    Config
    loop *agentloop.Loop
}

func New(
    cfg Config,
    store session.SessionStore,
    llm llmclient.Client,
    registry *tools.Registry,
    appCfg *config.Config,   // ← Global + Project 合并后的最终配置
    approvalMgr *approval.Manager,
    memoryMgr *memory.Manager,
    skillsMgr *skills.Manager,
    bgProcManager *tools.BackgroundProcessManager,
    mcpMgr *mcp.Manager,
    solidifier *skills.Solidifier,
) *Agent {
    a := &Agent{Config: cfg}
    a.loop = agentloop.New(store, llm, registry, appCfg,
        approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)
    return a
}

func Default(
    store session.SessionStore, llm llmclient.Client, registry *tools.Registry,
    appCfg *config.Config,
    approvalMgr *approval.Manager, memoryMgr *memory.Manager, skillsMgr *skills.Manager,
    bgProcManager *tools.BackgroundProcessManager, mcpMgr *mcp.Manager, solidifier *skills.Solidifier,
) *Agent {
    return New(Config{
        ID:           "devo-default",
        Name:         "Devo",
        Description:  "通用编程助手",
        SystemPrompt: defaultSystemPrompt(),
        Tools:        nil,
    }, store, llm, registry, appCfg, approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)
}
```

### 10.2 App 初始化（使用 Registry）

```go
// internal/cli/app.go
func NewApp(tuiMode, webMode bool, portFlag int, version string) (*App, error) {
    // ... 加载配置 ...
    cfg, _ := config.LoadFullConfig(wd)

    // 构造所有依赖
    store := initDB()
    llm := providers.NewClient(cfg, toolRegistry)
    approvalMgr := approval.NewManager()
    memoryMgr := memory.NewManager(...)
    skillsMgr := skills.NewManager(...)
    bgProcManager := tools.NewBackgroundProcessManager()
    mcpMgr := mcp.NewManager(wd)
    solidifier := skills.NewSolidifier(...)

    // 创建默认 Agent
    defaultAgent := agent.Default(store, llm, toolRegistry, cfg,
        approvalMgr, memoryMgr, skillsMgr, bgProcManager, mcpMgr, solidifier)

    // 创建 Registry（管理所有 Agent）
    agentRegistry := agent.NewRegistry(defaultAgent)

    // 可选：注册更多 Agent（从配置文件加载）
    // agentRegistry.Register(agent.New(agent.Config{...}, ...))

    // Handler 持有 Registry，不再绑定单一 Agent
    handler := rest.NewHandler(store, agentRegistry, memoryMgr, version)

    return &App{
        agentRegistry: agentRegistry,
        handler:       handler,
        // ...
    }, nil
}
```

### 10.3 Handler 使用 Registry 路由

```go
// internal/interfaces/rest/handler.go
type Handler struct {
    store         session.SessionStore
    agentRegistry *agent.Registry  // ← 替代 *agent.Agent
    memoryMgr     *memory.Manager
    // ...
}

func NewHandler(store session.SessionStore, agentRegistry *agent.Registry, ...) *Handler {
    return &Handler{
        store:         store,
        agentRegistry: agentRegistry,
        // ...
    }
}

// PostMessage：按 Session.AgentID 路由到正确的 Agent
func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    sess, err := h.store.Get(id)
    if err != nil {
        writeError(w, http.StatusNotFound, "session not found")
        return
    }

    agent := h.agentRegistry.Get(sess.AgentID)  // ← 按 AgentID 路由
    if err := agent.ProcessMessage(r.Context(), id, msg); err != nil {
        // ...
    }
}
```

### 10.4 CreateSession 接受 agent_id

```go
// internal/interfaces/rest/session_handler.go
type createSessionRequest struct {
    WorkingDirectory string `json:"working_directory"`
    Title            string `json:"title,omitempty"`
    AgentID          string `json:"agent_id,omitempty"`   // ← 新增
    // 不再有 ToolCallLimit, MaxContextTokens, KeepRecent, SystemPromptOverride
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
    var req createSessionRequest
    // ... parse request ...

    agentID := req.AgentID
    if agentID == "" {
        agentID = "devo-default"
    }

    // 验证 Agent 存在
    if _, ok := h.agentRegistry.Get(agentID); ok == nil {
        writeError(w, http.StatusBadRequest, "unknown agent: "+agentID)
        return
    }

    sess := &session.Session{
        ID:               session.GenerateID("sess"),
        Title:            req.Title,
        WorkingDirectory: workingDir,
        AgentID:          agentID,  // ← 存储 AgentID
        // 不再有 ToolCallLimit, MaxContextTokens, KeepRecent
        // ...
    }
    // ...
}
```

### 10.5 关键变化总览

| 变化 | 说明 |
|------|------|
| Agent Config 无行为字段 | `MaxToolCalls`、`Approval` 等全部在 `appCfg` 中 |
| `appCfg` 传入 Loop | Loop 构造函数接收 `*config.Config`，运行时直接读 |
| `appCfg` 已就绪 | `LoadFullConfig()` 已调用 `Merge()` + `ApplyDefaults()`，所有字段非零 |
| 无兜底逻辑 | `l.cfg.ToolCallLimit` 一定是有效值，不需要 `if <= 0` 检查 |
| **Registry 路由** | Handler 不绑定单一 Agent，按 `sess.AgentID` 从 Registry 获取 |
| **Session 选 Agent** | `CreateSession` 接受 `agent_id` 参数，Session 存储 `AgentID` |
| **依赖注入** | 所有依赖在构造 Agent 时一次性传入，零 SetXxx |

### 10.6 Agent 转发方法

Agent 对外暴露的所有操作，内部转发给 Loop：

```go
func (a *Agent) ProcessMessage(ctx context.Context, sessionID string, msg session.Message) error {
    return a.loop.ProcessMessage(ctx, sessionID, msg)
}
func (a *Agent) Pause(sessionID string) error    { return a.loop.Pause(sessionID) }
func (a *Agent) Resume(sessionID string) error   { return a.loop.Resume(sessionID) }
func (a *Agent) Cancel(sessionID string) error   { return a.loop.Cancel(sessionID) }
func (a *Agent) Complete(sessionID string) error { return a.loop.Complete(sessionID) }
func (a *Agent) Archive(sessionID string) error  { return a.loop.Archive(sessionID) }
func (a *Agent) Compact(sessionID string) error  { return a.loop.Compact(sessionID) }
func (a *Agent) Rollback(sessionID string, targetMessageID string) (*agentloop.RollbackResult, error) {
    return a.loop.Rollback(sessionID, targetMessageID)
}
func (a *Agent) UpdateConcurrencyConfig(sessionID string, maxConcurrentToolCalls, maxConcurrentSubprocesses int) error {
    return a.loop.UpdateConcurrencyConfig(sessionID, maxConcurrentToolCalls, maxConcurrentSubprocesses)
}
func (a *Agent) GetApprovalManager() *approval.Manager       { return a.loop.GetApprovalManager() }
func (a *Agent) ResolveApproval(...) (*approval.Result, error) { return a.loop.ResolveApproval(...) }
func (a *Agent) RecoverCrashedSessions() error               { return a.loop.RecoverCrashedSessions() }
func (a *Agent) EstimateInitialContextTokens(sess *session.Session) int {
    return a.loop.EstimateInitialContextTokens(sess)
}
```

### 10.4 工具选择

Agent 通过 `Tools` 字段决定可用工具：

```go
// 全部工具
agent.New(Config{Tools: nil}, ...)

// 只读工具
agent.New(Config{Tools: []string{"read_file", "glob", "search_codebase"}}, ...)
```

`nil` 表示全部工具可用。

---

## 11. Agent 执行流程细节

### 11.1 整体流程

一次完整的对话执行流程如下：

```
用户发送消息
  │
  ▼
Agent.ProcessMessage(ctx, sessionID, msg)
  │
  ▼
Loop.ProcessMessage(ctx, sessionID, msg)
  │
  ├── 1. 检查 Session 状态（必须为 Idle）
  ├── 2. 处理断点续传（上次因达到工具调用上限中断）
  ├── 3. 保存用户消息到存储
  ├── 4. 启动状态机（goroutine 异步执行）
  │
  └── 5. 返回（不等待执行完成，通过 EventBus 推送事件）
```

### 11.2 状态机执行流程

```
                        ┌──────────┐
                        │   Idle   │ ← 起始/结束状态
                        └────┬─────┘
                             │ ProcessMessage
                             ▼
                      ┌──────────────┐
                      │  Preparing   │ 组装 System Prompt + 消息历史
                      └──────┬───────┘
                             │
                             ▼
                      ┌──────────────┐
                      │  Thinking    │ 流式调用 LLM，实时推送 token
                      └──────┬───────┘
                             │
                             ▼
                   ┌───────────────────┐
              ┌───│ EvaluatingResult   │ 判断 LLM 返回的是工具调用还是纯文本
              │   └────────┬──────────┘
              │            │
              │     ┌──────┴──────┐
              │     │  有工具调用？  │
              │     └──────┬──────┘
              │      是    │    否
              │            │     │
              │            ▼     ▼
              │   ┌──────────────┐  ┌──────────────┐
              │   │ToolExecuting │  │TextResponse  │ → 保存回复 → Idle
              │   └──────┬───────┘  └──────────────┘
              │          │
              │   ┌──────┴──────┐
              │   │  需要审批？   │
              │   └──────┬──────┘
              │     是    │   否
              │      │    │    │
              │      ▼    │    │
              │ ┌──────────────┐ │
              │ │AwaitingApproval│ │
              │ └──────┬───────┘ │
              │        │ 审批通过 │
              │        ▼         │
              │   ┌──────────────┐
              │   │  执行工具     │
              │   └──────┬───────┘
              │          │
              │          ▼
              │   ┌──────────────┐
              └──►│  Preparing   │ ← 工具结果返回，重新组装上下文
                  └──────────────┘
                  继续下一轮循环...
```

### 11.3 各状态详解

| 状态 | Handler | 职责 |
|------|---------|------|
| **Preparing** | `preparingHandler` | 组装 System Prompt（从 Agent 读取）、加载消息历史、Token 计数、上下文压缩（如超限） |
| **Thinking** | `thinkingHandler` | 调用 LLM 流式 API，通过 EventBus 实时推送 `thinking_chunk` 事件（含推理链），前端展示打字机效果 |
| **EvaluatingResult** | `evaluatingResultHandler` | 解析 LLM 返回，判断是工具调用（`ToolCalls`）还是纯文本回复 |
| **ToolExecuting** | `toolExecutingHandler` | 保存 Assistant 消息（含工具调用），执行工具（串行/并行），通过 EventBus 推送 `tool_progress`、`tool_result` |
| **AwaitingApproval** | `awaitingApprovalHandler` | 高风险操作暂停，等待用户审批。支持超时自动拒绝 |
| **TextResponse** | `textResponseHandler` | 保存 Assistant 消息，发布 `message_complete` 事件，Session 切回 Idle |

### 11.4 事件驱动机制

Loop 不直接返回结果，而是通过 **EventBus** 推送事件，前端订阅：

```
EventBus 事件流：
  session_state_change → "thinking"
  thinking_chunk       → "I'll read the file..."  (流式 token)
  thinking_chunk       → "first..."                (推理链)
  session_state_change → "tool_executing"
  tool_progress        → {stage: "reading", message: "..."}
  tool_result          → {tool_name: "read_file", success: true}
  session_state_change → "idle"
  message_complete     → {full_text: "...", ...}
```

### 11.5 工具执行策略

- **串行执行**（默认）：逐个执行工具调用，一个完成后执行下一个
- **并行执行**：当 LLM 返回了多个工具调用时，使用 `errgroup` 并行执行
- **路径锁**：文件写入/编辑操作前获取路径锁，防止并发冲突
- **工具调用上限**：`l.cfg.ToolCallLimit` 限制单次 Loop 最大工具调用次数，超限后自动终止

### 11.6 审批流程

引入 Agent 后，审批策略解析的优先级链为三级：

```
工具执行前
  │
  ├── 低风险操作（read_file、glob 等）→ 自动通过，直接执行
  │
  └── 高风险操作（write_file、edit_file、exec_python）
        │
        ├── 信任级别 Elevated → 自动通过
        │
        └── 信任级别 Normal/Low
              │
              ├── Session 级审批策略覆盖？→ 按 Session 策略
              │
              ├── config.ApprovalPolicy (Global+Project 合并)？→ 按合并后策略
              │
              └── 无覆盖 → ApplyDefaults 兜底（DefaultApprovalPolicyMap）
                    │
                    ├── "always_ask" → 暂停，推送给前端等待用户审批
                    ├── "auto_approve" → 自动通过
                    └── 超时 → 自动拒绝
```

**优先级链**：`Session 覆盖 > config.ApprovalPolicy > ApplyDefaults 兜底`

- **ApplyDefaults 兜底**：`DefaultApprovalPolicyMap()` 提供最底层默认值，仅当 `config.ApprovalPolicy` 也未配置时生效
- **config.ApprovalPolicy**：Global + Project 合并后的审批策略
- **Session 覆盖**：用户通过 API 设置的 Session 级别覆盖，优先级最高

### 11.7 断点续传

当 Loop 因达到 `l.cfg.ToolCallLimit` 上限终止时：
1. Session 记录 `LastLoopTerminationReason = "tool_limit_reached"`
2. 下次 `ProcessMessage` 时检测到，自动注入一条 System 消息告知 LLM 从中断点继续
3. 重置 `ToolCallCount` 为 0，开始新一轮 Loop

## 12. 配置体系设计

### 12.1 核心原则

**只有一个配置来源**：`config.Config`（Global + Project 合并），所有行为参数从这里读。

Agent Config 只管身份，不管行为。两个 Config **字段不重叠**。

### 12.2 配置层次

```
┌──────────────────────────────────────────────────────┐
│ config.ApplyDefaults()  ← 代码中的硬编码默认值         │
│   ToolCallLimit=50, MaxContextTokens=128000,          │
│   KeepRecent=30, MaxConcurrentToolCalls=1,            │
│   MaxConcurrentSubprocesses=5,                        │
│   ApprovalPolicy=DefaultApprovalPolicy()              │
└──────────────┬───────────────────────────────────────┘
               │ 被覆盖
               ▼
┌──────────────────────────────────────────────────────┐
│ Global Config (~/.devo/config.json)                   │
│   用户全局设置：LLM keys、模型、审批策略、限制等        │
└──────────────┬───────────────────────────────────────┘
               │ Merge 覆盖
               ▼
┌──────────────────────────────────────────────────────┐
│ Project Config (<project>/.devo/config.json)          │
│   项目级覆盖：skills、MCP、审批策略、限制等            │
└──────────────┬───────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────┐
│ 最终 config.Config  ← LoadFullConfig() 返回           │
│   所有字段已就绪，注入 Loop，运行时直接读              │
└──────────────┬───────────────────────────────────────┘
               │ 运行时覆盖
               ▼
┌──────────────────────────────────────────────────────┐
│ Session 覆盖（仅并发 + 审批）                          │
│   MaxConcurrentToolCalls, MaxConcurrentSubprocesses,  │
│   ApprovalPolicy, ApprovalTimeoutSeconds              │
│   （创建时不写值，用户通过 API 运行时设置）             │
└──────────────────────────────────────────────────────┘
```

### 12.3 config.Config 变更

`config.Config` 新增两个字段，`ApplyDefaults()` 设置所有行为默认值：

```diff
// internal/config/config.go
type Config struct {
    LLM              LLMConfig
    DBPath           string
    LogPath          string
    LogLevel         string
    ApprovalPolicy   map[string]string
    Skills           []string
    MCP              []string
    ToolCallLimit    int
    MaxContextTokens int
    KeepRecent       int
+   MaxConcurrentToolCalls    int
+   MaxConcurrentSubprocesses int
}
```

```diff
// internal/config/config.go
func ApplyDefaults(cfg *Config) {
    // LLM 默认值（已有）
    if cfg.LLM.BaseURL == "" { cfg.LLM.BaseURL = DefaultLLMBaseURL }
    if cfg.LLM.Model == ""   { cfg.LLM.Model = DefaultLLMModel }
    if cfg.LLM.MaxTokens == 0 { cfg.LLM.MaxTokens = DefaultMaxTokens }

+   // 行为默认值（新增）
+   if cfg.ToolCallLimit == 0            { cfg.ToolCallLimit = 50 }
+   if cfg.MaxContextTokens == 0         { cfg.MaxContextTokens = 128000 }
+   if cfg.KeepRecent == 0              { cfg.KeepRecent = 30 }
+   if cfg.MaxConcurrentToolCalls == 0   { cfg.MaxConcurrentToolCalls = 1 }
+   if cfg.MaxConcurrentSubprocesses == 0 { cfg.MaxConcurrentSubprocesses = 5 }
+   if cfg.ApprovalPolicy == nil         { cfg.ApprovalPolicy = DefaultApprovalPolicyMap() }
}
```

`DefaultApprovalPolicyMap()` 提取自现有的 `DefaultApprovalPolicy()`（返回 `map[string]string` 版本）。

### 12.4 运行时读取逻辑

**Loop 中**：直接从 `l.cfg` 读，无需兜底

```go
// 工具调用上限
if sess.ToolCallCount >= l.cfg.ToolCallLimit { ... }

// 上下文压缩
result, err := l.compressor.Compress(ctx, sessionID, eventBus,
    systemPromptTokens, l.cfg.MaxContextTokens, l.cfg.KeepRecent)

// 并发工具执行
if l.cfg.MaxConcurrentToolCalls > 1 && len(lc.LLMResult.ToolCalls) > 1 {
    return l.executeToolsParallel(ctx, lc)
}
```

**Session 覆盖字段**：运行时判断

```go
// 并发数：Session 覆盖优先
maxConcurrent := l.cfg.MaxConcurrentToolCalls
if sess.MaxConcurrentToolCalls > 0 {
    maxConcurrent = sess.MaxConcurrentToolCalls
}
```

**审批策略**：3 层优先级

```
Session 覆盖 > cfg.ApprovalPolicy (Global+Project 合并) > 无（cfg 已保证非 nil）
```

### 12.5 与 Agent Config 的边界

| 配置类型 | 在哪定义 | 包含什么 | 示例 |
|---------|---------|---------|------|
| **Agent Config** | `agent.Config`（代码） | 身份、能力 | `ID="devo-default"`, `SystemPrompt="..."`, `Tools=nil` |
| **行为配置** | `config.Config`（文件） | 限制、策略 | `ToolCallLimit=50`, `ApprovalPolicy={...}` |
| **Session 覆盖** | `session.Session`（运行时） | 临时覆盖 | `MaxConcurrentToolCalls=3` |

**边界清晰，无字段重叠。**

### 12.6 用户视角

用户只需关心两个配置文件：

| 文件 | 位置 | 用途 |
|------|------|------|
| 全局配置 | `~/.devo/config.json` | LLM keys、模型、全局审批策略、全局限制 |
| 项目配置 | `<project>/.devo/config.json` | 项目级覆盖：skills、MCP、审批策略、限制 |

修改后重启生效。Session 运行时的并发数和审批策略可通过 API 动态调整。

---

## 13. REST API 变更影响

### 13.1 CreateSession API

**变更**：不再接受 `tool_call_limit`、`max_context_tokens`、`keep_recent`、`system_prompt_override`。

```json
// 之前
{
    "working_directory": "/path/to/project",
    "system_prompt_override": "...",
    "tool_call_limit": 50,
    "max_context_tokens": 128000,
    "keep_recent": 30
}

// 之后
{
    "working_directory": "/path/to/project",
    "title": "...",
    "agent_id": "devo-default"   // 可选，默认 "devo-default"
}
```

| 移除字段 | 替代方案 |
|---------|---------|
| `system_prompt_override` | 修改 Agent 配置文件的 `system_prompt` |
| `tool_call_limit` | 修改 `~/.devo/config.json` 或 `<project>/.devo/config.json` 的 `tool_call_limit` |
| `max_context_tokens` | 同上 |
| `keep_recent` | 同上 |

### 13.2 GetSession / ListSessions API

**移除的响应字段**：`tool_call_limit`、`max_context_tokens`、`system_prompt_override`

**新增的响应字段**：`agent_id`

### 13.3 UpdateConfig API

`PUT /api/v1/sessions/{id}/config` **精简**：

| 字段 | 变更 |
|------|------|
| `tool_call_limit` | **移除**，修改配置文件 |
| `max_context_tokens` | **移除**，修改配置文件 |
| `keep_recent` | **移除**，修改配置文件 |
| `max_concurrent_tool_calls` | **保留**，运行时调整 |
| `max_concurrent_subprocesses` | **保留**，运行时调整 |

### 13.4 Approval Policy API

**不变**。`PUT /api/v1/sessions/{id}/approval-policy` 和 `POST /api/v1/sessions/{id}/approve/{approval_id}` 继续支持。审批策略优先级：Session > config.ApprovalPolicy > ApplyDefaults 兜底。

### 13.5 Handler 重构

核心变更：`loop` → `agentRegistry`，按 Session.AgentID 路由。

```diff
type Handler struct {
    store         session.SessionStore
-   loop          *agentloop.Loop
+   agentRegistry *agent.Registry
    memoryManager *memory.Manager
    version       string
    // ...
}

func NewHandler(
    store session.SessionStore,
-   loop *agentloop.Loop,
+   agentRegistry *agent.Registry,
    memoryManager *memory.Manager,
    version string,
) *Handler {
    return &Handler{
        store:         store,
-       loop:          loop,
+       agentRegistry: agentRegistry,
        memoryManager: memoryManager,
        version:       version,
    }
}
```

**所有 Handler 方法中的路由模式**：

```go
// 之前：直接调用 h.loop
h.loop.ProcessMessage(ctx, id, msg)
h.loop.Pause(id)
h.loop.Resume(id)

// 之后：按 Session.AgentID 路由
agent := h.agentRegistry.Get(sess.AgentID)
agent.ProcessMessage(ctx, id, msg)
agent.Pause(id)
agent.Resume(id)
```

---

## 14. 前端与 TUI 变更

> **核心结论**：Settings 面板（Web 和 TUI）配置的是 Project Config / Global Config，本次重构不改变这两个 API 的行为，因此 Settings 面板**不受影响**。真正需要变更的只有 Session 类型定义和 Store 中的字段映射。

### 14.1 Web 前端变更

#### 14.1.1 Session 类型定义

**文件**：`web/src/types/session.ts`

当前实际代码：
```typescript
export interface Session {
  id: string
  title: string
  state: SessionState
  workingDirectory: string
  createdAt: string
  lastActiveAt: string
  messageCount: number
  tokenUsage: TokenUsage
  trustLevel: TrustLevel
  approvalPolicy: ApprovalPolicy
  toolCallLimit?: number
  keepRecent?: number
  maxContextTokens?: number
  currentContextTokens?: number
  lastMessageContent?: string
  lastMessageTime?: string
}
```

重构后变更：

```diff
export interface Session {
  id: string
  title: string
  state: SessionState
  workingDirectory: string
  createdAt: string
  lastActiveAt: string
  messageCount: number
  tokenUsage: TokenUsage
  trustLevel: TrustLevel
  approvalPolicy: ApprovalPolicy       // 保留，仅 Session 级覆盖
- toolCallLimit?: number               // 删除，迁移到 config.Config
- keepRecent?: number                  // 删除，迁移到 config.Config
- maxContextTokens?: number            // 删除，迁移到 config.Config
+ agentId: string                      // 新增，标识使用的 Agent
  currentContextTokens?: number        // 保留，运行时状态（SSE 推送）
  lastMessageContent?: string
  lastMessageTime?: string
}
```

**变更说明**：
- `toolCallLimit`、`keepRecent`、`maxContextTokens`：彻底删除。这些值现在由 `config.Config`（Global + Project 合并）管理，不再通过 Session API 返回
- `agentId`：新增，来源为 `data.agent_id`，默认 `"devo-default"`
- `approvalPolicy`：保留，但语义变为"仅 Session 级覆盖"，完整审批策略来自 `config.ApprovalPolicy`
- `currentContextTokens`：保留，运行时状态通过 SSE `token_usage` 事件推送

#### 14.1.2 Session Store

**文件**：`web/src/stores/session.ts`

**`mapSession()` 函数**（当前实际代码 L241-L253）：

```diff
function mapSession(data: any): Session {
  return {
    id: data.id,
    title: data.title || '未命名会话',
    state: data.state || 'idle',
    workingDirectory: data.working_directory || data.project_path || '',
    createdAt: data.created_at || new Date().toISOString(),
    lastActiveAt: data.last_active_at || new Date().toISOString(),
    messageCount: data.message_count || 0,
    tokenUsage: data.token_usage || { input: 0, output: 0 },
    trustLevel: data.trust_level || 'normal',
    approvalPolicy: data.approval_policy || {},
-   toolCallLimit: data.tool_call_limit,
-   keepRecent: data.keep_recent,
-   maxContextTokens: data.max_context_tokens,
+   agentId: data.agent_id || 'devo-default',
    currentContextTokens: data.current_context_tokens,
    lastMessageContent: data.last_message_content,
    lastMessageTime: data.last_message_time,
  }
}
```

**`createSession()` 函数**（当前实际代码 L60-L72）：

```diff
const session: Session = {
  // ...
  trustLevel: data.trust_level || 'normal',
  approvalPolicy: data.approval_policy || {},
- toolCallLimit: data.tool_call_limit,
- keepRecent: data.keep_recent,
- maxContextTokens: data.max_context_tokens,
+ agentId: data.agent_id || 'devo-default',
  currentContextTokens: data.current_context_tokens,
}
```

**`updateTokenUsage()` 函数**：**不受影响**。`currentContextTokens` 更新逻辑保留。

#### 14.1.3 Settings 面板

**不受影响**。`SettingsPanelController.ts` 中的 `tool_call_limit`、`max_context_tokens`、`keep_recent` 字段读取自 **Project Config API**（`GET /api/v1/project/config`）和 **Global Config API**（`GET /api/v1/global/config`），而非 Session API。本次重构不改变这两个配置 API。

同样，`saveProjectParams()` 和 `saveGlobalParams()` 写入的也是这两个 API，不受影响。

#### 14.1.4 Session 列表

**不受影响**。`listSessionsItem` 响应中不包含 `tool_call_limit`、`max_context_tokens`、`keep_recent`，当前已经是干净的。

#### 14.1.5 审批策略

**不受影响**。`setApprovalPolicy()` 调用 `PUT /api/v1/sessions/{id}/approval-policy`，这是 Session 级别的审批策略覆盖 API，保持不变。

#### 14.1.6 上下文 Token 显示

**不受影响**。`InputAreaController.ts`（L90）和 `MobileInputBarController.ts`（L55）通过 `sessionStore.currentSession?.currentContextTokens` 读取上下文 Token 数，此字段保留在 Session 中，通过 SSE 推送更新。

#### 14.1.7 测试夹具

**不受影响**。`web/src/test/fixtures/sessions.ts` 中的 `mockSession` 不包含 `toolCallLimit`、`keepRecent`、`maxContextTokens` 字段，无需修改。

### 14.2 TUI 变更

#### 14.2.1 SessionInfo 类型

**不受影响**。`internal/interfaces/tui/types/session.go` 中的 `SessionInfo` 结构体（当前实际代码）：

```go
type SessionInfo struct {
    ID                   string       `json:"id"`
    Title                string       `json:"title"`
    State                SessionState `json:"state"`
    WorkingDirectory     string       `json:"working_directory"`
    CreatedAt            string       `json:"created_at"`
    LastActiveAt         string       `json:"last_active_at"`
    MessageCount         int          `json:"message_count"`
    TokenUsage           TokenUsage   `json:"token_usage"`
    TrustLevel           TrustLevel   `json:"trust_level"`
    CurrentContextTokens int          `json:"current_context_tokens"`
    LastMessageContent   string       `json:"last_message_content"`
    LastMessageTime      string       `json:"last_message_time"`
}
```

结构体已不包含 `ToolCallLimit`、`MaxContextTokens`、`KeepRecent` 等字段，当前已经是干净的。需要新增 `AgentID` 字段。

#### 14.2.2 UpdateConfigRequest

**文件**：`internal/interfaces/tui/types/approval.go`

当前实际代码：
```go
type UpdateConfigRequest struct {
    ToolCallLimit int `json:"tool_call_limit,omitempty"`
}
```

```diff
type UpdateConfigRequest struct {
-   ToolCallLimit int `json:"tool_call_limit,omitempty"`
+   MaxConcurrentToolCalls    *int `json:"max_concurrent_tool_calls,omitempty"`
+   MaxConcurrentSubprocesses *int `json:"max_concurrent_subprocesses,omitempty"`
}
```

变更说明：`PUT /api/v1/sessions/{id}/config` API 移除了 `tool_call_limit`、`max_context_tokens`、`keep_recent`，仅保留并发控制字段。

#### 14.2.3 Settings 设置页

**不受影响**。`internal/interfaces/tui/overlays/settings.go` 中的 `SettingsPanel` 操作的是 `ProjectConfigInfo` 和 `GlobalConfigInfo`（通过 `api/client.go` 的 `GetProjectConfig()` / `GetGlobalConfig()` 获取），而非 Session 字段。这些配置 API 保持不变。

#### 14.2.4 API Client

**不受影响**。`internal/interfaces/tui/api/client.go` 中的 `ProjectConfigInfo` 和 `GlobalConfigInfo` 结构体：

```go
type ProjectConfigInfo struct {
    Skills           []string       `json:"skills"`
    MCP              []string       `json:"mcp"`
    ApprovalPolicy   ApprovalPolicy `json:"approval_policy"`
    ToolCallLimit    *int           `json:"tool_call_limit"`
    MaxContextTokens *int           `json:"max_context_tokens"`
    KeepRecent       *int           `json:"keep_recent"`
}

type GlobalConfigInfo struct {
    ToolCallLimit    *int            `json:"tool_call_limit"`
    MaxContextTokens *int            `json:"max_context_tokens"`
    KeepRecent       *int            `json:"keep_recent"`
    LLM              *LLMConfigInfo  `json:"llm"`
    ApprovalPolicy   ApprovalPolicy  `json:"approval_policy"`
}
```

这些结构体对应的是 Project/Global Config API 的响应，**不受本次重构影响**。配置合并（Global + Project → `config.Config`）发生在服务端，客户端无感知。

### 14.3 变更影响总览

| 组件 | 需要变更 | 变更内容 |
|------|---------|---------|
| **Web** `types/session.ts` | ✅ 是 | 移除 `toolCallLimit`、`keepRecent`、`maxContextTokens`；新增 `agentId` |
| **Web** `stores/session.ts` | ✅ 是 | `mapSession()` 和 `createSession()` 中移除旧字段映射，新增 `agentId` |
| **Web** `SettingsPanelController.ts` | ❌ 否 | 使用 Project/Global Config API，不受影响 |
| **Web** `InputAreaController.ts` | ❌ 否 | 只读 `currentContextTokens`，保留 |
| **Web** `MobileInputBarController.ts` | ❌ 否 | 只读 `currentContextTokens`，保留 |
| **Web** `AppController.ts` | ❌ 否 | `currentContextTokens` 通过 SSE 推送，保留 |
| **Web** 审批策略 | ❌ 否 | Session 级审批策略 API 保持不变 |
| **Web** 测试夹具 | ❌ 否 | 已不包含这些字段 |
| **TUI** `types/session.go` | ✅ 是 | 新增 `AgentID` 字段 |
| **TUI** `types/approval.go` | ✅ 是 | `UpdateConfigRequest` 移除 `ToolCallLimit`，替换为并发字段 |
| **TUI** `overlays/settings.go` | ❌ 否 | 使用 Project/Global Config，不受影响 |
| **TUI** `api/client.go` | ❌ 否 | `ProjectConfigInfo`/`GlobalConfigInfo` 保持不变 |

---

### 14.4 调用链全景对比（故事版）

> 以下用一次真实的用户消息请求，展示重构前后完整调用链的差异。每个节点标注了实际文件路径和行号，方便对照代码。

---

#### 14.4.1 故事：用户发送 "帮我写一个 Python 爬虫"

**用户操作**：在 Web UI 输入框键入 "帮我写一个 Python 爬虫"，按 Enter。

---

#### 14.4.2 重构前调用链（当前代码）

```
┌─ 阶段一：应用启动 ─────────────────────────────────────────────────────┐
│                                                                          │
│  [.\internal\cli\app.go:51] NewApp()                 │
│    │                                                                     │
│    ├─ [app.go:64] config.LoadFullConfig(wd) → cfg                        │
│    │   └─ [config.go:70] LoadGlobal() → 读 ~/.devo/config.json           │
│    │   └─ [config.go:77] LoadProjectConfig(wd) → 读 ./.devo/config.json  │
│    │   └─ [config.go:79] Merge(global, project) → cfg                    │
│    │   └─ [config.go:82] applyDefaults(cfg)                              │
│    │                                                                     │
│    ├─ [app.go:163] app.initLLM()                                         │
│    │   └─ [app.go:164] app.loop = agentloop.NewWithTools(store, llm,     │
│    │       registry)                                                     │
│    │       └─ [loop.go:73] prompt.NewAssembler()                         │
│    │           └─ [assembler.go:11] defaultBasePrompt = "You are Devo..."│
│    │               ← 硬编码 System Prompt，不可配置                       │
│    │       └─ [loop.go:74] approval.NewManager()                         │
│    │           ← 审批管理器无 userGlobalPolicy，待后续 SetXxx 注入        │
│    │                                                                     │
│    ├─ [app.go:167] app.initMemory()                                      │
│    │   └─ [app.go:173] a.loop.SetMemoryManager(memoryMgr)  ← ※ 补丁1    │
│    │                                                                     │
│    ├─ [app.go:178] app.initSkills()                                      │
│    │   └─ [app.go:186] a.loop.SetSkillsManager(skillsMgr)  ← ※ 补丁2    │
│    │                                                                     │
│    ├─ [app.go:147] app.initTools()                                       │
│    │   └─ [app.go:150] a.loop.SetBackgroundProcessManager(bg) ← ※ 补丁3 │
│    │                                                                     │
│    ├─ [app.go:202] app.initHandler()                                     │
│    │   └─ [handler.go:33] rest.NewHandler(store, a.loop, ...)            │
│    │       ← Handler 持有 *agentloop.Loop，没有 Agent 层                  │
│    │                                                                     │
│    └─ [app.go:219] app.loadUserApprovalPolicy()                          │
│        └─ [handler.go] h.LoadUserApprovalPolicy()                        │
│            └─ approvalManager.SetUserGlobalPolicy(policy) ← ※ 补丁4      │
│              ← 审批策略通过 SetXxx 注入，共 5 层                          │
│                                                                          │
└─ 阶段一结束 ────────────────────────────────────────────────────────────┘

┌─ 阶段二：用户创建 Session ──────────────────────────────────────────────┐
│                                                                          │
│  HTTP POST /api/v1/sessions                                              │
│  → [.\internal\interfaces\rest\session_handler.go:44]│
│    Handler.CreateSession()                                               │
│    │                                                                     │
│    ├─ 读请求体：                                                          │
│    │   req.ToolCallLimit, req.MaxContextTokens, req.KeepRecent            │
│    │   req.SystemPromptOverride  ← 6 个配置字段混在请求体中               │
│    │                                                                     │
│    ├─ [session_handler.go:77] cfg, _ := config.LoadFullConfig(workingDir)│
│    │   ← 再次加载配置（NewApp 时已加载过）                                │
│    │                                                                     │
│    ├─ [session_handler.go:78-88] 配置传递：                               │
│    │   if toolCallLimit <= 0 { toolCallLimit = cfg.ToolCallLimit }        │
│    │   if maxContextTokens <= 0 { maxContextTokens = cfg.MaxContextTokens }│
│    │   if keepRecent <= 0 { keepRecent = cfg.KeepRecent }                │
│    │                                                                     │
│    ├─ [session_handler.go:90-95] 终极兜底：                               │
│    │   if toolCallLimit <= 0 { toolCallLimit = 50 }        ← ※ 补丁5     │
│    │   if maxContextTokens <= 0 { maxContextTokens = 128000 } ← ※ 补丁6  │
│    │   if keepRecent <= 0 { keepRecent = 30 }              ← ※ 补丁7     │
│    │                                                                     │
│    ├─ [session_handler.go:97-113] 创建 Session：                          │
│    │   sess := &session.Session{                                         │
│    │       ToolCallLimit:          toolCallLimit,     ← 存到 Session      │
│    │       MaxContextTokens:       maxContextTokens,  ← 存到 Session      │
│    │       KeepRecent:             keepRecent,        ← 存到 Session      │
│    │       SystemPromptOverride:   req.SystemPromptOverride,              │
│    │       ApprovalPolicy:         make(map[string]string),               │
│    │       ...                                                           │
│    │   }                                                                 │
│    │                                                                     │
│    └─ 返回 createSessionResponse{                                        │
│        ToolCallLimit:  sess.ToolCallLimit,          ← 暴露给前端          │
│        MaxContextTokens: sess.MaxContextTokens,     ← 暴露给前端          │
│        KeepRecent:       sess.KeepRecent,           ← 暴露给前端          │
│    }                                                                     │
│                                                                          │
└─ 阶段二结束 ────────────────────────────────────────────────────────────┘

┌─ 阶段三：用户发送消息 → 进入 Loop ──────────────────────────────────────┐
│                                                                          │
│  HTTP POST /api/v1/sessions/{id}/messages                                │
│  → [.\internal\interfaces\rest\message_handler.go:20]│
│    Handler.PostMessage()                                                 │
│    │                                                                     │
│    └─ [message_handler.go:68] h.loop.ProcessMessage(ctx, id, msg)        │
│        ← 直接调用 Loop，无 Agent 层                                       │
│        │                                                                 │
│        └─ [.\internal\core\agentloop\loop.go:109]    │
│          Loop.ProcessMessage()                                           │
│          │                                                               │
│          ├─ [loop.go:110] sess, _ := l.store.Get(sessionID)              │
│          │                                                               │
│          ├─ 阶段 3a: System Prompt 组装                                   │
│          │   └─ [assembler.go:105]                                       │
│          │       if sess.SystemPromptOverride != "" {                     │
│          │           prompt += sess.SystemPromptOverride                  │
│          │       }                                                       │
│          │       ← 从 Session 读 SystemPromptOverride，非 Agent 配置     │
│          │                                                               │
│          ├─ 阶段 3b: 工具调用计数与限制                                    │
│          │   └─ [helpers.go:38] incrementToolCallCount()                 │
│          │       │                                                       │
│          │       ├─ [helpers.go:43] sess.ToolCallCount++                 │
│          │       │                                                       │
│          │       ├─ [helpers.go:44] if sess.ToolCallLimit <= 0 {         │
│          │       │       sess.ToolCallLimit = 50  ← ※ 补丁8：运行时兜底  │
│          │       │   }                                                   │
│          │       │                                                       │
│          │       └─ [helpers.go:48] if sess.ToolCallCount >=             │
│          │               sess.ToolCallLimit { stop }                     │
│          │           ← 从 Session 读 ToolCallLimit，非统一配置            │
│          │                                                               │
│          ├─ 阶段 3c: 上下文压缩                                            │
│          │   └─ [compressor.go:97] sess.MaxContextTokens                 │
│          │       └─ [compressor.go:98] if maxContext <= 0 {              │
│          │               maxContext = 128000  ← ※ 补丁9：运行时兜底       │
│          │           }                                                   │
│          │       └─ [compressor.go:109] sess.KeepRecent                  │
│          │       └─ [compressor.go:110] if keepRecent <= 0 {             │
│          │               keepRecent = 30  ← ※ 补丁10：运行时兜底          │
│          │           }                                                   │
│          │       ← 从 Session 读压缩参数，非统一配置                       │
│          │                                                               │
│          └─ 阶段 3d: 审批策略解析（5 层）                                  │
│              └─ [approval_handler.go:75] resolveEffectivePolicy()        │
│                  │                                                       │
│                  ├─ [approval_handler.go:77] 层1: sess.ApprovalPolicy     │
│                  │   ← Session 级覆盖                                     │
│                  │                                                       │
│                  ├─ [approval_handler.go:82] 层2:                        │
│                  │   config.LoadProjectConfig(sess.WorkingDirectory)     │
│                  │   ← 每次调用都重新加载项目配置！IO 浪费                  │
│                  │                                                       │
│                  ├─ [approval_handler.go:90] 层3:                        │
│                  │   approvalManager.ResolveEffectivePolicy(             │
│                  │       sessionPolicy, projectPolicy, opType)           │
│                  │   │                                                   │
│                  │   ├─ [approval.go:291] 层3: m.userGlobalPolicy        │
│                  │   │   ← 通过 SetUserGlobalPolicy() 注入，非构造时传入  │
│                  │   │                                                   │
│                  │   ├─ [approval.go:299] 层4:                           │
│                  │   │   m.userPolicyStore.GetFullTrust(opType)          │
│                  │   │   ← FullTrust 机制，可通过 API 设置               │
│                  │   │                                                   │
│                  │   └─ [approval.go:307] 层5: DefaultApprovalPolicy()   │
│                  │       ← 终极兜底                                       │
│                  │                                                       │
│                  └─ 返回最终策略                                          │
│                                                                          │
└─ 阶段三结束 ────────────────────────────────────────────────────────────┘
```

**统计：共 10 处补丁/问题，5 层审批链，3 次配置重复加载。**

---

#### 14.4.3 重构后调用链（目标架构）

```
┌─ 阶段一：应用启动 ─────────────────────────────────────────────────────┐
│                                                                          │
│  [.\internal\cli\app.go] NewApp()                    │
│    │                                                                     │
│    ├─ config.LoadFullConfig(wd) → cfg                                    │
│    │   └─ [config.go:70] LoadGlobal()                                    │
│    │   └─ [config.go:77] LoadProjectConfig(wd)                           │
│    │   └─ [config.go:79] Merge(global, project) → cfg                    │
│    │   └─ [config.go:82] applyDefaults(cfg)                              │
│    │       └─ ToolCallLimit=50, MaxContextTokens=128000, KeepRecent=30   │
│    │       └─ ApprovalPolicy 已填充默认值                                 │
│    │   ← cfg 已保证所有行为参数非零，下游无需再兜底                       │
│    │                                                                     │
│    ├─ 构造所有依赖（一次性完成）：                                         │
│    │   approvalMgr = approval.NewManager()                               │
│    │   memoryMgr = memory.NewManager(...)                                │
│    │   skillsMgr = skills.NewManager(...)                                │
│    │   bgProcManager = tools.NewBackgroundProcessManager()               │
│    │   mcpMgr = mcp.NewManager(wd)                                       │
│    │   solidifier = skills.NewSolidifier(...)                            │
│    │                                                                     │
│    ├─ 创建默认 Agent：                                                   │
│    │   defaultAgent = agent.Default(store, llm, toolRegistry, cfg,        │
│    │       approvalMgr, memoryMgr, skillsMgr, bgProcManager,              │
│    │       mcpMgr, solidifier)                                            │
│    │   └─ [agent.go] agent.loop = agentloop.New(                          │
│    │           store, llm, registry, cfg,                                 │
│    │           approvalMgr, memoryMgr, skillsMgr,                         │
│    │           bgProcManager, mcpMgr, solidifier,                         │
│    │       )                                                              │
│    │       └─ [loop.go] l.cfg = cfg  ← Loop 持有 merged config            │
│    │       └─ [loop.go] l.promptAssembler =                               │
│    │           prompt.NewAssembler(agent.Config.SystemPrompt)             │
│    │           ← SystemPrompt 注入，不再硬编码                             │
│    │       └─ 所有依赖在构造时注入，零 SetXxx                              │
│    │                                                                      │
│    ├─ 创建 Registry（管理所有 Agent）：                                    │
│    │   agentRegistry = agent.NewRegistry(defaultAgent)                    │
│    │   ← Registry 持有 defaultAgent，按 AgentID 路由                      │
│    │   ← 可选：agentRegistry.Register(anotherAgent)                       │
│    │                                                                      │
│    └─ handler = rest.NewHandler(store, agentRegistry, ...)                │
│        ← Handler 持有 *agent.Registry，不再绑定单一 Agent                  │
│                                                                           │
└─ 阶段一结束：零补丁，Registry 路由，一次注入完成 ──────────────────────────┘

┌─ 阶段二：用户创建 Session ──────────────────────────────────────────────┐
│                                                                          │
│  HTTP POST /api/v1/sessions                                              │
│  → [.\internal\interfaces\rest\session_handler.go]   │
│    Handler.CreateSession()                                               │
│    │                                                                     │
│    ├─ 读请求体：                                                          │
│    │   req.WorkingDirectory, req.Title, req.AgentID                       │
│    │   ← ToolCallLimit/MaxContextTokens/KeepRecent 已从请求体删除        │
│    │   ← SystemPromptOverride 已删除（SystemPrompt 归 Agent 管）         │
│    │                                                                     │
│    ├─ 验证 AgentID：                                                      │
│    │   agentID := req.AgentID                                            │
│    │   if agentID == "" { agentID = "devo-default" }                     │
│    │   if _, ok := h.agentRegistry.Get(agentID); ok == nil {             │
│    │       return error("unknown agent")                                 │
│    │   }                                                                 │
│    │   ← AgentID 在 Registry 中验证，确保 Agent 存在                     │
│    │                                                                     │
│    ├─ 不调用 config.LoadFullConfig() ← 不再重复加载配置                  │
│    │                                                                     │
│    ├─ 不写 if xx <= 0 兜底 ← cfg 已保证非零                              │
│    │                                                                     │
│    ├─ 创建 Session：                                                      │
│    │   sess := &session.Session{                                         │
│    │       ID:               session.GenerateID("sess"),                 │
│    │       Title:            title,                                      │
│    │       WorkingDirectory: workingDir,                                 │
│    │       AgentID:          "devo-default",  ← 新增                     │
│    │       ApprovalPolicy:   make(map[string]string), ← 仅 Session 覆盖  │
│    │       ...                                                           │
│    │   }                                                                 │
│    │   ← 无 ToolCallLimit, MaxContextTokens, KeepRecent,                 │
│    │     SystemPromptOverride 字段                                        │
│    │                                                                     │
│    └─ 返回 createSessionResponse{                                        │
│        AgentID: sess.AgentID,  ← 新增                                    │
│        ...                                                               │
│    }                                                                     │
│    ← 无 tool_call_limit, max_context_tokens, keep_recent 字段            │
│                                                                          │
└─ 阶段二结束：Session 瘦身完成 ───────────────────────────────────────────┘

┌─ 阶段三：用户发送消息 → 进入 Loop ──────────────────────────────────────┐
│                                                                          │
│  HTTP POST /api/v1/sessions/{id}/messages                                │
│  → [.\internal\interfaces\rest\message_handler.go]   │
│    Handler.PostMessage()                                                 │
│    │                                                                     │
│    └─ agent := h.agentRegistry.Get(sess.AgentID)  ← 按 AgentID 路由     │
│        │                                                                 │
│        └─ agent.ProcessMessage(ctx, id, msg)                             │
│              │                                                           │
│              └─ [.\internal\core\agentloop\loop.go]  │
│                Loop.ProcessMessage()                                     │
│                │                                                         │
│                ├─ 阶段 3a: System Prompt 组装                             │
│                │   └─ l.promptAssembler.Assemble(sess)                   │
│                │       ← 使用 Agent 注入的 SystemPrompt，不再读 Session   │
│                │       ← 不再有硬编码的 defaultBasePrompt                 │
│                │                                                         │
│                ├─ 阶段 3b: 工具调用计数与限制                              │
│                │   └─ incrementToolCallCount()                           │
│                │       │                                                 │
│                │       ├─ sess.ToolCallCount++                           │
│                │       │                                                 │
│                │       ├─ 不再需要 "if sess.ToolCallLimit <= 0" 兜底     │
│                │       │   ← l.cfg.ToolCallLimit 已由 ApplyDefaults 保证  │
│                │       │                                                 │
│                │       └─ if sess.ToolCallCount >= l.cfg.ToolCallLimit { │
│                │               stop                                     │
│                │           }                                             │
│                │           ← 直接从 l.cfg 读，统一配置源                  │
│                │                                                         │
│                ├─ 阶段 3c: 上下文压缩                                      │
│                │   └─ maxContext := l.cfg.MaxContextTokens               │
│                │       ← 不再需要 "if <= 0" 兜底                          │
│                │       └─ keepRecent := l.cfg.KeepRecent                 │
│                │           ← 不再需要 "if <= 0" 兜底                      │
│                │           ← 直接从 l.cfg 读，统一配置源                  │
│                │                                                         │
│                └─ 阶段 3d: 审批策略解析（3 层）                            │
│                    └─ resolveEffectivePolicy(sess, opType)               │
│                        │                                                 │
│                        ├─ 层1: if sess.ApprovalPolicy != nil {           │
│                        │          return sess.ApprovalPolicy[opType]     │
│                        │       }                                         │
│                        │       ← Session 级覆盖                           │
│                        │                                                 │
│                        ├─ 层2: if l.cfg.ApprovalPolicy != nil {          │
│                        │          return l.cfg.ApprovalPolicy[opType]    │
│                        │       }                                         │
│                        │       ← 直接从 cfg 读，已 merge Global+Project   │
│                        │       ← 不再每次调用 LoadProjectConfig()        │
│                        │                                                 │
│                        └─ 层3: return approval.DefaultApprovalPolicy()   │
│                                ← cfg 已保证非 nil，此分支理论上不触发     │
│                        ← 不再有 UserGlobalPolicy / FullTrust             │
│                                                                          │
└─ 阶段三结束：零兜底，3 层审批，配置单源 ─────────────────────────────────┘
```

**统计：零补丁，3 层审批链，配置只加载一次。**

---

#### 14.4.4 关键差异对比

| 维度 | 重构前（当前代码） | 重构后（目标架构） |
|------|-------------------|-------------------|
| **配置加载** | `LoadFullConfig` 在 NewApp 和 CreateSession 各调一次 | 只在 NewApp 调用一次，注入给 Agent/Loop |
| **配置读取** | Loop 从 `sess.ToolCallLimit` 等 Session 字段读 | Loop 从 `l.cfg.ToolCallLimit` 直接读 |
| **兜底代码** | `if <= 0` 兜底遍布 `helpers.go`、`compressor.go`、`session_handler.go` | 零兜底，`ApplyDefaults` 已保证非零 |
| **依赖注入** | 4 个 `SetXxx` 方法（`SetMemoryManager`、`SetSkillsManager`、`SetBackgroundProcessManager`、`SetSolidifier`） | 全部在构造时注入，零 `SetXxx` |
| **审批链** | 5 层：Session → ProjectConfig → UserGlobalPolicy → FullTrust → Default | 3 层：Session → `l.cfg.ApprovalPolicy` → Default |
| **审批配置加载** | `resolveEffectivePolicy` 内每次调用 `config.LoadProjectConfig()` | 无 IO，直接从 `l.cfg.ApprovalPolicy` 读 |
| **System Prompt** | `assembler.go` 硬编码 `defaultBasePrompt` | Agent Config 注入 |
| **Session 请求体** | 6 个配置字段（`tool_call_limit`、`max_context_tokens`、`keep_recent`、`system_prompt_override` 等） | 仅 `working_directory`、`title` |
| **Session 响应体** | 返回 `tool_call_limit`、`max_context_tokens`、`keep_recent` | 返回 `agent_id` |
| **Handler 持有** | `*agentloop.Loop` | `*agent.Agent` |
| **UpdateConfig API** | `PUT /api/v1/sessions/{id}/config` 修改 Session 字段 | 删除（Session 不再存配置默认值） |

---

## 15. 不做的事情

明确列出本设计**不包含**的内容，避免过度设计：

| 不做 | 原因 |
|------|------|
| Agent CRUD API | 当前只有 `devo-default`，无需管理 |
| Agent 数据库表 | 只有一个 Agent，配置在代码中定义即可 |
| Agent 选择 UI | 只有一个 Agent，无需选择 |
| 多 Agent 切换 | 当前不需要，但架构已为此预留空间 |
| Agent 绑定独立模型 | 模型选择是全局配置，不是 Agent 级别。当前项目已实现全局多模型管理（`/api/v1/global/config/models`），Agent 不介入模型选择 |
| `AgentManager` | 只有一个 Agent，不需要管理 |
| 自定义 Agent | 用户无法创建新 Agent |
| Agent 配置文件 | 配置在代码中，不需要外部文件。如需自定义，通过 Project Config 覆盖 |

这些能力可以在未来需要时，基于已有的 `Agent` 结构体扩展——每个 Agent 创建自己的 Loop 实例即可，架构不需要大改。

---

## 16. 风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| Session 字段移除导致序列化不兼容 | 前端/API 可能依赖旧字段 | 前端同步适配，彻底移除字段，不保留兼容层 |
| SQLite 迁移失败 | 旧 Session 无法加载 | 迁移脚本加 `IF NOT EXISTS`，旧列保留但不再写入，新 Session 不创建旧列 |
| 测试用例大量失败 | 测试中直接构造 Session 设置了已移除字段 | 逐文件修复，改为从 config.Config 读取 |
| 审批策略个性化丢失 | 用户可能在 Session 级别覆盖了审批策略 | 保留 Session 级别审批策略覆盖能力，优先级最高 |
| Session 创建时 `tool_call_limit` 等字段被移除 | 依赖旧 API 的客户端可能报错 | 前端同步适配 |
| `UpdateConfig` 移除字段导致功能退化 | 用户无法动态调整上下文窗口 | 引导用户通过配置文件调整 |

---

## 17. 补充细节（v1.5.0 新增）

### 17.1 `SessionModel`（SQLite）变更

当前 `SessionModel` 中存在以下问题需要一并修复：

**问题 1**：`MaxConcurrentToolCalls` 和 `MaxConcurrentSubprocesses` 在 `Session` 结构体中存在，但 `SessionModel` 中没有对应的数据库列，导致这两个字段无法持久化。需要在 `SessionModel` 中新增这两列。

**问题 2**：需要新增 `AgentID` 列。

```diff
// internal/storage/sqlite/models.go

type SessionModel struct {
    // ... 现有字段 ...
+   AgentID                   string `gorm:"size:64;default:devo-default"`
+   MaxConcurrentToolCalls    int    `gorm:"default:0"`
+   MaxConcurrentSubprocesses int    `gorm:"default:0"`
}
```

`ToDomain()` 和 `fromDomain()` 也需要同步更新：

```diff
func (m *SessionModel) ToDomain() *session.Session {
    sess := &session.Session{
        // ...
+       AgentID:                m.AgentID,
+       MaxConcurrentToolCalls:    m.MaxConcurrentToolCalls,
+       MaxConcurrentSubprocesses: m.MaxConcurrentSubprocesses,
    }
    // ...
}

func fromDomain(s *session.Session) *SessionModel {
    model := &SessionModel{
        // ...
+       AgentID:                   s.AgentID,
+       MaxConcurrentToolCalls:    s.MaxConcurrentToolCalls,
+       MaxConcurrentSubprocesses: s.MaxConcurrentSubprocesses,
    }
    // ...
}
```

**数据迁移**：旧 Session 的 `AgentID` 默认为 `"devo-default"`，`MaxConcurrentToolCalls` 和 `MaxConcurrentSubprocesses` 默认为 0（运行时从 Agent 读取默认值）。

### 17.2 `approval.Manager` 审批策略集成

当前 `ResolveEffectivePolicy` 的兜底值是 `DefaultApprovalPolicy()`（硬编码）。引入新架构后，兜底值来自 `config.ApplyDefaults()` 中的 `DefaultApprovalPolicyMap()`。

运行时优先级链（3 层）：

```
Session 覆盖 > config.ApprovalPolicy (Global+Project 合并) > ApplyDefaults 兜底
```

Loop 中调用时传入 `l.cfg.ApprovalPolicy`：

```go
func (l *Loop) resolveEffectivePolicy(sess *session.Session, opType approval.OperationType) approval.PolicyLevel {
    // 1. Session 覆盖
    if sess.ApprovalPolicy != nil {
        if policy, ok := sess.ApprovalPolicy[opType]; ok {
            return policy
        }
    }
    // 2. config.ApprovalPolicy（Global+Project 合并后的值）
    if l.cfg.ApprovalPolicy != nil {
        if policy, ok := l.cfg.ApprovalPolicy[opType]; ok {
            return policy
        }
    }
    // 3. DefaultApprovalPolicy 兜底（cfg 已保证非 nil，此分支理论上不会触发）
    return approval.PolicyAlwaysAsk
}
```

### 17.3 `loadUserApprovalPolicy` 移除

当前 `app.go` 中 `loadUserApprovalPolicy()` 将用户全局审批策略加载到 `approval.Manager` 的 `userGlobalPolicy` 中。新架构下，审批策略已通过 `config.LoadFullConfig()` 合并到 `config.Config.ApprovalPolicy` 中，`loadUserApprovalPolicy()` 和 `SetUserGlobalPolicy` 可以移除。

### 17.4 `project_config_handler.go` 变更

`GetProjectConfig` 和 `SetProjectConfig` 中读取/写入的 `ToolCallLimit`、`MaxContextTokens`、`KeepRecent` 字段**不受影响**。这些字段是 Project Config 级别的配置，与 Global Config 合并后形成最终 `config.Config`。API 行为不变。

但响应含义需要澄清：`GetProjectConfig` 返回的 `tool_call_limit`、`max_context_tokens`、`keep_recent` 是 Project Config 中显式设置的值。如果未设置（为 0），前端应理解为"使用 Global Config 或系统默认值"。

### 17.5 `state_handlers.go` 中并发控制的读取方式

改为从 `l.cfg` 读取，Session 覆盖优先：

```go
// 变更后
maxConcurrent := l.cfg.MaxConcurrentToolCalls
if sess.MaxConcurrentToolCalls > 0 {
    maxConcurrent = sess.MaxConcurrentToolCalls
}
if maxConcurrent > 1 && len(lc.LLMResult.ToolCalls) > 1 {
    return l.executeToolsParallel(ctx, lc, maxConcurrent)
}
```

### 17.6 `helpers.go` 中工具调用上限的变更

改为从 `l.cfg` 读取，`l.cfg.ToolCallLimit` 已有 `ApplyDefaults()` 保证非零：

```go
// 变更后
if sess.ToolCallCount >= l.cfg.ToolCallLimit {
    // 达到上限
}
```

### 17.7 `compressor.go` 中配置读取的变更

改为从 `l.cfg` 读取，通过参数传入：

```go
// compressor.go
func (c *Compressor) Compress(ctx context.Context, sessionID string, eventBus *session.EventBus,
    systemPromptTokens int, maxContextTokens int, keepRecent int) (*CompressResult, error) {
    // maxContextTokens 和 keepRecent 直接使用传入值，无需兜底
}

// Loop 调用时
result, err := l.compressor.Compress(ctx, lc.SessionID, lc.EventBus,
    systemPromptTokens, l.cfg.MaxContextTokens, l.cfg.KeepRecent)
```

### 17.8 `createSessionRequest` 中 `tool_call_limit` 等字段的处理

`POST /api/v1/sessions` 不再接受 `tool_call_limit`、`max_context_tokens`、`keep_recent`。这些值由 `config.Config` 管理，创建 Session 时不需要传入。

### 17.9 配置传递路径总结

**一条路径，无中间环节**：

```
config.LoadFullConfig() → Merge(Global, Project) → ApplyDefaults()
    → 得到 config.Config（所有字段已就绪）
    → 注入 Loop
    → 运行时直接读 l.cfg.XXX，零兜底
```

因为 `l.cfg` 已经是 Global + Project 合并后的最终配置，不存在"ProjectConfig 覆盖值无法传递"的问题。

---

## 附录 A：Session 字段迁移对照表

| 当前 Session 字段 | 类型 | 迁移目标 | 备注 |
|-------------------|------|---------|------|
| `ToolCallLimit` | `int` | `config.Config.ToolCallLimit` | ApplyDefaults 设 50，Session 不再存储 |
| `MaxContextTokens` | `int` | `config.Config.MaxContextTokens` | ApplyDefaults 设 128000，Session 不再存储 |
| `KeepRecent` | `int` | `config.Config.KeepRecent` | ApplyDefaults 设 30，Session 不再存储 |
| `SystemPromptOverride` | `string` | **删除** | Agent.Config.SystemPrompt 替代 |
| `MaxConcurrentToolCalls` | `int` | `config.Config.MaxConcurrentToolCalls` | ApplyDefaults 设 1，**保留 Session 字段**用于覆盖 |
| `MaxConcurrentSubprocesses` | `int` | `config.Config.MaxConcurrentSubprocesses` | ApplyDefaults 设 5，**保留 Session 字段**用于覆盖 |
| `ApprovalPolicy` | `map[string]string` | `config.Config.ApprovalPolicy` | 默认从 ApplyDefaults，**保留 Session 字段**仅存覆盖值 |
| `ToolCallCount` | `int` | **保留在 Session** | 运行时状态 |
| `MessageCount` | `int` | **保留在 Session** | 统计计数 |
| `LastLoopTerminationReason` | `string` | **保留在 Session** | 运行时状态 |
| `TokenUsage` | `TokenUsage` | **保留在 Session** | 累计统计 |
| `CompressionCount` | `int` | **保留在 Session** | 统计计数 |
| `CurrentContextTokens` | `int` | **保留在 Session** | 运行时估算 |
| `ActiveSSEConnections` | `int` | **保留在 Session** | 运行时状态 |
| `CancelRequested` | `bool` | **保留在 Session** | 控制标志 |
| `PauseRequested` | `bool` | **保留在 Session** | 控制标志 |
| `ChildPID` | `*int` | **保留在 Session** | 进程管理 |
| `BackgroundPIDs` | `[]int` | **保留在 Session** | 进程管理 |
| `ArchivePath` | `string` | **保留在 Session** | 归档路径 |
| `TrustLevel` | `string` | **保留在 Session** | 信任级别 |
| `ActiveSkills` | `[]string` | **保留在 Session** | 激活技能 |
| `ApprovalTimeoutSeconds` | `int` | **保留在 Session** | Session 级可覆盖 |
| `CachedDirectorySummary` | `*DirectorySummary` | **保留在 Session** | 仅内存缓存 |
| `EventBus` | `*EventBus` | **保留在 Session** | 仅内存 |
| — | `string` | **新增** `AgentID` | 默认 `"devo-default"` |

## 附录 B：Loop 配置读取对照表

Loop 中所有行为参数改为从 `l.cfg`（合并后的 `config.Config`）读取，Session 覆盖优先：

| 文件 | 原代码 | 新代码 |
|------|--------|--------|
| `helpers.go` | `sess.ToolCallLimit` | `l.cfg.ToolCallLimit`（不再从 Session 读） |
| `compressor/compressor.go` | `sess.MaxContextTokens` | 参数传入 `l.cfg.MaxContextTokens` |
| `compressor/compressor.go` | `sess.KeepRecent` | 参数传入 `l.cfg.KeepRecent` |
| `approval_handler.go` | `sess.ApprovalPolicy` | `l.cfg.ApprovalPolicy`（合并 Session 覆盖） |
| `state_handlers.go` | `sess.MaxConcurrentToolCalls` | `l.cfg.MaxConcurrentToolCalls`（Session 覆盖优先） |
| `loop.go` | `sess.ToolCallCount`（断点续传消息） | 保留从 Session 读取（运行时状态） |

---

## 18. 深度分析与重构建议（v1.5.0 新增）

本节跳出当前代码的局限，从"理想架构"的角度审视设计，提出更彻底的改进方向。

### 18.1 当前架构的核心问题

#### 问题一：审批系统层级过多

当前 `ResolveEffectivePolicy` 的优先级链为：

```
Session ApprovalPolicy > Project Config > UserGlobalPolicy > FullTrust > DefaultApprovalPolicy()
```

| 层 | 来源 | 问题 |
|----|------|------|
| Session ApprovalPolicy | `sess.ApprovalPolicy` | Session 级别覆盖，合理 |
| Project Config | `.devo/config.json` | 项目级配置，合理 |
| UserGlobalPolicy | `~/.devo/config.json` → `approval.Manager` | 与 config.ApprovalPolicy 功能重叠 |
| FullTrust | `UserPolicyStore`（无真实实现） | 概念模糊，可用 `auto_approve` 替代 |
| DefaultApprovalPolicy() | 硬编码常量 | 应改为 ApplyDefaults 提供 |

#### 问题二：Session 仍然过重

即使按设计移除了 4 个字段，Session 仍有约 20 个字段，其中混杂了三类不同生命周期的数据：

- **持久标识**：`ID`、`AgentID`、`Title`、`WorkingDirectory`、`CreatedAt`
- **运行时瞬态**：`State`、`ToolCallCount`、`CancelRequested`、`CurrentContextTokens`、`ActiveSSEConnections`、`ChildPID`、`BackgroundPIDs`
- **累计统计**：`TokenUsage`、`CompressionCount`、`MessageCount`

这三类数据每次 `store.Update(sess)` 都会被全量写回 SQLite，但实际上运行时瞬态数据不需要持久化。

#### 问题三：Agent 的转发方法

Agent 对外暴露的方法大部分是直接转发给 Loop：

```go
func (a *Agent) Pause(sessionID string) error   { return a.loop.Pause(sessionID) }
func (a *Agent) Resume(sessionID string) error  { return a.loop.Resume(sessionID) }
```

这些转发方法是纯样板代码。但 Agent 作为"身份 + 入口"的角色是合理的——它对外提供统一的 API 面，内部委托给 Loop 执行。

### 18.2 重构建议

#### 建议一：审批系统精简为 3 层

**目标架构**：

```
Session 覆盖 > config.ApprovalPolicy (Global+Project 合并) > ApplyDefaults 兜底
```

**具体变更**：

1. **移除 `UserGlobalPolicy` 层**——`config.LoadFullConfig()` 已经将 Global 和 Project 的 `ApprovalPolicy` 合并，不需要单独的 `loadUserApprovalPolicy()` → `SetUserGlobalPolicy` 调用链。

2. **移除 `FullTrust` / `UserPolicyStore`**——无真实实现，用户可通过在 `~/.devo/config.json` 中设置 `"file_write_new": "auto_approve"` 达到同样效果。

3. **`TrustLevel` 保留**——`TrustLevel = Elevated` 时跳过所有审批，作为快捷方式保留。

#### 建议二：已解决——配置统一由 `config.Config` 管理

v2.0.0 设计中，Loop 持有 `l.cfg *config.Config`（Global + Project 合并后的最终配置），所有行为参数直接从 `l.cfg` 读取。`ConfigResolver` 不再需要。

#### 建议三：Session 拆分持久/瞬态（可选）

将 Session 拆分为两个结构体，运行时状态不持久化：

```go
type Session struct {
    ID               string
    AgentID          string
    Title            string
    // ... 持久字段 ...
}

type SessionRuntime struct {
    SessionID    string
    ToolCallCount int
    // ... 仅内存字段 ...
}
```

**收益**：减少 SQLite 写入，语义更清晰。此项属于独立优化，与本次 Agent 抽象层重构正交，不纳入本次实施范围。

#### 建议四：Agent 通过接口暴露能力（可选）

```go
type SessionController interface {
    Pause(sessionID string) error
    Resume(sessionID string) error
    // ...
}

type Handler struct {
    agent       *agent.Agent
    sessionCtrl SessionController
}
```

减少样板代码，通过接口限制 Handler 对 Loop 的访问范围。

### 18.3 实施优先级

| 优先级 | 建议 | 理由 |
|--------|------|------|
| **P0（必须）** | 审批系统精简为 3 层 | 消除概念冗余，简化 `ResolveEffectivePolicy` |
| **P1** | 移除 `UserGlobalPolicy` / `loadUserApprovalPolicy` | 配置已由 `config.LoadFullConfig()` 统一管理 |
| **P1** | 移除 `FullTrust` / `UserPolicyStore` | 无真实实现 |
| **P2** | Agent 通过接口暴露能力 | 减少样板代码 |
| **P3** | Session 拆分持久/瞬态 | 较大改动，需评估收益 |