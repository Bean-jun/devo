# Devo Agent Loop 事件驱动架构重构方案

**版本**：1.0.0  
**状态**：设计阶段  
**关联文档**：[2-architecture.md](./2-architecture.md)

---

## 目录

1. [背景与动机](#1-背景与动机)
2. [现状分析](#2-现状分析)
3. [目标架构](#3-目标架构)
4. [状态机设计](#4-状态机设计)
5. [事件体系设计](#5-事件体系设计)
6. [核心组件详细设计](#6-核心组件详细设计)
7. [LLM 流式调用改造](#7-llm-流式调用改造)
8. [工具执行异步化](#8-工具执行异步化)
9. [审批流程事件化](#9-审批流程事件化)
10. [控制操作（暂停/取消/恢复）](#10-控制操作暂停取消恢复)
11. [文件结构变化](#11-文件结构变化)
12. [实施路径](#12-实施路径)
13. [测试策略](#13-测试策略)
14. [风险与应对](#14-风险与应对)

---

## 1. 背景与动机

### 1.1 当前问题

`internal/core/agentloop/run.go` 中的 `runAgentLoop` 方法是一个 **200+ 行的巨型 for 循环**，所有逻辑耦合在一起：

```
runAgentLoop()
  └─ for {
       ├─ 获取消息
       ├─ 轮询检查控制标志 (CancelRequested/PauseRequested)
       ├─ 压缩上下文
       ├─ 组装 prompt
       ├─ 同步调用 LLM (阻塞)
       ├─ 处理 ToolCalls
       │    ├─ 同步执行工具 (阻塞)
       │    └─ 同步等待审批 (select 阻塞)
       └─ 继续循环
     }
```

**五大痛点**：

| 痛点 | 描述 | 影响 |
|------|------|------|
| 控制标志轮询 | `checkControlFlags` 在循环中被动检查，有延迟 | 暂停/取消响应不及时 |
| LLM 同步调用 | `Complete()` 必须等完整结果 | 前端无法实时展示打字机效果 |
| 工具同步执行 | `Execute()` 阻塞等待 | 无法并行执行独立工具 |
| 逻辑耦合 | 所有逻辑在一个函数中 | 难以测试、难以扩展 |
| 隐式控制流 | 状态转换靠 if/continue 实现 | 新增状态需改整个循环 |

### 1.2 目标

将隐式的控制流**显式化为状态机 + 事件驱动**，实现：

- LLM 调用的流式响应，前端实时打字机效果
- 控制操作的即时响应（Channel 信号替代轮询）
- 工具执行的异步化与进度推送
- 每个状态独立可测试
- 架构可扩展，新增状态只需注册 handler

---

## 2. 现状分析

### 2.1 现有调用链路

```
HTTP POST /api/sessions/{id}/messages
  └─ message_handler.go: PostMessage()
       └─ loop.go: ProcessMessage()
            ├─ 状态检查 (Idle/Processing/AwaitingApproval)
            ├─ 添加用户消息
            └─ go l.runAgentLoop(context.Background(), sessionID, eventBus)  ← goroutine 启动
                 └─ run.go: runAgentLoop()
                      └─ for { ... }  ← 200+ 行循环
```

### 2.2 现有 Session 状态（保留不变）

```
Idle → Processing → Idle（正常完成）
                  → Paused（暂停）
                  → Idle（取消/错误）
     → AwaitingApproval → Processing（批准/拒绝后恢复）
     → Completed → Archived
```

### 2.3 现有 EventBus 机制（保留不变）

```go
type EventBus struct {
    mu          sync.RWMutex
    history     []Event           // 最多 200 条事件历史
    nextID      int64
    subscribers map[int64]chan Event
}

func (eb *EventBus) Publish(eventType string, data any)
func (eb *EventBus) Subscribe() (chan Event, func())
func (eb *EventBus) GetHistory(sinceID int64) []Event
```

### 2.4 现有控制操作（改为信号驱动）

| 方法 | 当前实现 | 改为 |
|------|---------|------|
| `Pause(sessionID)` | 设置 `PauseRequested=true`，轮询检查 | 发信号到 `PauseCh` |
| `Cancel(sessionID)` | 设置 `CancelRequested=true`，轮询检查 | 发信号到 `CancelCh` + kill 子进程 |
| `Resume(sessionID)` | 设置 `State=Processing` | 发信号到 `ResumeCh` |

### 2.5 现有 LLM 客户端接口

```go
type Client interface {
    Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*CompleteResult, error)
}
```

### 2.6 现有工具执行器接口

```go
type ToolExecutor interface {
    Execute(workingDir string, toolName string, params map[string]interface{}) (*tools.ToolResult, error)
    GetTool(name string) (tools.Tool, bool)
    ListTools() []tools.Tool
}
```

---

## 3. 目标架构

### 3.1 核心变化对比

| 维度 | 当前架构 | 目标架构 |
|------|---------|---------|
| 控制流 | for 循环 + if/continue | 状态机 + 事件回调 |
| LLM 调用 | 同步 `Complete()` | 异步 `CompleteStream(callback)` |
| 工具执行 | 同步 `Execute()` | 异步 `ExecuteAsync(ctx, callback)` |
| 控制操作 | 轮询 `checkControlFlags` | Channel 信号即时响应 |
| 审批等待 | select 阻塞 | 状态机暂停，等待事件 |
| 可测试性 | 需 mock 整个循环 | 每个 handler 独立测试 |
| 可扩展性 | 改循环体 | 注册新 handler |

### 3.2 架构图

```
                         ┌──────────────────────┐
                         │    ProcessMessage()   │
                         │  创建 LoopContext     │
                         │  启动 StateMachine    │
                         └──────────┬───────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────────────┐
│                        StateMachine.Run()                         │
│                                                                   │
│  ┌──────────┐   ┌───────────┐   ┌──────────┐   ┌──────────────┐  │
│  │ LoopIdle │◄──│ Preparing │──▶│ Thinking │──▶│ Evaluating   │  │
│  └──────────┘   └───────────┘   └──────────┘   │  Result      │  │
│       ▲              ▲                ▲         └──────┬───────┘  │
│       │              │                │                │          │
│       │              │                │    ┌───────────┤          │
│       │              │                │    │           │          │
│       │              │                │    ▼           ▼          │
│       │              │                │  ┌────────┐ ┌─────────┐  │
│       │              │                │  │ Tool   │ │ Text    │  │
│       │              │                │  │ Execute│ │ Response│  │
│       │              │                │  └───┬────┘ └────┬────┘  │
│       │              │                │      │           │       │
│       │              │                │      │  ┌────────┼───┐   │
│       │              │                │      ▼  ▼        │   │   │
│       │              │                │  ┌────────────┐  │   │   │
│       │              │                │  │Awaiting    │  │   │   │
│       │              │                │  │Approval    │  │   │   │
│       │              │                │  └─────┬──────┘  │   │   │
│       │              │                │        │          │   │   │
│       │              │                └────────┼──────────┘   │   │
│       │              │                         │ (回到Preparing) │   │
│       │              │                         └────────────────┘   │
│       │              │                                              │
│       │              │  (收到新消息)                                  │
│       └──────────────┘                                              │
│                                                                     │
│  控制信号: CancelCh / PauseCh / ResumeCh (所有状态都可响应)           │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. 状态机设计

### 4.1 Loop 内部状态定义

这些是**Loop 内部**的细粒度执行状态，替代当前的 for 循环隐式控制流。与 Session 级状态（Idle/Processing/AwaitingApproval 等）是**两个独立维度**。

```go
// internal/core/agentloop/state_machine.go

type LoopState string

const (
    LoopStateIdle              LoopState = "idle"               // 等待新消息
    LoopStatePreparing         LoopState = "preparing"          // 加载消息、压缩、组装 prompt
    LoopStateThinking          LoopState = "thinking"           // LLM 流式生成中
    LoopStateEvaluatingResult  LoopState = "evaluating_result"  // 判断 LLM 结果类型
    LoopStateToolExecuting     LoopState = "tool_executing"     // 执行工具调用
    LoopStateAwaitingApproval  LoopState = "awaiting_approval"  // 等待用户审批
    LoopStateTextResponse      LoopState = "text_response"      // 返回纯文本响应
    LoopStatePaused            LoopState = "paused"             // 暂停
    LoopStateCancelled         LoopState = "cancelled"          // 取消
    LoopStateError             LoopState = "error"              // 错误（可恢复）
)
```

### 4.2 状态转换表

| 当前状态 | 事件/条件 | 下一状态 | 说明 |
|---------|----------|---------|------|
| `LoopIdle` | 收到用户消息 | `Preparing` | 开始新循环 |
| `Preparing` | 准备完成 | `Thinking` | 开始 LLM 调用 |
| `Thinking` | LLM 流式完成 | `EvaluatingResult` | 评估返回结果 |
| `EvaluatingResult` | 有 tool_calls | `ToolExecuting` | 执行工具 |
| `EvaluatingResult` | 无 tool_calls (纯文本) | `TextResponse` | 返回文本 |
| `ToolExecuting` | 需要审批 | `AwaitingApproval` | 暂停等待审批 |
| `ToolExecuting` | 所有工具执行完 | `Preparing` | 回到循环 |
| `AwaitingApproval` | 审批通过/拒绝 | `ToolExecuting` | 继续执行 |
| `AwaitingApproval` | 审批超时 | `ToolExecuting` | 跳过该工具 |
| `TextResponse` | 保存完成 | `LoopIdle` | 循环结束 |
| **任意状态** | 收到取消信号 | `Cancelled` | 终止 |
| **任意状态** | 收到暂停信号 | `Paused` | 暂停 |
| `Paused` | 收到恢复信号 | 回到暂停前的状态 | 继续执行 |
| **任意状态** | 不可恢复错误 | `Error` | 错误处理 |

### 4.3 Session 级状态与 Loop 状态的对应关系

| Session State | 对应的 Loop 状态 |
|---------------|-----------------|
| `StateProcessing` | `Preparing`, `Thinking`, `EvaluatingResult`, `ToolExecuting`, `TextResponse` |
| `StateAwaitingApproval` | `AwaitingApproval` |
| `StatePaused` | `Paused` |
| `StateIdle` | `LoopIdle` |

---

## 5. 事件体系设计

### 5.1 事件分类

#### 保留的现有事件

```go
// 这些事件在现有代码中已存在，保持不变
"thinking"              // 开始处理
"token_usage"           // token 用量上报
"tool_call_request"     // 工具调用请求
"tool_result"           // 工具执行结果
"approval_auto"         // 自动批准
"approval_required"     // 需要审批
"approval_resolved"     // 审批已解决
"message_complete"      // 消息完成
"session_state_change"  // 会话状态变更
"error"                 // 错误
```

#### 新增事件

```go
// === LLM 流式事件 ===
"streaming_token"         // 单个 token/chunk，前端打字机效果
"streaming_complete"      // LLM 流式输出完成（含 tool_calls、finish_reason）

// === Loop 内部状态事件（调试/监控用） ===
"loop.state_change"       // Loop 内部状态变更
"loop.preparing_done"     // 准备阶段完成
"loop.thinking_started"   // 开始思考
"loop.thinking_complete"  // 思考完成
"loop.result_evaluated"   // 结果评估完成
"loop.tool_execution_done"// 工具执行完成
"loop.loop_completed"     // 整个 loop 完成

// === 控制事件 ===
"loop.paused"             // loop 已暂停（确认）
"loop.cancelled"          // loop 已取消（确认）

// === 工具执行进度 ===
"tool_progress"           // 工具执行进度（长时间运行的工具）
```

### 5.2 事件 Payload 详细定义

```go
// streaming_token
{
    "session_id":        "sess-xxx",
    "token":             "Hello",                    // 单个 token 文本
    "accumulated_text":  "Hello, wor",               // 累积文本
    "index":             5                           // token 序号
}

// streaming_complete
{
    "session_id":    "sess-xxx",
    "full_text":     "Hello, world!",                // 完整响应文本
    "tool_calls":    [{...}],                        // 累积的工具调用（可能为空）
    "finish_reason": "stop",                         // stop/tool_calls/length
    "token_usage":   {                               // token 用量
        "input_tokens":  150,
        "output_tokens": 50,
        "total_tokens":  200
    }
}

// loop.state_change
{
    "session_id": "sess-xxx",
    "old_state":  "Preparing",
    "new_state":  "Thinking",
    "timestamp":  "2026-06-23T10:30:00Z"
}

// tool_progress
{
    "tool_name":  "execute_command",
    "stage":      "running",                         // starting/running/writing/done
    "message":    "正在执行命令...",
    "progress":   0.5                                // 0.0 - 1.0
}

// loop.paused
{
    "session_id": "sess-xxx",
    "paused_at":  "Preparing",                       // 暂停时所在的状态
    "reason":     "user_requested"
}

// loop.cancelled
{
    "session_id": "sess-xxx",
    "cancelled_at": "ToolExecuting",
    "reason":       "user_requested"
}
```

### 5.3 事件流时序图

```
用户发送消息
  │
  ├─ "thinking"                     (ProcessMessage 入口)
  ├─ "session_state_change"         (Idle → Processing)
  ├─ "loop.state_change"            (Idle → Preparing)
  │
  ├─ "loop.preparing_done"          (准备完成)
  ├─ "loop.state_change"            (Preparing → Thinking)
  ├─ "loop.thinking_started"        (开始 LLM 调用)
  │
  ├─ "streaming_token" × N          (每个 token)
  ├─ "streaming_token"
  ├─ ...
  │
  ├─ "streaming_complete"           (LLM 完成)
  ├─ "token_usage"                  (token 用量)
  ├─ "loop.thinking_complete"
  ├─ "loop.state_change"            (Thinking → EvaluatingResult)
  │
  ├─ "loop.result_evaluated"
  ├─ "loop.state_change"            (EvaluatingResult → ToolExecuting)
  │
  ├─ "tool_call_request"            (工具调用)
  │   ├─ "approval_required"        (如果需审批)
  │   ├─ "session_state_change"     (Processing → AwaitingApproval)
  │   ├─ "loop.state_change"        (ToolExecuting → AwaitingApproval)
  │   ├─ "approval_resolved"        (审批结果)
  │   ├─ "session_state_change"     (AwaitingApproval → Processing)
  │   └─ "loop.state_change"        (AwaitingApproval → ToolExecuting)
  ├─ "tool_progress" × N            (执行进度)
  ├─ "tool_result"                  (执行结果)
  │
  ├─ "loop.tool_execution_done"
  ├─ "loop.state_change"            (ToolExecuting → Preparing)
  │
  └─ (回到 Preparing，继续循环)
      ...
      最终: "loop.state_change" → TextResponse → LoopIdle
         ├─ "message_complete"
         ├─ "session_state_change"  (Processing → Idle)
         └─ "loop.loop_completed"
```

---

## 6. 核心组件详细设计

### 6.1 LoopContext 结构体

替代当前在 `runAgentLoop` 函数中通过局部变量和闭包传递的状态。

```go
// internal/core/agentloop/loop_context.go

type LoopContext struct {
    // 基础信息
    SessionID       string
    EventBus        *session.EventBus

    // 循环状态追踪
    StepSeq         int
    TotalStepTokens int
    HasFileChange   bool

    // 当前循环的上下文数据
    ActiveMsgs      []session.Message
    DynamicPrompt   string
    LLMResult       *llmclient.CompleteResult

    // 审批上下文（用于 AwaitingApproval 状态恢复）
    PendingToolCall *session.ToolCall
    ApprovalCh      chan ApprovalDecision

    // 控制信号
    CancelCh        chan struct{}
    PauseCh         chan struct{}
    ResumeCh        chan struct{}

    // 暂停前状态（用于恢复）
    PausedInState   LoopState
}
```

### 6.2 StateHandler 类型定义

```go
// internal/core/agentloop/state_handler.go

// StateHandler 是一个状态处理函数
// 参数:
//   - ctx: 上下文，用于取消和超时控制
//   - lc:  LoopContext，循环上下文
// 返回:
//   - LoopState: 下一个状态
//   - error: 如果返回非 nil，状态机将进入 LoopStateError
type StateHandler func(ctx context.Context, lc *LoopContext) (LoopState, error)
```

### 6.3 StateMachine 引擎

```go
// internal/core/agentloop/state_machine.go

type StateMachine struct {
    handlers map[LoopState]StateHandler
    mu       sync.RWMutex
}

// NewStateMachine 创建状态机
func NewStateMachine() *StateMachine

// Register 注册状态处理器
func (sm *StateMachine) Register(state LoopState, handler StateHandler)

// Run 运行状态机主循环
// 从 LoopStatePreparing 开始，直到 LoopStateIdle 或 LoopStateCancelled
func (sm *StateMachine) Run(ctx context.Context, lc *LoopContext) error
```

`Run` 方法伪代码逻辑：

```
func (sm *StateMachine) Run(ctx, lc):
    currentState = LoopStatePreparing
    
    while currentState != LoopStateIdle and currentState != LoopStateCancelled:
        // 1. 检查控制信号（非阻塞）
        if signal from CancelCh:
            currentState = LoopStateCancelled
            break
        if signal from PauseCh:
            lc.PausedInState = currentState
            currentState = LoopStatePaused
            wait for ResumeCh or CancelCh
            if CancelCh:
                currentState = LoopStateCancelled
                break
            currentState = lc.PausedInState  // 恢复
        
        // 2. 执行当前状态 handler
        handler = sm.handlers[currentState]
        nextState, err = handler(ctx, lc)
        
        if err != nil:
            handle error → LoopStateError or LoopStateCancelled
            break
        
        // 3. 发布状态变更事件
        lc.EventBus.Publish("loop.state_change", {
            old_state: currentState,
            new_state: nextState
        })
        
        currentState = nextState
    
    return nil
```

### 6.4 各 StateHandler 详细设计

#### 6.4.1 PreparingHandler

**职责**：替代当前 `runAgentLoop` 中 for 循环的前半部分。

**迁移来源**：`run.go` 中 for 循环内的以下代码段：
- 获取消息 (`GetMessages`)
- 上下文压缩 (`Compressor.Compress`)
- 控制标志检查 (`checkControlFlags`)
- 组装 prompt (`promptAssembler.Assemble`)

```
PreparingHandler(ctx, lc):
    // 1. 获取消息
    msgs, _, err = store.GetMessages(sessionID, 0, 0)
    if err: return LoopStateError, err
    
    // 2. 上下文压缩
    result, err = compressor.Compress(ctx, sessionID, eventBus)
    if result.CompressedCount > 0:
        archiveManager.AppendSystemMessage(...)
    
    // 3. 重新获取消息（压缩后可能变化）
    msgs, _, err = store.GetMessages(sessionID, 0, 0)
    
    // 4. 过滤活跃消息
    sess, err = store.Get(sessionID)
    activeMsgs = compressor.FilterActiveMessages(msgs, sess.CompressionState)
    
    // 5. 组装 prompt
    dynamicPrompt = promptAssembler.Assemble(sess, hasFileChange)
    hasFileChange = false
    
    // 6. 保存到 LoopContext
    lc.ActiveMsgs = activeMsgs
    lc.DynamicPrompt = dynamicPrompt
    
    // 7. 发布事件
    eventBus.Publish("loop.preparing_done", nil)
    
    return LoopStateThinking, nil
```

#### 6.4.2 ThinkingHandler

**职责**：替代 `l.llmClient.Complete()` 同步调用，改为流式调用。

**迁移来源**：`run.go` 中的 `l.llmClient.Complete(ctx, activeMsgs, dynamicPrompt)`

```
ThinkingHandler(ctx, lc):
    eventBus.Publish("loop.thinking_started", nil)
    
    // 流式调用 LLM
    err = llmClient.CompleteStream(ctx, lc.ActiveMsgs, lc.DynamicPrompt, callback)
    
    callback(evt StreamEvent):
        switch evt.Type:
        case "token":
            eventBus.Publish("streaming_token", {
                token: evt.Token,
                accumulated_text: evt.FullText
            })
        case "done":
            lc.LLMResult = &CompleteResult{
                Text: evt.FullText,
                ToolCalls: evt.ToolCalls,
                TokenUsage: evt.TokenUsage
            }
            eventBus.Publish("streaming_complete", {
                full_text: evt.FullText,
                tool_calls: evt.ToolCalls,
                finish_reason: evt.FinishReason
            })
        case "error":
            eventBus.Publish("error", {message: evt.Err.Error()})
    
    if err: return LoopStateError, err
    
    // Token 用量记录
    lc.StepSeq++
    if result.TokenUsage != nil:
        tokenMeter.RecordStep(...)
        eventBus.Publish("token_usage", ...)
    
    eventBus.Publish("loop.thinking_complete", nil)
    return LoopStateEvaluatingResult, nil
```

#### 6.4.3 EvaluatingResultHandler

**职责**：判断 LLM 返回结果类型，决定下一步。

**迁移来源**：`run.go` 中的 `if len(result.ToolCalls) > 0 { ... } else { ... }` 分支。

```
EvaluatingResultHandler(ctx, lc):
    if len(lc.LLMResult.ToolCalls) > 0:
        eventBus.Publish("loop.result_evaluated", {type: "tool_calls"})
        return LoopStateToolExecuting, nil
    
    eventBus.Publish("loop.result_evaluated", {type: "text"})
    return LoopStateTextResponse, nil
```

#### 6.4.4 ToolExecutingHandler

**职责**：执行工具调用，处理审批。

**迁移来源**：`run.go` 中 for 循环内 `if len(result.ToolCalls) > 0 { ... }` 的整个代码块。

```
ToolExecutingHandler(ctx, lc):
    for each tc in lc.LLMResult.ToolCalls:
        // 1. 保存 assistant 消息（含 tool_calls）
        store.AddMessage(sessionID, assistantMsg)
        
        // 2. 获取工具定义
        tool, ok = toolExecutor.GetTool(tc.ToolName)
        if !ok: 处理未知工具，continue
        
        // 3. 检查是否需要审批
        if needsApproval:
            // 保存上下文，切换到审批状态
            lc.PendingToolCall = &tc
            return LoopStateAwaitingApproval, nil
        
        // 4. 自动批准
        if autoApproved:
            store.AddMessage(sessionID, systemNote)
            eventBus.Publish("approval_auto", ...)
        
        // 5. 路径锁
        lockPath = getLockPath(...)
        if lockPath != "":
            pathLockManager.Lock(lockPath)
        
        // 6. 异步执行工具
        result, err = toolExecutor.ExecuteAsync(ctx, workingDir, tc.ToolName, tc.Params, 
            onProgress(progress):
                eventBus.Publish("tool_progress", {
                    tool_name: tc.ToolName,
                    stage: progress.Stage,
                    message: progress.Message,
                    progress: progress.Progress
                })
        )
        
        if lockPath != "":
            pathLockManager.Unlock(lockPath)
        
        // 7. 处理结果
        store.AddMessage(sessionID, toolMsg)
        eventBus.Publish("tool_result", ...)
        
        // 8. 文件变更追踪
        if tc.ToolName in ["write_file", "edit_file"]:
            store.RecordFileModification(...)
            lc.HasFileChange = true
        
        // 9. 工具调用计数
        if incrementToolCallCount(sessionID, eventBus):
            return LoopStateIdle, nil  // 达到上限，结束
        
        // 10. 检查控制信号（非阻塞）
        select {
        case <-lc.CancelCh: return LoopStateCancelled, nil
        case <-lc.PauseCh: return LoopStatePaused, nil
        default:
        }
    
    eventBus.Publish("loop.tool_execution_done", nil)
    return LoopStatePreparing, nil  // 回到准备阶段，继续 LLM 调用
```

#### 6.4.5 AwaitingApprovalHandler

**职责**：等待用户审批，同时响应取消/暂停/超时。

**迁移来源**：`approval_handler.go` 中的 `handleApproval()` 方法内的 select 循环。

```
AwaitingApprovalHandler(ctx, lc):
    // 审批请求已在 ToolExecutingHandler 中创建
    
    // 创建审批 channel
    ch = make(chan ApprovalDecision, 1)
    approvalChannels[req.ID] = ch
    
    // 更新 session 状态
    sess.State = StateAwaitingApproval
    store.Update(sess)
    
    eventBus.Publish("session_state_change", {
        old_state: "Processing",
        new_state: "AwaitingApproval"
    })
    eventBus.Publish("approval_required", {
        approval_id: req.ID,
        operation_type: req.OperationType,
        ...
    })
    
    // 等待结果
    select {
    case decision = <-ch:
        if decision == "approve":
            restore session to Processing
            return LoopStateToolExecuting, nil  // 继续执行工具
        else:
            store.AddMessage(sessionID, rejectionMsg)
            restore session to Processing
            return LoopStateToolExecuting, nil  // 跳过该工具
    
    case <-timeoutCh:
        handle timeout → reject
        return LoopStateToolExecuting, nil
    
    case <-lc.CancelCh:
        return LoopStateCancelled, nil
    
    case <-lc.PauseCh:
        return LoopStatePaused, nil
    }
```

**关键改进**：不再需要 `cancelTicker` 轮询（当前是 500ms 间隔），通过 `lc.CancelCh` 即时响应。

#### 6.4.6 TextResponseHandler

**职责**：保存 assistant 的纯文本响应，结束 loop。

**迁移来源**：`run.go` 中 for 循环末尾的纯文本处理代码。

```
TextResponseHandler(ctx, lc):
    // 1. 保存 assistant 消息
    assistantMsg = session.Message{
        ID: GenerateID("msg"),
        Role: RoleAssistant,
        Content: lc.LLMResult.Text,
    }
    store.AddMessage(sessionID, assistantMsg)
    archiveManager.AppendAssistantMessage(sessionID, result.Text)
    
    // 2. 发布 message_complete 事件
    eventBus.Publish("message_complete", {
        message_id: assistantMsg.ID,
        full_text: result.Text,
        total_step_tokens: lc.TotalStepTokens,
        ...
    })
    
    // 3. 更新 session 状态为 Idle
    freshSess, err = store.Get(sessionID)
    freshSess.State = StateIdle
    freshSess.LastActiveAt = time.Now()
    store.Update(freshSess)
    
    eventBus.Publish("session_state_change", {
        old_state: "Processing",
        new_state: "Idle",
        reason: "completed"
    })
    
    eventBus.Publish("loop.loop_completed", nil)
    return LoopStateIdle, nil
```

---

## 7. LLM 流式调用改造

### 7.1 接口扩展

```go
// internal/taskexec/llmclient/client.go

// StreamEvent 流式事件
type StreamEvent struct {
    Type        string              // "token", "tool_call", "done", "error"
    Token       string              // 单个 token 文本
    ToolCalls   []session.ToolCall  // 累积的工具调用
    FullText    string              // 累积的完整文本
    FinishReason string             // stop/tool_calls/length
    TokenUsage  *tokenmeter.TokenUsage
    Err         error
}

// StreamCallback 流式回调
type StreamCallback func(event StreamEvent)

// Client 接口扩展
type Client interface {
    // Complete 同步调用（保留，向后兼容）
    Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*CompleteResult, error)
    
    // CompleteStream 流式调用（新增）
    CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback StreamCallback) error
}
```

### 7.2 MockClient 实现

```go
// MockClient.CompleteStream 降级实现
func (m *MockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback StreamCallback) error {
    // 先同步调用 Complete
    result, err := m.Complete(ctx, messages, systemPrompt)
    if err != nil {
        callback(StreamEvent{Type: "error", Err: err})
        return err
    }
    
    // 模拟逐 token 推送
    words := strings.Fields(result.Text)
    for i, word := range words {
        fullText := strings.Join(words[:i+1], " ")
        callback(StreamEvent{
            Type:      "token",
            Token:     word,
            FullText:  fullText,
        })
        // 模拟网络延迟
        time.Sleep(5 * time.Millisecond)
    }
    
    // 完成事件
    callback(StreamEvent{
        Type:       "done",
        FullText:   result.Text,
        ToolCalls:  result.ToolCalls,
        TokenUsage: result.TokenUsage,
    })
    
    return nil
}
```

### 7.3 OpenAI Provider 实现（示意）

OpenAI 的 Chat Completions API 原生支持 `stream: true`，通过 SSE 返回 `data: {"choices":[{"delta":{"content":"..."}}]}` 格式的流式数据。

Provider 需要：
1. 设置 `stream: true` 参数
2. 逐行读取 SSE 响应体
3. 解析每个 chunk 中的 `delta.content` 和 `delta.tool_calls`
4. 调用 `callback` 推送每个事件

---

## 8. 工具执行异步化

### 8.1 接口扩展

```go
// internal/core/agentloop/loop.go

// ToolProgress 工具执行进度
type ToolProgress struct {
    Stage    string  // "starting", "running", "writing", "done"
    Message  string
    Progress float64 // 0.0 - 1.0
}

// ToolExecutor 接口扩展
type ToolExecutor interface {
    Execute(workingDir string, toolName string, params map[string]interface{}) (*tools.ToolResult, error)
    
    // ExecuteAsync 异步执行（新增）
    ExecuteAsync(ctx context.Context, workingDir string, toolName string, params map[string]interface{}, onProgress func(ToolProgress)) (*tools.ToolResult, error)
    
    GetTool(name string) (tools.Tool, bool)
    ListTools() []tools.Tool
}
```

### 8.2 并行执行策略

当前 LLM 的 tool_calls 通常是**顺序依赖**的（后续工具调用依赖前一个工具的结果），所以默认**串行执行**。

通过 `session.MaxConcurrentToolCalls` 配置项，可以启用并行执行（适用于明确彼此独立的工具调用）。

```
ToolExecutingHandler:
    if session.MaxConcurrentToolCalls > 1:
        // 并行执行（使用 errgroup 或 semaphore）
        eg, ctx = errgroup.WithContext(ctx)
        eg.SetLimit(session.MaxConcurrentToolCalls)
        
        for _, tc := range toolCalls:
            tc := tc
            eg.Go(func() error:
                result, err = toolExecutor.ExecuteAsync(ctx, ...)
                // 处理结果...
            )
        
        err = eg.Wait()
    else:
        // 串行执行（默认）
        for _, tc := range toolCalls:
            result, err = toolExecutor.ExecuteAsync(ctx, ...)
```

### 8.3 长时间运行工具的进度推送

对于 `execute_command` 等长时间运行的工具，在执行过程中通过 `onProgress` 回调推送进度。

对于短期工具（如 `read_file`, `list_files`），`onProgress` 可简单地在开始和结束时各调用一次。

---

## 9. 审批流程事件化

### 9.1 当前方式 vs 新方式

| 操作 | 当前方式 | 新方式 |
|------|---------|--------|
| 创建审批请求 | `approvalManager.CreateRequest(...)` | 同左，不变 |
| 等待审批结果 | `handleApproval()` 内 select 循环阻塞 | `AwaitingApprovalHandler` 等待 channel |
| 取消审批 | `cancelTicker` 500ms 轮询 `CancelRequested` | `lc.CancelCh` 即时响应 |
| 超时处理 | `time.After(timeout)` | 同左，不变 |
| 用户决策 | `ResolveApproval()` 发到 channel | 同左，不变 |

### 9.2 审批上下文恢复

当从 `AwaitingApproval` 回到 `ToolExecuting` 时，需要知道是哪个 tool call 触发了审批。通过 `LoopContext.PendingToolCall` 保存：

```go
// 进入审批前
lc.PendingToolCall = &tc  // 保存当前 tool call

// 从审批恢复后
tc := lc.PendingToolCall
lc.PendingToolCall = nil

// 根据审批结果决定是否执行该工具
if approved:
    result, err = toolExecutor.ExecuteAsync(...)
else:
    store.AddMessage(sessionID, rejectionMsg)
```

---

## 10. 控制操作（暂停/取消/恢复）

### 10.1 设计原则

通过 **Channel 信号**替代轮询，实现即时响应。

### 10.2 Loop 结构体变更

```go
// internal/core/agentloop/loop.go

type Loop struct {
    // ... 现有字段 ...
    
    // 新增：活跃 loop 上下文管理
    activeLoops sync.Map  // map[string]*LoopContext
}
```

### 10.3 ProcessMessage 变更

```go
func (l *Loop) ProcessMessage(ctx context.Context, sessionID, content string) error {
    // ... 现有状态检查 ...
    
    // 创建 LoopContext
    lc := &LoopContext{
        SessionID: sessionID,
        EventBus:  eventBus,
        CancelCh:  make(chan struct{}, 1),
        PauseCh:   make(chan struct{}, 1),
        ResumeCh:  make(chan struct{}, 1),
    }
    
    // 注册到活跃 loop 管理器
    l.activeLoops.Store(sessionID, lc)
    defer l.activeLoops.Delete(sessionID)
    
    // 启动状态机
    go l.stateMachine.Run(ctx, lc)
    return nil
}
```

### 10.4 Pause/Cancel/Resume 变更

```go
func (l *Loop) Pause(sessionID string) error {
    // ... 现有状态检查 ...
    
    // 直接发信号到 channel
    lc, ok := l.activeLoops.Load(sessionID)
    if !ok {
        return fmt.Errorf("no active loop for session %s", sessionID)
    }
    
    select {
    case lc.PauseCh <- struct{}{}:
    default:
        // channel 已满，说明已经在暂停中
    }
    return nil
}

func (l *Loop) Cancel(sessionID string) error {
    // ... 现有状态检查 ...
    
    // 先 kill 子进程
    l.killChildProcesses(sessionID)
    
    // 发信号到 channel
    lc, ok := l.activeLoops.Load(sessionID)
    if !ok {
        return fmt.Errorf("no active loop for session %s", sessionID)
    }
    
    select {
    case lc.CancelCh <- struct{}{}:
    default:
    }
    return nil
}

func (l *Loop) Resume(sessionID string) error {
    // ... 现有状态检查 ...
    
    lc, ok := l.activeLoops.Load(sessionID)
    if !ok {
        return fmt.Errorf("no active loop for session %s", sessionID)
    }
    
    select {
    case lc.ResumeCh <- struct{}{}:
    default:
    }
    return nil
}
```

### 10.5 状态机中的暂停/恢复处理

在 `StateMachine.Run()` 的主循环中，每个状态执行前检查 `PauseCh`：

```
// 暂停处理
select {
case <-lc.PauseCh:
    lc.PausedInState = currentState
    eventBus.Publish("loop.paused", {
        paused_at: currentState,
        reason: "user_requested"
    })
    
    // 等待恢复或取消
    select {
    case <-lc.ResumeCh:
        currentState = lc.PausedInState
        continue  // 回到当前状态继续执行
    case <-lc.CancelCh:
        currentState = LoopStateCancelled
        break
    }
default:
}
```

**注意**：暂停/恢复检查位于 `StateMachine.Run()` 主循环中，而非每个 handler 内部。这样：
- 暂停在**状态之间**生效（原子性更好）
- 每个 handler 内部也可以通过 `select` 检查 `CancelCh` 实现在长时间操作中的取消

---

## 11. 文件结构变化

```
internal/core/agentloop/
├── loop.go                  # Loop 结构体（修改：新增 activeLoops、stateMachine 字段）
├── loop_context.go          # 🆕 LoopContext 结构体定义
├── state_machine.go         # 🆕 状态机引擎 (StateMachine + LoopState 定义)
├── state_handlers.go        # 🆕 各状态 handler 实现
│                            #   - PreparingHandler
│                            #   - ThinkingHandler
│                            #   - EvaluatingResultHandler
│                            #   - ToolExecutingHandler
│                            #   - AwaitingApprovalHandler
│                            #   - TextResponseHandler
├── run.go                   # 删除（或仅保留 runAgentLoop 兼容包装）
├── helpers.go               # 保留（工具函数不变）
├── approval_handler.go      # 修改（审批等待逻辑迁移到 AwaitingApprovalHandler）
├── state_control.go         # 修改（Pause/Cancel/Resume 改为 channel 信号）
├── crash_recovery.go        # 保留
├── rollback.go              # 保留
├── *_test.go                # 测试文件（新增 + 修改）
```

### 11.1 变更影响范围

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `loop.go` | 修改 | 新增 `activeLoops`、`stateMachine` 字段；`ProcessMessage` 改为创建 LoopContext + 启动状态机 |
| `loop_context.go` | 新增 | LoopContext 结构体 |
| `state_machine.go` | 新增 | LoopState 枚举、StateMachine 引擎 |
| `state_handlers.go` | 新增 | 6 个状态 handler |
| `run.go` | 删除 | 迁移到 `state_handlers.go` |
| `approval_handler.go` | 修改 | `handleApproval` 中的等待逻辑迁移到 handler |
| `state_control.go` | 修改 | `Pause`/`Cancel`/`Resume` 改为 channel 信号 |
| `helpers.go` | 保留 | 工具函数不变 |
| `llmclient/client.go` | 修改 | 新增 `CompleteStream` 方法和 `StreamEvent` 类型 |
| `llmclient/providers/openai/openai.go` | 修改 | 实现 `CompleteStream` |

---

## 12. 实施路径

### Phase 1：基础设施搭建（预计 1-2 天）

**目标**：状态机引擎可运行，但不改变现有行为。

| 步骤 | 内容 | 产出 |
|------|------|------|
| 1.1 | 创建 `loop_context.go`，定义 `LoopContext` 结构体 | 新文件 |
| 1.2 | 创建 `state_machine.go`，定义 `LoopState` 枚举和 `StateMachine` 引擎 | 新文件 |
| 1.3 | 写 `StateMachine` 单元测试 | 新测试文件 |
| 1.4 | 在 `Loop` 中新增 `stateMachine` 和 `activeLoops` 字段 | 修改 loop.go |

**验证标准**：`StateMachine` 的 `Register` 和 `Run` 方法可通过单元测试验证基本行为。

### Phase 2：迁移核心流程（预计 2-3 天）

**目标**：将 `runAgentLoop` 逻辑拆分为 6 个 handler，与旧代码行为一致。

| 步骤 | 内容 | 产出 |
|------|------|------|
| 2.1 | 创建 `state_handlers.go`，实现 `PreparingHandler` | 新函数 |
| 2.2 | 实现 `ThinkingHandler`（先使用同步 `Complete`，暂不改为流式） | 新函数 |
| 2.3 | 实现 `EvaluatingResultHandler` | 新函数 |
| 2.4 | 实现 `ToolExecutingHandler`（先使用同步 `Execute`） | 新函数 |
| 2.5 | 实现 `AwaitingApprovalHandler` | 新函数 |
| 2.6 | 实现 `TextResponseHandler` | 新函数 |
| 2.7 | 修改 `ProcessMessage` 改为启动状态机（保留旧代码作为 fallback） | 修改 loop.go |
| 2.8 | 写集成测试：确保新旧代码行为一致 | 测试文件 |

**验证标准**：所有现有测试通过；新状态机产生的消息和事件序列与旧代码一致。

### Phase 3：LLM 流式化（预计 1-2 天）

**目标**：LLM 调用改为流式，前端可实时看到打字机效果。

| 步骤 | 内容 | 产出 |
|------|------|------|
| 3.1 | 扩展 `llmclient.Client` 接口，新增 `CompleteStream` | 修改 client.go |
| 3.2 | `MockClient` 实现 `CompleteStream`（模拟 token 推送） | 修改 client.go |
| 3.3 | 修改 `ThinkingHandler` 使用 `CompleteStream` | 修改 state_handlers.go |
| 3.4 | 写 `streaming_token` 事件的单元测试 | 测试文件 |
| 3.5 | OpenAI Provider 实现 `CompleteStream`（如需要） | 修改 openai.go |

**验证标准**：`streaming_token` 事件按顺序发布；`streaming_complete` 包含完整结果。

### Phase 4：控制操作优化（预计 1 天）

**目标**：Pause/Cancel/Resume 改为 channel 信号，移除轮询。

| 步骤 | 内容 | 产出 |
|------|------|------|
| 4.1 | 修改 `state_control.go`：`Pause`/`Cancel`/`Resume` 改为 channel 信号 | 修改文件 |
| 4.2 | 在 `StateMachine.Run()` 中实现暂停/恢复逻辑 | 修改 state_machine.go |
| 4.3 | 在各 handler 中添加 `CancelCh` 检查（长时间操作） | 修改 state_handlers.go |
| 4.4 | 移除 `checkControlFlags` 和相关轮询逻辑 | 清理代码 |
| 4.5 | 写暂停/取消/恢复的单元测试 | 测试文件 |

**验证标准**：暂停/取消在 10ms 内生效（vs 当前的 500ms 轮询延迟）。

### Phase 5：工具异步化（预计 1-2 天）

**目标**：工具执行支持异步和进度推送。

| 步骤 | 内容 | 产出 |
|------|------|------|
| 5.1 | 扩展 `ToolExecutor` 接口，新增 `ExecuteAsync` | 修改 loop.go |
| 5.2 | 修改 `ToolExecutingHandler` 使用 `ExecuteAsync` | 修改 state_handlers.go |
| 5.3 | 实现 `tool_progress` 事件推送 | 修改 state_handlers.go |
| 5.4 | 支持 `MaxConcurrentToolCalls` 并行执行 | 修改 state_handlers.go |
| 5.5 | 写异步执行的单元测试 | 测试文件 |

**验证标准**：工具执行期间发布 `tool_progress` 事件；并行执行时多个工具同时运行。

### Phase 6：清理与收尾（预计 1 天）

**目标**：删除旧代码，确保所有测试通过。

| 步骤 | 内容 | 产出 |
|------|------|------|
| 6.1 | 删除 `run.go` 中的 `runAgentLoop`（如果已完全迁移） | 删除 |
| 6.2 | 清理 `approval_handler.go` 中迁移到 handler 的代码 | 修改 |
| 6.3 | 全量回归测试 | 测试通过 |
| 6.4 | 更新相关文档 | 文档更新 |

---

## 13. 测试策略

### 13.1 测试分层

```
┌─────────────────────────────────────────┐
│          E2E 测试（端到端）               │
│  通过 HTTP API 发送消息，验证完整流程      │
│  (现有 message_handler_test.go 等)        │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│          集成测试                         │
│  验证状态机完整流程，事件序列正确          │
│  (新增 state_machine_integration_test.go) │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│          单元测试                         │
│  每个 handler 独立测试                    │
│  (新增 state_handlers_test.go)            │
└─────────────────────────────────────────┘
```

### 13.2 单元测试清单

#### 13.2.1 StateMachine 引擎测试

```go
// state_machine_test.go

func TestStateMachine_BasicFlow(t *testing.T)
// 测试正常流程：Preparing → Thinking → EvaluatingResult → TextResponse → LoopIdle

func TestStateMachine_ToolCallFlow(t *testing.T)
// 测试工具调用流程：... → EvaluatingResult → ToolExecuting → Preparing → ... → LoopIdle

func TestStateMachine_ApprovalFlow(t *testing.T)
// 测试审批流程：ToolExecuting → AwaitingApproval → ToolExecuting → ...

func TestStateMachine_CancelDuringThinking(t *testing.T)
// 测试在 Thinking 状态中取消

func TestStateMachine_CancelDuringToolExecution(t *testing.T)
// 测试在 ToolExecuting 状态中取消

func TestStateMachine_PauseAndResume(t *testing.T)
// 测试暂停和恢复：任意状态 → Paused → 恢复后继续

func TestStateMachine_PauseAndCancel(t *testing.T)
// 测试暂停后取消

func TestStateMachine_ErrorHandling(t *testing.T)
// 测试各状态中的错误处理

func TestStateMachine_HandlerNotFound(t *testing.T)
// 测试未注册 handler 的状态

func TestStateMachine_StateChangeEvent(t *testing.T)
// 测试每次状态转换都发布 loop.state_change 事件
```

#### 13.2.2 各 Handler 单元测试

```go
// state_handlers_test.go

// PreparingHandler
func TestPreparingHandler_BasicMessages(t *testing.T)
func TestPreparingHandler_WithCompression(t *testing.T)
func TestPreparingHandler_GetMessagesError(t *testing.T)

// ThinkingHandler
func TestThinkingHandler_StreamingTokens(t *testing.T)
// 验证 streaming_token 事件按顺序发布
func TestThinkingHandler_CompleteWithToolCalls(t *testing.T)
// 验证 tool_calls 正确保存到 LoopContext
func TestThinkingHandler_CompleteWithTextOnly(t *testing.T)
// 验证纯文本响应
func TestThinkingHandler_StreamError(t *testing.T)
// 验证流式错误处理
func TestThinkingHandler_CancelDuringStreaming(t *testing.T)
// 验证流式调用中的取消

// EvaluatingResultHandler
func TestEvaluatingResultHandler_WithToolCalls(t *testing.T)
func TestEvaluatingResultHandler_WithTextOnly(t *testing.T)

// ToolExecutingHandler
func TestToolExecutingHandler_SingleTool(t *testing.T)
func TestToolExecutingHandler_MultipleTools(t *testing.T)
func TestToolExecutingHandler_UnknownTool(t *testing.T)
func TestToolExecutingHandler_NeedsApproval(t *testing.T)
// 验证需要审批时返回 AwaitingApproval
func TestToolExecutingHandler_ToolCallLimitReached(t *testing.T)
func TestToolExecutingHandler_FileChangeDetection(t *testing.T)
func TestToolExecutingHandler_CancelDuringExecution(t *testing.T)

// AwaitingApprovalHandler
func TestAwaitingApprovalHandler_Approved(t *testing.T)
func TestAwaitingApprovalHandler_Rejected(t *testing.T)
func TestAwaitingApprovalHandler_Timeout(t *testing.T)
func TestAwaitingApprovalHandler_CancelWhileWaiting(t *testing.T)
func TestAwaitingApprovalHandler_PauseWhileWaiting(t *testing.T)

// TextResponseHandler
func TestTextResponseHandler_SavesMessage(t *testing.T)
func TestTextResponseHandler_ReturnsToIdle(t *testing.T)
func TestTextResponseHandler_PublishesEvents(t *testing.T)
```

#### 13.2.3 LLM 流式调用测试

```go
// llmclient/client_test.go (新增)

func TestMockClient_CompleteStream(t *testing.T)
// 验证 MockClient 的 CompleteStream 正确推送 token 事件

func TestMockClient_CompleteStream_ToolCalls(t *testing.T)
// 验证包含 tool_calls 的流式响应

func TestMockClient_CompleteStream_Error(t *testing.T)
// 验证错误事件的推送
```

#### 13.2.4 控制操作测试

```go
// state_control_test.go (修改)

func TestPause_ChannelSignal(t *testing.T)
// 验证 Pause 通过 channel 发送信号

func TestCancel_ChannelSignal(t *testing.T)
// 验证 Cancel 通过 channel 发送信号

func TestCancel_KillsChildProcesses(t *testing.T)
// 验证取消时 kill 子进程

func TestResume_ChannelSignal(t *testing.T)
// 验证 Resume 通过 channel 发送信号
```

### 13.3 集成测试清单

```go
// state_machine_integration_test.go (新增)

func TestIntegration_FullConversation(t *testing.T)
// 完整对话流程：发送消息 → 收到回复 → 状态回到 Idle

func TestIntegration_ToolCallLoop(t *testing.T)
// 工具调用循环：LLM 返回 tool_calls → 执行 → 结果返回 LLM → 继续

func TestIntegration_ApprovalRequired(t *testing.T)
// 审批流程：工具需要审批 → 等待 → 批准 → 继续执行

func TestIntegration_EventSequence(t *testing.T)
// 验证事件序列：compare expected vs actual event types in order

func TestIntegration_StateTransitions(t *testing.T)
// 验证状态转换序列：Preparing → Thinking → ... → LoopIdle

func TestIntegration_PauseResumeMidLoop(t *testing.T)
// 在 loop 中途暂停和恢复

func TestIntegration_CancelMidLoop(t *testing.T)
// 在 loop 中途取消

func TestIntegration_ToolCallLimit(t *testing.T)
// 达到工具调用上限后停止

func TestIntegration_ErrorRecovery(t *testing.T)
// 错误恢复：LLM 调用失败 → 错误状态 → 回到 Idle
```

### 13.4 测试辅助工具

```go
// test_helpers.go (新增)

// mockStateHandler 创建一个返回固定状态的 mock handler
func mockStateHandler(nextState LoopState, err error) StateHandler

// newTestLoopContext 创建测试用的 LoopContext
func newTestLoopContext(sessionID string, store *session.InMemoryStore) *LoopContext

// assertEventPublished 断言指定类型的事件已发布
func assertEventPublished(t *testing.T, eventBus *session.EventBus, eventType string, timeout time.Duration)

// assertEventSequence 断言事件序列与预期一致
func assertEventSequence(t *testing.T, eventBus *session.EventBus, expectedTypes []string, timeout time.Duration)

// sendCancelSignal 发送取消信号
func sendCancelSignal(lc *LoopContext)

// sendPauseSignal 发送暂停信号
func sendPauseSignal(lc *LoopContext)

// sendResumeSignal 发送恢复信号
func sendResumeSignal(lc *LoopContext)
```

### 13.5 现有测试兼容性

| 测试文件 | 影响 | 处理方式 |
|---------|------|---------|
| `loop_test.go` | 中等 | `ProcessMessage` 行为不变，应通过；可能需要调整等待时间 |
| `approval_test.go` | 中等 | 审批流程逻辑不变，但内部实现变化，验证事件序列 |
| `tool_test.go` | 中等 | 工具执行行为不变，验证事件序列 |
| `state_control_test.go` | 高 | Pause/Cancel/Resume 改为 channel 信号，需重写 |
| `token_test.go` | 低 | 行为不变，应通过 |
| `crash_recovery_test.go` | 低 | 行为不变，应通过 |
| `rollback_test.go` | 低 | 行为不变，应通过 |

---

## 14. 风险与应对

### 14.1 风险矩阵

| 风险 | 等级 | 概率 | 影响 | 应对措施 |
|------|------|------|------|---------|
| 状态转换遗漏 | 中 | 中 | 导致 loop 卡死或跳过步骤 | 状态机有严格的状态转换表；集成测试覆盖所有路径 |
| 现有测试大量失败 | 高 | 高 | 回归风险 | 分阶段迁移；Phase 2 保留旧代码并行运行；全量回归测试 |
| 工具执行顺序依赖 | 中 | 中 | 并行执行导致结果错误 | 默认串行执行；并行化通过配置显式启用 |
| Context 生命周期管理 | 中 | 中 | goroutine 泄漏 | 使用 `defer` 清理；每个 handler 接收 ctx，cancel 时级联取消 |
| Channel 死锁 | 低 | 低 | loop 卡死 | 所有 channel 使用带缓冲的（buffer=1）；select+default 非阻塞发送 |
| 审批流程复杂度 | 高 | 中 | 审批上下文丢失 | `PendingToolCall` 保存审批上下文；恢复时从上下文继续 |
| 向后兼容性 | 中 | 低 | 外部调用方受影响 | `ProcessMessage` 签名不变；`Complete` 方法保留 |
| 性能退步 | 低 | 低 | 状态机开销大于简单循环 | 状态机开销极小（map 查找 + 函数调用），实测对比 |

### 14.2 回滚策略

如果新架构出现问题，可以快速回退到旧代码：

1. Phase 2 期间保留 `runAgentLoop` 原代码，通过 feature flag 控制使用新旧代码
2. 所有状态 handler 是从旧代码**抽取**而非重写，逻辑等价
3. 每个 Phase 完成后打 tag，便于回退到任意阶段

### 14.3 监控指标

重构后需要关注的指标：

| 指标 | 说明 | 目标 |
|------|------|------|
| Loop 完成时间 | 从 ProcessMessage 到 LoopIdle 的总耗时 | 不低于重构前 |
| 控制响应延迟 | 从 Cancel 调用到 loop 停止的时间 | < 10ms（当前 ~500ms） |
| 首 token 延迟 | 从 Thinking 开始到第一个 streaming_token 的时间 | 可与重构前对比 |
| 事件序列正确性 | 事件类型和顺序 | 100% 与旧代码一致 |
| 测试覆盖率 | agentloop 包的测试覆盖率 | ≥ 80% |

---

## 附录 A：新旧代码对比示例

### A.1 当前 runAgentLoop 简化版

```go
func (l *Loop) runAgentLoop(ctx context.Context, sessionID string, eventBus *session.EventBus) {
    // ... defer recover ...
    // ... get session ...
    
    for {
        // 1. 获取消息
        msgs, _, _ := l.store.GetMessages(sessionID, 0, 0)
        
        // 2. 检查控制标志（轮询）
        if l.checkControlFlags(sessionID, eventBus) { return }
        
        // 3. 压缩上下文
        l.compressor.Compress(ctx, sessionID, eventBus)
        
        // 4. 组装 prompt
        dynamicPrompt := l.promptAssembler.Assemble(sess, hasFileChange)
        
        // 5. LLM 调用（同步阻塞）
        result, err := l.llmClient.Complete(ctx, activeMsgs, dynamicPrompt)
        
        // 6. 处理结果
        if len(result.ToolCalls) > 0 {
            // ... 工具执行（同步阻塞）...
            // ... 审批等待（select 阻塞）...
            continue
        }
        
        // 7. 文本响应
        // ... 保存消息，更新状态为 Idle ...
        return
    }
}
```

### A.2 新架构等价代码

```go
// StateMachine.Run() 替代 for 循环
// PreparingHandler 替代步骤 1-4
// ThinkingHandler 替代步骤 5（改为流式）
// EvaluatingResultHandler 替代步骤 6 的分支判断
// ToolExecutingHandler 替代步骤 6 的工具执行
// AwaitingApprovalHandler 替代步骤 6 的审批等待
// TextResponseHandler 替代步骤 7

// 控制标志检查 → 改为 StateMachine.Run() 中的 select 检查 CancelCh/PauseCh
```

---

## 附录 B：新增文件完整清单

| 文件 | 类型 | 预计行数 |
|------|------|---------|
| `loop_context.go` | 新增 | ~50 行 |
| `state_machine.go` | 新增 | ~120 行 |
| `state_handlers.go` | 新增 | ~400 行 |
| `state_machine_test.go` | 新增 | ~300 行 |
| `state_handlers_test.go` | 新增 | ~500 行 |
| `state_machine_integration_test.go` | 新增 | ~300 行 |
| `test_helpers.go` | 新增 | ~100 行 |
| `loop.go` | 修改 | +20 行 |
| `state_control.go` | 修改 | -30 行, +30 行 |
| `approval_handler.go` | 修改 | -50 行 |
| `run.go` | 删除 | -367 行 |
| `llmclient/client.go` | 修改 | +50 行 |

---
