# Devo TUI 交互层架构设计文档

**版本**：3.0.0

**定位**：基于 bubbletea 的全屏终端交互界面（TUI），作为 Devo 编码代理系统的第一方交互入口。TUI 与 Devo 服务运行在同一进程内，服务以协程方式内嵌启动，通过 HTTP 协议（localhost 随机端口）驱动所有操作。用户在项目目录中执行 `devo --tui` 即可开始与 Agent 对话。

---

## 1. 总体架构

### 1.1 进程模型

TUI 不重新实现核心逻辑，而是与 Devo 服务器**运行在同一进程内**，通过 **goroutine** 内嵌 HTTP 服务，TUI 通过 localhost HTTP 与之通信。

```
┌──────────────────────────────────────────────────────────┐
│                    单一进程 (devo --tui)                   │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │  main()                                            │  │
│  │  ├── 初始化 DB、工具、LLM、AgentLoop、REST Handler │  │
│  │  ├── 分配随机端口                                   │  │
│  │  ├── go server.ListenAndServe()  ← 协程启动 HTTP   │  │
│  │  ├── 等待服务就绪 (poll /api/v1/sessions)          │  │
│  │  └── tui.Launch(baseURL)                           │  │
│  └────────────────────────────────────────────────────┘  │
│                         │                                │
│  ┌──────────────────────┼──────────────────────────────┐ │
│  │              TUI (bubbletea)                        │ │
│  │  ┌──────────────┐   │   ┌───────────┐              │ │
│  │  │  API Client  │←──┼──→│ HTTP 服务  │              │ │
│  │  │  (REST 调用) │   │   │ (goroutine)│              │ │
│  │  └──────────────┘   │   └───────────┘              │ │
│  │  ┌──────────────┐   │                               │ │
│  │  │  SSE Client  │←──┼──→ SSE 事件流                 │ │
│  │  │  (事件流)    │   │   localhost:<随机端口>         │ │
│  │  └──────────────┘   │                               │ │
│  │                     │                               │ │
│  │  ┌──────────────────────────────────────────────┐  │ │
│  │  │         TUI 渲染层 (bubbletea)               │  │ │
│  │  │  ChatScreen · SessionSidebar · ApprovalModal │  │ │
│  │  └──────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

**核心原则**：
- 单一进程，单一入口 `cmd/devo/main.go`，通过 `--tui` flag 切换模式
- Devo Server 以 **goroutine** 方式内嵌运行，与 TUI 共享进程生命周期
- 通信仅通过 HTTP（REST + SSE），不共享内存，不直接引用内部包
- 端口随机分配，避免冲突，支持多实例并行
- 会话工作目录**自动设为 TUI 启动时的当前目录**，无需用户指定
- 服务初始化代码**只写一次**（在 main.go），TUI 不重复服务初始化逻辑

### 1.2 启动流程

```
用户: cd /path/to/my-project
用户: devo --tui
         │
         ▼
┌─────────────────────────────────────┐
│ 1. 初始化服务组件（main.go）         │
│    · 打开 SQLite 数据库              │
│    · 注册工具集                      │
│    · 初始化 LLM Client              │
│    · 创建 AgentLoop                 │
│    · 创建 REST Handler + Mux        │
├─────────────────────────────────────┤
│ 2. 分配随机端口                      │
│    port = net.Listen(":0")          │
│    例如: 52341                      │
├─────────────────────────────────────┤
│ 3. 协程启动 HTTP 服务                │
│    go server.ListenAndServe()       │
├─────────────────────────────────────┤
│ 4. 等待服务就绪                      │
│    poll GET /api/v1/sessions        │
│    超时 10s，失败则 log.Fatal       │
├─────────────────────────────────────┤
│ 5. 启动 TUI 主界面                   │
│    tui.Launch(baseURL)             │
│    bubbletea 接管终端                │
│    TUI Init() 自动创建会话           │
│    用户可立即开始对话                 │
└─────────────────────────────────────┘
```

**退出流程**：
```
用户: Ctrl+Q
    │
    ▼
1. TUI 退出，bubbletea 恢复终端
2. main() 中 server.Close() 关闭 HTTP 服务
3. store.Close() 关闭数据库
4. 进程结束（无残留子进程）
```

---

## 2. 技术选型

| 库 | 用途 | 说明 |
| :--- | :--- | :--- |
| `github.com/charmbracelet/bubbletea` | TUI 框架 | Elm 架构，Model/Update/View 分离 |
| `github.com/charmbracelet/bubbles` | TUI 组件库 | viewport、textarea、table、spinner 等即用组件 |
| `github.com/charmbracelet/lipgloss` | 终端样式 | 声明式样式系统，支持自适应宽度 |
| `net` (标准库) | 端口分配 | `net.Listen("tcp", "127.0.0.1:0")` 获取随机端口 |
| `net/http` (标准库) | HTTP 服务/客户端 | 协程内嵌服务 + REST API 调用 + SSE 长连接 |

---

## 3. 屏幕设计

### 3.1 整体布局

```
┌──────────────────────────────────────────────────────────┐
│ Devo · my-project · Idle · 12.5K tok · :52341 ✓          │  ← StatusBar
├────────────┬─────────────────────────────────────────────┤
│            │                                             │
│  Sessions  │  ┌───────────────────────────────────────┐  │
│            │  │  [User] 14:32                          │  │
│  ● my-proj │  │  帮我修复 utils.go 中的空指针问题       │  │
│    bug-fix │  │                                        │  │
│    refactor│  │  [Thinking] 正在分析代码...             │  │
│            │  │                                        │  │
│  [+ New]   │  │  ┌ Tool: read_file ───────────────┐   │  │
│            │  │  │ utils.go · 156 lines · ✓        │   │  │
│            │  │  └─────────────────────────────────┘   │  │
│            │  │                                        │  │
│            │  │  ┌ Tool: write_file ──────────────┐   │  │
│            │  │  │ utils.go                        │   │  │
│            │  │  │ + if x == nil { return }        │   │  │
│            │  │  │ Approved ✓ · 1.2s               │   │  │
│            │  │  └─────────────────────────────────┘   │  │
│            │  │                                        │  │
│            │  │  [Assistant] 14:32                      │  │
│            │  │  已修复 utils.go 中的空指针问题，        │  │
│            │  │  在 oldFunc() 开头添加了 nil 检查...    │  │
│            │  │                                        │  │
│            │  └───────────────────────────────────────┘  │
│            │  ┌───────────────────────────────────────┐  │
│            │  │ > 继续优化这个函数                      │  │
│            │  └───────────────────────────────────────┘  │
├────────────┴─────────────────────────────────────────────┤
│ ^S sessions  ^N new  ^C cancel  ^P pause  ^Q quit        │  ← HelpBar
└──────────────────────────────────────────────────────────┘
```

### 3.2 屏幕组件树

```
App (bubbletea.Model)
│
├── StatusBar                         顶部状态栏
│   ├── AppName ("Devo")
│   ├── SessionTitle (目录名)
│   ├── SessionState (Idle/Processing/AwaitingApproval/...)
│   ├── TokenUsage (累计消耗)
│   └── ServerIndicator (端口号 + 连接状态)
│
├── MainContent                       主内容区
│   ├── SessionSidebar                左侧会话列表 (宽 20 列, 可折叠)
│   │   ├── SessionList (可滚动)
│   │   │   ├── SessionItem (当前选中高亮 ●)
│   │   │   └── SessionItem
│   │   └── NewSessionButton ("[+ New]")
│   │
│   └── ChatView                      右侧聊天主区
│       ├── MessageViewport (可滚动)
│       │   ├── UserBubble
│       │   ├── AssistantBubble
│       │   ├── SystemNotice (灰色)
│       │   ├── ToolCallCard (可折叠)
│       │   │   ├── Header: tool_name · status · duration
│       │   │   ├── Body: params summary / result summary
│       │   │   └── ExpandHint: "Enter to expand"
│       │   └── ApprovalCard
│       │       ├── Header: operation_type · risk_level
│       │       ├── DiffView (语法高亮 diff)
│       │       └── AutoApproved badge (若自动批准)
│       └── InputArea
│           ├── TextArea (多行输入, 最小 3 行)
│           └── SendHint ("Enter to send · Shift+Enter newline")
│
├── ApprovalModal                     审批弹窗 (覆盖层)
│   ├── Overlay (半透明遮罩)
│   ├── ModalBox
│   │   ├── Title: "⚠ Approval Required"
│   │   ├── OperationType + RiskLevel badge
│   │   ├── DiffView (完整 diff, 可滚动)
│   │   ├── CommandPreview (若为 execute_command)
│   │   └── ActionBar: "[Y] Approve  [N] Reject  [D] View Full Diff"
│
├── HelpBar                           底部快捷键栏
│   └── KeyBindings (根据当前上下文动态显示)
│
└── Overlays (按优先级叠加)
    ├── ErrorToast (临时浮动提示, 3s 自动消失)
    ├── InfoToast (操作确认提示)
    └── Spinner (Processing 状态时在输入区上方显示)
```

---

## 4. 组件详细设计

### 4.1 App — 顶层模型

```go
// app.go

type App struct {
    // HTTP 通信
    apiClient *APIClient
    sseClient *SSEClient

    // 会话
    sessions      []SessionInfo
    activeSession *SessionInfo
    messages      []Message

    // 子组件
    statusBar     StatusBar
    sidebar       SessionSidebar
    chatView      ChatView
    helpBar       HelpBar
    approvalModal *ApprovalModal

    // 状态
    state        AppState
    showSidebar  bool
    width        int
    height       int
    err          error
}

type AppState int

const (
    StateReady             AppState = iota  // 就绪，等待输入
    StateProcessing                         // 消息已发送，等待 Agent 响应
    StateAwaitingApproval                   // 审批弹窗已打开
    StateQuitting                           // 正在退出
)
```

**Update 消息类型**：

```go
// messages/messages.go

// 来自 SSE 的事件消息
type SSEEvent struct {
    Type string
    Data map[string]interface{}
}

// API 调用完成的响应
type APIResponse struct {
    Kind     string          // "session_created", "message_sent", "approval_done", ...
    Data     interface{}
    Err      error
}

// 审批弹窗中的用户决策
type ApprovalDecision struct {
    Approved bool
}

// 定时器事件
type TickMsg time.Time
```

**Update 核心逻辑**：

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        a.width = msg.Width
        a.height = msg.Height
        return a, nil

    case tea.KeyMsg:
        if a.state == StateAwaitingApproval {
            return a.handleApprovalKey(msg)
        }
        return a.handleGlobalKey(msg)

    case messages.SSEEvent:
        return a.handleSSEEvent(msg)

    case messages.APIResponse:
        return a.handleAPIResponse(msg)
    }
    return a, nil
}
```

### 4.2 Launcher — 启动入口

```go
// launcher.go

// Launch 接收 baseURL，初始化 TUI 并交给 bubbletea 运行
func Launch(baseURL string) {
    app, err := NewAppWithURL(baseURL)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to initialize TUI: %v\n", err)
        os.Exit(1)
    }

    p := tea.NewProgram(app, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
    }
}
```

HTTP 服务启动、端口分配、就绪等待均在 [main.go](file:///c:/Users/bean/Desktop/Devo/cmd/devo/main.go) 中完成，Launcher 只负责接收 `baseURL` 并启动 TUI。

### 4.3 APIClient — REST 调用封装

```go
// apiclient.go

type APIClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
    return &APIClient{
        baseURL:    strings.TrimRight(baseURL, "/"),
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

// 会话管理
func (c *APIClient) CreateSession(workingDir, title string) (*SessionInfo, error)
func (c *APIClient) ListSessions() ([]SessionInfo, error)
func (c *APIClient) GetSession(id string) (*SessionInfo, error)

// 消息
func (c *APIClient) SendMessage(sessionID, content string) error
func (c *APIClient) GetMessages(sessionID string, limit, offset int) ([]Message, error)

// 状态控制
func (c *APIClient) Pause(sessionID string) error
func (c *APIClient) Resume(sessionID string) error
func (c *APIClient) Cancel(sessionID string) error
func (c *APIClient) Complete(sessionID string) error
func (c *APIClient) Archive(sessionID string) error

// 审批
func (c *APIClient) Approve(sessionID, approvalID string) error
func (c *APIClient) Reject(sessionID, approvalID string) error
func (c *APIClient) SetTrustLevel(sessionID, level string) error
func (c *APIClient) SetApprovalPolicy(sessionID, opType, level string) error

// 其他
func (c *APIClient) GetFiles(sessionID string) ([]FileInfo, error)
```

### 4.4 SSEClient — SSE 事件消费

```go
// sseclient.go

type SSEClient struct {
    sessionID string
    eventCh   chan SSEEvent
    errCh     chan error
    done      chan struct{}
}

func (s *SSEClient) Connect(sessionID string) error
func (s *SSEClient) Disconnect()
func (s *SSEClient) Events() <-chan SSEEvent
func (s *SSEClient) Errors() <-chan error
```

解析 SSE 协议：`id:` / `event:` / `data:` 行，空行表示事件结束，通过 `eventCh` 发送给 bubbletea 主循环。

### 4.5 消息渲染规则

| 消息角色 | 样式 | 前缀 |
| :--- | :--- | :--- |
| `user` | 青色边框，右对齐 | `[You]` |
| `assistant` | 默认色，左对齐 | `[Assistant]` |
| `system` | 灰色，左对齐 | `[System]` |
| `tool` | 黄色边框，缩进 | `[Tool Result]` |
| 工具调用卡片 | 可折叠块，绿色(成功)/红色(失败) | `┌ Tool: read_file ─┐` |
| 审批卡片 | 红色边框，橙色标题 | `⚠ Approval Required` |

### 4.6 审批弹窗

```go
// approvalmodal.go

type ApprovalModal struct {
    visible  bool
    request  *ApprovalRequest
    viewport viewport.Model
}

type ApprovalRequest struct {
    ApprovalID     string
    OperationType  string
    RiskLevel      string
    Summary        string
    Diff           string
    CommandPreview string
}

func (m *ApprovalModal) Update(msg tea.Msg) (ApprovalModal, tea.Cmd) {
    if !m.visible {
        return *m, nil
    }
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "y", "Y":
            return *m, func() tea.Msg { return messages.ApprovalDecision{Approved: true} }
        case "n", "N", "esc":
            return *m, func() tea.Msg { return messages.ApprovalDecision{Approved: false} }
        case "d", "D":
            m.viewport.GotoTop()
        }
    }
    return *m, nil
}
```

---

## 5. 包结构

### 5.1 目录布局

```
cmd/
└── devo/main.go                   # 唯一入口（--tui flag 切换模式）

internal/interfaces/
├── rest/                          # 现有：REST API 处理器（不变）
└── tui/                           # 新增：TUI 交互层
    ├── launcher.go                # Launch()：接收 baseURL，启动 TUI
    ├── app.go                     # bubbletea App 顶层 Model
    ├── apiclient.go               # APIClient：REST API 调用封装
    ├── sseclient.go               # SSEClient：SSE 事件流消费
    │
    ├── components/                # 可复用 UI 组件
    │   ├── statusbar.go           # 顶部状态栏
    │   ├── sidebar.go             # 会话列表侧栏
    │   ├── chatview.go            # 聊天主区
    │   ├── messageview.go         # 消息列表
    │   ├── inputarea.go           # 输入区域
    │   ├── approvalmodal.go       # 审批弹窗
    │   ├── helpbar.go             # 底部快捷键栏
    │   ├── toast.go               # 浮动提示
    │   ├── spinner.go             # 加载动画
    │   └── styles.go              # lipgloss 全局样式定义
    │
    ├── messages/                  # bubbletea 自定义消息类型
    │   └── messages.go            # SSEEvent, APIResponse, ApprovalDecision, ...
    │
    └── types/                     # 数据传输类型
        └── types.go               # SessionInfo, Message, ApprovalRequest, ...
```

### 5.2 依赖关系

```
cmd/devo/main.go
    ├── --tui 模式:
    │   ├── go server.ListenAndServe()     ← 协程启动 HTTP 服务
    │   └── tui.Launch(baseURL)            ← 传入 URL 启动 TUI
    └── 默认模式:
        └── http.ListenAndServe()          ← 直接阻塞运行

internal/interfaces/tui/
    ├── launcher.go     → app.go
    ├── app.go          → apiclient.go, sseclient.go, components/*
    ├── apiclient.go    → net/http (标准库)
    ├── sseclient.go    → net/http (标准库)
    ├── components/*    → bubbletea, bubbles, lipgloss
    ├── messages/*      → (纯类型，无外部依赖)
    └── types/*         → (纯类型，无外部依赖)
```

**关键约束**：
- `internal/interfaces/tui/` **不依赖** `internal/core/`、`internal/taskexec/`、`internal/storage/`
- 与现有 `internal/interfaces/rest/` 并列，互不引用
- `cmd/devo/main.go` 是唯一入口，所有服务初始化代码集中于此

### 5.3 入口文件

```go
// cmd/devo/main.go
package main

import (
    "flag"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "time"

    "devo/internal/core/agentloop"
    "devo/internal/interfaces/rest"
    "devo/internal/interfaces/tui"
    "devo/internal/storage/sqlite"
    "devo/internal/taskexec/llmclient/providers"
    "devo/internal/taskexec/tools"
)

func main() {
    tuiMode := flag.Bool("tui", false, "Launch TUI mode")
    flag.Parse()

    // ... 初始化 DB、工具、LLM、AgentLoop、Handler ...

    if *tuiMode {
        // 分配随机端口，协程启动 HTTP 服务
        tuiPort, _ := findFreePort()
        server := &http.Server{
            Addr:    fmt.Sprintf("127.0.0.1:%d", tuiPort),
            Handler: mux,
        }
        go server.ListenAndServe()

        baseURL := fmt.Sprintf("http://127.0.0.1:%d", tuiPort)
        waitForReady(baseURL, 10*time.Second)

        tui.Launch(baseURL)   // 启动 TUI，阻塞直到用户退出

        server.Close()
        store.Close()
        return
    }

    // 默认模式：直接启动 HTTP 服务
    log.Printf("Devo server starting on :%s", port)
    http.ListenAndServe(":"+port, mux)
}
```

### 5.4 go.mod 新增依赖

```
github.com/charmbracelet/bubbletea v1.x
github.com/charmbracelet/bubbles v0.x
github.com/charmbracelet/lipgloss v1.x
```

---

## 6. 数据流设计

### 6.1 发送消息 → 接收回复

```
用户在 InputArea 按 Enter
    │
    ▼
ChatView 产生 tea.KeyMsg{Enter}
    │
    ▼
App.Update() 识别为消息提交
    │
    ├─→ 1. 切换到 StateProcessing
    ├─→ 2. 启动 SSE 连接（若未连接）
    │      GET /api/v1/sessions/{id}/events
    │
    └─→ 3. 发送 API 请求（异步 Cmd）
           POST /api/v1/sessions/{id}/messages
           Body: {"content": "..."}
           │
           ▼
       APIResponse{Kind: "message_sent"}
           │
           ▼
       App.Update() 收到确认，开始等待 SSE 事件
           │
           ▼
       SSE 事件流到达（SSEClient goroutine → SSEEvent）
           │
           ├─→ "thinking"           → 追加到消息列表，显示 spinner
           ├─→ "tool_call_request"  → 追加 ToolCallCard
           ├─→ "tool_result"        → 更新 ToolCallCard 状态
           ├─→ "approval_required"  → 切换到 StateAwaitingApproval，打开弹窗
           ├─→ "approval_auto"      → 追加系统通知
           ├─→ "message_complete"   → 追加 AssistantBubble
           ├─→ "token_usage"        → 更新 StatusBar Token 计数
           └─→ "session_state_change"
                  │
                  ├─→ reason: "completed" → 切换到 StateReady
                  └─→ reason: "tool_limit_reached"
                         → 追加系统提示，等待用户输入
```

### 6.2 审批交互

```
SSE 事件: approval_required
    │
    ▼
App.Update() 收到 SSEEvent
    │
    ├─→ 1. 解析审批请求详情
    │      (approval_id, operation_type, risk_level, diff, ...)
    │
    ├─→ 2. 切换到 StateAwaitingApproval
    │
    └─→ 3. approvalModal.Show(request)
           │
           ▼
       用户按 Y 或 N
           │
           ▼
       ApprovalModal.Update() 产生 ApprovalDecision
           │
           ▼
       App.Update() 收到决策
           │
           ├─→ Approved = true:
           │      POST /api/v1/sessions/{id}/approve/{approval_id}
           │      → 关闭弹窗，切换回 StateProcessing
           │
           └─→ Approved = false:
                  POST /api/v1/sessions/{id}/approve/{approval_id}
                  (body: decision=reject)
                  → 关闭弹窗，显示拒绝 toast
```

### 6.3 会话切换

```
用户在 SessionSidebar 按 Enter
    │
    ▼
SessionSidebar.Update() 产生 switchSessionCmd
    │
    ▼
App.Update()
    │
    ├─→ 1. 断开当前 SSE 连接
    ├─→ 2. GET /api/v1/sessions/{new_id}
    ├─→ 3. GET /api/v1/sessions/{new_id}/messages?limit=50
    ├─→ 4. 更新 activeSession + messages
    ├─→ 5. 重新连接 SSE
    └─→ 6. MessageViewport 滚动到底部
```

### 6.4 新建会话

```
用户按 Ctrl+N
    │
    ▼
自动使用当前工作目录创建会话
    │
    ├─→ 工作目录：自动填入当前工作目录
    └─→ 标题：自动填入目录名
           │
           ▼
       POST /api/v1/sessions
       Body: {"working_directory": "<当前目录>", "title": "<目录名>"}
           │
           ▼
       更新 sessions 列表，切换到新会话
```

---

## 7. 键盘快捷键

### 7.1 全局快捷键

| 快捷键 | 说明 |
| :--- | :--- |
| `Ctrl+S` | 切换会话列表侧栏显示/隐藏 |
| `Ctrl+N` | 新建会话（使用当前目录） |
| `Ctrl+C` | 取消当前 Agent 操作 |
| `Ctrl+P` | 暂停/恢复当前会话 |
| `Ctrl+Q` | 退出（关闭子进程并退出） |
| `Ctrl+L` | 清屏（重新渲染） |

### 7.2 聊天区快捷键

| 快捷键 | 说明 |
| :--- | :--- |
| `Enter` | 发送消息 |
| `Shift+Enter` | 输入框内换行 |
| `Esc` | 取消输入焦点 |
| `↑` / `↓` | 消息历史滚动 |
| `PgUp` / `PgDn` | 消息翻页 |
| `Home` / `End` | 跳到消息开头/末尾 |
| `Enter` (在 ToolCallCard 上) | 展开/折叠工具调用详情 |

### 7.3 会话列表快捷键

| 快捷键 | 说明 |
| :--- | :--- |
| `↑` / `↓` 或 `j` / `k` | 移动光标 |
| `Enter` | 切换到选中会话 |
| `d` | 删除选中会话（确认后） |

### 7.4 审批弹窗快捷键

| 快捷键 | 说明 |
| :--- | :--- |
| `Y` | 批准 |
| `N` 或 `Esc` | 拒绝 |
| `D` | 展开完整 diff |

---

## 8. 样式系统

### 8.1 色彩体系

```go
// styles.go

var (
    ColorPrimary   = lipgloss.Color("#7C3AED")  // 紫色，主色调
    ColorSuccess   = lipgloss.Color("#10B981")  // 绿色，成功
    ColorWarning   = lipgloss.Color("#F59E0B")  // 橙色，警告
    ColorDanger    = lipgloss.Color("#EF4444")  // 红色，危险/错误
    ColorInfo      = lipgloss.Color("#3B82F6")  // 蓝色，信息
    ColorMuted     = lipgloss.Color("#6B7280")  // 灰色，次要文字
    ColorBg        = lipgloss.Color("#1F2937")  // 深色背景
    ColorSurface   = lipgloss.Color("#374151")  // 卡片背景
    ColorBorder    = lipgloss.Color("#4B5563")  // 边框色

    StateColors = map[string]lipgloss.Color{
        "Idle":             ColorSuccess,
        "Processing":       ColorInfo,
        "AwaitingApproval": ColorWarning,
        "Paused":           ColorMuted,
        "Completed":        ColorSuccess,
        "Archived":         ColorMuted,
    }
)
```

### 8.2 组件样式

```go
// 状态栏
var statusBarStyle = lipgloss.NewStyle().
    Background(ColorBg).
    Foreground(lipgloss.Color("#FFFFFF")).
    Padding(0, 1)

// 用户消息气泡
var userBubbleStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(ColorPrimary).
    Padding(0, 1).
    Margin(0, 0, 1, 4)

// 助手消息气泡
var assistantBubbleStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(ColorBorder).
    Padding(0, 1).
    Margin(0, 4, 1, 0)

// 工具调用卡片
var toolCardStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder()).
    BorderForeground(ColorInfo).
    Padding(0, 1).
    Margin(0, 0, 1, 2)

// 审批弹窗
var modalOverlayStyle = lipgloss.NewStyle().
    Background(lipgloss.Color("#00000088"))

var modalBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(ColorWarning).
    Padding(2, 3)
```

---

## 9. 实现计划

| 编号 | 任务 | 内容 | 预估 | 依赖 |
| :--- | :--- | :--- | :--- | :--- |
| TUI-01 | 项目骨架 + 入口 | `launcher.go`、`types/`、`messages/`，main.go 加 `--tui` flag | 1h | — |
| TUI-02 | 协程内嵌服务 | main.go 中协程启动 HTTP 服务、随机端口分配、就绪等待 | 1h | TUI-01 |
| TUI-03 | APIClient + SSEClient | `apiclient.go`：封装全部 REST 端点；`sseclient.go`：SSE 事件解析 | 2h | TUI-01 |
| TUI-04 | 样式系统 | `styles.go`：lipgloss 色彩、组件样式、状态色映射 | 1h | TUI-01 |
| TUI-05 | StatusBar + HelpBar | 顶部状态栏 + 底部快捷键栏，动态内容 | 0.5h | TUI-04 |
| TUI-06 | ChatView | MessageViewport + InputArea，消息渲染、输入处理 | 2h | TUI-04 |
| TUI-07 | SessionSidebar | 会话列表、光标导航、切换/新建 | 1h | TUI-03, TUI-04 |
| TUI-08 | ApprovalModal | 审批弹窗覆盖层、diff 展示、Y/N 决策 | 1.5h | TUI-04 |
| TUI-09 | App 主流程 | `app.go`：整合所有组件，状态机、SSE 事件消费、消息分发 | 2.5h | TUI-02~08 |
| TUI-10 | Toast + 异常处理 | 浮动提示、连接断开恢复、异常提示 | 1h | TUI-09 |
| TUI-11 | 集成测试 | 端到端验证：启动→会话创建→对话→审批→切换→退出 | 2h | TUI-10 |

**总计**：约 15 小时。

---

## 10. 验收标准

1. **启动流程**：在项目目录下执行 `devo --tui`，自动初始化服务、协程启动 HTTP 服务、创建会话、进入聊天界面
2. **消息对话**：输入文本后发送，Agent 返回回复，消息气泡正确渲染
3. **实时事件**：SSE 事件实时展示（thinking、tool_call_request、tool_result、message_complete）
4. **工具调用展示**：工具调用卡片正确显示工具名、参数摘要、结果摘要、耗时
5. **审批交互**：
   - 高风险操作触发审批弹窗，展示 diff 摘要
   - 按 `Y` 批准执行，按 `N` 拒绝
   - 自动批准的操作显示 badge 标记
6. **会话管理**：侧栏可切换会话、新建会话（自动使用当前目录）
7. **状态控制**：`Ctrl+P` 暂停/恢复、`Ctrl+C` 取消，状态栏实时更新
8. **退出清理**：`Ctrl+Q` 退出时，TUI 恢复终端，main() 依次关闭 HTTP 服务和数据库，无残留协程
9. **异常处理**：服务启动失败时 log.Fatal 退出；SSE 断连时自动重连
10. **跨平台**：Windows、macOS、Linux 终端均可正常运行

---

## 附录 A：与 REST API 的映射

| TUI 操作 | HTTP 方法 | API 路径 |
| :--- | :--- | :--- |
| 自动创建会话 | `POST` | `/api/v1/sessions` |
| 列出会话 | `GET` | `/api/v1/sessions` |
| 获取会话详情 | `GET` | `/api/v1/sessions/{id}` |
| 发送消息 | `POST` | `/api/v1/sessions/{id}/messages` |
| 获取消息历史 | `GET` | `/api/v1/sessions/{id}/messages` |
| 获取文件列表 | `GET` | `/api/v1/sessions/{id}/files` |
| SSE 事件流 | `GET` | `/api/v1/sessions/{id}/events` |
| 批准操作 | `POST` | `/api/v1/sessions/{id}/approve/{approval_id}` |
| 暂停 | `POST` | `/api/v1/sessions/{id}/pause` |
| 恢复 | `POST` | `/api/v1/sessions/{id}/resume` |
| 取消 | `POST` | `/api/v1/sessions/{id}/cancel` |
| 完成 | `POST` | `/api/v1/sessions/{id}/complete` |
| 归档 | `POST` | `/api/v1/sessions/{id}/archive` |
| 设置信任级别 | `PUT` | `/api/v1/sessions/{id}/trust` |
| 设置审批策略 | `PUT` | `/api/v1/sessions/{id}/approval-policy` |

---

## 附录 B：关键设计决策

| 决策 | 选项 | 选择 | 理由 |
| :--- | :--- | :--- | :--- |
| TUI 与 Server 关系 | 父子进程 | **协程内嵌** | 单一进程，统一入口；无需维护子进程生命周期；服务初始化代码不重复 |
| 端口分配 | 固定端口 | **随机端口** | 避免多实例冲突；无需用户配置 |
| 会话工作目录 | 用户输入 | **自动获取 cwd** | 零配置启动；符合"在项目目录下运行"的使用场景 |
| REPL vs TUI | 先 REPL 再 TUI | **直接 Full TUI** | 减少重复工作；bubbletea 成熟度高 |
| 多会话支持 | 仅单会话 | **多会话** | 侧栏切换成本低；测试多会话场景是刚需 |
| SSE 客户端 | 标准库 | **标准库** | bufio.Scanner 逐行解析 SSE 足够，无需第三方库 |
| 样式方案 | ANSI 码 | **lipgloss** | 声明式 API 更易维护；自适应宽度；与 bubbletea 生态一致 |
| 入口模式 | 独立二进制 | **单一入口 + flag** | `devo --tui` 启动 TUI；`devo` 启动纯服务；统一构建 |