# Devo TUI 交互层架构设计文档

**版本**：4.0.0

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
| `github.com/charmbracelet/bubbles` | TUI 组件库 | viewport、textarea、spinner 等即用组件 |
| `github.com/charmbracelet/lipgloss` | 终端样式 | 声明式样式系统，支持自适应宽度 |
| `github.com/charmbracelet/glamour` | Markdown 渲染 | 助手消息 Markdown 渲染 |
| `net` (标准库) | 端口分配 | `net.Listen("tcp", "127.0.0.1:0")` 获取随机端口 |
| `net/http` (标准库) | HTTP 服务/客户端 | 协程内嵌服务 + REST API 调用 + SSE 长连接 |

---

## 3. MVC 架构设计

### 3.1 三层职责

```
┌─────────────────────────────────────────────────────────┐
│                    Model (model.go)                     │
│  状态持有：消息列表、会话列表、覆盖层状态、输入状态        │
│  业务逻辑：消息跳转定位、命令路由、Toast 定时器            │
│  数据初始化：Mock 数据 / API 数据加载                     │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────────┐
│                  Controller (update.go)                  │
│  事件分发：KeyMsg → 快捷键/覆盖层/输入框                  │
│  命令路由：/ 命令 → OverlayType                          │
│  窗口调整：WindowSizeMsg → 重新计算布局                   │
│  API 回调：SSEEvent → 更新消息列表                        │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────────┐
│                     View (view.go)                      │
│  纯渲染：StatusBar + ChatView + InputArea + Overlay     │
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
    ├─ 覆盖层?  → handleOverlayKey() → 面板内导航
    ├─ 输入框?  → textarea.Update() → 有内容时直接输入
    └─ 其他?    → 转发给 viewport.Update()
    │
    ▼
model 状态更新
    │
    ▼
view.go: View()
    ├── statusBar.Render()
    ├── overlay.IsOpen()? → renderOverlay() : viewport.View()
    └── renderInputArea()
    │
    ▼
返回完整字符串 → bubbletea 渲染到终端
```

---

## 4. 屏幕设计

### 4.1 整体布局

```
┌──────────────────────────────────────────────────────────┐
│ ● Devo · my-project · Idle · YOLO · :52341 已连接    🌙  │  ← StatusBar
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
│                                                          │
├──────────────────────────────────────────────────────────┤
│ > 继续优化这个函数                              ↩         │  ← InputArea
└──────────────────────────────────────────────────────────┘
```

### 4.2 覆盖层模式

当打开命令面板、会话选择器、帮助等面板时，覆盖层替换聊天视口区域，StatusBar 和 InputArea 保持可见：

```
┌──────────────────────────────────────────────────────────┐
│ ● Devo · my-project · Idle · :52341 已连接               │  ← StatusBar（始终可见）
├──────────────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────────────┐   │
│ │ ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │   │  ← 半透明遮罩
│ │ ░░ ┌─ Commands ───────────────────────────────┐ ░░ │   │
│ │ ░░ │                                            │ ░░ │   │
│ │ ░░ │  Chat                                      │ ░░ │   │  ← 面板内容
│ │ ░░ │  /new        新建会话                      │ ░░ │   │
│ │ ░░ │  /switch     切换会话                      │ ░░ │   │
│ │ ░░ │  ...                                       │ ░░ │   │
│ │ ░░ │                                            │ ░░ │   │
│ │ ░░ │              [Esc] 关闭                    │ ░░ │   │
│ │ ░░ └────────────────────────────────────────────┘ ░░ │   │
│ │ ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │   │
│ └────────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────┤
│ >                                                       │  ← InputArea（始终可见）
└──────────────────────────────────────────────────────────┘
```

---

## 5. 组件详细设计

### 5.1 Model — 顶层模型

```go
// model.go

type model struct {
    // bubbletea 内置组件
    viewport viewport.Model
    textarea textarea.Model

    // 业务数据
    renderer  *msgRenderer
    messages  []Message
    sessions  []Session

    // 子组件
    statusBar    StatusBar
    inputArea    InputArea
    overlay      OverlayStack
    toast        Toast

    // 覆盖层面板
    cmdSheet     CommandSheet
    sessPicker   SessionPicker
    approval     ApprovalModal
    helpPanel    HelpPanel
    filesPanel   FilesPanel
    skillsPanel  SkillsPanel
    mcpPanel     MCPPanel
    memoryPanel  MemoryPanel
    wsPanel      WorkspacePanel
    renameModal  RenameModal
    rollback     RollbackPicker
    newSessModal NewSessionModal

    // 终端尺寸
    width  int
    height int
    ready  bool
}
```

### 5.2 StatusBar — 顶部状态栏

```go
// components/statusbar.go

type StatusBar struct {
    AppName    string   // "Devo"
    Session    string   // 会话标题
    Processing bool     // 处理中状态
    Paused     bool     // 暂停状态
    Yolo       bool     // YOLO 模式
    Connected  bool     // 服务连接状态
    Width      int
}
```

**显示内容**（从左到右）：
- 状态圆点 ●（颜色对应状态）
- 应用名 "Devo"
- 会话名
- 状态文本（Idle / Thinking / ToolExecuting / AwaitingApproval / Paused）
- YOLO 标签（开启时显示）
- 连接状态 + 端口号
- 主题切换按钮 🌙/☀️

### 5.3 InputArea — 输入区域

```go
// components/inputarea.go

type InputArea struct {
    textarea   textarea.Model
    width      int
    processing bool
}
```

- 3 行高度，自适应宽度
- 无行号，无边框
- 上下分隔线区分输入区与聊天区
- 右侧显示发送按钮（处理中显示 spinner）
- 支持多行输入，Enter 发送

### 5.4 OverlayStack — 覆盖层管理器

```go
// overlays/stack.go

type OverlayType int

const (
    OverlayNone OverlayType = iota
    OverlayApproval
    OverlayHelp
    OverlayCommand
    OverlaySession
    OverlayFiles
    OverlaySkills
    OverlayMCP
    OverlayMemory
    OverlayWorkspace
    OverlayNewSession
    OverlayRename
    OverlayRollback
)

type OverlayStack struct {
    current OverlayType
}

func (os *OverlayStack) Open(t OverlayType)
func (os *OverlayStack) Close() bool
func (os *OverlayStack) IsOpen() bool
```

所有覆盖层统一使用 `Render()` / `CursorUp()` / `CursorDown()` 接口，支持键盘导航。

### 5.5 覆盖层面板列表

| 面板 | 触发方式 | 文件 | 说明 |
| :--- | :--- | :--- | :--- |
| 命令面板 | `/`（输入框为空） | `overlays/command.go` | 分类命令列表，支持搜索过滤 |
| 会话选择器 | `Ctrl+S` | `overlays/session.go` | 会话切换，支持搜索 |
| 帮助面板 | `?` | `overlays/help.go` | 所有快捷键说明 |
| 文件管理 | `/files` | `overlays/files.go` | 工作区文件浏览 |
| 技能管理 | `/skills` | `overlays/skills.go` | 技能启用/禁用 |
| MCP 管理 | `/mcp` | `overlays/mcp.go` | MCP 服务器连接管理 |
| 记忆管理 | `/memory` | `overlays/memory.go` | 记忆条目查看 |
| 工作区管理 | `/workspace` | `overlays/workspace.go` | 工作区切换 |
| 新建会话 | `/new` | `overlays/new_session.go` | 输入名称创建会话 |
| 重命名 | `/rename` | `overlays/rename.go` | 重命名当前会话 |
| 回滚选择 | `/rollback` | `overlays/rollback.go` | 选择回滚点 |
| 审批弹窗 | SSE 触发 | `overlays/approval.go` | 审批操作确认 |

### 5.6 消息渲染

```go
// renderer/renderer.go

type msgRenderer struct {
    cache    *renderCache
    width    int
    mdRender glamour.TermRenderer
}

type renderCache struct {
    cache []string
    dirty int
}
```

**渲染规则**：

| 消息角色 | 样式 | 前缀 |
| :--- | :--- | :--- |
| `user` | 右对齐，accent 色边框 | `[User] HH:MM` |
| `assistant` | 左对齐，Markdown 渲染 | `[Assistant] HH:MM` |
| `system` | 居中，灰色 | 无前缀 |
| 工具调用 | 可折叠卡片，绿色(成功)/红色(失败) | `┌ Tool: read_file ─┐` |

**CJK 文本处理**（`renderer/cjk.go`）：
- 中英文混排自动换行，基于字符显示宽度计算
- 先 wrapCJK 预换行，再交给 glamour 做 Markdown 渲染
- 避免中文长文本不换行问题

**消息跳转**：
- `Ctrl+U`：跳转到上一条用户消息
- `Ctrl+D`：跳转到下一条用户消息
- 基于渲染行偏移量定位，输入框为空时生效

---

## 6. 目录结构

```
internal/interfaces/tui/
├── launcher.go                    # Launch()：接收 baseURL，启动 TUI
├── launcher_unix.go               # Unix 平台选项（WithAltScreen）
├── launcher_windows.go            # Windows 平台选项
│
├── model.go                       # MVC-Model：状态、数据、业务逻辑
├── view.go                        # MVC-View：主渲染入口
├── update.go                      # MVC-Controller：事件处理、命令路由
│
├── components/                    # 可复用 UI 组件
│   ├── statusbar.go               # 顶部状态栏
│   ├── inputarea.go               # 输入区域
│   ├── toast.go                   # Toast 浮动提示
│   └── styles.go                  # 主题系统 + 全局样式
│
├── overlays/                      # 覆盖层面板（每个面板一个文件）
│   ├── stack.go                   # OverlayStack 管理器
│   ├── command.go                 # 命令面板
│   ├── session.go                 # 会话选择器
│   ├── help.go                    # 帮助面板
│   ├── files.go                   # 文件管理
│   ├── skills.go                  # 技能管理
│   ├── mcp.go                     # MCP 管理
│   ├── memory.go                  # 记忆管理
│   ├── workspace.go               # 工作区管理
│   ├── new_session.go             # 新建会话
│   ├── rename.go                  # 重命名
│   ├── rollback.go                # 回滚选择
│   └── approval.go                # 审批弹窗
│
├── renderer/                      # 消息渲染层
│   ├── renderer.go                # 渲染器 + 缓存
│   └── cjk.go                     # CJK 文本换行
│
├── api/                           # API 对接层（参考 Web 前端接口）
│   ├── client.go                  # HTTP 客户端
│   ├── sse.go                     # SSE 客户端
│   ├── sessions.go                # 会话相关 API
│   ├── messages.go                # 消息相关 API
│   └── workspace.go               # 工作区相关 API
│
├── types/                         # 类型定义（参考 Web 前端 types/）
│   ├── session.go                 # SessionInfo 会话类型
│   ├── message.go                 # Message 消息类型
│   ├── sse.go                     # SSEEvent SSE 事件类型
│   └── approval.go                # ApprovalRequest 审批类型
│
├── model_test.go                  # Model 层测试
├── view_test.go                   # View 渲染测试
├── update_test.go                 # Controller 测试
├── components/*_test.go           # 组件测试
├── overlays/*_test.go             # 面板测试
├── renderer/*_test.go             # 渲染器测试
└── types/*_test.go                # 类型测试
```

### 依赖关系

```
cmd/devo/main.go
    ├── --tui 模式:
    │   ├── go server.ListenAndServe()     ← 协程启动 HTTP 服务
    │   └── tui.Launch(baseURL)            ← 传入 URL 启动 TUI
    └── 默认模式:
        └── http.ListenAndServe()          ← 直接阻塞运行

internal/interfaces/tui/
    ├── launcher.go     → model.go
    ├── model.go        → components/*, overlays/*, renderer/*, types/*, api/*
    ├── view.go         → components/*, overlays/*, renderer/*
    ├── update.go       → model.go, api/*
    ├── components/*    → bubbletea, bubbles, lipgloss
    ├── overlays/*      → lipgloss
    ├── renderer/*      → glamour, lipgloss
    ├── api/*           → net/http (标准库)
    └── types/*         → (纯类型，无外部依赖)
```

**关键约束**：
- `internal/interfaces/tui/` **不依赖** `internal/core/`、`internal/taskexec/`、`internal/storage/`
- 与现有 `internal/interfaces/rest/` 并列，互不引用
- `cmd/devo/main.go` 是唯一入口，所有服务初始化代码集中于此

---

## 7. API 接口

参考 Web 前端 REST API，TUI 对接以下接口：

### 7.1 会话管理

| 方法 | 路径 | 用途 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/sessions` | 获取会话列表 |
| `POST` | `/api/v1/sessions` | 创建新会话 |
| `GET` | `/api/v1/sessions/{id}` | 获取会话详情 |
| `PUT` | `/api/v1/sessions/{id}` | 重命名会话 |
| `DELETE` | `/api/v1/sessions/{id}` | 删除会话 |
| `PUT` | `/api/v1/sessions/{id}/config` | 更新会话配置 |
| `PUT` | `/api/v1/sessions/{id}/trust` | 设置信任级别 |
| `PUT` | `/api/v1/sessions/{id}/approval-policy` | 设置审批策略 |

### 7.2 消息

| 方法 | 路径 | 用途 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions/{id}/messages` | 发送消息 |
| `GET` | `/api/v1/sessions/{id}/messages` | 获取消息历史 |
| `GET` | `/api/v1/sessions/{id}/events` | SSE 事件流 |

### 7.3 状态控制

| 方法 | 路径 | 用途 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions/{id}/cancel` | 取消操作 |
| `POST` | `/api/v1/sessions/{id}/pause` | 暂停 |
| `POST` | `/api/v1/sessions/{id}/resume` | 恢复 |
| `POST` | `/api/v1/sessions/{id}/rollback` | 回滚到指定消息 |
| `POST` | `/api/v1/sessions/{id}/compact` | 压缩上下文 |
| `POST` | `/api/v1/sessions/{id}/complete` | 完成会话 |

### 7.4 审批

| 方法 | 路径 | 用途 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions/{id}/approve/{approval_id}` | 批准/拒绝操作 |

### 7.5 工作区与文件

| 方法 | 路径 | 用途 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/current-workspace` | 获取当前工作区 |
| `POST` | `/api/v1/current-workspace` | 切换工作区 |
| `GET` | `/api/v1/workspace` | 工作区列表 |
| `DELETE` | `/api/v1/workspace` | 删除工作区 |
| `GET` | `/api/v1/files` | 获取文件列表 |

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
| `Shift+↑` | 上一条输入历史 |
| `Shift+↓` | 下一条输入历史 |

### 8.3 模式切换

| 快捷键 | 说明 |
| :--- | :--- |
| `Ctrl+T` | 切换主题（暗色 / 亮色） |
| `Ctrl+Y` | 切换 YOLO 模式 |
| `Ctrl+P` | 暂停 / 恢复 |

### 8.4 覆盖层

| 快捷键 | 说明 |
| :--- | :--- |
| `Esc` | 关闭当前覆盖层 / 面板 |
| `?` | 打开帮助面板 |
| `↑` / `↓` / `j` / `k` | 面板内光标移动 |
| `Enter` | 面板内确认选择 |

### 8.5 系统

| 快捷键 | 说明 |
| :--- | :--- |
| `Ctrl+C` | 退出 |
| `Ctrl+Q` | 退出 |

---

## 9. 主题系统

### 9.1 主题定义

```go
// components/styles.go

type Theme struct {
    Name          string
    BgPrimary     lipgloss.Color
    TextPrimary   lipgloss.Color
    TextSecondary lipgloss.Color
    TextTertiary  lipgloss.Color
    Accent        lipgloss.Color
    Success       lipgloss.Color
    Warning       lipgloss.Color
    Error         lipgloss.Color
    Border        lipgloss.Color
}

var Dark = Theme{
    Name:          "dark",
    BgPrimary:     "#0d1117",
    TextPrimary:   "#e6edf3",
    TextSecondary: "#8b949e",
    TextTertiary:  "#6e7681",
    Accent:        "#58a6ff",
    Success:       "#3fb950",
    Warning:       "#d29922",
    Error:         "#f85149",
    Border:        "#30363d",
}

var Light = Theme{
    Name:          "light",
    BgPrimary:     "#ffffff",
    TextPrimary:   "#1f2328",
    TextSecondary: "#656d76",
    TextTertiary:  "#8b949e",
    Accent:        "#0969da",
    Success:       "#1a7f37",
    Warning:       "#9a6700",
    Error:         "#cf222e",
    Border:        "#d0d7de",
}
```

### 9.2 色彩语义

| Token | 用途 |
| :--- | :--- |
| `Accent` | 主色调：用户消息、选中高亮、快捷键 |
| `Success` | 成功：工具执行成功、连接状态 |
| `Warning` | 警告：审批等待、YOLO 模式 |
| `Error` | 错误：执行失败、断连 |
| `TextSecondary` (Muted) | 次要文本：分隔线、提示文字、时间戳 |
| `Border` | 边框：卡片边框、面板分隔 |

### 9.3 样式常量

- `PanelHeaderStyle(w)` — 面板标题栏（粗体、accent 色）
- `PanelFooterStyle(w)` — 面板底部栏（灰色、居中）
- `dimBgColor()` — 遮罩层背景色（比主背景略深，模拟半透明）

---

## 10. 实现计划

| 阶段 | 编号 | 任务 | 内容 | 预估 | 依赖 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1** | TUI-01 | 目录结构 + 类型定义 | 创建目录结构，`types/` 下定义所有类型 | 1h | — |
| **1** | TUI-02 | 样式系统 | `components/styles.go`：主题定义、色彩函数、样式常量 | 0.5h | TUI-01 |
| **1** | TUI-03 | 基础组件 | `components/`：StatusBar、InputArea、Toast | 1.5h | TUI-02 |
| **2** | TUI-04 | 渲染器 | `renderer/`：消息渲染、CJK 换行、缓存 | 2h | TUI-02 |
| **2** | TUI-05 | 覆盖层框架 | `overlays/stack.go`：OverlayStack 管理器 | 0.5h | TUI-02 |
| **2** | TUI-06 | 覆盖层面板 | `overlays/`：命令面板、会话选择器、帮助面板等 12 个面板 | 3h | TUI-05 |
| **3** | TUI-07 | Model 层 | `model.go`：状态管理、业务逻辑、消息跳转 | 2h | TUI-03,04,06 |
| **3** | TUI-08 | View 层 | `view.go`：主渲染，整合所有组件 | 1.5h | TUI-03,04,06 |
| **3** | TUI-09 | Controller 层 | `update.go`：事件处理、快捷键、命令路由 | 2h | TUI-07 |
| **4** | TUI-10 | API 对接层 | `api/`：HTTP 客户端、SSE 客户端、各接口封装 | 2h | TUI-07 |
| **4** | TUI-11 | 启动入口 | `launcher.go`：对接 main.go 的 --tui flag | 0.5h | TUI-09,10 |
| **5** | TUI-12 | 测试 | 所有 `*_test.go`：组件、面板、渲染、路由、API Mock | 3h | TUI-03~11 |
| **5** | TUI-13 | 旧代码清理 | 删除旧 `internal/interfaces/tui/` 代码 | 0.5h | TUI-12 |

**总计**：约 20 小时。

---

## 11. 验收标准

1. **启动流程**：在项目目录下执行 `devo --tui`，自动初始化服务、协程启动 HTTP 服务、创建会话、进入聊天界面
2. **消息对话**：输入文本后发送，Agent 返回回复，消息气泡正确渲染（含 CJK 文本）
3. **实时事件**：SSE 事件实时展示（thinking、tool_call_request、tool_result、message_complete）
4. **工具调用展示**：工具调用卡片正确显示工具名、参数摘要、结果摘要、耗时，Tab 键展开/折叠
5. **审批交互**：
   - 高风险操作触发审批弹窗，展示 diff 摘要
   - 按 `Y` 批准执行，按 `N` 拒绝
   - YOLO 模式下自动批准，显示 badge 标记
6. **会话管理**：命令面板切换会话、新建会话（自动使用当前目录）
7. **状态控制**：`Ctrl+P` 暂停/恢复、`Ctrl+U`/`Ctrl+D` 消息跳转，状态栏实时更新
8. **主题切换**：`Ctrl+T` 切换暗色/亮色主题，所有组件即时响应
9. **覆盖层交互**：覆盖层仅替换视口区域，StatusBar 和 InputArea 保持可见，按 `Esc` 关闭
10. **退出清理**：`Ctrl+Q` 退出时，TUI 恢复终端，main() 依次关闭 HTTP 服务和数据库，无残留协程
11. **异常处理**：服务启动失败时 log.Fatal 退出；SSE 断连时自动重连
12. **跨平台**：Windows、macOS、Linux 终端均可正常运行
13. **测试覆盖**：所有组件、面板、渲染器、路由逻辑有对应测试，`go test ./...` 全部通过

---

## 附录 A：关键设计决策

| 决策 | 选项 | 选择 | 理由 |
| :--- | :--- | :--- | :--- |
| TUI 与 Server 关系 | 父子进程 | **协程内嵌** | 单一进程，统一入口；无需维护子进程生命周期 |
| 端口分配 | 固定端口 | **随机端口** | 避免多实例冲突；无需用户配置 |
| 架构模式 | 单文件 App | **MVC 三层拆分** | model/view/update 职责清晰，便于测试和维护 |
| 组件拆分 | 单文件 | **按组件分层** | components/overlays/renderer/api 各司其职 |
| 覆盖层管理 | 独立 bool 标志 | **OverlayStack** | 统一管理，支持嵌套（预留），代码更简洁 |
| 样式方案 | 硬编码色值 | **Theme 系统** | Dark/Light 切换，色彩语义化，与 Web 端对齐 |
| 文本渲染 | 仅 glamour | **CJK wrap + glamour** | 中文长文本不换行，先预换行再 Markdown 渲染 |
| 消息跳转 | 逐行滚动 | **行偏移量定位** | 基于渲染缓存计算偏移，精准跳转到用户消息 |
| 入口模式 | 独立二进制 | **单一入口 + flag** | `devo --tui` 启动 TUI；`devo` 启动纯服务；统一构建 |
| API 设计 | 自定义 | **参考 Web 前端** | 与 REST API 完全对齐，复用同一套接口 |

---

## 附录 B：与 REST API 的映射

| TUI 操作 | HTTP 方法 | API 路径 |
| :--- | :--- | :--- |
| 自动创建会话 | `POST` | `/api/v1/sessions` |
| 列出会话 | `GET` | `/api/v1/sessions` |
| 获取会话详情 | `GET` | `/api/v1/sessions/{id}` |
| 重命名会话 | `PUT` | `/api/v1/sessions/{id}` |
| 删除会话 | `DELETE` | `/api/v1/sessions/{id}` |
| 发送消息 | `POST` | `/api/v1/sessions/{id}/messages` |
| 获取消息历史 | `GET` | `/api/v1/sessions/{id}/messages` |
| 获取文件列表 | `GET` | `/api/v1/files` |
| SSE 事件流 | `GET` | `/api/v1/sessions/{id}/events` |
| 批准操作 | `POST` | `/api/v1/sessions/{id}/approve/{approval_id}` |
| 暂停 | `POST` | `/api/v1/sessions/{id}/pause` |
| 恢复 | `POST` | `/api/v1/sessions/{id}/resume` |
| 取消 | `POST` | `/api/v1/sessions/{id}/cancel` |
| 回滚 | `POST` | `/api/v1/sessions/{id}/rollback` |
| 压缩上下文 | `POST` | `/api/v1/sessions/{id}/compact` |
| 完成 | `POST` | `/api/v1/sessions/{id}/complete` |
| 设置信任级别 | `PUT` | `/api/v1/sessions/{id}/trust` |
| 设置审批策略 | `PUT` | `/api/v1/sessions/{id}/approval-policy` |
| 切换工作区 | `POST` | `/api/v1/current-workspace` |
| 工作区列表 | `GET` | `/api/v1/workspace` |
| 删除工作区 | `DELETE` | `/api/v1/workspace` |