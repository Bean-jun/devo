# Devo 前端 Web 应用设计方案

**版本**：1.3.0（2026-06-26 更新：基于实际实现状态全面更新——三栏面板布局已完工，6 个面板全部实现，工作区管理（列表/切换/删除/确认）、会话管理（创建/重命名/删除/归档）、文件预览（大小限制+类型过滤+图片预览）、主题切换动画全部落地）

**定位**：同一份 Vue 3 应用，通过 `usePlatform` 模式分流同时支持浏览器（完整控制中心，三栏面板布局）和 VSCode Webview（极简聊天窗口）。后端 API 完全复用，不做任何区分。

---

## 一、系统全景回顾

### 后端已实现

| 模块 | 能力 | 前端接入状态 |
| :--- | :--- | :--- |
| 会话管理 | 创建/切换/暂停/恢复/取消/归档/重命名/删除 | ✅ 已接入 |
| 对话循环 | 发消息/LLM 回复/工具调用 | ✅ 已接入 |
| SSE 事件流 | 20+ 事件类型实时推送 | ✅ 已接入 |
| 审批门控 | 6 种操作类型 + 超时 + 自动批准 | ✅ 已接入 |
| 工具调用 | 读文件/写文件/编辑/搜索/执行命令 | ✅ 已接入 |
| 上下文压缩 | 自动压缩 + 事件通知 | ✅ 已接入 |
| 消息回滚 | 回滚 + 吸附逻辑 + 文件警告 | ✅ 已接入 |
| 系统提示词 | agents.md + 目录摘要 + 占位点 | ✅ 已接入 |
| 工作区管理 | 当前目录查询/切换、工作区列表去重、按工作区删除会话 | ✅ 已接入 |
| 文件浏览 | 目录树、文件内容读取、图片 base64 返回 | ✅ 已接入 |
| 长期记忆 | 用户记忆/项目记忆 CRUD + 自动更新 | 🔶 待接入 |
| Skills 管理器 | 三种来源加载 + 渐进式披露 + 动态启停 | 🔶 待接入 |
| 经验固化器 | 完成会话 → 分析 → 生成 SKILL.md | 🔶 待接入 |
| Token 计量 | 统计/查询/聚合 | 🔶 待接入 |
| MCP 客户端 | 外部工具发现和调用、自动重连 | 🔶 待接入 |
| 审批策略 | 按操作类型配置 | 🔶 待接入 |

### 后端未来

| 模块 | 能力 | 前端关系 |
| :--- | :--- | :--- |
| 团队统计 | Token 用量上报/聚合/预算预警 | 预留 |
| 团队协作 | 审批中转/会话旁观 | 预留 |

---

## 二、核心设计原则

### 2.1 核心设计理念：聊天是主场，面板是辅助

浏览器模式采用**三栏面板布局**，聊天始终在中间，不可离开。右侧面板通过 Tab 切换承载 files/skills/memory/dashboard/settings/terminal 等功能，用户切换面板时不离开聊天上下文。

```
浏览器模式的三栏布局：
┌──────────┬──────────────────────┬──────────────────┐
│ 左侧栏   │ 中间（聊天主场）      │ 右侧面板（Tab）   │
│          │                      │                  │
│ 📁 项目A  │ StatusBar（会话名/状态）│ Files│Skills     │
│ 📁 项目B  │ ──────────────────── │ Memory│Dashboard │
│ ──────── │ 消息列表             │ Settings│Terminal │
│ 💬 会话1  │ Tool Call            │                  │
│ 💬 会话2  │ Assistant 回答       │ ┌──────────────┐ │
│ + 新建   │ ──────────────────── │ │ 面板内容     │ │
│          │ InputArea + 命令      │ │              │ │
│          │                      │ └──────────────┘ │
└──────────┴──────────────────────┴──────────────────┘
```

### 2.2 模式分流

```
浏览器访问  →  http://localhost:8080/             → 完整功能（三栏面板）
VSCode 访问  →  http://localhost:8080/?mode=vscode  → 仅聊天
```

同一个 Vue 应用，入口根据 `usePlatform` 检测结果选择不同的布局壳。**组件、Store、API 层完全复用**。

### 2.3 职责分离

| | Web（浏览器） | VSCode 扩展 |
| :--- | :--- | :--- |
| 定位 | 控制中心 / 配置台（三栏面板） | 终端 / 对话窗口 |
| 工作区 | 多工作区切换、管理、删除 | 只看当前工作区 |
| 会话 | 创建/删除/重命名/归档 | 创建/切换 |
| 技能 | 浏览、安装、启停、删除（面板） | 无（仅被动消费 Web 配置） |
| 记忆 | 浏览、增删改查（面板） | 无（仅被动消费 Web 配置） |
| 文件 | 文件树浏览、内容预览（面板） | 无 |
| 仪表盘 | Token 用量、趋势图表（面板） | 仅状态栏显示 |
| 终端 | 内嵌终端执行命令（面板） | 无 |
| 审批 | 弹窗处理 | 弹窗处理 |
| 聊天 | 有（中间主场） | 有（唯一功能） |

### 2.4 scope 分层体系

Skills 和 Memory 具有全局（`global`）和项目（`workspace:<id>`）两个层级，项目级叠加全局级。

```typescript
type Scope = 'global' | `workspace:${string}`

interface Skill {
  name: string
  scope: Scope
  // ...
}

interface Memory {
  key: string
  scope: Scope
  // ...
}
```

### 2.5 渐进式实施

不是一个巨大的 PR，而是分阶段叠加：

```
Phase 1: 模式分流 + 三栏面板布局重构 ✅ 已完成
Phase 2: 右侧面板实现（全部 6 个面板）✅ 已完成
Phase 3: 交互优化（拖拽、动画、快捷键）✅ 已完成
```

---

## 三、目录结构（当前实现状态）

```
web/src/
├── main.ts                          ← 入口
├── App.vue                          ← 模式分流 + SSE 事件处理 + 初始化
│
├── layouts/                         ← 布局层
│   ├── BrowserLayout.vue            ← 浏览器三栏布局（可拖拽调整宽度）
│   └── VscodeLayout.vue             ← VSCode 极简布局（仅聊天）
│
├── components/                      ← 共享组件
│   ├── chat/
│   │   ├── ChatPanel.vue            ← 聊天面板（消息列表 + 输入区）
│   │   ├── InputArea.vue            ← 输入区（/ 命令 + 发送 + 停止）
│   │   ├── MessageBubble.vue        ← 消息气泡（Markdown 渲染）
│   │   ├── MessageList.vue          ← 消息列表（自动滚动）
│   │   ├── ThinkingIndicator.vue    ← 思考指示器（动画）
│   │   ├── ToolCallCard.vue         ← 工具调用卡片
│   │   └── ToolCallGroup.vue        ← 工具调用分组
│   ├── command/
│   │   └── CommandPalette.vue       ← 命令面板（/ 触发）
│   ├── layout/
│   │   ├── AppSidebar.vue           ← 左侧栏（工作区列表 + 会话列表）
│   │   ├── AppHeader.vue            ← 顶部栏（已从 BrowserLayout 移除，保留待用）
│   │   ├── GlobalModals.vue         ← 全局弹窗容器（Teleport）
│   │   ├── RightPanel.vue           ← 右侧面板（Tab 切换框架）
│   │   ├── StatusBar.vue            ← 顶部状态栏（会话名/状态/主题切换）
│   │   ├── ToastContainer.vue       ← 浮动提示容器
│   │   └── ToastItem.vue            ← 单个提示
│   └── modal/
│       ├── ApprovalModal.vue        ← 审批弹窗
│       ├── HelpPanel.vue            ← 帮助面板
│       ├── RollbackPicker.vue       ← 回滚选择器
│       └── SessionPicker.vue        ← 会话选择器
│
├── panels/                          ← 右侧面板组件
│   ├── files/
│   │   └── FilesPanel.vue           ← 文件树 + 预览（大小限制/类型过滤/图片预览）
│   ├── skills/
│   │   └── SkillsPanel.vue          ← 技能管理面板（scope 筛选）
│   ├── memory/
│   │   └── MemoryPanel.vue          ← 记忆管理面板（scope 筛选）
│   ├── dashboard/
│   │   └── DashboardPanel.vue       ← 仪表盘面板
│   ├── settings/
│   │   └── SettingsPanel.vue        ← 设置面板
│   └── terminal/
│       └── TerminalPanel.vue        ← 终端面板
│
├── views/                           ← 路由视图
│   ├── ChatView.vue                 ← 聊天视图（包装 ChatPanel）
│   ├── ApprovalPolicyView.vue       ← 审批策略配置页
│   ├── DashboardView.vue            ← 仪表盘独立页
│   ├── McpSettingsView.vue          ← MCP 设置页
│   ├── MemoryView.vue               ← 记忆独立页
│   ├── ProjectSettingsView.vue      ← 项目设置页
│   ├── SessionArchiveView.vue       ← 会话存档页
│   ├── SessionListView.vue          ← 会话列表页
│   ├── SkillDetailView.vue          ← 技能详情页
│   └── SkillsListView.vue           ← 技能列表页
│
├── composables/                     ← 可复用逻辑
│   ├── useApi.ts                    ← API 请求封装
│   ├── useSSE.ts                    ← SSE 事件流消费
│   ├── useCommand.ts                ← 命令处理
│   ├── useKeyboard.ts               ← 键盘快捷键
│   ├── useAutoScroll.ts             ← 自动滚动
│   ├── useSession.ts                ← 会话逻辑
│   ├── useThemeTransition.ts        ← 主题切换动画
│   └── usePlatform.ts               ← 平台模式检测
│
├── stores/                          ← Pinia 状态管理
│   ├── ui.ts                        ← UI 状态（主题/工作区/面板/Toast/连接状态）
│   ├── session.ts                   ← 会话状态（CRUD/切换/Token 用量）
│   ├── chat.ts                      ← 聊天状态（消息/流式/工具调用）
│   ├── approval.ts                  ← 审批状态
│   ├── command.ts                   ← 命令状态
│   ├── skills.ts                    ← 技能状态（scope 分层）
│   ├── memory.ts                    ← 记忆状态（scope 分层）
│   └── mcp.ts                       ← MCP 状态
│
├── types/                           ← TypeScript 类型
│   ├── session.ts                   ← 会话类型（含 trustLevel/approvalPolicy）
│   ├── message.ts                   ← 消息类型
│   ├── sse.ts                       ← SSE 事件类型
│   ├── approval.ts                  ← 审批类型
│   ├── tool.ts                      ← 工具类型
│   ├── api.ts                       ← API 类型
│   ├── skills.ts                    ← 技能类型（含 scope 字段）
│   ├── memory.ts                    ← 记忆类型（含 scope 字段）
│   └── workspace.ts                 ← 工作区类型
│
├── router/                          ← 路由
│   └── index.ts                     ← 仅 /chat 路由
│
├── utils/                           ← 工具函数
│   ├── constants.ts                 ← 常量（API_BASE 等）
│   ├── formatters.ts                ← 格式化工具
│   └── markdown.ts                  ← Markdown 渲染
│
└── styles/                          ← 样式
    ├── variables.css                ← CSS 变量（主题色/间距/字体）
    ├── reset.css                    ← CSS Reset
    ├── base.css                     ← 基础样式
    ├── typography.css               ← 排版
    └── animations.css               ← 动画（主题过渡等）
```

---

## 四、模式分流设计

### 4.1 入口检测（usePlatform.ts）

```typescript
// 检测逻辑
const params = new URLSearchParams(window.location.search)
const mode = params.get('mode') === 'vscode' ? 'vscode' : 'browser'
// 也检测 window.acquireVsCodeApi 是否存在（VSCode Webview 环境）
```

### 4.2 布局切换

```
App.vue
  ├── isVscodeMode === true
  │   └── <VscodeLayout>
  │       ├── StatusBar          ← 顶部状态
  │       ├── ChatView           ← 全屏聊天
  │       └── GlobalModals       ← 弹窗层
  │
  └── isVscodeMode === false
      └── <BrowserLayout>
          ├── AppSidebar         ← 左侧栏（工作区 + 会话）
          ├── StatusBar          ← 顶部状态栏
          ├── <router-view />    ← 主内容区（/chat → ChatView）
          ├── RightPanel         ← 右侧面板（Tab 切换）
          └── GlobalModals       ← 弹窗层
```

### 4.3 初始化流程

```
页面加载
  ↓
detectMode() → 设置 isVscodeMode
  ↓
registerThemeTransition() → 注册主题切换涟漪动画
  ↓
sessionStore.fetchWorkspace() → GET /api/v1/current-workspace → 获取当前目录
  ↓
uiStore.fetchWorkspaceList() → GET /api/v1/workspace → 工作区列表
  ↓
uiStore.setActiveWorkspace(currentDir) → 选中当前工作区（覆盖 localStorage）
  ↓
sessionStore.fetchSessions(currentDir) → GET /api/v1/sessions?project=xxx → 当前工作区会话
  ↓
自动选中或创建会话
  ↓
右侧面板默认展示 Files tab
  ↓
watch(activeWorkspace) → 切换工作区时自动刷新会话列表
```

---

## 五、路由设计

浏览器模式不再通过路由跳转切换功能，而是用**右侧面板 Tab 切换**。聊天始终在中间，路由仅分 `/chat`。

```typescript
// router/index.ts
const routes = [
  {
    path: "/",
    redirect: "/chat",
  },
  {
    path: "/chat",
    name: "chat",
    component: () => import("@/views/ChatView.vue"),
    meta: { title: "对话" },
  },
]
```

`views/` 目录下的其他视图文件（`ApprovalPolicyView`, `DashboardView`, `MemoryView` 等）为预留独立页面，当前未注册到路由，功能通过右侧面板承载。

---

## 六、各面板详细设计（已实现）

### 6.1 左侧栏（AppSidebar）

```
┌──────────────┐
│ 📁 devo-web    │  ← 工作区列表（显示名称 + 路径小字）
│   /home/...    │
│ 📁 my-project│
│ ──────────── │
│ 💬 修复登录页  │  ← 当前工作区会话列表
│ 💬 重构数据库  │
│ 💬 添加 API  │
│ + 新建会话    │
└──────────────┘
```

**已实现功能**：
- 工作区列表：从 `uiStore.workspaceList` 读取，显示名称 + 灰色小字路径
- 点击工作区：切换工作区 → `POST /api/v1/current-workspace` 通知后端 → 刷新会话列表
- 工作区删除：二次确认弹窗，要求用户输入完整路径确认，否则删除按钮 disabled
- 会话列表：按最后活动时间倒序，当前会话高亮
- 会话 hover 显示 ✕ 删除按钮，点击弹出确认弹窗
- 新建会话：弹窗填写名称（可选），确认后创建并跳转聊天
- 右侧面板默认展示 Files tab

**工作区删除弹窗**：
```
┌──────────────────────────────────────┐
│  ⚠️ 删除工作区                        │
│                                      │
│  此操作将删除 pi-web 下的所有          │
│  会话和记录，不可恢复。                │
│                                      │
│  请输入以下路径以确认删除：             │
│  ┌──────────────────────────────────┐│
│  │ /home/user/projects/pi-web      ││
│  └──────────────────────────────────┘│
│  ┌──────────────────────────────────┐│
│  │ 输入路径以确认...                ││  ← 必须精确匹配
│  └──────────────────────────────────┘│
│                    [取消]  [删除]     │  ← 路径不匹配时 disabled
└──────────────────────────────────────┘
```

**会话删除弹窗**：
```
┌──────────────────────────┐
│  删除会话                 │
│                          │
│  确定要删除会话           │
│  "修复登录 Bug" 吗？     │
│  此操作不可恢复。          │
│                          │
│          [取消]  [删除]  │
└──────────────────────────┘
```

### 6.2 聊天视图（ChatView.vue）

包装现有 `ChatPanel` 组件，**不做任何业务逻辑改动**。VSCode 模式和浏览器模式共用同一个组件。

### 6.3 右侧面板（RightPanel.vue）

**Tab 标签**（全部 6 个，始终可见）：

```
📁 Files  ⚡ Skills  🧠 Memory  📊 Dashboard  ⚙ Settings  🖥 Terminal
```

所有 6 个 Tab 始终可见，不区分模式。面板组件通过 `defineAsyncComponent` 懒加载。

### 6.4 文件面板（FilesPanel.vue）

**功能**：
- 文件树展示：递归懒加载，点击展开目录，点击文件预览内容
- 文件类型图标：根据扩展名映射不同图标（🔷 .ts, 🐍 .py, 🔵 .go, 🎨 .css 等）
- 文件大小格式化：B / KB / MB
- 刷新按钮

**预览功能**：
- 预览区域可拖拽调整高度（80px ~ 600px）
- 关闭按钮收起预览

**预览限制**：

| 条件 | 行为 |
| :--- | :--- |
| 文件 > 5MB | 显示 `[文件过大 (xx MB)，无法预览]`，不发起请求 |
| 不支持的扩展名 | 显示 `[此文件类型不支持预览]`，不发起请求 |
| 无后缀 & ≤ 1MB | 默认以文本模式预览 |
| 图片（png/jpg/gif/svg/webp/bmp/ico） | 显示 `<img>` 预览，后端返回 base64 data URL |
| 代码/脚本（150+ 种扩展名） | 显示 `<pre><code>` 文本 |

**可预览扩展名**：ts, tsx, js, jsx, mjs, cjs, vue, svelte, astro, solid, py, go, rs, java, kt, kts, scala, cs, fs, fsx, vb, c, cpp, cxx, cc, h, hpp, hxx, hh, css, scss, less, sass, styl, pcss, html, htm, xml, xhtml, mjml, json, jsonc, yaml, yml, toml, ini, cfg, conf, config, md, mdx, txt, log, rst, adoc, asciidoc, org, pod, tex, bib, sh, bash, zsh, fish, ps1, bat, cmd, psd1, psm1, sql, graphql, gql, proto, prisma, env, envrc, gitignore, dockerfile, dockerignore, makefile, justfile, earthfile, containerfile, rb, erb, rake, gemspec, php, phtml, swift, lua, r, rmd, rnw, jl, dart, ex, exs, eex, heex, leex, elm, hs, lhs, ml, mli, clj, cljs, cljc, edn, zig, nim, nims, v, cr, pl, pm, t, raku, p6, nqp, coffee, litcoffee, pug, jade, ejs, haml, slim, tf, tfvars, hcl, nomad, gradle, groovy, res, resi, purs, dhall, nix, cabal, s, asm, nasm, wat, wast, f, f90, f95, f03, f08, for, cob, cbl, cpy, diff, patch, lock, csv, tsv, psv, puml, plantuml, wsd, nginx, htaccess, apache, cmake, meson, bazel, bzl, build, eslintrc, prettierrc, babelrc, stylelintrc, npmrc, yarnrc, properties, prop, xsl, xslt, xsd, dtd, wsdl, re, rei, sas, stata, do, ado, matlab, m, octave, scheme, scm, ss, rkt, lisp, lsp, fasl, sc, scd, pde, ino, odin, pony, move, smithy, cue, wgsl, slint, wit

### 6.5 技能面板（SkillsPanel.vue）

**功能**：
- scope 筛选：全部 / 全局 / 项目
- 展示技能卡片：名称、描述、scope 标签、状态（启用/禁用）
- 启用/禁用切换、删除按钮
- 安装新技能功能

**API 交互**：
- 加载：`GET /api/v1/skills`
- 启停：`PUT /api/v1/sessions/{id}/skills/toggle`
- 安装：`POST /api/v1/skills/install`
- 删除：`DELETE /api/v1/skills/{name}`

### 6.6 记忆面板（MemoryPanel.vue）

**功能**：
- scope 筛选：全部 / 个人 / 项目
- 展示记忆卡片：key、value、scope 标签
- 编辑、删除按钮
- 添加新记忆功能

**API 交互**：
- 加载：`GET /api/v1/memory`
- 新建：`POST /api/v1/memory`
- 更新：`PUT /api/v1/memory/{key}`
- 删除：`DELETE /api/v1/memory/{key}`

### 6.7 仪表盘面板（DashboardPanel.vue）

展示 Token 用量统计、趋势数据、会话分布等。当前为占位实现，数据通过 `GET /api/v1/stats/dashboard` 获取。

### 6.8 设置面板（SettingsPanel.vue）

**功能**：
- 项目设置：工作目录、审批策略、MCP 工具管理、agents.md 内容
- 全局设置：主题、快捷键、默认模型、语言偏好

### 6.9 终端面板（TerminalPanel.vue）

内嵌终端，工作目录为当前 workspace。通过 `POST /api/v1/sessions/{id}/terminal` 执行命令。

---

## 七、Store 设计（已实现）

### 7.1 ui Store

| 状态 | 类型 | 说明 |
| :--- | :--- | :--- |
| `theme` | `'light' | 'dark'` | 当前主题，持久化 localStorage |
| `isVscodeMode` | `boolean` | 是否为 VSCode 模式 |
| `activeWorkspace` | `string | null` | 当前选中工作区，持久化 localStorage |
| `activeRightTab` | `RightTabType` | 右侧面板激活 Tab，默认 `'files'` |
| `rightPanelVisible` | `boolean` | 右侧面板是否可见 |
| `workspaceList` | `WorkspaceEntry[]` | 工作区列表 |
| `toasts` | `Toast[]` | 浮动提示列表 |
| `activeModal` | `ModalType` | 当前激活弹窗 |
| `connectionStatus` | `'connected' | 'disconnected' | 'connecting'` | 连接状态 |
| `focusInputCounter` | `number` | 输入框聚焦计数器 |
| `pendingCommand` | `string | null` | 待执行命令 |

**关键 actions**：
- `fetchWorkspaceList()` — `GET /api/v1/workspace`
- `addWorkspace(path)` — 添加工作区到列表
- `removeWorkspace(id)` — 删除工作区（调 `DELETE /api/v1/workspace`）
- `setActiveWorkspace(id)` — 设置当前工作区（持久化）
- `setActiveRightTab(tab)` — 切换右侧面板 Tab
- `toggleThemeWithTransition(x, y)` — 主题切换（涟漪动画从按钮位置扩散）
- `showToast(type, message)` — 显示浮动提示

### 7.2 session Store

| 状态 | 类型 | 说明 |
| :--- | :--- | :--- |
| `currentSession` | `Session | null` | 当前会话 |
| `sessions` | `Session[]` | 会话列表 |
| `isLoading` | `boolean` | 加载状态 |
| `workingDirectory` | `string` | 当前工作目录 |

**关键 actions**：
- `createSession(request)` — `POST /api/v1/sessions`（支持 title 参数）
- `fetchWorkspace()` — `GET /api/v1/current-workspace`
- `fetchSessions(project?)` — `GET /api/v1/sessions?project=xxx`
- `switchSessionById(id)` — 切换会话
- `renameSession(id, title)` — `PUT /api/v1/sessions/{id}`
- `archiveSession(id)` — `POST /api/v1/sessions/{id}/archive`
- `deleteSession(id)` — `DELETE /api/v1/sessions/{id}`
- `updateSessionState(id, state)` — 更新会话状态
- `updateTokenUsage(id, usage)` — 更新 Token 用量

**关键 getters**：`isProcessing`, `isAwaitingApproval`, `isPaused`, `isArchived`, `sessionStatus`, `canPause`, `canResume`, `canCancel`

### 7.3 chat Store

管理消息列表、流式内容、工具调用状态。支持：
- `appendUserMessage(content)` — 添加用户消息
- `appendAssistantMessage(content, usage)` — 添加助手消息
- `startStreaming()` / `appendStreamChunk(chunk)` / `finishStreaming(usage)` — 流式处理
- `appendToolCallMessage(toolCall)` / `updateToolCallStatus(id, status, result)` — 工具调用
- `clearMessages()` / `fetchMessages(sessionId)` — 消息管理

### 7.4 skills Store

| 状态 | 类型 | 说明 |
| :--- | :--- | :--- |
| `skills` | `Skill[]` | 技能列表 |
| `isLoading` | `boolean` | 加载状态 |

**Getters**：`globalSkills`（scope=global）、`workspaceSkills(workspaceId)`（scope=workspace:id）

**Actions**：`fetchSkills()`, `toggleSkill()`, `installSkill()`, `deleteSkill()`

### 7.5 memory Store

| 状态 | 类型 | 说明 |
| :--- | :--- | :--- |
| `memories` | `Memory[]` | 记忆列表 |
| `isLoading` | `boolean` | 加载状态 |

**Getters**：`globalMemories`（scope=global）、`workspaceMemories(workspaceId)`（scope=workspace:id）

**Actions**：`fetchMemories()`, `addMemory()`, `updateMemory()`, `deleteMemory()`

### 7.6 mcp Store

| 状态 | 类型 | 说明 |
| :--- | :--- | :--- |
| `tools` | `McpTool[]` | MCP 工具列表 |
| `isLoading` | `boolean` | 加载状态 |

**Actions**：`fetchTools()` — `GET /api/v1/mcp/tools`

### 7.7 approval Store

管理审批请求状态：`setApproval()`, `clearApproval()`, `setAutoApproved()`, `approveRequest()`, `rejectRequest()`

### 7.8 command Store

管理命令面板状态：`openPalette()`, `closePalette()`, `executeCommand()`

---

## 八、StatusBar — 顶部状态栏

**显示内容**（从左到右）：
- 会话名称（可点击重命名，支持 inline 编辑 + Enter 确认）
- 会话状态指示器（圆点 + 文字）：
  - 🟢 Idle（空闲）
  - 🔵 Processing（处理中，带脉冲动画）
  - 🟡 AwaitingApproval（等待审批）
  - ⏸️ Paused（已暂停）
  - ✅ Completed（已完成）
  - 📦 Archived（已归档）
- 连接状态：🟢 已连接 / 🔴 未连接
- 主题切换按钮：🌙（暗色模式）/ ☀️（亮色模式），涟漪动画从按钮位置扩散

**位置**：在 BrowserLayout 中位于聊天区域上方（AppHeader 已移除，StatusBar 直接紧贴顶部）。

---

## 九、SSE 事件处理

当前 `App.vue` 已处理的事件：

| 事件 | 前端处理 |
| :--- | :--- |
| `thinking` | `chatStore.startStreaming()` |
| `streaming_token` | `chatStore.appendStreamChunk()` |
| `streaming_complete` | 等待 message_complete |
| `message_complete` | `chatStore.finishStreaming()` + 更新会话状态 |
| `token_usage` | `sessionStore.updateTokenUsage()` |
| `tool_call_request` | `chatStore.appendToolCallMessage()` |
| `tool_result` | `chatStore.updateToolCallStatus()` |
| `tool_progress` | `chatStore.updateToolCallStatus('executing')` |
| `approval_required` | `approvalStore.setApproval()` |
| `approval_auto` | Toast 通知 |
| `approval_resolved` | `approvalStore.clearApproval()` |
| `session_state_change` | `sessionStore.updateSessionState()` |
| `context_compressed` | Toast 通知 |
| `file_state_warning` | Toast 通知 |
| `error` | Toast 通知 |

**SSE 事件分发**：在 `App.vue` 中统一管理，事件触发后同时调用对应 Store 的 action。

---

## 十、API 端点汇总

| 方法 | 路径 | 前端用途 |
| :--- | :--- | :--- |
| GET | `/api/v1/version` | 版本号 |
| GET | `/api/v1/current-workspace` | 获取当前工作目录 |
| POST | `/api/v1/current-workspace` | 切换工作目录（切换工作区时调用） |
| GET | `/api/v1/workspace` | 获取工作区列表 |
| DELETE | `/api/v1/workspace?path=xxx` | 删除工作区下所有会话 |
| POST | `/api/v1/sessions` | 创建会话 |
| GET | `/api/v1/sessions` | 获取会话列表（支持 `?project=` 过滤） |
| GET | `/api/v1/sessions/{id}` | 获取单个会话 |
| PUT | `/api/v1/sessions/{id}` | 重命名会话 |
| DELETE | `/api/v1/sessions/{id}` | 删除会话 |
| POST | `/api/v1/sessions/{id}/archive` | 归档会话 |
| POST | `/api/v1/sessions/{id}/messages` | 发送消息 |
| GET | `/api/v1/sessions/{id}/messages` | 获取消息列表 |
| GET | `/api/v1/sessions/{id}/events` | SSE 事件流 |
| POST | `/api/v1/sessions/{id}/pause` | 暂停会话 |
| POST | `/api/v1/sessions/{id}/resume` | 恢复会话 |
| POST | `/api/v1/sessions/{id}/cancel` | 取消会话 |
| GET | `/api/v1/sessions/{id}/files?path=xxx` | 获取文件列表/内容 |
| POST | `/api/v1/sessions/{id}/rollback` | 回滚消息 |
| GET | `/api/v1/skills` | 获取技能列表 |
| POST | `/api/v1/skills/install` | 安装技能 |
| DELETE | `/api/v1/skills/{name}` | 删除技能 |
| PUT | `/api/v1/sessions/{id}/skills/toggle` | 启停技能 |
| GET | `/api/v1/memory` | 获取记忆列表 |
| POST | `/api/v1/memory` | 添加记忆 |
| PUT | `/api/v1/memory/{key}` | 更新记忆 |
| DELETE | `/api/v1/memory/{key}` | 删除记忆 |
| GET | `/api/v1/mcp/tools` | 获取 MCP 工具列表 |
| GET | `/api/v1/stats/dashboard` | 获取仪表盘数据 |

---

## 十一、关键设计决策总结

| 决策 | 方案 | 理由 |
| :--- | :--- | :--- |
| 模式分流 | URL 参数 `?mode=vscode` + `acquireVsCodeApi` | 零侵入，同一份代码，部署简单 |
| 布局 | 三栏面板模式 | 聊天是主场不可离开，面板是辅助 |
| 路由 | vue-router 仅 `/chat` 路由 | 功能切换走右侧面板 Tab，不跳转页面 |
| 工作区初始化 | `current-workspace` API 决定默认选中 | 确保前端与后端进程当前目录一致 |
| 工作区切换 | 前端 + 后端同步（`POST /api/v1/current-workspace`） | 后端 `os.Chdir()` 切换目录 |
| 工作区删除 | 输入路径精确匹配确认 | 防止误删，不可恢复 |
| 会话删除 | 简单确认弹窗 | 单会话删除风险可控 |
| scope 分层 | `global` + `workspace:<id>` | Skills 和 Memory 支持全局和项目两层 |
| 文件预览 | 5MB 上限 + 150+ 扩展名白名单 + 无后缀 ≤1MB 文本 | 安全可控，覆盖主流编程语言 |
| 图片预览 | 后端 base64 返回 `data:image/xxx;base64,...` | 避免额外请求，一次性加载 |
| 主题切换 | 涟漪动画从按钮位置扩散 | 视觉连贯，双向统一 |
| 审批弹窗 | 双模式都支持 | 审批是对话流程的一部分 |
| SSE 事件分发 | App.vue 统一处理 + 写入对应 Store | 无论哪个面板，数据都能实时刷新 |
| AppHeader | 已移除 | StatusBar 已显示会话名称，冗余 |
| StatusBar 位置 | 顶部（聊天区域上方） | 会话状态一目了然 |
| 样式 | 手写 CSS，零 UI 框架 | 轻量、可控、无依赖 |