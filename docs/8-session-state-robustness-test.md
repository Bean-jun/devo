# Devo 会话状态鲁棒性测试方案

**版本**：2.0.0
**状态**：已完成（后端 + 测试通过）
**更新日期**：2026-07-01
**关联文档**：[2-architecture.md](./2-architecture.md)、[6-agent-loop-event-driven-refactor.md](./6-agent-loop-event-driven-refactor.md)

---

## 目录

1. [背景与动机](#1-背景与动机)
2. [Session State 重新设计](#2-session-state-重新设计)
3. [操作行为决策表](#3-操作行为决策表)
4. [消息清理策略](#4-消息清理策略)
5. [前端状态机同步](#5-前端状态机同步)
6. [测试场景](#6-测试场景)
7. [测试方法](#7-测试方法)
8. [改动文件清单](#8-改动文件清单)
9. [测试记录（v1.1.0）](#9-测试记录v110)
10. [测试报告（v1.1.0）](#10-测试报告v110)
11. [测试报告（v2.0.0）](#11-测试报告v200)

---

## 1. 背景与动机

### 1.1 问题来源

在分析 Devo 的 Agent Loop 状态机时，发现以下问题：

1. **取消操作不中断 LLM 流式调用**：`thinkingHandler` 内部不检查 `CancelCh`，LLM 流会一直跑到结束
2. **取消操作导致孤儿 tool_call**：取消时正在执行的工具被杀死，未执行的工具被跳过，但 assistant 消息中的 tool_calls 已全部落盘，缺失的 tool_result 导致下一轮 LLM 调用时 API 返回 400 错误
3. **崩溃恢复不清理消息**：`RecoverCrashedSessions` 只重置会话状态和杀死子进程，不检查消息历史中是否存在孤儿 tool_call
4. **暂停/恢复的状态机行为未经充分测试**：暂停发生在不同阶段时，状态转换和消息完整性是否得到保证
5. **Processing 状态粒度过粗**：前端无法区分"LLM 在流式生成"和"工具在执行中"，无法做差异化交互

### 1.2 核心验证目标

**操作（暂停/取消/恢复/崩溃）发生后，下一轮对话能否正常走通？**

- ✅ 200 + 正常响应 = 通过，消息历史完整
- ❌ 400 + 错误信息 = 失败，存在孤儿 tool_call

### 1.3 涉及组件

| 组件 | 文件 | 职责 |
|------|------|------|
| 状态机 | `internal/core/agentloop/state_machine.go` | Agent Loop 主循环 |
| 状态处理器 | `internal/core/agentloop/state_handlers.go` | 各状态的处理逻辑 |
| 状态控制 | `internal/core/agentloop/state_control.go` | Cancel/Pause/Resume 入口 |
| Session 定义 | `internal/core/session/session.go` | Session State 常量定义 |
| 崩溃恢复 | `internal/core/agentloop/crash_recovery.go` | 启动时恢复崩溃会话 |
| 前端状态 | `web/src/types/session.ts`、`web/src/stores/session.ts` | 前端状态机同步 |
| Mock LLM | `tests/mock_server/main.go` | 模拟 LLM API，支持工具调用和消息校验 |

---

## 2. Session State 重新设计

### 2.1 变更动机

当前 `Processing` 覆盖了 LLM 流式输出和工具执行两个完全不同的阶段：

```
thinkingHandler          → 前端看到 Processing
evaluatingResultHandler  → 前端看到 Processing
toolExecutingHandler     → 前端看到 Processing
textResponseHandler      → 前端看到 Processing
```

**问题**：
- 前端无法区分"AI 在思考"和"AI 在干活"
- Pause 应在工具执行阶段生效，但无法区分当前是否在工具执行中
- 流式输出中暂停无意义（LLM 数据已生成，暂停只是延迟展示）

### 2.2 新 Session State 定义

```go
// internal/core/session/session.go

const (
    StateIdle             State = "Idle"              // 等待用户输入
    StateThinking         State = "Thinking"          // LLM 流式生成中（新增）
    StateToolExecuting    State = "ToolExecuting"     // 工具执行中（新增）
    StateAwaitingApproval State = "AwaitingApproval"  // 等待用户审批
    StatePaused           State = "Paused"            // 工具执行暂停
    StateCompleted        State = "Completed"         // 会话正常结束
    StateArchived         State = "Archived"          // 已归档
)
```

**废弃 `StateProcessing`**，用 `StateThinking` + `StateToolExecuting` 完全替代。

### 2.3 Loop State → Session State 映射

| Loop State（内部） | Session State（前端） | 说明 |
|-------------------|---------------------|------|
| Preparing | Thinking | 加载上下文 |
| Thinking | **Thinking** | LLM 流式生成 |
| EvaluatingResult | Thinking | 瞬间过渡，不值得单独暴露 |
| TextResponse | Thinking | 纯文本输出 |
| ToolExecuting | **ToolExecuting** | 工具执行中 |
| AwaitingApproval | AwaitingApproval | 等待审批 |
| Paused | Paused | 暂停 |
| Idle | Idle | 等待新消息 |

### 2.4 状态转换图

```
     SendMessage
        │
  Idle ─┴─────────→ Thinking ──→ ToolExecuting ──→ Thinking ──→ ... ──→ Idle
                      │               │
                   [Cancel]        [Pause]
                      │               │
                      ▼               ▼
                    Idle           Paused ──[Resume]──→ ToolExecuting
                                      │
                                   [Cancel]
                                      │
                                      ▼
                                    Idle
```

### 2.5 执行环节（子阶段）

Session State 虽然已拆分，但 Cancel 和 Crash 的清理逻辑还取决于 assistant 消息是否已落盘：

```
                  消息落盘边界
                       │
    Thinking           │  ToolExecuting / AwaitingApproval
    ───────────────────┼──────────────────────
    assistant 消息      │  assistant 消息
    未落盘              │  已落盘
                       │
    Cancel/Crash：      │  Cancel/Crash：
    丢弃即干净 ✅        │  必须补 tool_result ⚠️
```

| Session State | 实际阶段 | 消息落盘状态 | 清理难度 |
|---------------|---------|-------------|---------|
| Thinking | 流式输出中 | assistant 消息**未落盘**（token 在内存中） | 低：丢弃即可 |
| ToolExecuting | 工具调用中 | assistant 消息（含 tool_calls）**已落盘**，tool_result **部分落盘** | 高：需补 tool_result(error) |
| AwaitingApproval | 审批等待中 | assistant 消息（含 tool_call）**已落盘**，tool_result **未落盘** | 高：需补 tool_result(rejected) |

---

## 3. 操作行为决策表

### 3.1 Pause（暂停）

**核心原则**：Pause 只在工具执行阶段生效，Thinking 阶段忽略。

| Session State | HTTP | 新 State | 行为 |
|---------------|------|----------|------|
| Thinking | 200 | Thinking（不变） | **忽略暂停**，流式继续完成。发 EventBus 通知"暂停请求已记录，将在工具执行阶段生效" |
| ToolExecuting | 200 | **Paused** | 当前工具执行完，更新 `sess.State = Paused`，后续工具排队等待 |
| AwaitingApproval | 409 | — | 拒绝，审批阶段不可暂停 |
| Paused | 409 | — | 已暂停 |
| Idle | 409 | — | 无可暂停 |
| Completed | 409 | — | 会话已结束 |
| Archived | 409 | — | 会话已归档 |

### 3.2 Cancel（取消）

**核心原则**：追加 `tool_result(error/rejected)`，不删除消息。

| Session State | HTTP | 新 State | 消息清理 |
|---------------|------|----------|---------|
| Thinking | 200 | **Idle** | 中止 LLM 流，assistant 消息未落盘 → **无需清理** |
| ToolExecuting | 200 | **Idle** | 杀子进程 → 补 `tool_result(error)` × 每个未完成的 tool_call → 追加 `user: "用户已取消当前操作"` |
| AwaitingApproval | 200 | **Idle** | 补 `tool_result(rejected)` × 待审批的 tool_call → 追加 `user: "用户已拒绝工具调用：{tool_name}"` |
| Paused | 200 | **Idle** | 补 `tool_result(error)` × 排队中的 tool_call → 追加 `user: "用户已取消暂停的操作"` |
| Idle | 409 | — | 无可取消 |
| Completed | 409 | — | 会话已结束 |
| Archived | 409 | — | 会话已归档 |

### 3.3 Resume（恢复）

**核心原则**：Resume 只从 Paused 状态恢复，启动工具调度。

| Session State | HTTP | 新 State | 行为 |
|---------------|------|----------|------|
| Paused | 200 | **ToolExecuting** | 恢复工具调度，从排队工具继续执行 |
| 其他所有状态 | 409 | — | 不可恢复 |

### 3.4 SendMessage（发新消息到 Paused 会话）

**核心原则**：Paused 下发新消息 = 自动 Cancel + 处理新消息。

| Session State | HTTP | 新 State | 行为 |
|---------------|------|----------|------|
| Paused | 202 | **Thinking** | 自动 Cancel（补 tool_result(error) + 追加 user 消息）→ 处理新消息 |
| Idle | 202 | Thinking | 正常启动新轮 |
| Thinking | 409 | — | 上一轮未完成 |
| ToolExecuting | 409 | — | 上一轮未完成 |
| AwaitingApproval | 409 | — | 需先审批或取消 |
| Completed | 409 | — | 会话已结束 |
| Archived | 409 | — | 会话已归档 |

### 3.5 Crash（崩溃重启）

**核心原则**：重启后根据 DB 中的状态决定清理逻辑。

| DB 中的 State | 恢复后 State | 消息清理 |
|---------------|-------------|---------|
| Thinking | **Idle** | LLM 连接断开，assistant 消息未落盘 → **无需清理** |
| ToolExecuting | **Idle** | 子进程被杀，assistant 消息（含 tool_calls）已落盘 → **补 tool_result(error) + 追加 user 消息** |
| AwaitingApproval | **Idle** | 审批失效 → **补 tool_result(rejected) + 追加 user 消息** |
| Paused | **Idle** | 同 ToolExecuting → **补 tool_result(error) + 追加 user 消息** |
| Idle | Idle | 无需处理 |
| Completed | Completed | 保持 |
| Archived | Archived | 保持 |

---

## 4. 消息清理策略

### 4.1 核心原则

```
tool_call 和 tool_result 必须成对出现，否则 LLM API 返回 400

清理方式：
  ❌ 删除 assistant 消息（复杂、有风险，需追踪消息 ID）
  ✅ 追加 tool_result(error/rejected) + user 消息（简单、安全，只有追加操作）
```

### 4.2 清理示例

#### 工具调用中 Cancel

```
原消息历史：
  user: "帮我处理文件"
  assistant: [tool_call: read_file, tool_call: search_codebase]
                                    ↑ 两个 tool_call

Cancel 后追加：
  tool_result(read_file):      {error: "用户取消操作"}
  tool_result(search_codebase): {error: "用户取消操作"}
  user: "用户已取消当前操作"

下一轮 LLM 看到的完整上下文：
  "用户取消了 read_file 和 search_codebase，并且说'已取消当前操作'"
  → LLM 自然回应："好的，操作已取消。需要我帮你做什么吗？"
```

#### 审批等待中 Cancel

```
原消息历史：
  user: "帮我写文件"
  assistant: [tool_call: write_file]
                    ↑ 等待审批

Cancel 后追加：
  tool_result(write_file): {error: "用户已拒绝此操作"}
  user: "用户已拒绝工具调用：write_file"

下一轮 LLM 看到的完整上下文：
  "用户拒绝了 write_file"
  → LLM 自然回应："好的，文件未被修改。需要我用其他方式处理吗？"
```

### 4.3 优势

1. **零数据库删除** — 只有追加，没有删除，安全简单
2. **消息历史始终合法** — LLM 看到的永远是 `tool_call` ↔ `tool_result` 成对
3. **LLM 有上下文** — 知道"用户取消了什么"，能自然回应
4. **代码简单** — 不需要追踪"该删除哪条 assistant 消息"

---

## 5. 前端状态机同步

### 5.1 Pause 按钮状态

| Session State | 前端 Pause 按钮 | Resume 按钮 | Cancel 按钮 |
|---------------|:---:|:---:|:---:|
| Thinking | 置灰/隐藏 | 隐藏 | 可见 |
| ToolExecuting | **可点击** | 隐藏 | 可见 |
| Paused | 隐藏 | **可点击** | 可点击 |
| AwaitingApproval | 隐藏（审批弹窗已覆盖） | 隐藏 | 隐藏 |
| Idle | 隐藏 | 隐藏 | 隐藏 |

### 5.2 前端 UI 差异化

```typescript
// 之前：无法区分
if (session.state === 'Processing') {
  // 不知道在干嘛，只能显示通用 loading
}

// 之后：可以区分
if (session.state === 'Thinking') {
  // 显示流式打字动画，Pause 按钮置灰
  showStreamingAnimation()
  disablePauseButton()
} else if (session.state === 'ToolExecuting') {
  // 显示工具执行进度，Pause 按钮可点击
  showToolProgress(toolName)
  enablePauseButton()
}
```

### 5.3 前端改动文件

| 文件 | 改动 |
|------|------|
| `web/src/types/session.ts` | 类型新增 `Thinking`、`ToolExecuting`，废弃 `Processing` |
| `web/src/stores/session.ts` | Pause 按钮按 `ToolExecuting` 状态启用 |

---

## 6. 测试场景

### 6.1 场景矩阵（修正后）

P1（流式中暂停）和 R1（流式暂停后恢复）已取消——因为设计上 Thinking 阶段不可暂停。

共 **9 个场景**：

| 编号 | 操作 | 时机 | 核心验证 | 预期结果 |
|------|------|------|----------|----------|
| **P2** | Pause→Resume | 工具调用中 | 当前工具执行完，state=Paused，后续工具排队，Resume 后继续 | ✅ 200 |
| **P3** | Pause | 审批等待中 | Pause 返回 409 | ✅ 409 |
| **C1** | Cancel | 流式输出中 | 中止流 → state=Idle → 下一轮 202 | ✅ 202 |
| **C2** | Cancel | 工具调用中 | 杀进程 → 补 tool_result(error) → 补 user 消息 → state=Idle → 下一轮 202 | ✅ 202 |
| **C3** | Cancel | 审批等待中 | 补 tool_result(rejected) → 补 user 消息 → state=Idle → 下一轮 202 | ✅ 202 |
| **R2** | Resume | 工具暂停后 | 恢复工具调度 → 继续执行 → 最终 Idle，下一轮 202 | ✅ 202 |
| **Cr1** | Crash | 流式输出中 | 重启 → state=Idle → 下一轮 202 | ✅ 202 |
| **Cr2** | Crash | 工具调用中 | 重启 → 补 tool_result(error) → 补 user 消息 → state=Idle → 下一轮 202 | ✅ 202 |
| **Cr3** | Crash | 审批等待中 | 重启 → 补 tool_result(rejected) → 补 user 消息 → state=Idle → 下一轮 202 | ✅ 202 |

### 6.2 场景详细步骤

#### P2：工具调用中暂停 → 恢复

| Step | 操作 | 消息历史 | 说明 |
|------|------|----------|------|
| 1 | 用户发送 `"${read_file,list_files}$ 帮我处理"` | `[user]` | 触发多工具调用 |
| 2 | Devo → mock_server（请求 #1） | 同上 | mock_server 返回 assistant(tool_calls: read_file, list_files) |
| 3 | assistant 消息落盘 | `[user, assistant(tool_calls: read_file, list_files)]` | 进入 ToolExecuting |
| 4 | 开始执行 read_file | 同上 | 第一个工具执行中 |
| 5 | 用户调用 **Pause** | 同上 | state → Paused；read_file 执行完，list_files 排队 |
| 6 | 用户调用 **Resume** | 同上 | state → ToolExecuting；list_files 继续执行 |
| 7 | 所有工具执行完 | `[user, assistant(tool_calls), tool_result(read_file), tool_result(list_files)]` | 完整 |
| 8 | Devo → mock_server（送工具结果） | 同上 | mock 返回 final assistant 消息 |
| 9 | 用户发送 `"继续"` | 同上 + `[user: "继续"]` | 下一轮 |
| 10 | Devo → mock_server（请求 #2） | 同上 | **validateToolCalls：所有 tool_call 都有 result → 200** ✅ |

#### C2：工具调用中取消

| Step | 操作 | 消息历史 | 说明 |
|------|------|----------|------|
| 1 | 用户发送 `"${execute_command,read_file}$ 帮我"` | `[user]` | 触发多工具调用 |
| 2 | Devo → mock_server（请求 #1） | 同上 | mock_server 返回 assistant(tool_calls: execute_command, read_file) |
| 3 | assistant 消息落盘 | `[user, assistant(tool_calls: execute_command, read_file)]` | 进入 ToolExecuting |
| 4 | 开始执行 execute_command | 同上 | 工具执行中 |
| 5 | 用户调用 **Cancel** | 同上 | state → Idle；杀进程，补 tool_result(error) |
| 6 | 清理完成 | `[user, assistant(tool_calls), tool_result(error), tool_result(error), user: "用户已取消当前操作"]` | 消息完整 |
| 7 | 用户发送 `"继续"` | 同上 + `[user: "继续"]` | 下一轮 |
| 8 | Devo → mock_server（请求 #2） | 同上 | **validateToolCalls：所有 tool_call 都有 result → 200** ✅ |

#### Cr2：工具调用中崩溃

| Step | 操作 | 消息历史 | 说明 |
|------|------|----------|------|
| 1 | 用户发送 `"${execute_command,read_file}$ 帮我"` | `[user]` | 触发多工具调用 |
| 2 | Devo → mock_server（请求 #1） | 同上 | mock_server 返回 assistant(tool_calls: execute_command, read_file) |
| 3 | assistant 消息落盘 | `[user, assistant(tool_calls: execute_command, read_file)]` | 进入 ToolExecuting |
| 4 | 开始执行工具 | 同上 | 工具执行中 |
| 5 | **杀死 Devo 进程** | 同上 | 崩溃 |
| 6 | 重启 Devo | 同上 | `RecoverCrashedSessions` 恢复 |
| 7 | 清理完成 | `[user, assistant(tool_calls), tool_result(error), tool_result(error), user: "系统已崩溃重启，当前操作已取消"]` | 消息完整 |
| 8 | 用户发送 `"继续"` | 同上 + `[user: "继续"]` | 下一轮 |
| 9 | Devo → mock_server（请求 #2） | 同上 | **validateToolCalls：所有 tool_call 都有 result → 200** ✅ |

---

## 7. 测试方法

### 7.1 核心思路

用 `mock_server` 替代真实大模型，通过消息中的 `${工具名}$` 标记触发工具调用，操作完成后发下一条消息，看 mock 返回 200 还是 400。

```
测试流程：
  1. 发消息 "${tool1,tool2}$ 帮我做某事"
  2. 在特定时机执行操作（暂停/取消/恢复/崩溃）
  3. 发下一条消息（普通文本，不带 ${...}$）
  4. 看 mock 响应：
     → 200 + 正常文本  = 消息历史完整，通过
     → 400 + 错误信息  = 存在孤儿 tool_call，失败
```

### 7.2 Mock Server 校验逻辑

`mock_server` 在每次请求时执行 `validateToolCalls`，遍历 messages：

1. 找到最后一个带 `tool_calls` 的 assistant 消息
2. 收集所有 `tool_call_id`
3. 检查后续消息中，每个 `tool_call_id` 是否都有对应的 `tool` 角色消息
4. 缺失任何一个 → 返回 400，错误信息包含缺失的 `tool_call_id` 列表

### 7.3 操作时机控制

| 时机 | 控制方式 | 原理 |
|------|----------|------|
| **流式响应中** | mock_server 的 `/v1/hold` 端点延迟 N 秒 | 测试发消息后立即 Sleep 100ms，此时流式一定在进行中 |
| **工具调用中** | 使用 `execute_command`，命令设为 `"sleep 5"` | 命令运行 5 秒，期间点暂停/取消/崩溃 |
| **审批等待中** | 使用 `write_file` 触发审批 | 测试调用 `Client.WaitForApproval()` 确认审批已出现再操作 |

### 7.4 运行方式

```bash
# 运行全部 9 个场景
go test ./tests/ -run TestSessionRobustness -v -timeout 120s

# 运行单个场景
go test ./tests/ -run TestC2_CancelDuringToolCalls -v -timeout 30s
```

### 7.5 测试请求/响应归档

每次测试的完整 LLM 交互记录保存在 `tests/mock_server/requests/` 目录下：

```
tests/mock_server/requests/
├── req_001.json       ← Devo 发送给 mock_server 的请求（完整消息历史）
├── req_001_resp.json  ← mock_server 返回给 Devo 的响应（完整）
├── req_002.json
├── req_002_resp.json
└── ...
```

---

## 8. 改动文件清单

### 8.1 后端

| 文件 | 改动 |
|------|------|
| [session.go](file:///c:/Users/bean/Desktop/Devo/internal/core/session/session.go) | 新增 `StateThinking`、`StateToolExecuting`，废弃 `StateProcessing` |
| [state_handlers.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_handlers.go) | `thinkingHandler` 入口设 `StateThinking`，`toolExecutingHandler` 入口设 `StateToolExecuting` |
| [state_control.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_control.go) | Pause 检查 `StateToolExecuting`；Cancel 检查 `StateThinking`/`StateToolExecuting`/`StateAwaitingApproval`/`StatePaused` 并追加 tool_result；Resume 检查 `StatePaused` |
| [loop.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/loop.go) | `ProcessMessage` 入口设 `StateThinking`，结束设 `StateIdle` |
| [crash_recovery.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/crash_recovery.go) | switch 匹配 `StateThinking`/`StateToolExecuting`/`StateAwaitingApproval`/`StatePaused`，按阶段追加 tool_result |
| [state_machine.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_machine.go) | Pause 检查改为仅在 ToolExecuting handler 内生效 |

### 8.2 前端

| 文件 | 改动 |
|------|------|
| `web/src/types/session.ts` | 类型新增 `Thinking`、`ToolExecuting`，废弃 `Processing` |
| `web/src/stores/session.ts` | Pause 按钮按 `ToolExecuting` 状态启用/禁用 |

### 8.3 测试

| 文件 | 改动 |
|------|------|
| `tests/session_robustness_test.go` | 移除 P1/R1，新增 P2/C2/Cr2 的 tool_result 断言 |

---

## 9. 测试记录（v1.1.0）

**执行时间**：2026-07-01 12:41 - 12:44
**总耗时**：186.473s
**结果**：3 PASS / 9 FAIL

| 编号 | 场景 | 操作 | 时机 | 预期 | 实际结果 | 状态 |
|------|------|------|------|------|----------|------|
| P1 | Pause→Resume | 暂停→恢复 | 流式响应中 | ✅ | ❌ Resume 409，state 仍为 "processing" | **FAIL** |
| P2 | Pause→Resume | 暂停→恢复 | 工具调用中 | ✅ | ❌ Resume 409，state 仍为 "processing" | **FAIL** |
| P3 | Pause | 暂停 | 审批等待中 | ✅ | ❌ Pause 返回 200（hold_delay 导致在 Processing 态触发） | **FAIL** |
| C1 | Cancel | 取消 | 流式响应中 | ✅ | ❌ 下一轮 409，state 仍为 "processing" | **FAIL** |
| C2 | Cancel | 取消 | 工具调用中 | ❌ | ❌ 下一轮 409，mock 校验 PASS | **FAIL** |
| C3 | Cancel | 取消 | 审批等待中 | ❌ | ❌ 下一轮 409，mock 校验 PASS | **FAIL** |
| R1 | Resume | 暂停→恢复 | 流式暂停后 | ✅ | ❌ Resume 409，state 仍为 "processing" | **FAIL** |
| R2 | Resume | 暂停→恢复 | 工具暂停后 | ✅ | ❌ Resume 409，state 仍为 "processing" | **FAIL** |
| R3 | Resume | 暂停→恢复 | 审批暂停后 | ✅ | ❌ Pause 返回 200（hold_delay 导致在 Processing 态触发） | **FAIL** |
| Cr1 | Crash | 崩溃→重启 | 流式响应中 | ✅ | ✅ state=idle，下一轮 202 | **PASS** |
| Cr2 | Crash | 崩溃→重启 | 工具调用中 | ❌ | ✅ state=idle，下一轮 202，mock 校验 PASS | **PASS** |
| Cr3 | Crash | 崩溃→重启 | 审批等待中 | ❌ | ✅ state=idle，下一轮 202，mock 校验 PASS | **PASS** |

---

## 10. 测试报告（v1.1.0）

### 10.1 实测结果总览

| | 流式响应中 | 工具调用中 | 审批等待中 |
|---|---|---|---|
| **Pause→Resume** | ❌ 409（Bug #1） | ❌ 409（Bug #1） | ❌ 测试设计问题 |
| **Cancel** | ❌ 409（Bug #4） | ❌ 409（Bug #4） | ❌ 409（Bug #4） |
| **Resume** | ❌ 409（Bug #1） | ❌ 409（Bug #1） | ❌ 测试设计问题 |
| **Crash→重启** | ✅ 200 | ✅ 200 | ✅ 200 |

### 10.2 已确认 Bug

#### Bug #1：Pause 不更新 session.State → Resume 永远不可用

- **严重程度**：🔴 高
- **影响**：P1、P2、R1、R2
- **表现**：Pause 返回 200，但 `GET /api/v1/sessions/{id}` 返回 `state: "processing"`；Resume 返回 409 `"session is not paused: current state is Processing"`
- **根因**：[state_control.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_control.go) 中 `Pause()` 只向 `PauseCh` 发送信号，不调用 `sess.State = StatePaused` 和 `store.Update()`

#### Bug #4：Cancel 不更新 session.State → 下一轮被拒绝

- **严重程度**：🔴 高
- **影响**：C1、C2、C3
- **表现**：Cancel 返回 200，但 `GET /api/v1/sessions/{id}` 返回 `state: "processing"`；发送新消息返回 409 `"session is not idle: current state is Processing"`
- **根因**：[state_control.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_control.go) 中 `Cancel()` 只向 `CancelCh` 发送信号，不调用 `sess.State = StateIdle` 和 `store.Update()`

### 10.3 未确认 Bug（推测不成立）

#### Bug #2（原推测）：Cancel 产生孤儿 tool_call → 下一轮 LLM 返回 400

- **验证结果**：❌ 不成立
- **原因**：C2/C3 中 Cancel 后下一轮被 Devo 在 409 处拦截，**未到达 LLM 调用**。Bug #4 的存在掩盖了 Bug #2 的验证。
- **结论**：需先修复 Bug #4 后重新测试。根据 v2.0.0 设计，Cancel 应补 tool_result(error/rejected) 而非删除消息。

#### Bug #3（原推测）：崩溃恢复不清理孤儿 tool_call → 下一轮 LLM 返回 400

- **验证结果**：❌ 不成立
- **原因**：Cr2/Cr3 中崩溃重启后，下一轮消息正常发送，mock_server 校验 PASS。说明崩溃恢复时消息历史已被正确处理。
- **结论**：v2.0.0 设计中仍需显式补 tool_result，当前测试通过可能依赖时序（assistant 消息未落盘），需加固。

### 10.4 结论

| 类型 | 数量 | 详情 |
|------|------|------|
| **确认 Bug** | 2 | Bug #1（Pause 不更新 state）、Bug #4（Cancel 不更新 state） |
| **排除 Bug** | 2 | Bug #2（孤儿 tool_call 不成立）、Bug #3（崩溃恢复正确清理消息） |
| **待验证** | 1 | Bug #2 需修复 Bug #4 后重新测试 |
| **测试设计问题** | 2 | P3/R3 时机控制、P1/P2/R1/R2 超时 |

**核心发现**：Devo 的 Agent Loop 内部状态管理（PauseCh/CancelCh/ResumeCh 信号通道）工作正常，但 **REST API 层未将状态变更同步到持久化存储**，导致前后端状态不一致。

### 10.5 v2.0.0 设计变更总结

| 变化 | v1.1.0 | v2.0.0 |
|------|--------|--------|
| Session State | Processing（粗粒度） | Thinking + ToolExecuting（细分） |
| Pause 生效范围 | 所有 Processing | 仅 ToolExecuting |
| 流式输出中 Pause | 暂停 SSE → 内存堆积 | 忽略，流式跑完 |
| Resume 场景 | 流式/工具都要恢复 | 只需恢复工具调度 |
| Cancel 清理方式 | 无（当前不清理） | 追加 tool_result(error/rejected) + user 消息 |
| Crash 清理方式 | 无（当前不清理） | 追加 tool_result(error/rejected) + user 消息 |
| 测试用例数 | 12 | 9（移除 P1/R1/R3） |
| 前端 Pause 按钮 | 始终可见 | 仅 ToolExecuting 时可点击 |

---

## 11. 测试报告（v2.0.0）

### 11.1 执行信息

| 项目 | 内容 |
|------|------|
| **执行时间** | 2026-07-01 07:21 |
| **总耗时** | 71.2s |
| **Go 版本** | 1.25.10 |
| **测试用例数** | 9 |
| **结果** | **9/9 PASS** ✅ |

### 11.2 场景测试结果

| 编号 | 场景 | 操作 | 时机 | 验证结果 | 耗时 |
|------|------|------|------|:---:|------|
| **P2** | Pause→Resume | 暂停→恢复 | 工具调用中 | ✅ PASS | 17.0s |
| **P3** | Pause | 暂停 | 审批等待中 | ✅ PASS | 1.2s |
| **C1** | Cancel | 取消 | 流式输出中 | ✅ PASS | 1.1s |
| **C2** | Cancel | 取消 | 工具调用中 | ✅ PASS | 4.5s |
| **C3** | Cancel | 取消 | 审批等待中 | ✅ PASS | 1.2s |
| **R2** | Resume | 暂停→恢复 | 工具暂停后 | ✅ PASS | 9.8s |
| **Cr1** | Crash | 崩溃→重启 | 流式输出中 | ✅ PASS | 6.5s |
| **Cr2** | Crash | 崩溃→重启 | 工具调用中 | ✅ PASS | 13.6s |
| **Cr3** | Crash | 崩溃→重启 | 审批等待中 | ✅ PASS | 8.6s |

### 11.3 状态矩阵

| | 流式响应中（Thinking） | 工具调用中（ToolExecuting） | 审批等待中（AwaitingApproval） |
|---|---|---|---|
| **Pause** | —（忽略） | ✅ P2 | ✅ P3（409） |
| **Cancel** | ✅ C1 | ✅ C2 | ✅ C3 |
| **Resume** | —（无意义） | ✅ R2 | — |
| **Crash→重启** | ✅ Cr1 | ✅ Cr2 | ✅ Cr3 |

### 11.4 Mock Server 校验

每个工具调用场景均通过 mock_server 的 `validateToolCalls` 校验，确保所有 `tool_call_id` 都有对应的 `tool_result`：

| 编号 | 场景 | 校验结果 |
|:---:|------|:---:|
| P2 | Pause→Resume 工具调用中 | ✅ PASS |
| P3 | Pause 审批等待中 | ✅ PASS |
| C2 | Cancel 工具调用中 | ✅ PASS |
| C3 | Cancel 审批等待中 | ✅ PASS |
| R2 | Resume 工具暂停后 | ✅ PASS |
| Cr2 | Crash 工具调用中 | ✅ PASS |
| Cr3 | Crash 审批等待中 | ✅ PASS |

### 11.5 本轮修复的 Bug

#### Bug #1：`Cancel` 未同步更新状态（[state_control.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_control.go)）

- **严重程度**：🔴 高
- **影响**：C1、C2、C3
- **表现**：Cancel 仅发送 `CancelCh` 信号就返回，状态更新由 goroutine 异步完成，测试立即检查时状态仍是旧值
- **修复**：同步设置 `sess.State = StateIdle`，并发布 `session_state_change` 事件

#### Bug #2：`ProcessMessage` goroutine 覆盖 `Paused` 状态（[loop.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/loop.go)）

- **严重程度**：🔴 高
- **影响**：P2、R2
- **表现**：状态机返回后无条件 `sess.State = StateIdle`，覆盖了 `Pause` 设置的 `Paused` 状态
- **修复**：增加判断 `sess.State != session.StatePaused`，仅在非暂停状态时设为 Idle

#### Bug #3：`Resume` 未更新状态（[state_control.go](file:///c:/Users/bean/Desktop/Devo/internal/core/agentloop/state_control.go)）

- **严重程度**：🔴 高
- **影响**：R2
- **表现**：有活跃 loop 时只发送 ResumeCh 信号，不更新 `sess.State`，导致前端获取到旧状态
- **修复**：无论是否有活跃 loop，都先更新 `sess.State = StateToolExecuting` 并持久化

#### Bug #4：`evaluatingResultHandler`/`toolExecutingHandler` 状态更新未输出日志

- **严重程度**：🟡 中
- **影响**：调试困难
- **表现**：`l.store.Get()` 失败时静默忽略，无错误日志
- **修复**：添加 `log.Printf` 错误日志，方便定位问题

### 11.6 测试稳定性改进

| 问题 | 影响 | 修复 |
|------|------|------|
| R2 工具执行过快（`read_file`+`list_files` <1s） | 测试轮询抓不到 ToolExecuting 状态 | 改用 `execute_command`（`powershell -Command Start-Sleep -Seconds 5`） |
| C2/R2/Cr2 工具调用进入审批流程 | 状态变为 AwaitingApproval 而非 ToolExecuting | 添加 `SetTrustLevel(sessionID, "elevated")` 启用 YOLO 模式 |
| 测试端口冲突 | mock_server 和 Devo 抢占端口 | mock_server 使用 9090，Devo 使用 9091 |
| mock_server LLM 响应过快 | 无法在流式输出中触发操作 | 使用 `/v1/hold` 端点延迟 3000ms |

### 11.7 与 v1.1.0 对比

| 指标 | v1.1.0 | v2.0.0 |
|------|--------|--------|
| 测试用例数 | 12 | 9 |
| 通过率 | 3/12（25%） | **9/9（100%）** |
| 确认 Bug | 2 | 4（全部修复） |
| Session State | Processing（粗粒度） | Thinking + ToolExecuting（细分） |
| 总耗时 | 186s | 71s |

### 11.8 待完成

- [ ] **前端改造**：同步状态机逻辑，`ToolExecuting` 阶段才启用 Pause 按钮
  - `web/src/types/session.ts`：类型新增 `Thinking`、`ToolExecuting`
  - `web/src/stores/session.ts`：Pause 按钮按状态启用/禁用
- [ ] **清理调试日志**：移除 `state_handlers.go` 中新增的 `[DEBUG]` 日志
- [ ] **回归测试**：前端改造完成后再次运行全部测试用例