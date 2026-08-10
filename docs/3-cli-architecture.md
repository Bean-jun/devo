# Devo TUI 交互层架构设计文档

**版本**：4.0.0

**定位**：基于 bubbletea 的全屏终端交互界面（TUI），作为 Devo 编码代理系统的第一方交互入口。TUI 与 Devo 服务运行在同一进程内，服务以 goroutine 方式内嵌启动，通过 HTTP 协议（localhost 随机端口）驱动所有操作。用户在项目目录中执行 `devo --tui` 即可开始与 Agent 对话。

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
│  │  ├── go server.ListenAndServe()  ← goroutine 启动  │  │
│  │  ├── 等待服务就绪 (poll /api/v1/sessions)          │  │
│  │  └── tui.Launch(baseURL, version)                  │  │
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
│  │  │  StatusBar · ChatView · OverlayStack         │  │ │
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
│ 3. goroutine 启动 HTTP 服务          │
│    go server.ListenAndServe()       │
├─────────────────────────────────────┤
│ 4. 等待服务就绪                      │
│    poll GET /api/v1/sessions        │
│    超时 10s，失败则 log.Fatal       │
├─────────────────────────────────────┤
│ 5. 启动 TUI 主界面                   │
│    tui.Launch(baseURL, version)    │
│    bubbletea 接管终端                │
│    TUI Init() 开始                  │
│    用户可立即开始对话                 │
└─────────────────────────────────────┘
```

**退出流程**：
```
用户: Ctrl+Q / Ctrl+C
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
| `charm.land/bubbletea/v2` | TUI 框架 | Elm 架构，Model/Update/View 分离 |
| `charm.land/bubbles/v2` | TUI 组件库 | viewport、textarea 等即用组件 |
| `charm.land/lipgloss/v2` | 终端样式 | 声明式样式系统，支持自适应宽度 |
| `charm.land/glamour/v2` | Markdown 渲染 | 助手消息 Markdown 渲染 |
| `golang.org/x/term` | 终端尺寸 | 获取终端宽度/高度 |
| `net` (标准库) | 端口分配 | `net.Listen("tcp", "127.0.0.1:0")` 获取随机端口 |
| `net/http` (标准库) | HTTP 服务/客户端 | goroutine 内嵌服务 + REST API 调用 + SSE 长连接 |

---

## 3. MVC 架构设计

### 3.1 三层职责

```
┌─────────────────────────────────────────────────────────┐
│                    Model (model.go)                     │
│  状态持有：消息列表、会话列表、覆盖层状态、输入状态        │
│  业务逻辑：消息跳转定位、命令路由、Toast 定时器            │
│  API 调用：30+ 后端 API 方法封装为 tea.Cmd               │
│  数据适配：applySessionsData / applyMessagesData 等       │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────────┐
│                  Controller (update.go)                  │
│  事件分发：KeyMsg → 快捷键/覆盖层/输入框                  │
│  命令路由：/ 命令 → routeCommand() → OverlayType         │
│  窗口调整：WindowSizeMsg → applySize() 重新计算布局       │
│  API 回调：apiResponseMsg → 处理 30+ 种响应类型           │
│  SSE 事件：sseEventMsg → 处理 20+ 种 SSE 事件类型         │
│  覆盖层交互：handleOverlayKey / handleOverlayEnter        │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────────┐
│                     View (view.go)                      │
│  纯渲染：Toast + StatusBar + Viewport/Overlay + InputArea │
│  不修改任何状态，只读取 model 字段生成字符串               │
│  委托组件渲染：components/*、overlays/*、renderer/*       │
└─────────────────────────────────────────────────────────┘
```

### 3.2 数据流

```
用户按键 → tea.KeyMsg
    │
    ▼
update.go: Update()
    │
    ├─ 快捷键?  → model.jumpToPrevUserMessage() / toggleTheme() / ...
    ├─ 覆盖层?  → handleOverlayKey() → 面板内导航/编辑
    ├─ 输入框?  → textarea.Update() → 有内容时直接输入
    └─ 其他?    → 转发给 viewport.Update()
    │
    ▼
model 状态更新 → 返回 (model, tea.Cmd)
    │
    ▼
view.go: View()
    ├── renderToastLine()      → toast 浮动提示
    ├── statusBar.Render()     → 顶部状态栏
    ├── overlay.IsOpen()?      → renderOverlay() : viewport.View()
    └── renderInputArea()      → 输入区 + footer
    │
    ▼
返回 tea.View{AltScreen: true} → bubbletea 渲染到终端
```

### 3.3 初始化流程（Init）

[model.go:L23-L31](file:///home/bean/Desktop/devo/internal/interfaces/tui/model.go#L23-L31)

```go
func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        textarea.Blink,             // 输入框光标闪烁
        tea.Tick(200ms, TickMsg),   // Toast 倒计时 200ms/tick
        tea.Tick(50ms, resizeTickMsg),  // 终端尺寸检测 50ms/tick
    )
}
```

首次 `WindowSizeMsg` 到达时触发 `initFromAPI()` 加载会话列表，若已有空会话则直接使用，否则自动创建新会话。

---

## 4. 屏幕设计

### 4.1 整体布局

```
┌──────────────────────────────────────────────────────────┐
│ ● Devo · my-project · Idle · YOLO · :52341 已连接    🌙  │  ← StatusBar（始终可见）
├──────────────────────────────────────────────────────────┤
│                                                          │
│  [User] 14:32                                            │
│  帮我修复 utils.go 中的空指针问题                          │
│                                                          │
│  ┌ Tool: read_file ──────────────────────────────────┐   │
│  │ utils.go · 156 lines · ✓                          │   │
│  └───────────────────────────────────────────────────┘   │
│                                                          │
│  [Assistant] 14:32                                       │
│  已修复 utils.go 中的空指针问题，                          │
│  在 oldFunc() 开头添加了 nil 检查...                      │
│                                                          │
├──────────────────────────────────────────────────────────┤
│ > 继续优化这个函数                              ↩         │  ← InputArea（始终可见）
│ ──────────────────────────────────────────────────────── │  ← 分隔线
│ Context 1.2k · Tokens 3.5k (↑2.1k ↓1.4k) · /home/...   │  ← Footer
└──────────────────────────────────────────────────────────┘
```

### 4.2 布局计算

View 层将屏幕分为三个固定区域：

| 区域 | 高度 | 说明 |
| :--- | :--- | :--- |
| Toast 行 | 1 行 | 浮动通知，仅在有消息时显示 |
| StatusBar + 分隔线 | 2 行 | 顶部状态栏，始终可见 |
| Viewport / Overlay | `height - 9` | 聊天区域或覆盖层区域 |
| InputArea + 分隔线 + Footer | 6 行 | 输入框（3行）+ 分隔线 + footer |

### 4.3 覆盖层模式

当打开命令面板、会话选择器、帮助等面板时，覆盖层替换聊天视口区域，StatusBar 和 InputArea 保持可见。覆盖层通过 `OverlayStack` 管理，支持嵌套遮挡；关闭时回到上一层面板。覆盖层内容居中显示在 viewport 区域内：

```
┌──────────────────────────────────────────────────────────┐
│ ● Devo · my-project · Idle · :52341 已连接               │  ← StatusBar（始终可见）
├──────────────────────────────────────────────────────────┤
│                                                          │
│              ┌─ Commands ──────────────────────┐         │
│              │                                  │         │
│              │  Chat                            │         │  ← 面板居中
│              │  /new        新建会话             │         │
│              │  /switch     切换会话             │         │
│              │  ...                             │         │
│              │                                  │         │
│              │         [Esc] 关闭               │         │
│              └──────────────────────────────────┘         │
│                                                          │
├──────────────────────────────────────────────────────────┤
│ >                                                       │  ← InputArea（始终可见）
└──────────────────────────────────────────────────────────┘
```

---

## 5. 组件详细设计

### 5.1 Model — 顶层模型

[model.go:L1-L69](file:///home/bean/Desktop/devo/internal/interfaces/tui/model.go#L1-L69)

```go
type Model struct {
    // bubbletea 内置组件
    viewport viewport.Model
    textarea textarea.Model

    // 渲染器
    renderer *renderer.MsgRenderer

    // 业务数据
    messages        []types.Message
    sessions        []types.SessionInfo
    activeSessionID string

    // 子组件
    statusBar components.StatusBar
    overlay   overlays.OverlayStack
    toast     components.Toast

    // 覆盖层面板（17 个面板实例）
    cmdSheet        overlays.CommandSheet
    sessPicker      overlays.SessionPicker
    approval        overlays.ApprovalModal
    helpPanel       overlays.HelpPanel
    skillsPanel     overlays.SkillsPanel
    mcpPanel        overlays.MCPPanel
    memoryPanel     overlays.MemoryPanel
    wsPanel         overlays.WorkspacePanel
    renameModal     overlays.RenameModal
    rollback        overlays.RollbackPicker
    newSessModal    overlays.NewSessionModal
    statusPanel     overlays.StatusPanel
    versionPanel    overlays.VersionPanel
    backgroundPanel overlays.BackgroundPanel
    dashboardPanel  overlays.DashboardPanel
    settingsPanel   overlays.SettingsPanel

    // API 通信
    apiClient *api.Client
    sseClient *api.SSEClient

    // 元数据
    baseURL    string
    version    string
    workingDir string

    // 终端尺寸
    width  int
    height int
    ready  bool

    // 状态
    initialized bool
    loading     map[overlays.OverlayType]bool
}
```

**Model 层核心方法分类**：

| 类别 | 方法 | 说明 |
| :--- | :--- | :--- |
| API 调用 | `createSessionFromAPI`, `fetchSessionsFromAPI`, `fetchMessagesFromAPI`, `sendMessageToAPI`, `renameSessionFromAPI`, `deleteSessionFromAPI`, `exportSessionFromAPI`, `pauseSessionFromAPI`, `resumeSessionFromAPI`, `cancelSessionFromAPI`, `archiveSessionFromAPI`, `compactSessionFromAPI`, `approveFromAPI`, `rollbackFromAPI`, `fetchSkillsFromAPI`, `toggleSkillFromAPI`, `installSkillFromAPI`, `fetchMCPServersFromAPI`, `toggleMCPServerFromAPI`, `addMCPServerFromAPI`, `fetchMemoriesFromAPI`, `upsertMemoryFromAPI`, `deleteMemoryFromAPI`, `fetchWorkspacesFromAPI`, `switchWorkspaceFromAPI`, `fetchBackgroundProcessesFromAPI`, `stopBackgroundProcessFromAPI`, `fetchDashboardDataFromAPI`, `fetchProjectConfigFromAPI`, `fetchGlobalConfigFromAPI`, `saveProjectConfigFromAPI`, `saveGlobalConfigFromAPI` | 所有后端通信封装为 `tea.Cmd` |
| SSE | `connectSSE`, `listenSSE` | SSE 连接管理与事件监听 |
| 数据适配 | `applySessionsData`, `applyMessagesData`, `applySkillsData`, `applyMCPServersData`, `applyMemoriesData`, `applyWorkspacesData` | 将 API 返回数据适配到面板结构 |
| 布局 | `applySize`, `applyTermSize`, `overlayPanelWidth`, `refreshViewport`, `refreshViewportToBottom`, `buildFooterText` | 终端尺寸适配与视口刷新 |
| 导航 | `findUserMessageYOffsets`, `jumpToPrevUserMessage`, `jumpToNextUserMessage` | 消息跳转定位 |
| 命令路由 | `routeCommand` | 18 个命令映射到对应面板/操作 |
| 编辑状态 | `isEditing`, `appendEditChar`, `setLoading`, `isLoading` | 覆盖层内编辑模式管理 |
| 状态查询 | `updateStatusInfo` | 构建 StatusPanel 所需信息 |

### 5.2 StatusBar — 顶部状态栏

[components/statusbar.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/components/statusbar.go)

```go
type StatusBar struct {
    AppName    string   // "Devo"
    Session    string   // 会话标题
    Processing bool     // 处理中状态
    Paused     bool     // 暂停状态
    Yolo       bool     // YOLO 模式
    Connected  bool     // 服务连接状态
    Width      int      // 渲染宽度
    ServerPort string   // 服务端口号
}
```

**显示内容**（从左到右）：
- 会话名
- 状态圆点 + 状态文本（idle / Processing / Paused）
- YOLO 标签（开启时显示黄色 "YOLO" badge）
- 处理中时居中显示 ⌛ spinner + "Processing..."
- 右侧：✓ 连接图标 + 端口号

**颜色语义**：
- 空闲：绿色圆点
- 处理中：蓝色圆点 + 居中动画
- 暂停：灰色圆点

### 5.3 InputArea — 输入区域

输入区域由 `renderInputArea` 方法渲染，包含：
- **分隔线（上）**：`─` 字符，宽度自适应
- **Textarea**：3 行高度，无行号，无边框，placeholder "输入消息..."
- **分隔线（下）**：`─` 字符
- **Footer**：显示 Context tokens、Token 用量（输入/输出）、工作目录路径，右侧显示发送图标（空闲时 `⏎`，处理中时 `■`）

### 5.4 OverlayStack — 覆盖层管理器

[overlays/stack.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/overlays/stack.go)

```go
type OverlayType int

const (
    OverlayNone OverlayType = iota
    OverlayApproval
    OverlayHelp
    OverlayCommand
    OverlaySession
    OverlayToast
    OverlaySkills
    OverlayMCP
    OverlayMemory
    OverlayWorkspace
    OverlayNewSession
    OverlayRename
    OverlayRollback
    OverlayStatus
    OverlayVersion
    OverlayBackground
    OverlayDashboard
    OverlaySettings
)

type OverlayStack struct {
    stack   []OverlayType    // 支持嵌套的栈
    Current OverlayType      // 当前活动面板
}
```

`Open(t)` 将面板类型压入栈并设为当前面板；`Close()` 弹出栈顶并恢复上一层，栈空时回到聊天视图。`IsOpen()` 判断当前是否有面板打开。

### 5.5 Toast — 浮动提示

[components/toast.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/components/toast.go)

```go
type Toast struct {
    Message  string   // 提示消息
    Type     string   // "error" / "success"
    Duration int      // 剩余 tick 数（200ms/tick）
    Width    int      // 渲染宽度
}
```

- `Show(msg, isError)` — 显示消息，error 类型显示 5 ticks（1s），success 类型显示 3 ticks（0.6s）
- `Tick()` — 每 200ms 递减 Duration
- `Render()` — 仅在 Duration > 0 时渲染，右对齐显示

---

## 6. 源代码结构

```
internal/interfaces/tui/
│
├── launcher.go                   # TUI 入口：Launch(baseURL, version)
├── launcher_unix.go              # Unix 平台选项（build tag: !windows）
├── launcher_windows.go           # Windows 平台选项（build tag: windows）
├── logger.go                     # 日志系统：文件重定向
├── model.go                      # 顶层 Model：状态、API 调用、数据适配
├── update.go                     # 控制器：事件分发、SSE 处理、覆盖层交互
├── view.go                       # 视图：布局渲染、面板委托
├── model_test.go                 # Model 层测试
├── update_test.go                # Update 层测试
│
├── api/
│   ├── client.go                 # REST API 客户端（30+ 端点）
│   └── sse.go                    # SSE 客户端（事件流）
│
├── types/
│   ├── message.go                # Message、ToolCall、SendMessageRequest 等
│   ├── session.go                # SessionInfo、SessionState、TokenUsage 等
│   ├── sse.go                    # SSEEvent、APIResponse、TickMsg
│   └── approval.go               # ApprovalRequest、ApproveRequest、FileInfo 等
│
├── components/
│   ├── statusbar.go              # 顶部状态栏
│   ├── styles.go                 # 主题系统（Dark/Light）+ 全局样式函数
│   ├── toast.go                  # Toast 浮动提示（自动消失）
│   ├── statusbar_test.go
│   ├── styles_test.go
│   └── toast_test.go
│
├── overlays/                     # 覆盖层面板（17 个面板）
│   ├── stack.go                  # OverlayStack 管理器（支持嵌套栈）
│   ├── command.go                # 命令面板（4 组分类，实时过滤）
│   ├── session.go                # 会话选择器（含消息预览）
│   ├── help.go                   # 帮助面板（5 组快捷键）
│   ├── approval.go               # 审批弹窗（风险等级 + Diff）
│   ├── skills.go                 # 技能管理（启用/禁用 + 安装）
│   ├── mcp.go                    # MCP 服务器管理（连接/断开 + 添加）
│   ├── memory.go                 # 记忆管理（CRUD）
│   ├── workspace.go              # 工作区切换
│   ├── modals.go                 # NewSession、Rename、Rollback 三个面板
│   ├── background.go             # 后台进程管理（展开/停止）
│   ├── dashboard.go              # 仪表盘（Token 用量条形图）
│   ├── status.go                 # 状态面板（会话信息一览）
│   ├── version.go                # 版本面板
│   ├── settings.go               # 设置面板（项目/全局配置，int/枚举编辑）
│   └── *_test.go                 # 各面板测试文件
│
└── renderer/                     # 消息渲染引擎
    ├── renderer.go               # 消息渲染、CJK 换行、增量缓存、Markdown
    └── renderer_test.go          # 渲染器测试
```

---

## 7. API 通信层

### 7.1 API Client

[api/client.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/api/client.go)

```go
type Client struct {
    BaseURL    string
    httpClient *http.Client    // 5s 超时
}
```

**通用 HTTP 方法**：
- `do(method, path, body, result)` — 通用请求，自动 JSON 序列化/反序列化
- `get(path, result)` — GET 请求
- `post(path, body, result)` — POST 请求
- `put(path, body, result)` — PUT 请求
- `del(path)` — DELETE 请求

**接口分类**：

| 类别 | 方法 | API 路径 |
| :--- | :--- | :--- |
| 会话管理 | `CreateSession`, `ListSessions`, `GetSession`, `RenameSession`, `DeleteSession`, `ArchiveSession`, `ExportSession` | `/api/v1/sessions` |
| 消息 | `SendMessage`, `GetMessages`, `Rollback` | `/api/v1/sessions/{id}/messages` |
| 文件 | `GetFiles` | `/api/v1/files` |
| 技能 | `GetSkills`, `SetSessionSkills`, `RemoveSessionSkill`, `InstallSkill` | `/api/v1/skills` |
| MCP | `GetMCPServers`, `ToggleMcpServer`, `AddMCPServer` | `/api/v1/mcp/servers` |
| 记忆 | `GetMemories`, `UpsertMemory`, `DeleteMemory` | `/api/v1/sessions/{id}/memory` |
| 工作区 | `GetWorkspaces`, `GetCurrentWorkspace`, `SetWorkspace`, `DeleteWorkspace` | `/api/v1/workspace` |
| 会话控制 | `Pause`, `Resume`, `Cancel`, `Complete` | `/api/v1/sessions/{id}/...` |
| 审批 | `Approve`, `SetTrustLevel` | `/api/v1/sessions/{id}/approve` |
| 配置 | `GetConfigStatus`, `UpdateConfig`, `GetProjectConfig`, `GetGlobalConfig`, `UpdateProjectConfig`, `UpdateGlobalConfig` | `/api/v1/config` |
| 用量 | `GetSessionUsage`, `GetProjectUsage` | `/api/v1/sessions/{id}/usage` |
| 后台进程 | `GetBackgroundProcesses`, `StopBackgroundProcess` | `/api/v1/sessions/{id}/background` |
| 其他 | `GetVersion`, `SyncArchive`, `CompactSession` | `/api/v1/version` 等 |

### 7.2 SSE Client

[api/sse.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/api/sse.go)

```go
type SSEClient struct {
    baseURL string
    eventCh chan types.SSEEvent    // 缓冲 100
    errCh   chan error             // 缓冲 10
    done    chan struct{}          // 关闭信号
}
```

**工作流程**：
1. `Connect(sessionID)` — 发起 GET `/api/v1/sessions/{id}/events`，设置 `Accept: text/event-stream`
2. `readLoop(resp)` — 在 goroutine 中逐行读取 SSE 数据
3. 解析 `data:` 前缀行，提取 JSON wrapper 中的 `type` 和 `data` 字段
4. 通过 `eventCh` 发送 `SSEEvent`，通过 `errCh` 发送错误
5. `Disconnect()` — 关闭 `done` channel 终止读取循环

**SSE 事件类型**（20+ 种）：

| 事件类型 | 说明 | 处理方式 |
| :--- | :--- | :--- |
| `thinking` | Agent 开始思考 | 设置 Processing 状态 |
| `reasoning_token` | 推理过程 token 流 | 追加到消息 `Thinking` 字段 |
| `reasoning_complete` | 推理完成 | 用完整推理文本替换 |
| `streaming_token` | 响应 token 流 | 追加到消息 `Content` 字段 |
| `streaming_complete` | 流式响应完成 | 标记 `IsStreaming = false` |
| `tool_call_request` | 工具调用请求 | 创建 tool 消息，显示工具卡片 |
| `tool_result` | 工具执行结果 | 更新工具卡片状态和输出 |
| `tool_progress` | 工具执行进度 | 更新工具卡片阶段信息 |
| `tool_chunk` | 工具输出流式块 | 追加到工具输出内容 |
| `message_complete` | 消息完成 | 停止 Processing，更新 Token 用量 |
| `token_usage` | Token 用量更新 | 更新会话 TokenUsage 和 ContextTokens |
| `session_state_change` | 会话状态变更 | 更新状态栏（idle/cancelled/paused/thinking） |
| `context_compressed` | 上下文压缩 | 添加 system 消息提示 |
| `file_state_warning` | 文件状态警告 | 添加 system 消息提示 |
| `skill_solidified` | 技能固化 | Toast 提示 |
| `memory_updated` | 记忆更新 | Toast 提示 |
| `mcp_tool_discovered` | 发现 MCP 工具 | Toast 提示 |
| `background_output` | 后台进程输出 | 添加 system 消息显示输出 |
| `approval_required` | 需要审批 | 打开审批弹窗 |
| `approval_auto` | 自动审批 | Toast 提示（非 YOLO 策略时） |
| `approval_resolved` | 审批已解决 | 关闭审批弹窗 |
| `error` | SSE 错误 | Toast 错误提示 |
| `done` / `complete` | 完成 | 停止 Processing |
| `delta` / `message` | 消息增量 | 追加到消息内容 |
| `tool_use` | 工具使用 | 添加工具调用到助手消息的 ToolCalls |
| `loop.completed_with_reason` | Agent Loop 完成 | 根据原因显示 Toast |

### 7.3 API 响应处理

Model 层通过 `apiResponseMsg` 结构传递 API 响应：

```go
type apiResponseMsg struct {
    kind      string       // 响应类型标识（30+ 种）
    data      interface{}  // 响应数据
    err       error        // 错误信息
    sessionID string       // 关联的会话 ID
    title     string       // 重命名标题
    path      string       // 工作区路径
    key       string       // 记忆键
    id        string       // 记忆 ID
}
```

`handleAPIResponse` 方法根据 `kind` 分类处理，典型流程：
1. 类型断言获取数据
2. 更新 Model 对应字段
3. 显示 Toast 提示
4. 必要时触发后续操作（如创建会话后加载消息 + 连接 SSE）

---

## 8. 键盘快捷键

### 8.1 导航

| 快捷键 | 说明 |
| :--- | :--- |
| `↑` / `↓` | 行滚动 |
| `PgUp` / `PgDn` | 页滚动 |
| `Ctrl+U` | 跳转到上一条用户消息（输入框为空时） |
| `Ctrl+D` | 跳转到下一条用户消息（输入框为空时） |
| `Tab` | 展开/折叠工具调用卡片 |

### 8.2 聊天

| 快捷键 | 说明 |
| :--- | :--- |
| `Enter` | 发送消息 |
| `/` | 打开命令面板（输入框为空时） |
| `Ctrl+N` | 新建会话 |
| `Ctrl+S` | 会话列表 |
| `F2` | 重命名当前会话 |

### 8.3 模式切换

| 快捷键 | 说明 |
| :--- | :--- |
| `Ctrl+T` | 切换主题（暗色 / 亮色） |
| `Ctrl+Y` | 切换 YOLO 模式 |
| `Ctrl+P` | 暂停 / 恢复 |

### 8.4 覆盖层

| 快捷键 | 说明 |
| :--- | :--- |
| `Esc` | 关闭当前覆盖层 / 面板（编辑模式中先取消编辑） |
| `?` | 打开帮助面板 |
| `↑` / `↓` / `j` / `k` | 面板内光标移动（编辑模式下 j/k 作为字符输入） |
| `Enter` | 面板内确认选择 |
| `Space` | 技能面板：切换启用状态；MCP 面板：连接/断开；设置面板：切换审批级别 |
| `a` | 技能/MCP/记忆面板：进入编辑模式（添加新项） |
| `Backspace` | 命令面板/重命名/编辑模式：删除字符；会话面板：删除当前会话 |
| `Delete` | 记忆面板/会话面板：删除当前选中项 |
| `Tab` | 后台进程面板：展开/折叠进程详情 |
| `y` / `Y` | 审批面板：批准操作 |
| `n` / `N` | 审批面板：拒绝操作 |

### 8.5 系统

| 快捷键 | 说明 |
| :--- | :--- |
| `Ctrl+C` | 退出 |
| `Ctrl+Q` | 退出 |

---

## 9. 命令面板

命令面板通过 `/` 键打开（输入框为空时），内置 4 组命令：

| 分组 | 命令 | 说明 |
| :--- | :--- | :--- |
| **SESSION** | `/new` | 创建新会话 |
| | `/switch` | 切换会话 |
| | `/rename` | 重命名会话 |
| | `/export` | 导出会话 |
| | `/rollback` | 回滚到消息 |
| | `/pause` | 暂停当前会话 |
| | `/resume` | 恢复当前会话 |
| | `/cancel` | 取消当前操作 |
| | `/compact` | 压缩会话上下文 |
| | `/help` | 显示帮助 |
| **PANEL** | `/skills` | 技能管理 |
| | `/mcp` | MCP 管理 |
| | `/memory` | 记忆管理 |
| | `/background` | 后台任务 |
| | `/dashboard` | 仪表盘 |
| | `/settings` | 设置 |
| **WORKSPACE** | `/workspace-switch` | 切换工作区 |
| **APP** | `/toggle-theme` | 切换主题 |
| | `/status` | 查看当前状态 |
| | `/version` | 查看应用版本 |

**CommandSheet 核心结构**：

[overlays/command.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/overlays/command.go)

```go
type CommandSheet struct {
    Width        int
    Height       int
    Filter       string           // 实时过滤文本
    Selected     int              // 当前选中索引
    Groups       []CommandGroup   // 4 组命令定义
    FlatCommands []FlatCommand    // 过滤后的扁平列表
}
```

**支持实时过滤**：输入任意字符过滤命令名称和描述，按 `Enter` 执行选中命令。过滤逻辑为大小写不敏感的包含匹配，匹配范围包括命令名称和描述。

---

## 10. 主题系统

### 10.1 主题定义

[components/styles.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/components/styles.go)

```go
type Theme struct {
    Name          string
    IsDark        bool
    BgPrimary     color.Color
    TextPrimary   color.Color
    TextSecondary color.Color
    TextTertiary  color.Color
    Accent        color.Color
    Success       color.Color
    Warning       color.Color
    Error         color.Color
    Border        color.Color
}
```

两套主题：**Dark**（默认）和 **Light**。通过 `Ctrl+T` 或 `/toggle-theme` 切换。主题切换时需重建渲染器（`renderer.New`）以适配 Markdown 渲染主题。

**Dark 主题色值**：
- BgPrimary: `#0d1117`（GitHub Dark 背景）
- Accent: `#58a6ff`（蓝色）
- Success: `#3fb950`（绿色）
- Warning: `#d29922`（黄色）
- Error: `#f85149`（红色）

**Light 主题色值**：
- BgPrimary: `#ffffff`
- Accent: `#0969da`（蓝色）
- Success: `#1a7f37`（绿色）
- Warning: `#9a6700`（黄色）
- Error: `#cf222e`（红色）

### 10.2 色彩语义

| Token | 用途 |
| :--- | :--- |
| `Accent` | 主色调：用户消息、选中高亮、快捷键、面板标题 |
| `Success` | 成功：工具执行成功、连接状态、技能已启用 |
| `Warning` | 警告：审批等待、YOLO 模式 badge |
| `Error` | 错误：执行失败、断连、审批高风险 |
| `TextSecondary` (Muted) | 次要文本：分隔线、提示文字、时间戳、描述 |
| `Border` | 边框：卡片边框、面板分隔、Diff 分隔线 |

### 10.3 样式函数

所有样式通过延迟求值的函数提供，确保主题切换后即时生效：

| 样式函数 | 用途 |
| :--- | :--- |
| `StatusBarBg()` | 状态栏背景样式 |
| `InputBoxStyle()` | 输入框样式（圆角边框） |
| `UserPrefix()` / `AsstPrefix()` | 用户/助手消息前缀 |
| `ThinkStyle()` / `SysStyle()` | 思考/系统消息样式 |
| `ToolOK()` / `ToolFail()` / `ToolWait()` / `ToolExec()` | 工具调用状态样式 |
| `ToolNameStyle()` / `ToolDetail()` | 工具名称和详情样式 |
| `OverlayStyle()` / `OverlayBoxStyle()` | 覆盖层容器样式（圆角边框） |
| `OverlayTitleStyle()` / `OverlayMutedStyle()` | 覆盖层标题/次要文本 |
| `PanelHeaderStyle(w)` / `PanelFooterStyle(w)` | 面板标题栏/底部栏 |
| `PanelSeparator(w)` | 面板分隔线 |
| `ToastError()` / `ToastSuccess()` / `ToastInfo()` | Toast 通知样式 |
| `DiffStyle()` | Diff 内容样式 |
| `TimeStyle()` | 时间戳样式 |

---

## 11. 消息渲染引擎

### 11.1 渲染器架构

[renderer/renderer.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/renderer/renderer.go)

```go
type MsgRenderer struct {
    width      int
    mdRenderer *glamour.TermRenderer   // Markdown 渲染器
    cache      *RenderCache            // 增量渲染缓存
    isDark     bool
}

type RenderCache struct {
    cache   []string   // 每条消息的渲染结果
    dirty   int        // -1 表示干净，>=0 表示从该索引起需要重渲染
    content string     // 拼接后的完整内容
    count   int        // 缓存的消息数量
}
```

### 11.2 增量渲染机制

- `Render(messages)` — 检查缓存状态，仅重渲染 dirty 标记之后的消息
- `Invalidate(idx)` — 标记从指定索引开始需要重渲染
- 缓存按索引存储每条消息的渲染字符串，避免全量重渲染
- 当消息数量变化（新增）时，追加渲染新消息

### 11.3 消息角色渲染

| 角色 | 前缀 | 渲染方式 |
| :--- | :--- | :--- |
| `user` | ▶（蓝色粗体） | 纯文本 + CJK 换行 |
| `assistant` | ●（粗体） | Glamour Markdown 渲染 + CJK 换行 + Thinking 文本 + ToolCalls 卡片 |
| `tool` | — | 工具调用卡片渲染，如无 ToolCalls 则按 system 渲染 |
| `system` | — | 斜体灰色文本 |

### 11.4 CJK 文本换行

`WrapCJK(text, width)` 函数实现 CJK 字符宽度感知的换行：
- ASCII 字符宽度计为 1
- CJK 字符（`utf8.RuneLen(r) > 1`）宽度计为 2
- 按指定宽度截断换行，保留 `\n` 换行符

### 11.5 工具调用卡片渲染

每个 `ToolCall` 渲染为一行状态卡片，包含：
- 状态图标：✓（成功）、✗（失败）、○（待处理/执行中）
- 工具名称（粗体）
- 摘要 + 状态 + 耗时（灰色）
- 展开时额外显示 Diff 内容和 Output 内容

### 11.6 消息跳转定位

`FindUserMessageYOffsets(messages)` 基于渲染缓存计算每条 user 消息的 Y 偏移量，用于 `Ctrl+U`/`Ctrl+D` 快速跳转。

---

## 12. 覆盖层面板详情

### 12.1 面板功能总览

| 面板 | 文件名 | 结构化数据 | 交互方式 |
| :--- | :--- | :--- | :--- |
| CommandSheet | command.go | 4 组 20 个命令 | 实时过滤 + ↑↓/j/k 导航 + Enter 执行 |
| SessionPicker | session.go | 会话列表 + 消息预览 | ↑↓ 导航 + Enter 切换 + Backspace/Delete 删除 |
| HelpPanel | help.go | 5 组快捷键 | 只读，Esc 关闭 |
| ApprovalModal | approval.go | 操作名 + 风险等级 + Diff | Y 批准 / N 拒绝 |
| SkillsPanel | skills.go | 技能列表（名称+描述+启用） | Space 切换 + a 安装 + Enter 确认 |
| MCPPanel | mcp.go | MCP 服务器列表（名称+URL+状态） | Space 连接/断开 + a 添加 |
| MemoryPanel | memory.go | 记忆列表（类型+键+内容） | Del 删除 + a 添加 |
| WorkspacePanel | workspace.go | 工作区列表（名称+路径） | Enter 切换 |
| NewSessionModal | modals.go | — | Enter 确认 / Esc 取消 |
| RenameModal | modals.go | 当前名称 + 新名称输入框 | 实时编辑 + Enter 确认 |
| RollbackPicker | modals.go | 消息列表（角色+内容+时间） | ↑↓ 导航 + Enter 确认回滚 |
| BackgroundPanel | background.go | 进程列表（PID+命令+状态） | Tab 展开 + Enter 停止 |
| DashboardPanel | dashboard.go | 会话/项目 Token 用量 | 条形图渲染 + 只读 |
| StatusPanel | status.go | 9 个会话状态字段 | 只读 |
| VersionPanel | version.go | 版本号 | 只读 |
| SettingsPanel | settings.go | 项目/全局配置字段 | ↑↓ 导航 + Enter 编辑数字 + Space 切换枚举 |

### 12.2 编辑模式

多个面板支持编辑模式，通过 `Editing` 状态和 `EditBuffer` 管理：

- **SkillsPanel**：`a` 键进入编辑，输入技能路径或 URL，Enter 确认安装
- **MCPPanel**：`a` 键进入编辑，输入 `server_id endpoint`，Enter 确认添加
- **MemoryPanel**：`a` 键进入编辑，输入 `key content`，Enter 确认 upsert
- **SettingsPanel**：Enter 进入编辑（仅 int 字段），输入数字，Enter 确认保存
- **CommandSheet**：任意字符直接过滤，Backspace 删除过滤字符
- **RenameModal**：任意字符直接输入新名称，Backspace 删除

编辑模式优先级：Esc 键先退出编辑模式，再退出面板。

### 12.3 Settings 面板

[overlays/settings.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/overlays/settings.go)

SettingsPanel 是功能最丰富的面板，支持以下配置项：

**项目配置**（`/api/v1/project/config`）：
- 工具调用上限（int）
- 最大上下文 tokens（int）
- 保留最近消息数（int）
- 审批策略（enum：始终询问 / 会话信任 / 永久信任 / 自动批准）

**全局配置**（`/api/v1/global/config`）：
- 工具调用上限（int）
- 最大上下文 tokens（int）
- 保留最近消息数（int）
- LLM 最大 tokens（int）
- 审批策略（enum）

int 字段：Enter 进入编辑 → 输入数字 → Enter 确认保存
enum 字段：Space 循环切换 → 即时保存

---

## 13. 日志系统

TUI 模式下，所有日志输出被重定向到文件，避免干扰终端渲染：

[logger.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/logger.go)

```go
func initLogger()     // 初始化日志文件（DEVO_LOG_PATH 或默认路径）
func RedirectStdLog()  // 重定向标准 log 和 slog 到文件
func Log(format, args...) // 写入调试日志
func LogFilePath() string // 获取日志文件路径
```

日志文件路径：`$DEVO_LOG_PATH` 环境变量指定，或默认 `~/.devo/devo.log`。

---

## 14. 平台适配

[launcher.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/launcher.go) / [launcher_unix.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/launcher_unix.go) / [launcher_windows.go](file:///home/bean/Desktop/devo/internal/interfaces/tui/launcher_windows.go)

```go
// launcher.go
func Launch(baseURL string, version string) {
    model := NewModel(baseURL, version)
    model.applyTermSize()
    opts := append([]tea.ProgramOption{tea.WithOutput(os.Stderr)}, programOptions()...)
    p := tea.NewProgram(&model, opts...)
    p.Run()
}
```

平台特定选项通过 build tags 分离：
- `launcher_unix.go`：`//go:build !windows`
- `launcher_windows.go`：`//go:build windows`

当前两个平台均无额外配置（`programOptions()` 返回 nil）。

---

## 15. 测试覆盖

### 15.1 测试文件列表

| 文件 | 测试内容 |
| :--- | :--- |
| `model_test.go` | Model 初始化、窗口尺寸、面板宽度、视口刷新、命令路由（18 个命令全覆盖）、消息跳转、View 渲染、加载状态、编辑模式、字符输入、端口提取 |
| `update_test.go` | 窗口变化、退出键（Ctrl+C/Q）、命令面板打开、消息发送、空输入、帮助面板、覆盖层 Esc/Up/Down/j/k、Space 切换技能、Enter 确认、光标移动、Toast Tick、消息跳转、编辑模式 j/k 行为、审批面板 Y/N |
| `components/statusbar_test.go` | 状态栏渲染 |
| `components/styles_test.go` | 主题切换、颜色函数 |
| `components/toast_test.go` | Toast 显示/隐藏/Tick |
| `overlays/approval_test.go` | 审批弹窗渲染 |
| `overlays/command_test.go` | 命令面板过滤、导航 |
| `overlays/help_test.go` | 帮助面板渲染 |
| `overlays/mcp_test.go` | MCP 面板渲染 |
| `overlays/memory_test.go` | 记忆面板渲染 |
| `overlays/session_test.go` | 会话选择器渲染 |
| `overlays/skills_test.go` | 技能面板渲染 |
| `overlays/workspace_test.go` | 工作区面板渲染 |
| `overlays/stack_test.go` | OverlayStack 打开/关闭/嵌套 |
| `overlays/new_session_test.go` | 新建会话弹窗 |
| `overlays/rename_test.go` | 重命名弹窗 |
| `overlays/rollback_test.go` | 回滚选择器 |
| `overlays/settings_test.go` | 设置面板 |
| `renderer/renderer_test.go` | 消息渲染、CJK 换行 |