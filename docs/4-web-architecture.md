# Devo Web 前端工程架构文档

**版本**：1.3.0（2026-07-20 更新：会话状态拆分 thinking/tool_executing，信任级别改为 low/normal/elevated，审批策略补充 auto_approve，新增 MobileLayout 和 backgroundStore，SSE 事件补充 tool_chunk/background_output/loop.completed_with_reason，项目结构补全遗漏组件）

---

## 一、技术栈

| 技术 | 版本 | 说明 |
| :--- | :--- | :--- |
| Vue 3 | 3.4+ | Composition API + `<script setup>` |
| TypeScript | 5.4+ | 全量类型覆盖 |
| Vite | 5.3+ | 构建工具 |
| Pinia | 2.1+ | 状态管理 |
| Vue Router | 4.3+ | 路由（仅 `/chat` 路由） |
| marked | 最新 | Markdown 渲染 |
| highlight.js | 最新 | 代码高亮 |

**无 UI 框架**：全部手写 CSS，零外部 UI 库依赖。

---

## 二、项目结构

```
web/
├── public/                          ← 静态资源
├── src/
│   ├── main.ts                      ← 入口：创建 app + 注册 router/pinia
│   ├── App.vue                      ← 根组件：模式分流 + SSE 全局事件处理 + 初始化
│   │
│   ├── layouts/                     ← 布局层
│   │   ├── BrowserLayout.vue        ← 浏览器三栏布局
│   │   ├── VscodeLayout.vue         ← VSCode 极简布局
│   │   └── MobileLayout.vue         ← 移动端布局
│   │
│   ├── components/                  ← 共享组件
│   │   ├── chat/                    ← 聊天相关组件
│   │   │   ├── ChatPanel.vue        ← 聊天面板（消息列表 + 输入区）
│   │   │   ├── FloatingNavPanel.vue ← 浮动导航面板
│   │   │   ├── InputArea.vue        ← 输入区（/ 命令 + 发送 + 停止）
│   │   │   ├── MessageBubble.vue    ← 消息气泡（Markdown 渲染）
│   │   │   ├── MessageList.vue      ← 消息列表（自动滚动）
│   │   │   ├── ThinkingIndicator.vue← 思考指示器
│   │   │   ├── ToolCallCard.vue     ← 工具调用卡片
│   │   │   ├── ToolCallGroup.vue    ← 工具调用分组
│   │   │   └── VirtualMessageItem.vue← 虚拟滚动消息项
│   │   ├── command/
│   │   │   └── CommandPalette.vue   ← 命令面板
│   │   ├── common/
│   │   │   └── AppIcon.vue          ← 应用图标
│   │   ├── editor/
│   │   │   └── MonacoEditor.vue     ← Monaco 代码编辑器
│   │   ├── layout/
│   │   │   ├── AppSidebar.vue       ← 左侧栏（工作区 + 会话列表）
│   │   │   ├── AppHeader.vue        ← 顶部栏（已从 BrowserLayout 移除，保留待用）
│   │   │   ├── GlobalModals.vue     ← 全局弹窗容器（Teleport 到 body）
│   │   │   ├── RightPanel.vue       ← 右侧面板（Tab 切换框架）
│   │   │   ├── StatusBar.vue        ← 顶部状态栏
│   │   │   ├── ToastContainer.vue   ← 浮动提示容器
│   │   │   └── ToastItem.vue        ← 单个提示
│   │   ├── mobile/                  ← 移动端组件
│   │   │   ├── MobileCommandSheet.vue   ← 移动端命令面板
│   │   │   ├── MobileInputBar.vue       ← 移动端输入栏
│   │   │   ├── MobilePanelDrawer.vue    ← 移动端面板抽屉
│   │   │   ├── MobileSessionPicker.vue  ← 移动端会话选择器
│   │   │   └── MobileWorkspacePicker.vue← 移动端工作区选择器
│   │   └── modal/
│   │       ├── ApprovalModal.vue    ← 审批弹窗
│   │       ├── ConfigWarningDialog.vue  ← 配置警告弹窗
│   │       ├── ConfirmDeleteDialog.vue  ← 删除确认弹窗
│   │       ├── HelpPanel.vue        ← 帮助面板
│   │       ├── RollbackPicker.vue   ← 回滚选择器
│   │       └── SessionPicker.vue    ← 会话选择器
│   │
│   ├── panels/                      ← 右侧面板组件
│   │   ├── background/
│   │   │   └── BackgroundPanel.vue  ← 后台进程面板
│   │   ├── files/
│   │   │   └── FilesPanel.vue       ← 文件树 + 预览
│   │   ├── mcp/
│   │   │   └── McpPanel.vue         ← MCP 管理
│   │   ├── skills/
│   │   │   └── SkillsPanel.vue      ← 技能管理
│   │   ├── memory/
│   │   │   └── MemoryPanel.vue      ← 记忆管理
│   │   ├── dashboard/
│   │   │   └── DashboardPanel.vue   ← 仪表盘
│   │   ├── settings/
│   │   │   └── SettingsPanel.vue    ← 设置
│   │   └── terminal/
│   │       └── TerminalPanel.vue    ← 终端
│   │
│   ├── views/                       ← 路由视图
│   │   ├── ChatView.vue             ← 聊天视图（当前唯一注册路由）
│   │   ├── ApprovalPolicyView.vue   ← 审批策略视图
│   │   ├── DashboardView.vue        ← 仪表盘视图
│   │   ├── McpSettingsView.vue      ← MCP 设置视图
│   │   ├── MemoryView.vue           ← 记忆视图
│   │   ├── ProjectSettingsView.vue  ← 项目设置视图
│   │   ├── SessionArchiveView.vue   ← 会话存档视图
│   │   ├── SessionListView.vue      ← 会话列表视图
│   │   ├── SkillDetailView.vue      ← 技能详情视图
│   │   └── SkillsListView.vue       ← 技能列表视图
│   │
│   ├── composables/                 ← 可复用逻辑
│   │   ├── useApi.ts                ← HTTP 请求封装
│   │   ├── useAudio.ts              ← 音频提示
│   │   ├── useSSE.ts                ← SSE 连接管理
│   │   ├── useCommand.ts            ← 命令处理
│   │   ├── useFps.ts                ← FPS 监控
│   │   ├── useKeyboard.ts           ← 键盘快捷键
│   │   ├── useAutoScroll.ts         ← 自动滚动
│   │   ├── useSession.ts            ← 会话逻辑
│   │   ├── useThemeTransition.ts    ← 主题切换动画
│   │   ├── usePlatform.ts           ← 平台模式检测
│   │   └── useVirtualScroll.ts      ← 虚拟滚动
│   │
│   ├── stores/                      ← Pinia 状态管理
│   │   ├── ui.ts                    ← UI 全局状态
│   │   ├── session.ts               ← 会话状态
│   │   ├── chat.ts                  ← 聊天状态
│   │   ├── approval.ts              ← 审批状态
│   │   ├── background.ts            ← 后台进程状态
│   │   ├── command.ts               ← 命令状态
│   │   ├── skills.ts                ← 技能状态
│   │   ├── memory.ts                ← 记忆状态
│   │   └── mcp.ts                   ← MCP 状态
│   │
│   ├── types/                       ← TypeScript 类型定义
│   │   ├── session.ts               ← 会话类型
│   │   ├── message.ts               ← 消息类型
│   │   ├── sse.ts                   ← SSE 事件类型
│   │   ├── approval.ts              ← 审批类型
│   │   ├── tool.ts                  ← 工具类型
│   │   ├── api.ts                   ← API 类型
│   │   ├── skills.ts                ← 技能类型
│   │   ├── memory.ts                ← 记忆类型
│   │   └── workspace.ts             ← 工作区类型
│   │
│   ├── router/                      ← 路由
│   │   └── index.ts                 ← 仅 `/chat` 路由
│   │
│   ├── utils/                       ← 工具函数
│   │   ├── constants.ts             ← 常量（API_BASE 等）
│   │   ├── formatters.ts            ← 格式化工具
│   │   ├── languageMap.ts           ← 语言映射
│   │   └── markdown.ts              ← Markdown 渲染
│   │
│   └── styles/                      ← 样式
│       ├── variables.css            ← CSS 变量
│       ├── reset.css                ← CSS Reset
│       ├── base.css                 ← 基础样式
│       ├── typography.css           ← 排版
│       └── animations.css           ← 动画
│
├── index.html                       ← HTML 入口
├── package.json
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts                   ← Vite 配置
└── env.d.ts                         ← 环境类型声明
```

---

## 三、架构全景图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          App.vue                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  detectMode() → BrowserLayout / VscodeLayout / MobileLayout   │  │
│  │  onMounted: fetchWorkspace() → fetchWorkspaceList() →         │  │
│  │             setActiveWorkspace() → fetchSessions() →          │  │
│  │             connectSSE()                                       │  │
│  │  ┌─────────────────────────────────────────────────────────┐  │  │
│  │  │  SSE 事件分发（14+ 事件类型）                             │  │  │
│  │  │  事件 → chatStore / sessionStore / approvalStore / Toast │  │  │
│  │  └─────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
            ┌───────┴───────┐           ┌───────┴───────┐           ┌───────┴───────┐
            │ BrowserLayout │           │ VscodeLayout  │           │ MobileLayout  │
            │ (三栏+拖拽)    │           │ (极简聊天)    │           │ (移动端适配)  │
            └───────┬───────┘           └───────┬───────┘           └───────┬───────┘
                    │                           │
    ┌───────┬───────┼───────┬───────┐   ┌───────┼───────┐
    │       │       │       │       │   │       │       │
   Sidebar Chat  RightPanel  │      StatusBar Chat  Modals
   (左)   (中)   (右)        │
                        GlobalModals
```

---

## 四、组件树

### 4.1 浏览器模式

```
BrowserLayout.vue
├── AppSidebar.vue
│   ├── 工作区列表（名称 + 路径小字 + 删除）
│   │   ├── 点击 → POST /api/v1/current-workspace → 刷新会话
│   │   └── 删除 → 确认弹窗 → 输入路径匹配 → DELETE /api/v1/workspace
│   ├── 会话列表（按时间倒序，高亮当前）
│   │   ├── 点击 → 切换会话 + 加载消息 + 连接 SSE
│   │   ├── 删除 → 确认弹窗 → DELETE /api/v1/sessions/{id}
│   │   └── 重命名 → inline 编辑
│   └── 新建会话 → 弹窗填写名称 → POST /api/v1/sessions
│
├── <div class="chat-area">
│   ├── StatusBar.vue
│   │   ├── 会话名称（可点击重命名）
│   │   ├── 会话状态指示器（圆点 + 文字）
│   │   ├── 连接状态
│   │   └── 主题切换按钮（涟漪动画）
│   │
│   └── <router-view> → ChatView.vue
│       └── ChatPanel.vue
│           ├── MessageList.vue
│           │   ├── MessageBubble.vue（Markdown 渲染）
│           │   ├── ToolCallCard.vue / ToolCallGroup.vue
│           │   └── ThinkingIndicator.vue
│           └── InputArea.vue
│               └── CommandPalette.vue
│
├── RightPanel.vue（Tab 切换框架）
│   ├── FilesPanel.vue       ← 📁 文件树 + 预览（5MB 限制/类型过滤）
│   ├── SkillsPanel.vue      ← ⚡ 技能管理（scope 筛选）
│   ├── McpPanel.vue         ← 🔌 MCP 管理
│   ├── MemoryPanel.vue      ← 🧠 记忆管理（scope 筛选）
│   ├── BackgroundPanel.vue  ← 🔄 后台进程
│   ├── DashboardPanel.vue   ← 📊 仪表盘
│   ├── SettingsPanel.vue    ← ⚙ 设置
│   └── TerminalPanel.vue    ← 🖥 终端
│
└── GlobalModals.vue（Teleport to body）
    ├── ApprovalModal.vue
    ├── ConfigWarningDialog.vue
    ├── ConfirmDeleteDialog.vue
    ├── SessionPicker.vue
    ├── RollbackPicker.vue
    └── HelpPanel.vue
```

### 4.2 VSCode 模式

```
VscodeLayout.vue
├── StatusBar.vue（同上）
├── ChatView.vue → ChatPanel.vue（同上）
└── GlobalModals.vue（同上）
```

### 4.3 移动端模式

```
MobileLayout.vue
├── StatusBar.vue（同上）
├── ChatView.vue → ChatPanel.vue（同上）
│   └── InputArea.vue → MobileInputBar.vue
├── MobileCommandSheet.vue
├── MobilePanelDrawer.vue
├── MobileSessionPicker.vue
├── MobileWorkspacePicker.vue
└── GlobalModals.vue（同上）
```

---

## 五、Store 状态管理架构

### 5.1 Store 依赖关系

```
                   ┌──────────────┐
                   │   uiStore    │ ← 全局单例：主题、工作区、面板、Toast
                   └──────┬───────┘
                          │ 被多数 Store 引用
         ┌────────────────┼──────────────────┐
         │                │                  │
  ┌──────┴──────┐  ┌──────┴──────┐  ┌───────┴──────┐
  │ sessionStore│  │  chatStore  │  │ approvalStore│
  │ (会话状态)  │  │ (聊天状态)  │  │ (审批状态)   │
  └──────┬──────┘  └─────────────┘  └──────────────┘
         │
         │ 创建会话时引用 uiStore.activeWorkspace
         │
    ┌────┴─────┐
    │ ws 目录   │ ← workingDirectory（来自 current-workspace API）
    └──────────┘

  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
  │  skillsStore │  │  memoryStore │  │   mcpStore   │
  │  (技能)      │  │  (记忆)      │  │  (MCP 工具)  │
  └──────────────┘  └──────────────┘  └──────────────┘
         │                  │
         └────────┬─────────┘
                  │
           scope 分层体系
           ├── global（全局）
           └── workspace:<id>（项目级）

  ┌──────────────┐
  │backgroundStore│ ← 后台进程输出管理（SSE 驱动）
  └──────────────┘

### 5.2 uiStore（核心）

```typescript
// 文件：stores/ui.ts
export const useUiStore = defineStore('ui', () => {
  // ─── 主题 ───
  const theme = ref<ThemeType>(loadTheme())           // 持久化 localStorage
  // ─── 布局 ───
  const layoutMode = ref<'browser' | 'vscode' | 'mobile'>('browser')  // 平台模式
  const sidebarCollapsed = ref(false)                  // 侧栏折叠（持久化）
  // ─── 工作区 ───
  const activeWorkspace = ref<string | null>(loadActiveWorkspace())  // 持久化
  const workspaceList = ref<WorkspaceEntry[]>([])
  // ─── 右侧面板 ───
  const activeRightTab = ref<RightTabType>('files')   // 默认展示文件
  //     RightTabType = 'files' | 'skills' | 'mcp' | 'memory' | 'background' | 'dashboard' | 'settings'
  const rightPanelVisible = ref(false)                 // 持久化
  // ─── UI 状态 ───
  const toasts = ref<Toast[]>([])
  const activeModal = ref<ModalType>(null)
  //     ModalType = 'approval' | 'session-picker' | 'rollback-picker' | 'help' | 'config-warning'
  const connectionStatus = ref<'connected' | 'disconnected' | 'connecting'>('disconnected')
  const focusInputCounter = ref(0)                     // 控制输入框聚焦
  const pendingCommand = ref<string | null>(null)      // 待执行命令
  const activity = ref<string>('')                     // 当前活动状态文本
  let activityTimeout: ReturnType<typeof setTimeout> | null = null

  // 关键 Actions
  async function fetchWorkspaceList()           // GET /api/v1/workspace
  function addWorkspace(path)                   // 添加工作区到列表
  function removeWorkspace(id)                  // 删除工作区 + DELETE API
  function setActiveWorkspace(id)               // 设置当前工作区（持久化）
  function setActiveRightTab(tab)               // 切换右侧面板
  function toggleThemeWithTransition(x, y)      // 主题切换（涟漪动画）
  function showToast(type, message)             // 显示 Toast
  function setActivity(text)                    // 设置活动状态文本
  function clearActivity()                      // 清除活动状态
  // ...
})
```

### 5.3 sessionStore

```typescript
// 文件：stores/session.ts
export const useSessionStore = defineStore('session', () => {
  const currentSession = ref<Session | null>(null)
  const sessions = ref<Session[]>([])
  const isLoading = ref(false)
  const workingDirectory = ref('')               // 来自 current-workspace API

  // 关键 Actions
  async function createSession(request)     // POST /api/v1/sessions
  async function fetchWorkspace()           // GET /api/v1/current-workspace
  async function fetchSessions(project?)    // GET /api/v1/sessions?project=xxx
  function switchSession(session)           // 切换会话（本地）
  async function switchSessionById(id)      // 切换会话 + 加载消息
  async function renameSession(id, title)   // PUT /api/v1/sessions/{id}
  async function archiveSession(id)         // POST /api/v1/sessions/{id}/archive
  async function deleteSession(id)          // DELETE /api/v1/sessions/{id}
  function updateSessionState(id, state)    // 更新状态（SSE 驱动）
  function updateTokenUsage(id, usage)      // 更新 Token 用量（SSE 驱动）

  // 关键 Getters
  const isProcessing         // state === 'processing' || state === 'thinking' || state === 'tool_executing'
  const isThinking           // state === 'thinking'
  const isToolExecuting      // state === 'tool_executing'
  const isAwaitingApproval   // state === 'awaiting_approval'
  const isPaused             // state === 'paused'
  const isArchived           // state === 'archived'
  const isSessionActive      // 活跃状态判定（thinking / tool_executing / awaiting_approval）
  const sessionStatus        // 当前状态字符串
  const yoloEnabled          // trustLevel === 'elevated'（YOLO 模式）
  const canPause             // state === 'tool_executing'（仅在工具执行期间可暂停）
  const canResume            // state === 'paused'
  const canCancel            // 可取消状态判定
})
```

### 5.4 chatStore

```typescript
// 文件：stores/chat.ts
export const useChatStore = defineStore('chat', () => {
  const messages = ref<Message[]>([])
  const streamingContent = ref('')
  const isStreaming = ref(false)
  const isLoading = ref(false)

  // 关键 Actions
  function appendUserMessage(content)                // 添加用户消息
  function appendAssistantMessage(content, usage)    // 添加助手消息
  function startStreaming()                          // 开始流式
  function appendStreamChunk(chunk)                  // 追加流式片段
  function finishStreaming(usage)                    // 结束流式
  function appendToolCallMessage(toolCall)           // 添加工具调用
  function updateToolCallStatus(id, status, result)  // 更新工具调用状态
  function clearMessages()                           // 清空消息
  async function fetchMessages(sessionId)            // 获取消息列表
  // ...
})
```

### 5.5 其他 Store

| Store | 文件 | 核心职责 |
| :--- | :--- | :--- |
| `approvalStore` | `stores/approval.ts` | 审批请求管理：setApproval/clearApproval/approve/reject |
| `commandStore` | `stores/command.ts` | 命令面板：openPalette/closePalette/executeCommand |
| `skillsStore` | `stores/skills.ts` | 技能管理：fetch/list/toggle/install/delete + scope 分层 |
| `memoryStore` | `stores/memory.ts` | 记忆管理：fetch/add/update/delete + scope 分层 |
| `mcpStore` | `stores/mcp.ts` | MCP 工具：fetchTools/updateToolsFromEvent |
| `backgroundStore` | `stores/background.ts` | 后台进程：register/appendOutput/terminate（SSE 驱动） |

---

## 六、SSE 事件流架构

### 6.1 连接管理

```typescript
// composables/useSSE.ts
export function useSSE() {
  const { connect, disconnect, onEvent, offEvent } = createSSEConnection()

  // connect(sessionId): 建立 SSE 连接到 /api/v1/sessions/{id}/events
  // disconnect(): 断开连接
  // onEvent(name, handler): 注册事件监听
  // offEvent(name, handler): 移除事件监听
}
```

### 6.2 事件分发（App.vue）

```
SSE 事件流
│
├── thinking               → chatStore.startStreaming() + uiStore.setActivity(message)
├── streaming_token        → chatStore.appendStreamChunk(content) + uiStore.setActivity(content)
├── streaming_complete     → (内部标记流式完成，实际消息落盘在 message_complete)
├── message_complete       → chatStore.finishStreaming(usage) + sessionStore.updateSessionState('idle')
│                             + uiStore.clearActivity()
│                             （若没有流式内容，则通过 appendAssistantMessage 落盘完整消息）
├── token_usage            → sessionStore.updateTokenUsage(id, usage)
├── tool_call_request      → chatStore.appendToolCallMessage(toolCall) + uiStore.setActivity(toolName)
├── tool_result            → chatStore.updateToolCallStatus(id, success/failed, result)
│                             + 后台进程注册（exec_python background mode 时提取 PID）
│                             + uiStore.setActivity(toolName + ' 完成')
├── tool_progress          → chatStore.updateToolProgress(id, stage) + uiStore.setActivity(stage)
├── tool_chunk             → chatStore.appendToolStreamChunk(id, chunk)（工具流式输出）
├── approval_required      → approvalStore.setApproval(data) + uiStore.setActiveModal('approval')
├── approval_auto          → 系统消息（若 policy 为 yolo 则跳过，避免冗余通知）
├── approval_resolved      → approvalStore.clearApproval() + uiStore.clearModal()
├── session_state_change   → sessionStore.updateSessionState(id, state)
│                             （cancelled 原因特殊处理：state 设为 'cancelled'；
│                             tool_limit_reached 追加提示消息；error 追加错误消息）
├── context_compressed     → 系统消息 + Toast 通知
├── file_state_warning     → 系统消息
├── skill_solidified       → skillsStore.updateSkillFromEvent(data)
├── memory_updated         → memoryStore.updateMemoryFromEvent(data)
├── mcp_tool_discovered    → mcpStore.updateToolsFromEvent(tools)
├── loop.completed_with_reason → reason === 'completed' 时播放提示音
├── background_output      → backgroundStore.appendOutput(pid, stream, chunk)
└── error                  → Toast 通知
```

### 6.3 事件分发原则

- **所有 SSE 事件在 App.vue 中统一处理**，不分散到各个组件
- 事件触发后直接调用对应 Store 的 action
- Store 的响应式数据变化自动驱动 UI 更新
- 切换会话时断开旧连接、建立新连接

---

## 七、API 请求层

### 7.1 请求封装

```typescript
// composables/useApi.ts
// 基于 fetch 的轻量封装，统一 BASE_URL 和错误处理
```

### 7.2 API 端点全览

| 分类 | 方法 | 端点 | 调用方 |
| :--- | :--- | :--- | :--- |
| 版本 | GET | `/api/v1/version` | - |
| 配置 | GET | `/api/v1/config/status` | App.vue 初始化时检查 LLM 配置 |
| 工作区 | GET | `/api/v1/current-workspace` | sessionStore.fetchWorkspace |
| 工作区 | POST | `/api/v1/current-workspace` | AppSidebar（切换工作区） |
| 工作区 | GET | `/api/v1/workspace` | uiStore.fetchWorkspaceList |
| 工作区 | DELETE | `/api/v1/workspace?path=xxx` | uiStore.removeWorkspace |
| 会话 | POST | `/api/v1/sessions` | sessionStore.createSession |
| 会话 | GET | `/api/v1/sessions?project=xxx` | sessionStore.fetchSessions |
| 会话 | GET | `/api/v1/sessions/{id}` | sessionStore.fetchSession |
| 会话 | PUT | `/api/v1/sessions/{id}` | sessionStore.renameSession |
| 会话 | DELETE | `/api/v1/sessions/{id}` | sessionStore.deleteSession |
| 会话 | POST | `/api/v1/sessions/{id}/archive` | sessionStore.archiveSession |
| 消息 | POST | `/api/v1/sessions/{id}/messages` | chatStore / InputArea |
| 消息 | GET | `/api/v1/sessions/{id}/messages` | chatStore.fetchMessages |
| SSE | GET | `/api/v1/sessions/{id}/events` | useSSE.connect |
| 会话控制 | POST | `/api/v1/sessions/{id}/pause` | StatusBar |
| 会话控制 | POST | `/api/v1/sessions/{id}/resume` | StatusBar |
| 会话控制 | POST | `/api/v1/sessions/{id}/cancel` | StatusBar |
| 文件 | GET | `/api/v1/sessions/{id}/files?path=xxx` | FilesPanel |
| 回滚 | POST | `/api/v1/sessions/{id}/rollback` | RollbackPicker |
| 技能 | GET | `/api/v1/skills` | skillsStore.fetchSkills |
| 技能 | POST | `/api/v1/skills/install` | skillsStore.installSkill |
| 技能 | DELETE | `/api/v1/skills/{name}` | skillsStore.deleteSkill |
| 技能 | PUT | `/api/v1/sessions/{id}/skills/toggle` | skillsStore.toggleSkill |
| 记忆 | GET | `/api/v1/memory` | memoryStore.fetchMemories |
| 记忆 | POST | `/api/v1/memory` | memoryStore.addMemory |
| 记忆 | PUT | `/api/v1/memory/{key}` | memoryStore.updateMemory |
| 记忆 | DELETE | `/api/v1/memory/{key}` | memoryStore.deleteMemory |
| MCP | GET | `/api/v1/mcp/tools` | mcpStore.fetchTools |
| 统计 | GET | `/api/v1/stats/dashboard` | DashboardPanel |

---

## 八、数据流

### 8.1 初始化流程

```
用户打开 http://localhost:8080/
  │
  ▼
App.vue onMounted
  ├── detectMode()
  │   ├── URL 参数 ?mode=vscode → layoutMode = 'vscode'
  │   ├── 移动端检测 → layoutMode = 'mobile'
  │   └── 否则 → layoutMode = 'browser'
  │
  ├── registerThemeTransition() → 注册主题切换涟漪动画
  │
  ├── sessionStore.fetchWorkspace()
  │   └── GET /api/v1/current-workspace
  │       └── 返回 { working_directory: "/home/user/projects/pi-web" }
  │           └── sessionStore.workingDirectory = value
  │
  ├── uiStore.fetchWorkspaceList()
  │   └── GET /api/v1/workspace
  │       └── 返回 { workspaces: [{ id, name, path, exists }, ...] }
  │           └── uiStore.workspaceList = value
  │
  ├── uiStore.setActiveWorkspace(currentDir)
  │   └── 从 localStorage 读取 + 与 current-workspace 对比
  │       └── 优先使用 current-workspace 的值（覆盖 localStorage）
  │
  ├── sessionStore.fetchSessions(activeWorkspace)
  │   └── GET /api/v1/sessions?project=/home/user/projects/pi-web
  │       └── sessionStore.sessions = [...]
  │
  ├── 自动选中或创建会话
  │   └── URL 参数 ?session=xxx → 切换该会话
  │   └── 有会话 → 选第一个
  │   └── 无会话 → createSession() → 选新建的
  │
  ├── 配置检查
  │   └── GET /api/v1/config/status
  │       └── llm_configured === false → uiStore.setActiveModal('config-warning')
  │
  └── watch(activeWorkspace) → 切换时自动刷新
```

### 8.2 发送消息流程

```
用户在 InputArea 输入消息 → 回车
  │
  ▼
InputArea 发出事件
  │
  ▼
ChatPanel 处理：
  ├── chatStore.appendUserMessage(content)
  │   └── UI 立即显示用户消息
  │
  └── POST /api/v1/sessions/{id}/messages
      └── body: { content }
      └── 后端开始处理，SSE 开始推送事件
```

### 8.3 工作区切换流程

```
用户点击 AppSidebar 中的工作区
  │
  ▼
AppSidebar 调用：
  ├── uiStore.setActiveWorkspace(id)
  │   └── localStorage.setItem('devo-active-workspace', id)
  │
  └── POST /api/v1/current-workspace
      └── body: { path: id }
      └── 后端：os.Chdir(path) → 切换工作目录
      └── 返回：{ working_directory: path }
          │
          ▼
      App.vue watch(activeWorkspace) 触发：
          ├── sessionStore.fetchSessions(newWorkspace)
          └── 自动选中第一个会话
```

### 8.4 文件预览流程

```
用户点击 FilesPanel 中的文件
  │
  ▼
FilesPanel 检查：
  ├── 文件 > 5MB → 显示错误提示，不发起请求
  ├── 不支持扩展名 → 显示错误提示，不发起请求
  └── 通过检查 →
      │
      GET /api/v1/sessions/{id}/files?path=xxx
      │
      └── 后端返回：
          ├── 目录 → { items: [...] }
          ├── 图片（<5MB）→ { content: "data:image/png;base64,..." }
          └── 文本 → { content: "文本内容..." }
```

---

## 九、布局与样式架构

### 9.1 CSS 变量体系

```css
/* styles/variables.css */
:root {
  /* 主色调 */
  --color-primary: #6366f1;
  --color-primary-hover: #4f46e5;

  /* 背景 */
  --color-bg-primary: #ffffff;
  --color-bg-secondary: #f8fafc;
  --color-bg-tertiary: #f1f5f9;

  /* 文字 */
  --color-text-primary: #0f172a;
  --color-text-secondary: #475569;
  --color-text-tertiary: #94a3b8;

  /* 边框 */
  --color-border: #e2e8f0;

  /* 间距 */
  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 12px;
  --spacing-lg: 16px;
  --spacing-xl: 24px;

  /* 圆角 */
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;

  /* 字体 */
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
}

[data-theme='dark'] {
  --color-bg-primary: #0f172a;
  --color-bg-secondary: #1e293b;
  --color-bg-tertiary: #334155;
  --color-text-primary: #f1f5f9;
  --color-text-secondary: #94a3b8;
  --color-text-tertiary: #64748b;
  --color-border: #334155;
}
```

### 9.2 主题切换动画

```
主题切换按钮点击
  │
  ▼
uiStore.toggleThemeWithTransition(clientX, clientY)
  │
  ▼
useThemeTransition 注册的 handler:
  ├── 在按钮位置创建圆形遮罩元素
  ├── CSS clip-path: circle(0% at x y) → circle(150% at x y)
  ├── 动画结束后切换 theme 变量
  └── 移除遮罩元素
```

### 9.3 BrowserLayout 三栏布局

```
┌──────────────────────────────────────────────────────┐
│  <div class="browser-layout">                        │
│    display: flex; height: 100vh;                    │
│                                                      │
│    ├── AppSidebar     (width: 260px, 可拖拽)         │
│    ├── .main-content  (flex: 1)                     │
│    │   ├── StatusBar  (height: 40px)                │
│    │   └── <router-view> (flex: 1)                  │
│    └── RightPanel     (width: 360px, 可拖拽)         │
└──────────────────────────────────────────────────────┘
```

关键 CSS 约束：
- 左侧栏最小宽度：200px，最大宽度：400px
- 右侧面板最小宽度：280px，最大宽度：500px
- 中间区域使用 `flex: 1` 自适应
- 拖拽分隔线使用 `resize` 或自定义拖拽逻辑

---

## 十、关键设计决策

| 决策 | 方案 | 技术细节 |
| :--- | :--- | :--- |
| 状态管理 | Pinia + Composition API | 9 个独立 Store，无循环依赖 |
| SSE 事件分发 | App.vue 统一处理 | 事件 → Store action，不分散到组件 |
| 模式分流 | usePlatform composable | 检测 VSCode API + 移动端 + 浏览器默认 |
| 布局切换 | 三套 Layout 组件 | 组件/Store/API 完全复用 |
| 路由 | 仅 `/chat` 路由 | 功能切换走右侧面板 Tab |
| 工作区持久化 | localStorage + API | 前端存储 activeWorkspace，API 验证 |
| 工作区同步 | POST /api/v1/current-workspace | 确保前后端目录一致 |
| 文件预览 | 客户端白名单过滤 | 5MB 上限 + 150+ 扩展名 + 图片 base64 |
| 主题切换 | CSS 变量 + 涟漪动画 | `clip-path` 圆形扩散，0.5s 过渡 |
| 组件加载 | defineAsyncComponent | 右侧面板组件懒加载 |
| 样式方案 | 手写 CSS + CSS 变量 | 零 UI 框架依赖 |
| 构建工具 | Vite 5 | 快速 HMR，TypeScript 原生支持 |

---

## 十一、构建与部署

### 11.1 开发模式

```bash
cd web
pnpm dev
# 启动在 http://localhost:5173
# 代理 /api/v1 到后端 http://localhost:8080
```

### 11.2 生产构建

```bash
cd web
pnpm build
# 输出到 web/dist/
# 静态文件由后端 http.FileServer 托管
```

### 11.3 Vite 配置要点

```typescript
// vite.config.ts
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': '/src'
    }
  },
  server: {
    proxy: {
      '/api/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
```

---

## 十二、类型体系

### 12.1 Session 类型

```typescript
// types/session.ts
export type SessionState = 'idle' | 'thinking' | 'tool_executing'
  | 'processing'  // deprecated，保留兼容
  | 'awaiting_approval' | 'paused' | 'completed' | 'archived'

export type ApprovalPolicyLevel = 'always_ask' | 'session_trust' | 'full_trust' | 'auto_approve'
export type TrustLevel = 'low' | 'normal' | 'elevated'

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
  maxContextTokens?: number
  currentContextTokens?: number
  toolCallLimit?: number
  keepRecent?: number
}

export interface ApprovalPolicy {
  [toolName: string]: ApprovalPolicyLevel
}

export interface TokenUsage {
  input: number
  output: number
}
```

### 12.2 Skills 类型（scope 分层）

```typescript
// types/skills.ts
export type Scope = 'global' | `workspace:${string}`

export interface Skill {
  name: string
  description: string
  scope: Scope
  status: 'active' | 'inactive'
  source: 'builtin' | 'user' | 'project'
}
```

### 12.3 Memory 类型（scope 分层）

```typescript
// types/memory.ts
export type Scope = 'global' | `workspace:${string}`

export interface Memory {
  key: string
  value: string
  scope: Scope
  createdAt: string
  updatedAt: string
}
```

### 12.4 Workspace 类型

```typescript
// types/workspace.ts
export interface WorkspaceEntry {
  id: string        // 路径（用作唯一标识）
  name: string      // 显示名称（路径最后一段）
  path: string      // 完整路径
  exists: boolean   // 路径是否在文件系统中存在
}
```

---

## 十三、安全性

| 措施 | 实现 |
| :--- | :--- |
| XSS 防护 | Vue 默认转义 + Markdown 渲染使用 marked 安全配置 |
| 文件预览限制 | 客户端 5MB 上限 + 扩展名白名单，不预览不明文件 |
| API 认证 | 无（本地单用户工具，不对外暴露） |
| 路径遍历 | 后端处理，前端传递路径时做 encodeURIComponent |
| 敏感信息 | 无硬编码密钥，无日志泄露 |

---

## 十四、已知限制与待优化

| 限制 | 影响 | 计划 |
| :--- | :--- | :--- |
| Vite 构建大 chunk 警告 | 首屏加载时间 | 后续代码分割优化 |
| 文件预览仅支持文本和图片 | 不支持 PDF/Office | 后续考虑 |
| 终端面板占位 | 命令执行功能待完善 | 待后端接口就绪 |
| 仪表盘数据简单 | 缺少图表 | 后续集成 ECharts |
| 无路由懒加载优化 | 影响不大（仅一个路由） | 低优先级 |
| 无 PWA 支持 | 离线不可用 | 后续考虑 |
| 无国际化 | 仅中文 | 后续考虑 |