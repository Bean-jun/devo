# Devo Web 前端架构设计文档

**版本**：1.0.0

**定位**：基于 Vue 3 + Vite + TypeScript 的浏览器端单页应用（SPA），作为 Devo 编码代理系统的 Web 交互入口。Web 前端与 Devo Go 后端通过 HTTP 协议（REST + SSE）通信，部署在 `devo --web` 模式下由 Go 服务内嵌静态文件服务提供。用户在浏览器中直接操作，也可封装为 VS Code Webview 扩展使用。

---

## 1. 总体架构

### 1.1 进程模型

Web 前端与 Devo 服务器的关系与 TUI 模式一致——**同一进程内**，Go 后端通过 `--web` 模式启动 HTTP 服务，同时将 Vue 构建产物作为静态文件托管。浏览器访问 `http://localhost:<port>` 即可加载页面。

```
┌──────────────────────────────────────────────────────────┐
│                    单一进程 (devo --web)                   │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │  main()                                            │  │
│  │  ├── 初始化 DB、工具、LLM、AgentLoop、REST Handler │  │
│  │  ├── 分配端口（默认 8080 或随机）                    │  │
│  │  ├── 注册静态文件路由 → web/dist/                   │  │
│  │  ├── 注册 API 路由 → /api/v1/*                     │  │
│  │  └── http.ListenAndServe()  ← 阻塞运行             │  │
│  └────────────────────────────────────────────────────┘  │
│                         │                                │
│  ┌──────────────────────┼──────────────────────────────┐ │
│  │              浏览器 (Browser)                        │ │
│  │  ┌──────────────┐   │   ┌───────────────────┐      │ │
│  │  │  Vue SPA     │←──┼──→│ REST API          │      │ │
│  │  │  (静态文件)  │   │   │ /api/v1/*         │      │ │
│  │  └──────────────┘   │   └───────────────────┘      │ │
│  │  ┌──────────────┐   │                               │ │
│  │  │  EventSource │←──┼──→ SSE 事件流                 │ │
│  │  │  (SSE 消费)  │   │   /api/v1/sessions/{id}/events│ │
│  │  └──────────────┘   │                               │ │
│  │                     │                               │ │
│  │  ┌──────────────────────────────────────────────┐  │ │
│  │  │         Vue 渲染层                            │  │ │
│  │  │  ChatPanel · CommandPalette · ApprovalModal  │  │ │
│  │  └──────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

**核心原则**：
- 单一进程，单一入口 `cmd/devo/main.go`，通过 `--web` flag 切换模式
- Go 后端同时服务 API 和静态文件，不依赖额外的前端开发服务器
-有效通信仅通过 HTTP（REST + SSE），前端不直接引用 Go 内部包
- 生产环境：`devo --web` → 自动打开浏览器；开发环境：`npm run dev` + 代理到 Go 后端
- 同一套代码，浏览器直接运行，也可嵌入 VS Code Webview

### 1.2 启动流程

```
用户：devo --web
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
│ 2. 分配端口                          │
│    默认 :8080，被占用则随机           │
│    port = resolvePort()             │
├─────────────────────────────────────┤
│ 3. 注册静态文件路由                   │
│    mux.Handle("/", FileServer(      │
│      embed.FS(web/dist)))            │
│    SPA fallback：所有非 /api 路径    │
│    返回 index.html                   │
├─────────────────────────────────────┤
│ 4. 启动 HTTP 服务（阻塞）            │
│    http.ListenAndServe(port, mux)   │
│    打印访问地址，自动打开浏览器       │
└─────────────────────────────────────┘
```

**退出流程**：
```
用户关闭浏览器标签页 / Ctrl+C
    │
    ▼
1. 浏览器断开 SSE 连接
2. 后端检测到 SSE 断连（不影响会话状态）
3. 用户 Ctrl+C 终止服务
4. store.Close() 关闭数据库
5. 进程结束
```

### 1.3 开发模式

```
开发阶段：
  Terminal 1:  npm run dev          → Vite dev server (localhost:5173)
  Terminal 2:  devo --api-only      → Go 后端 (localhost:8080, 仅 API)
  Vite proxy:  /api → localhost:8080

生产阶段：
  npm run build                     → 输出到 web/dist/
  devo --web                        → 单进程，API + 静态文件
```

---

## 2. 技术选型

| 技术 | 用途 | 说明 |
| :--- | :--- | :--- |
| Vue 3 (Composition API) | 前端框架 | `<script setup lang="ts">`，响应式 + 组合式 |
| Vite | 构建工具 | 快速 HMR，ESBuild 编译 |
| TypeScript | 类型系统 | 严格模式，接口约束 API 数据结构 |
| Pinia | 状态管理 | 轻量级，支持 DevTools，模块化 Store |
| EventSource | SSE 消费 | 浏览器原生 API，自动重连，零依赖 |
| marked + highlight.js | Markdown 渲染 | 轻量 Markdown 解析 + 代码语法高亮 |
| 手写 CSS | 样式 | 零 UI 框架依赖，CSS 变量 + 模块化 |

**不使用的技术及理由**：

| 不选用 | 理由 |
| :--- | :--- |
| Element Plus / Naive UI | 要求手写 CSS，零 UI 依赖 |
| Tailwind CSS | 同上，要求手写 CSS |
| Vue Router | 单页单视图，无需路由 |
| Axios | fetch 原生支持足够，减少依赖 |
| WebSocket | SSE 单向推送已满足需求 |

---

## 3. 组件树

```
App.vue
├── StatusBar.vue                    # 顶部状态栏
│   ├── 会话状态指示器
│   ├── 当前会话名称
│   ├── Token 用量显示
│   └── 连接状态指示器
│
├── ChatPanel.vue                    # 聊天主面板（核心）
│   ├── MessageList.vue              # 消息列表
│   │   ├── MessageBubble.vue        # 消息气泡（用户/助手/系统）
│   │   │   ├── 用户消息：右对齐，蓝色气泡
│   │   │   ├── 助手消息：左对齐，Markdown 渲染
│   │   │   └── 系统消息：居中，灰色提示
│   │   ├── ToolCallCard.vue         # 工具调用卡片
│   │   │   ├── 工具名称 + 图标
│   │   │   ├── 参数摘要（可折叠）
│   │   │   ├── 执行结果（可折叠）
│   │   │   └── 耗时显示
│   │   ├── ThinkingIndicator.vue    # 思考中动画
│   │   └── RollbackMarker.vue       # 回滚标记
│   │
│   └── InputArea.vue                # 底部输入区
│       ├── 多行文本输入框
│       ├── 发送按钮
│       ├── 停止按钮（处理中时显示）
│       └── 字符计数 / Token 估算
│
├── CommandPalette.vue               # / 命令面板
│   ├── 命令输入框
│   ├── 命令列表（/new, /sessions, /switch, /archive, /rollback, /help）
│   ├── 自动补全建议
│   └── 搜索结果列表
│
├── ApprovalModal.vue                # 审批弹窗
│   ├── 操作类型 + 风险等级
│   ├── Diff 对比展示
│   ├── 命令内容展示
│   ├── 文件路径显示
│   ├── 批准 / 拒绝 按钮
│   └── 超时倒计时
│
├── SessionPicker.vue                # 会话选择器（/sessions 命令触发）
│   ├── 会话列表
│   ├── 搜索过滤
│   ├── 状态标签（active/archived）
│   └── 选择确认
│
├── RollbackPicker.vue               # 回滚选择器（/rollback 命令触发）
│   ├── 消息时间线
│   ├── 选中预览
│   └── 确认回滚
│
├── ToastContainer.vue               # 浮动提示容器
│   └── ToastItem.vue                # 单条提示（success/error/info/warning）
│
└── HelpPanel.vue                    # 帮助面板（/help 命令触发）
    ├── 快捷键列表
    ├── / 命令列表
    └── 版本信息
```

---

## 4. 组件详细设计

### 4.1 App.vue — 顶层布局

```
┌──────────────────────────────────────────────┐
│  StatusBar                                   │
├──────────────────────────────────────────────┤
│                                              │
│                                              │
│              ChatPanel                       │
│          (占据全部剩余空间)                    │
│                                              │
│                                              │
├──────────────────────────────────────────────┤
│  InputArea                                   │
└──────────────────────────────────────────────┘

覆盖层（按需出现）：
  · CommandPalette —— 输入 / 时从顶部滑入
  · ApprovalModal  —— 审批时居中弹出
  · SessionPicker  —— /sessions 命令时居中弹出
  · RollbackPicker —— /rollback 命令时居中弹出
  · HelpPanel      —— /help 命令时居中弹出
  · ToastContainer —— 右上角固定浮动
```

**布局约束**：
- 全屏布局，100vw × 100vh，无滚动条
- StatusBar 固定高度 40px，顶部固定
- InputArea 固定高度（最小 48px，最大 200px，可拖拽调整），底部固定
- ChatPanel 填充剩余空间，内部 MessageList 可滚动
- 所有弹窗/面板使用 `<Teleport to="body">` 渲染，z-index 层级管理

### 4.2 StatusBar — 顶部状态栏

**显示内容**（从左到右）：
- 会话名称（如 "my-project" 或 "未命名会话"）
- 会话状态指示器（圆点 + 文字）：
  - 🟢 Idle（空闲）
  - 🔵 Processing（处理中，带脉冲动画）
  - 🟡 AwaitingApproval（等待审批）
  - ⏸️ Paused（已暂停）
  - ✅ Completed（已完成）
  - 📦 Archived（已归档）
- Token 用量：`已用: 12,345 / 上限: 100,000`
- 连接状态：🟢 在线 / 🔴 离线

**交互**：
- 点击会话名称：可重命名会话
- 点击状态指示器：可暂停/恢复/取消会话

**CSS 要点**：
- 背景：深色半透明，与主面板形成层次
- 状态指示器脉冲动画：`@keyframes pulse` 缩放 + 透明度变化
- 字体：等宽数字（Token 计数），系统默认字体

### 4.3 ChatPanel — 聊天面板

**核心容器**，包含 MessageList 和 InputArea。

**MessageList 滚动行为**：
- 新消息到达时自动滚动到底部
- 用户手动上滚查看历史时，不强制滚动（"新消息" 提示按钮）
- 虚拟滚动优化（消息超过 500 条时启用）

**空状态**：
- 首次进入时显示欢迎语 + 快速开始提示
- 输入 `/help` 查看可用命令

### 4.4 MessageBubble — 消息气泡

**消息类型与渲染**：

| 类型 | 对齐 | 样式 | 内容渲染 |
| :--- | :--- | :--- | :--- |
| 用户消息 | 右对齐 | 蓝色气泡，圆角 | 纯文本，URL 自动链接 |
| 助手消息 | 左对齐 | 浅灰气泡，圆角 | Markdown 渲染（代码块 + 语法高亮） |
| 系统消息 | 居中 | 灰色小字，无气泡 | 纯文本（如 "会话已创建"、"已回滚到..."） |
| 工具调用 | 左对齐 | 卡片样式，边框 | 工具名 + 参数 + 结果 |
| 思考过程 | 左对齐 | 斜体，灰色，折叠 | 推理文本（可折叠，默认折叠） |

**Markdown 渲染**：
- 使用 `marked` 解析 Markdown
- 代码块使用 `highlight.js` 进行语法高亮
- 支持的语言：Go、Python、JavaScript、TypeScript、Shell、SQL、YAML、JSON、Markdown、Diff
- 代码块右上角显示语言标签 + 复制按钮
- 支持表格、列表、引用、链接、图片

**消息元数据**：
- 每条消息显示时间戳（hover 显示完整时间）
- 助手消息显示 Token 消耗（小字灰色）

### 4.5 ToolCallCard — 工具调用卡片

**结构**：
```
┌──────────────────────────────────────────────┐
│  🔧 write_file  ·  file.txt  ·  234ms        │
│  ────────────────────────────────────────────│
│  ▶ 参数（可折叠）                             │
│  ────────────────────────────────────────────│
│  ▶ 结果（可折叠，默认展开）                    │
│    · 成功写入 1,234 字节                       │
│    · 文件路径：/project/src/file.txt          │
└──────────────────────────────────────────────┘
```

**状态样式**：
- 执行中：橙色边框，旋转图标
- 成功：绿色边框，✅ 图标
- 失败：红色边框，❌ 图标
- 被拒绝：灰色边框，🚫 图标

**可折叠区域**：
- 参数区默认折叠（点击展开）
- 结果区默认展开（内容超过 10 行则折叠，显示 "展开全部" 按钮）

### 4.6 ThinkingIndicator — 思考指示器

**动画状态**：
- AI 思考中：三个圆点依次弹跳动画（`...`）
- 工具调用中：工具名称 + 旋转图标
- 重试中：`第 N 次尝试...` 文字 + 脉冲动画

**CSS 动画**：
- `@keyframes bounce`：三个圆点依次缩放（0% → 50% → 100%）
- `@keyframes pulse`：透明度变化（0.3 → 1.0 → 0.3）
- `@keyframes spin`：旋转 360°

### 4.7 InputArea — 输入区

**功能**：
- `<textarea>` 多行输入，自动调整高度
- Enter 发送消息（Shift+Enter 换行）
- 发送按钮（右侧，圆角，蓝色）
- 停止按钮（Processing 状态时替换发送按钮，红色）
- 字符计数：右下角显示 `{count}/{max}`（默认上限 10000 字符）
- Token 估算：右下角显示约 `~{n} tokens`

**`/` 命令触发**：
- 输入 `/` 时，自动弹出 CommandPalette
- 继续输入匹配命令，Tab 自动补全
- 非命令文本时正常发送消息

**禁用状态**：
- 会话状态为 Processing 时禁用输入（但可点击停止按钮）
- 会话状态为 AwaitingApproval 时禁用输入（提示 "请先处理审批请求"）
- 会话状态为 Archived 时禁用输入（提示 "会话已归档，不可操作"）

### 4.8 CommandPalette — 命令面板

**触发方式**：
- 在 InputArea 中输入 `/` 自动弹出
- 快捷键 `Ctrl+K` 或 `Cmd+K`（Mac）

**命令列表**：

| 命令 | 参数 | 功能 | 说明 |
| :--- | :--- | :--- | :--- |
| `/new` | `[name]` | 创建新会话 | 可选会话名称，自动切换 |
| `/sessions` | `[filter]` | 查看会话列表 | 可搜索过滤 |
| `/switch` | `<id或name>` | 切换会话 | 支持 ID 或名称模糊匹配 |
| `/rename` | `<name>` | 重命名当前会话 | |
| `/archive` | — | 归档当前会话 | 需二次确认 |
| `/rollback` | — | 回滚消息 | 打开 RollbackPicker |
| `/pause` | — | 暂停当前会话 | |
| `/resume` | — | 恢复当前会话 | |
| `/cancel` | — | 取消当前操作 | |
| `/help` | — | 显示帮助 | 打开 HelpPanel |
| `/clear` | — | 清屏 | 仅视觉上清空消息列表，不删除数据 |

**交互**：
- 输入 `/` 后显示命令列表，模糊匹配过滤
- 上下箭头选择，Enter 确认，Esc 关闭
- 命令参数提示（如 `/switch` 后显示会话列表）
- 选中项高亮，右侧显示命令描述

**CSS 要点**：
- 从 InputArea 上方滑入，高度最大 300px
- 半透明背景 + 模糊效果（`backdrop-filter: blur()`）
- 命令项左对齐，描述右对齐，灰色小字
- 选中项蓝色背景，白色文字

### 4.9 ApprovalModal — 审批弹窗

**触发条件**：
- SSE 事件 `approval_required` 到达
- 自动弹出模态框，聚焦

**显示内容**：
- 标题：`审批请求 - {风险等级}`
- 风险等级标签（颜色区分）：
  - 🔴 高风险：红色标签
  - 🟡 中风险：黄色标签
  - 🟢 低风险：绿色标签
- 操作类型图标 + 名称（如 🔧 write_file）
- 文件路径 / 命令内容
- Diff 展示（`write_file` 时显示，使用 diff 语法高亮）
- 命令内容展示（`execute_command` 时显示，代码块样式）

**操作按钮**：
- 批准（绿色按钮，快捷键 `Y`）
- 拒绝（红色按钮，快捷键 `N`）
- 超时倒计时（如 "30s 后自动拒绝"）

**信任策略快捷操作**：
- "本次会话全部自动批准" 复选框
- （如果配置了 full_trust 策略，此弹窗不出现，仅显示通知）

**CSS 要点**：
- 居中模态框，宽 600px，最大高 80vh
- 背景遮罩，半透明黑色（`rgba(0,0,0,0.5)`）
- Diff 展示区使用等宽字体，增/删行用绿/红背景
- 按钮：批准（绿色，左边），拒绝（红色，右边）
- 入场动画：`@keyframes modalIn` 缩放 + 透明度
- 出场动画：`@keyframes modalOut` 缩放 + 透明度

### 4.10 SessionPicker — 会话选择器

**触发方式**：
- 输入 `/sessions` 或 `/switch` 命令

**显示内容**：
- 搜索框（顶部）
- 会话列表（可滚动）
  - 会话名称
  - 状态标签（active/archived）
  - 最后活动时间
  - 消息数量
- 新建会话按钮（底部）

**排序**：
- 默认按最后活动时间倒序
- 当前会话排在最前，高亮显示

**交互**：
- 点击会话行：切换到该会话
- 点击新建按钮：创建新会话
- 搜索过滤：按名称模糊匹配

**CSS 要点**：
- 居中面板，宽 480px，最大高 60vh
- 会话行 hover 高亮，当前会话蓝色边框
- 状态标签小圆角徽章

### 4.11 RollbackPicker — 回滚选择器

**触发方式**：
- 输入 `/rollback` 命令

**显示内容**：
- 消息时间线（垂直时间轴）
  - 每条消息显示：序号、角色、内容摘要、时间
  - 当前选中位置高亮
- 选中位置预览：显示将回滚到哪条消息
- 回滚后不可恢复的警告文字
- 确认 / 取消 按钮

**CSS 要点**：
- 居中面板，宽 500px，最大高 70vh
- 时间轴左侧竖线，选中点放大 + 蓝色
- 确认按钮红色（危险操作）

### 4.12 ToastContainer — 浮动提示

**位置**：右上角固定，`position: fixed; top: 48px; right: 16px;`

**类型**：

| 类型 | 图标 | 颜色 | 持续时间 | 示例 |
| :--- | :--- | :--- | :--- | :--- |
| success | ✅ | 绿色 | 3s | "会话创建成功" |
| error | ❌ | 红色 | 5s | "发送失败：网络错误" |
| info | ℹ️ | 蓝色 | 3s | "已切换到会话 'my-project'" |
| warning | ⚠️ | 橙色 | 5s | "Token 用量已达 80%" |

**堆叠行为**：
- 多条 Toast 垂直堆叠，间距 8px
- 最多显示 5 条，超出自动移除最早的一条
- 入场：从右侧滑入 + 透明度（`@keyframes slideInRight`）
- 出场：向右侧滑出 + 透明度（`@keyframes slideOutRight`）

---

## 5. 数据流设计

### 5.1 状态管理（Pinia Store）

```
stores/
├── session.ts          # 会话状态
│   ├── state: currentSession, sessions[], isLoading
│   ├── actions: createSession, switchSession, fetchSessions, renameSession, archiveSession
│   └── getters: isProcessing, isAwaitingApproval, sessionStatus
│
├── chat.ts             # 聊天状态
│   ├── state: messages[], isStreaming, pendingApproval
│   ├── actions: sendMessage, appendMessage, clearMessages, rollbackTo
│   └── getters: lastMessage, messageCount, canRollback
│
├── approval.ts         # 审批状态
│   ├── state: currentApproval, approvalHistory[]
│   ├── actions: approve, reject, setTrustLevel
│   └── getters: hasPendingApproval, approvalTimeout
│
├── command.ts          # 命令面板状态
│   ├── state: isOpen, query, suggestions[], selectedIndex
│   ├── actions: open, close, filter, select
│   └── getters: filteredCommands
│
└── ui.ts               # UI 全局状态
    ├── state: toasts[], activeModal, connectionStatus
    ├── actions: showToast, removeToast, setConnectionStatus
    └── getters: isConnected, isModalOpen
```

### 5.2 API 通信层

```
composables/
├── useApi.ts           # REST API 封装
│   ├── get<T>(path): Promise<T>
│   ├── post<T>(path, body): Promise<T>
│   ├── put<T>(path, body): Promise<T>
│   └── 统一错误处理、超时、重试
│
├── useSSE.ts           # SSE 事件流
│   ├── connect(sessionId): void
│   ├── disconnect(): void
│   ├── onMessage(callback): void
│   ├── onEvent(type, callback): void
│   └── 自动重连（指数退避，最大 30s）
│
└── useSession.ts       # 会话操作组合式函数
    ├── 封装 createSession、switchSession 等
    └── 同时更新 store 和调用 API
```

### 5.3 数据流示意图

```
用户输入 "帮我写一个排序函数"
    │
    ▼
InputArea.vue
    │ emit('send', text)
    ▼
ChatPanel.vue
    │ chatStore.sendMessage(text)
    ▼
useApi.ts                          useSSE.ts
    │ POST /api/v1/sessions/{id}/     │ EventSource 连接
    │   messages                       │
    ▼                                 ▼
Go 后端                              SSE 事件流
    │                                 │
    ▼                                 ▼
SSE 事件到达 ←───────────────────────┘
    │
    ▼
useSSE.onEvent('thinking', (data) => {
    chatStore.appendMessage({ type: 'thinking', content: data })
})
useSSE.onEvent('tool_call_request', (data) => {
    chatStore.appendMessage({ type: 'tool_call', ...data })
})
useSSE.onEvent('message_complete', (data) => {
    chatStore.appendMessage({ type: 'assistant', content: data })
    chatStore.isStreaming = false
})
useSSE.onEvent('approval_required', (data) => {
    approvalStore.currentApproval = data
    uiStore.activeModal = 'approval'
})
```

### 5.4 SSE 事件类型映射

| SSE 事件类型 | 触发时机 | 前端处理 |
| :--- | :--- | :--- |
| `thinking` | AI 开始思考/推理 | 显示 ThinkingIndicator，更新思考文本 |
| `tool_call_request` | AI 准备调用工具 | 显示 ToolCallCard（执行中状态） |
| `tool_result` | 工具执行完成 | 更新 ToolCallCard（成功/失败状态） |
| `tool_call_approved` | 工具调用被批准 | 更新 ToolCallCard 状态 |
| `tool_call_rejected` | 工具调用被拒绝 | 更新 ToolCallCard（拒绝状态） |
| `message_chunk` | 助手消息流式片段 | 追加到当前助手消息气泡 |
| `message_complete` | 助手消息完成 | 标记消息完成，停止流式动画 |
| `approval_required` | 需要用户审批 | 弹出 ApprovalModal |
| `session_status` | 会话状态变更 | 更新 StatusBar 状态指示器 |
| `token_usage` | Token 用量更新 | 更新 StatusBar Token 计数 |
| `error` | 发生错误 | 显示 Toast 错误提示 |
| `rollback` | 回滚完成 | 清除回滚点之后的消息，显示系统提示 |

---

## 6. 目录结构

```
web/                                # Vue 前端项目根目录
├── public/
│   └── favicon.svg                 # 网站图标
│
├── src/
│   ├── main.ts                     # Vue 应用入口
│   ├── App.vue                     # 根组件
│   │
│   ├── components/                 # UI 组件
│   │   ├── layout/
│   │   │   ├── StatusBar.vue       # 顶部状态栏
│   │   │   ├── ToastContainer.vue  # 浮动提示容器
│   │   │   └── ToastItem.vue       # 单条提示
│   │   │
│   │   ├── chat/
│   │   │   ├── ChatPanel.vue       # 聊天主面板
│   │   │   ├── MessageList.vue     # 消息列表
│   │   │   ├── MessageBubble.vue   # 消息气泡
│   │   │   ├── ToolCallCard.vue    # 工具调用卡片
│   │   │   ├── ThinkingIndicator.vue # 思考指示器
│   │   │   ├── RollbackMarker.vue  # 回滚标记
│   │   │   └── InputArea.vue       # 输入区
│   │   │
│   │   ├── command/
│   │   │   ├── CommandPalette.vue  # 命令面板
│   │   │   └── CommandItem.vue     # 命令项
│   │   │
│   │   ├── modal/
│   │   │   ├── ApprovalModal.vue   # 审批弹窗
│   │   │   ├── SessionPicker.vue   # 会话选择器
│   │   │   ├── RollbackPicker.vue  # 回滚选择器
│   │   │   └── HelpPanel.vue       # 帮助面板
│   │   │
│   │   └── shared/
│   │       ├── DiffView.vue        # Diff 对比视图
│   │       ├── CodeBlock.vue       # 代码块（语法高亮）
│   │       ├── MarkdownView.vue    # Markdown 渲染器
│   │       └── Spinner.vue         # 加载动画
│   │
│   ├── composables/                # 组合式函数
│   │   ├── useApi.ts               # REST API 封装
│   │   ├── useSSE.ts               # SSE 事件流
│   │   ├── useSession.ts           # 会话操作
│   │   ├── useCommand.ts           # 命令面板逻辑
│   │   ├── useKeyboard.ts          # 键盘快捷键
│   │   └── useAutoScroll.ts        # 自动滚动
│   │
│   ├── stores/                     # Pinia 状态管理
│   │   ├── session.ts              # 会话状态
│   │   ├── chat.ts                 # 聊天状态
│   │   ├── approval.ts            # 审批状态
│   │   ├── command.ts             # 命令面板状态
│   │   └── ui.ts                  # UI 全局状态
│   │
│   ├── types/                      # TypeScript 类型定义
│   │   ├── api.ts                  # API 请求/响应类型
│   │   ├── session.ts             # 会话类型
│   │   ├── message.ts             # 消息类型
│   │   ├── tool.ts                # 工具类型
│   │   ├── approval.ts            # 审批类型
│   │   └── sse.ts                 # SSE 事件类型
│   │
│   ├── utils/                      # 工具函数
│   │   ├── markdown.ts            # Markdown 渲染配置
│   │   ├── highlight.ts           # 代码高亮配置
│   │   ├── formatters.ts          # 格式化（时间、字节、Token）
│   │   └── constants.ts           # 常量定义
│   │
│   └── styles/                     # 全局样式
│       ├── variables.css           # CSS 变量（颜色、间距、字体）
│       ├── reset.css               # 浏览器重置样式
│       ├── base.css                # 基础样式
│       ├── animations.css          # 动画定义
│       └── typography.css          # 排版样式
│
├── index.html                      # HTML 入口
├── package.json                    # 项目配置
├── tsconfig.json                   # TypeScript 配置
├── vite.config.ts                  # Vite 配置
└── env.d.ts                        # 环境变量类型声明
```

### 依赖关系

```
App.vue
├── StatusBar           → useSession(), useSSE()
├── ChatPanel           → chatStore, useApi(), useSSE()
│   ├── MessageList     → chatStore
│   │   ├── MessageBubble → MarkdownView, CodeBlock
│   │   ├── ToolCallCard  → DiffView
│   │   └── ThinkingIndicator
│   └── InputArea       → commandStore, chatStore
├── CommandPalette      → commandStore, useSession()
├── ApprovalModal       → approvalStore, useApi()
├── SessionPicker       → sessionStore, useSession()
├── RollbackPicker      → chatStore, useApi()
├── ToastContainer      → uiStore
└── HelpPanel           → (纯展示)
```

**关键约束**：
- 组件不直接调用 API，统一通过 composables 或 stores
- SSE 事件处理集中在 `useSSE` composable 中，组件只消费 store 中的响应式数据
- 所有类型定义在 `types/` 中集中管理，store 和 composable 引用类型
- CSS 不交叉引用，组件样式通过 `<style scoped>` 隔离，全局变量在 `variables.css`

---

## 7. 样式系统

### 7.1 设计原则

- **简洁**：去除多余装饰，突出内容
- **现代**：扁平化风格，微妙阴影和圆角
- **专业**：适合开发者使用，类 IDE 风格
- **响应式**：适配不同屏幕尺寸（最小 800px 宽）

### 7.2 色彩方案

**亮色主题**（默认）：

| 角色 | 色值 | 用途 |
| :--- | :--- | :--- |
| 主背景 | `#ffffff` | 页面背景 |
| 次背景 | `#f5f5f7` | 卡片、面板背景 |
| 主文字 | `#1d1d1f` | 正文 |
| 次文字 | `#86868b` | 辅助信息、时间戳 |
| 主色调 | `#0071e3` | 按钮、链接、选中态 |
| 主色调悬停 | `#0077ed` | 按钮悬停 |
| 成功色 | `#34c759` | 成功状态、批准按钮 |
| 警告色 | `#ff9500` | 警告状态、中风险 |
| 错误色 | `#ff3b30` | 错误状态、高风险、拒绝按钮 |
| 边框色 | `#d2d2d7` | 分割线、边框 |
| 用户气泡 | `#0071e3` | 用户消息气泡背景 |
| 助手气泡 | `#f5f5f7` | 助手消息气泡背景 |
| 代码块背景 | `#1e1e1e` | 代码块（暗色） |

**暗色主题**（后续支持）：

| 角色 | 色值 | 用途 |
| :--- | :--- | :--- |
| 主背景 | `#1e1e1e` | 页面背景 |
| 次背景 | `#252526` | 卡片、面板背景 |
| 主文字 | `#cccccc` | 正文 |
| 次文字 | `#858585` | 辅助信息 |
| 主色调 | `#0078d4` | 按钮、链接、选中态 |

### 7.3 CSS 变量定义

```css
/* variables.css */
:root {
  /* 颜色 */
  --color-bg-primary: #ffffff;
  --color-bg-secondary: #f5f5f7;
  --color-bg-tertiary: #e8e8ed;
  --color-text-primary: #1d1d1f;
  --color-text-secondary: #86868b;
  --color-text-tertiary: #aeaeb2;
  --color-accent: #0071e3;
  --color-accent-hover: #0077ed;
  --color-success: #34c759;
  --color-warning: #ff9500;
  --color-error: #ff3b30;
  --color-border: #d2d2d7;
  --color-user-bubble: #0071e3;
  --color-user-bubble-text: #ffffff;
  --color-assistant-bubble: #f5f5f7;
  --color-code-bg: #1e1e1e;

  /* 阴影 */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.08);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.12);
  --shadow-modal: 0 16px 48px rgba(0, 0, 0, 0.2);

  /* 圆角 */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  --radius-full: 9999px;

  /* 间距 */
  --space-xs: 4px;
  --space-sm: 8px;
  --space-md: 12px;
  --space-lg: 16px;
  --space-xl: 24px;
  --space-2xl: 32px;

  /* 字体 */
  --font-sans: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
               'Microsoft YaHei', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas',
               'Monaco', monospace;
  --font-size-xs: 11px;
  --font-size-sm: 12px;
  --font-size-base: 14px;
  --font-size-lg: 16px;
  --font-size-xl: 18px;
  --font-size-2xl: 24px;

  /* 动画 */
  --transition-fast: 150ms ease;
  --transition-base: 250ms ease;
  --transition-slow: 400ms ease;

  /* 布局 */
  --statusbar-height: 40px;
  --input-min-height: 48px;
  --input-max-height: 200px;
}
```

### 7.4 动画定义

```css
/* animations.css */
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes fadeOut {
  from { opacity: 1; }
  to { opacity: 0; }
}

@keyframes slideInRight {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}

@keyframes slideOutRight {
  from { transform: translateX(0); opacity: 1; }
  to { transform: translateX(100%); opacity: 0; }
}

@keyframes slideInUp {
  from { transform: translateY(100%); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

@keyframes modalIn {
  from { transform: scale(0.95); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

@keyframes modalOut {
  from { transform: scale(1); opacity: 1; }
  to { transform: scale(0.95); opacity: 0; }
}

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
```

---

## 8. 键盘快捷键

| 快捷键 | 作用域 | 功能 |
| :--- | :--- | :--- |
| `Enter` | InputArea | 发送消息 |
| `Shift+Enter` | InputArea | 换行 |
| `Escape` | 全局 | 关闭弹窗/面板 |
| `Ctrl+K` / `Cmd+K` | 全局 | 打开命令面板 |
| `Y` | ApprovalModal | 批准操作 |
| `N` | ApprovalModal | 拒绝操作 |
| `Ctrl+P` / `Cmd+P` | 全局 | 暂停/恢复当前会话 |
| `Ctrl+C` | 全局 | 取消当前操作 |
| `Ctrl+L` | 全局 | 清屏（视觉上） |
| `Ctrl+[` | 全局 | 切换到上一个会话 |
| `Ctrl+]` | 全局 | 切换到下一个会话 |

---

## 9. 与 REST API 的映射

| 前端操作 | HTTP 方法 | API 路径 |
| :--- | :--- | :--- |
| 创建会话 | `POST` | `/api/v1/sessions` |
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

## 10. 实现计划

| 编号 | 阶段 | 任务 | 内容 | 预估 |
| :--- | :--- | :--- | :--- | :--- |
| WEB-01 | 项目骨架 | 初始化 Vue 项目 | Vite + Vue 3 + TS 脚手架，目录结构，基础配置 | 0.5h |
| WEB-02 | 项目骨架 | 全局样式系统 | CSS 变量、reset、base、animation、typography | 0.5h |
| WEB-03 | 项目骨架 | 类型定义 | `types/` 下全部类型文件 | 0.5h |
| WEB-04 | 基础设施 | API 通信层 | `useApi` 封装、`useSSE` 封装 | 1h |
| WEB-05 | 基础设施 | 状态管理 | Pinia stores（session、chat、approval、command、ui） | 1.5h |
| WEB-06 | 核心组件 | App.vue + StatusBar | 顶层布局 + 状态栏 | 1h |
| WEB-07 | 核心组件 | ChatPanel + MessageList | 聊天面板框架 + 消息列表容器 | 1h |
| WEB-08 | 核心组件 | MessageBubble | 消息气泡（用户/助手/系统） | 1.5h |
| WEB-09 | 核心组件 | Markdown 渲染 | CodeBlock + MarkdownView + 语法高亮 | 1h |
| WEB-10 | 核心组件 | ToolCallCard | 工具调用卡片 + 折叠展开 | 1h |
| WEB-11 | 核心组件 | ThinkingIndicator | 思考动画 + 流式效果 | 0.5h |
| WEB-12 | 核心组件 | InputArea | 输入框 + 发送/停止按钮 + `/` 命令触发 | 1h |
| WEB-13 | 命令面板 | CommandPalette | 命令面板 + 全部命令实现 | 1.5h |
| WEB-14 | 审批 | ApprovalModal | 审批弹窗 + Diff 展示 + 批准/拒绝 | 1.5h |
| WEB-15 | 会话管理 | SessionPicker | 会话选择器 + 新建/切换 | 1h |
| WEB-16 | 回滚 | RollbackPicker | 回滚选择器 + 时间线 | 1h |
| WEB-17 | 辅助 | ToastContainer | 浮动提示系统 | 0.5h |
| WEB-18 | 辅助 | HelpPanel | 帮助面板 | 0.5h |
| WEB-19 | 辅助 | 键盘快捷键 | 全局快捷键绑定 | 0.5h |
| WEB-20 | 后端适配 | --web 模式 | Go 后端添加 `--web` flag、静态文件托管、SPA fallback | 1h |
| WEB-21 | 后端适配 | 开发代理配置 | Vite proxy 配置，开发模式联调 | 0.5h |
| WEB-22 | 集成测试 | 端到端测试 | 启动→会话创建→对话→审批→回滚→切换→归档 | 2h |

**总计**：约 20 小时。

---

## 11. 验收标准

1. **启动流程**：在项目目录下执行 `devo --web`，自动启动服务、打开浏览器、加载页面、创建会话、进入聊天界面
2. **消息对话**：输入文本后发送，Agent 流式返回回复，消息气泡正确渲染，Markdown 正确解析
3. **代码高亮**：代码块支持 Go、Python、JavaScript、TypeScript、Shell、SQL、YAML、JSON、Markdown、Diff 语法高亮
4. **实时事件**：SSE 事件实时展示（thinking、tool_call_request、tool_result、message_complete），动画流畅
5. **工具调用展示**：工具调用卡片正确显示工具名、参数摘要、结果摘要、耗时，折叠/展开正常
6. **审批交互**：
   - 高风险操作触发审批弹窗，展示 diff 摘要
   - 点击"批准"执行，点击"拒绝"取消
   - 自动批准的操作显示 badge 标记
7. **命令面板**：
   - 输入 `/` 弹出命令面板，模糊匹配过滤
   - 支持 `/new`、`/sessions`、`/switch`、`/rename`、`/archive`、`/rollback`、`/pause`、`/resume`、`/cancel`、`/help`、`/clear`
   - 键盘上下选择，Enter 确认，Esc 关闭
8. **会话管理**：通过命令面板创建、切换、重命名、归档会话
9. **消息回滚**：通过 `/rollback` 命令打开回滚选择器，选择回滚点后确认执行
10. **状态控制**：`Ctrl+P` 暂停/恢复、`Ctrl+C` 取消，状态栏实时更新
11. **异常处理**：后端断连时显示 Toast 提示，SSE 自动重连
12. **响应式布局**：800px 以上宽度正常显示，消息列表正确滚动
13. **跨浏览器**：Chrome、Edge、Firefox 均可正常运行

---

## 12. 关键设计决策

| 决策 | 选项 | 选择 | 理由 |
| :--- | :--- | :--- | :--- |
| 会话管理方式 | 侧边栏 | **`/` 命令面板** | 用户需求；减少视觉干扰，聚焦聊天 |
| CSS 方案 | UI 框架 | **手写 CSS** | 用户需求；完全控制样式，零依赖 |
| 路由方案 | Vue Router | **单视图** | 单页单功能，无需路由 |
| 静态文件部署 | 独立前端服务器 | **Go 内嵌静态文件** | 与 TUI 模式一致，单进程，零配置 |
| Markdown 渲染 | 自研 | **marked + highlight.js** | 轻量、成熟、可定制 |
| SSE 客户端 | 第三方库 | **EventSource 原生** | 浏览器原生支持，零依赖 |
| 状态管理 | 自研 reactive | **Pinia** | Vue 官方推荐，DevTools 支持，模块化 |
| 暗色主题 | 首版支持 | **先亮色，后续迭代** | 降低首版复杂度，CSS 变量已预留 |
| 消息回滚 | 直接掉 API | **RollbackPicker 可视化** | 提升用户体验，比 TUI 更强的交互 |
| 开发模式 | `npm run dev` 独立 | **Vite proxy 到 Go 后端** | 开发时 HMR 快速迭代，生产时单进程 |

---

## 附录 A：与 TUI 的差异对比

| 维度 | TUI | Web |
| :--- | :--- | :--- |
| 渲染层 | bubbletea + lipgloss | Vue 3 + 手写 CSS |
| 会话管理 | 固定侧边栏 | `/` 命令面板 |
| 代码展示 | 纯文本，无高亮 | 语法高亮 + 复制按钮 |
| 审批交互 | 终端弹窗，Y/N 按键 | 模态框，鼠标点击 + 键盘快捷键 |
| 消息回滚 | 快捷键触发 | 可视化时间线选择器 |
| 输入方式 | 纯键盘 | 键盘 + 鼠标 |
| 动画效果 | 终端限制，无动画 | CSS 动画（弹跳、脉冲、滑入） |
| 多会话 | 侧栏切换 | 命令面板切换 |
| 帮助系统 | 快捷键栏 | 帮助面板 + 命令面板 |
| 可扩展性 | 受终端限制 | 全浏览器能力（图片、视频、富文本） |