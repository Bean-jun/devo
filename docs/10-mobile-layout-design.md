# Devo 移动端布局设计方案

**版本**：1.0.0（2026-07-11 初版：命令驱动 + 右侧滑入面板 + 底部 Sheet 选择器，移动端完整功能，不做阉割）

**定位**：在现有 Vue 3 应用中新增第三种布局模式 `MobileLayout`，通过 `usePlatform` 检测 `?mode=mobile` 或屏幕宽度自动切换。默认界面为极简聊天窗口，通过 `[/]` 按钮触发命令面板，所有功能（工作区管理、会话管理、6 个面板）均可通过命令按需触达。组件、Store、API 层完全复用，不做任何区分。

---

## 一、设计哲学

### 1.1 核心原则：命令驱动，按需触达

```
桌面端 VS Code 的精髓：
  按 Ctrl+Shift+P → 输入 → 一切皆可达

移动端 Devo 的对应：
  点 [/] 按钮 → 选择命令 → 一切皆可达
```

移动端不做功能阉割，而是做**空间重组**。桌面端用水平空间换取效率（三栏并排），移动端用命令面板换取简洁（单栏 + 按需调出）。默认界面只有一个 `[/]` 按钮，所有功能通过命令面板触达，用完即走。

### 1.2 三种模式对比

```
                      BrowserLayout    VscodeLayout    MobileLayout
─────────────────────┼───────────────┼───────────────┼───────────────
核心入口              │ 左侧栏+右侧Tab │ 无（仅聊天）   │ [/] 命令按钮
工作区管理            │ 左侧栏常驻     │      ❌        │ 命令面板 → 选择器
会话管理              │ 左侧栏常驻     │      ❌        │ 命令面板 → 选择器
6个面板               │ 右侧常驻+Tab   │      ❌        │ 右侧滑入+Tab
面板间切换            │ 点击Tab        │      ❌        │ 面板内二级Tab
返回聊天              │ 始终可见       │ 始终可见       │ ← 返回 / 右滑关闭
审批弹窗              │ 居中Modal      │ 居中Modal      │ 底部Sheet
─────────────────────┼───────────────┼───────────────┼───────────────
默认界面复杂度        │ 高（三栏）     │ 极低（纯聊天） │ 极低（纯聊天）
功能完整度            │ 100%          │ ~30%          │ 100%
触达步数              │ 1步（直接点） │ 不可达         │ 2步（[/]→选择）
```

---

## 二、布局全景图

### 2.1 默认界面（极简聊天）

```
┌─────────────────────────────────────┐
│ StatusBar                           │  ← 只读信息，最小化
│ 会话名 · 工作区名           ● idle  │
├─────────────────────────────────────┤
│                                     │
│           纯聊天                     │
│                                     │
│   MessageList（消息列表）            │
│   ToolCallCard（工具调用卡片）       │
│   ThinkingIndicator（思考指示器）    │
│                                     │
├─────────────────────────────────────┤
│  [/] │ 输入消息...           [发送] │  ← 唯一交互入口
├─────────────────────────────────────┤
│ Context 12.5K · Tokens 45.2K ↑34 ↓11 · 60 FPS · /home/project │
│          ← 输入栏底部信息栏（与桌面端/VSCode 端完全一致）→              │
└─────────────────────────────────────┘
```

StatusBar 仅展示当前会话名、工作区名和连接状态，不承载交互入口。整个界面唯一的交互入口是输入框左侧的 `[/]` 按钮。

输入栏底部信息栏与桌面端/VSCode 端完全一致，包含：

| 项目 | 内容 | 来源 |
| :--- | :--- | :--- |
| Context | 当前上下文 token 用量 | `sessionStore.currentSession.currentContextTokens` |
| Tokens | 会话累计 token（输入 ↑ + 输出 ↓） | `sessionStore.currentSession.tokenUsage` |
| FPS | 帧率计数器 | `useFps` composable |
| 工作区目录 | 当前会话绑定的工作区路径 | `sessionStore.currentSession.workingDirectory` |

> 移动端相比桌面端**去掉版本号**（`v1.0`），因为移动端屏幕窄，版本号信息价值低。各项文字字号缩小至 11px，分隔符用 `·`，超出宽度时工作区目录末尾省略号截断，整体保持单行。

### 2.2 命令面板（底部 Sheet）

点击 `[/]` 后，从底部弹出命令面板 Sheet，包含全部 19 个命令（桌面端 9 个 + 移动端 10 个）：

```
┌─────────────────────────────────────┐
│ (聊天区域变暗，半透明遮罩)           │
│                                     │
├─────────────────────────────────────┤  ← 底部 Sheet（半屏起步，可拖拽至全屏）
│  ─── 拖动条 ───                     │
│  🔍 搜索命令...                     │  ← 输入过滤
│                                     │
│  ── 会话（桌面端现有 9 个）───────── │
│  💬 /new            创建新会话      │
│  💬 /switch         切换会话        │
│  💬 /rename         重命名当前会话   │
│  💬 /export         导出会话记录    │
│  💬 /rollback       回滚消息        │
│  💬 /pause          暂停当前会话    │
│  💬 /resume         恢复当前会话    │
│  💬 /cancel         取消当前操作    │
│  💬 /help           显示帮助        │
│                                     │
│  ── 面板（移动端新增 6 个）───────── │
│  📂 /files          打开文件面板    │
│  🧩 /skills         技能管理        │
│  🧠 /memory         记忆管理        │
│  📊 /dashboard      仪表盘          │
│  ⚙️ /settings       设置            │
│  💻 /terminal       终端            │
│                                     │
│  ── 工作区（移动端新增 3 个）─────── │
│  📁 /workspace-switch  切换工作区    │
│  ➕ /workspace-add     添加工作区    │
│  🗑 /workspace-delete  删除工作区    │
│                                     │
│  ── 应用（移动端新增 1 个）───────── │
│  🌙 /toggle-theme   切换主题        │
└─────────────────────────────────────┘
```

### 2.3 右侧滑入面板

执行面板命令后，面板从右侧滑入，覆盖聊天区域：

```
┌─────────────────────────────────┐
│ ← 返回  │  Files               │  ← 面板顶部标题栏
├─────────────────────────────────┤
│  Files│Skills│Memory│Dash│Set│Ter│  ← 二级 Tab Bar（横向滚动）
│ ─────────────────────────────── │
│                                 │
│     ┌─────────────────────────┐│
│     │                         ││
│     │   面板完整内容            ││  ← 与桌面端 RightPanel 内容一致
│     │                         ││
│     │   FilesPanel / Skills   ││
│     │   Memory / Dashboard    ││
│     │   Settings / Terminal   ││
│     │                         ││
│     └─────────────────────────┘│
│                                 │
└─────────────────────────────────┘
```

关键交互：

| 交互 | 行为 |
| :--- | :--- |
| 面板内 Tab 切换 | 顶部横向滚动的二级 Tab，面板间自由切换 |
| 返回聊天 | ① 点击左上角 `← 返回`；② 右滑手势关闭面板 |
| 关闭后 | 回到聊天界面，聊天状态保持不变 |

### 2.4 工作区 / 会话选择器（底部 Sheet）

执行切换命令后，弹出专用选择器：

**工作区选择器：**

```
┌─────────────────────────────────┐
│  ─── 拖动条 ───                 │
│  切换工作区                      │
│                                 │
│  ● /home/user/project-a  ←当前  │
│    /home/user/project-b         │
│    + 添加工作区...               │
│                                 │
│  [取消]                         │
└─────────────────────────────────┘
```

**会话选择器：**

```
┌─────────────────────────────────┐
│  ─── 拖动条 ───                 │
│  切换会话                 [+新建]│
│                                 │
│  ● 修复登录bug          ←当前   │
│    重构用户模块                  │
│    新功能开发                    │
│                                 │
│  [取消]                         │
└─────────────────────────────────┘
```

选择工作区或会话后，自动关闭选择器，执行切换并跳回聊天界面。

### 2.5 审批弹窗（底部 Sheet）

桌面端的居中 Modal 改为移动端的底部 Sheet：

```
┌─────────────────────────────────┐
│ (聊天区域变暗)                   │
├─────────────────────────────────┤
│  ─── 拖动条 ───                 │
│  ⚠️ 审批请求                     │
│                                 │
│  操作：修改文件                  │
│  风险等级：中                   │
│  文件路径：/path/to/file.ts     │
│                                 │
│  ┌─────────────────────────┐   │
│  │ diff 内容预览...         │   │
│  └─────────────────────────┘   │
│                                 │
│  [拒绝]           [批准]        │
└─────────────────────────────────┘
```

---

## 三、组件树

### 3.1 移动端模式

```
MobileLayout.vue
├── StatusBar.vue（只读模式，无交互入口）
├── MobileInputBar.vue（新建）
│   ├── [/] CommandButton       ← 触发命令面板
│   ├── 输入框                    ← 复用 InputArea 核心逻辑
│   └── 发送按钮
├── ChatView.vue → ChatPanel.vue（完全复用）
│   ├── MessageList.vue
│   │   ├── MessageBubble.vue
│   │   ├── ToolCallCard.vue / ToolCallGroup.vue
│   │   └── ThinkingIndicator.vue
│   └── InputArea.vue（桌面端触发 / 命令，移动端不触发）
│
├── MobileCommandSheet.vue（新建）
│   ├── 搜索框
│   ├── 命令分组列表（面板 / 会话 / 工作区 / 应用）
│   └── 命令项（图标 + 标签 + 描述）
│
├── MobilePanelDrawer.vue（新建）
│   ├── 顶部标题栏（← 返回 + 面板名）
│   ├── 二级 Tab Bar（横向滚动）
│   └── <component :is="activePanel" />（动态渲染 6 个面板）
│
├── GlobalModals.vue（复用）
│   ├── ApprovalModal（Sheet 模式）
│   ├── HelpPanel（Sheet 模式）
│   └── 其他弹窗（Sheet 模式）
│
└── ToastContainer.vue（复用）
```

### 3.2 与桌面端组件复用关系

```
桌面端组件                          移动端复用
─────────────────────────────────┼─────────────────────────────
ChatView.vue                      │ 直接复用，零改动
ChatPanel.vue                     │ 直接复用
MessageList.vue                   │ 直接复用
MessageBubble.vue                 │ 直接复用
ToolCallCard.vue                  │ 直接复用
ToolCallGroup.vue                 │ 直接复用
ThinkingIndicator.vue             │ 直接复用
InputArea.vue                     │ 逻辑复用，移动端用 MobileInputBar 包装
StatusBar.vue                     │ 只读模式复用（compact prop）
─────────────────────────────────┼─────────────────────────────
RightPanel.vue                    │ 逻辑复用，移动端用 MobilePanelDrawer 包装
FilesPanel.vue                    │ 直接复用
SkillsPanel.vue                   │ 直接复用
MemoryPanel.vue                   │ 直接复用
DashboardPanel.vue                │ 直接复用
SettingsPanel.vue                 │ 直接复用
TerminalPanel.vue                 │ 直接复用
─────────────────────────────────┼─────────────────────────────
AppSidebar.vue                    │ 抽离 WorkspaceList + SessionList 子组件后复用
WorkspaceList.vue（新建）          │ 工作区选择器复用
SessionList.vue（新建）            │ 会话选择器复用
─────────────────────────────────┼─────────────────────────────
CommandPalette.vue                │ 命令注册逻辑复用，UI 用 MobileCommandSheet
useCommand.ts                     │ 扩展 group/mobileLabel 字段后复用
─────────────────────────────────┼─────────────────────────────
ApprovalModal.vue                 │ 增加 Sheet 模式（variant prop）
GlobalModals.vue                  │ 直接复用
ToastContainer.vue                │ 直接复用
```

---

## 四、核心交互流程

### 4.1 打开面板

```
用户点击 [/] 
  → 底部弹出 MobileCommandSheet
  → 用户选择 “📂 文件面板”
  → Sheet 关闭
  → MobilePanelDrawer 从右侧滑入
  → 默认显示 Files 面板
  → 用户可在面板内切换其他 Tab
  → 用户点击 ← 返回或右滑
  → 面板关闭，回到聊天
```

### 4.2 切换工作区

```
用户点击 [/]
  → 底部弹出 MobileCommandSheet
  → 用户选择 “📁 切换工作区”
  → Sheet 关闭
  → 底部弹出工作区选择器 Sheet
  → 用户选择目标工作区
  → POST /api/v1/current-workspace
  → 刷新会话列表
  → 选择器关闭
  → Toast 提示 “已切换到 /home/user/project-b”
  → 回到聊天界面
```

### 4.3 切换会话

```
用户点击 [/]
  → 底部弹出 MobileCommandSheet
  → 用户选择 “💬 切换会话”
  → Sheet 关闭
  → 底部弹窗会话选择器 Sheet
  → 用户选择目标会话
  → 加载消息 + 连接 SSE
  → 选择器关闭
  → 回到聊天界面，显示新会话消息
```

### 4.4 审批处理

```
SSE 推送 approval_required
  → 底部弹出审批 Sheet（覆盖聊天区域）
  → 用户查看操作详情和 diff
  → 用户点击 [批准] 或 [拒绝]
  → POST /api/v1/approvals/{id}/resolve
  → Sheet 关闭
  → 回到聊天，继续对话
```

---

## 五、命令体系设计

### 5.1 命令注册扩展

在现有 `useCommand` 和 `Command` 接口基础上扩展 `group` 和 `mobileLabel` 字段，使命令可被移动端命令面板分类展示：

```typescript
// 现有接口（来自 stores/command.ts）
interface Command {
  id: string
  name: string
  description: string
  placeholder?: string
}

// 移动端扩展字段
interface MobileCommandMeta {
  group: 'panel' | 'session' | 'workspace' | 'app'  // 新增：命令分类
  mobileLabel: string    // 新增：移动端展示标签，如 '📂 文件面板'
}
```

### 5.2 完整命令清单（桌面端 9 个 + 移动端新增 10 个 = 共 19 个命令）

#### 5.2.1 会话命令（全部来自桌面端现有命令，移动端完整继承）

| 命令 | 桌面端 | 移动端 | 行为 |
| :--- | :---: | :---: | :--- |
| `/new` | ✅ | ✅ | 创建新会话（带 placeholder `[名称]`） |
| `/switch` | ✅ | ✅ | 弹出会话选择器 Sheet |
| `/rename` | ✅ | ✅ | 内联重命名当前会话（带 placeholder `<新名称>`） |
| `/export` | ✅ | ✅ | 导出当前会话记录 |
| `/rollback` | ✅ | ✅ | 回滚消息（弹出 RollbackPicker Sheet） |
| `/pause` | ✅ | ✅ | 暂停当前会话 |
| `/resume` | ✅ | ✅ | 恢复当前会话 |
| `/cancel` | ✅ | ✅ | 取消当前操作 |
| `/help` | ✅ | ✅ | 弹出帮助面板 Sheet |

#### 5.2.2 面板命令（移动端新增）

| 命令 | 桌面端 | 移动端 | 行为 |
| :--- | :---: | :---: | :--- |
| `/files` | ❌ | ✅ | 右侧滑入文件面板 |
| `/skills` | ❌ | ✅ | 右侧滑入技能管理面板 |
| `/memory` | ❌ | ✅ | 右侧滑入记忆管理面板 |
| `/dashboard` | ❌ | ✅ | 右侧滑入仪表盘面板 |
| `/settings` | ❌ | ✅ | 右侧滑入设置面板 |
| `/terminal` | ❌ | ✅ | 右侧滑入终端面板 |

#### 5.2.3 工作区命令（移动端新增）

| 命令 | 桌面端 | 移动端 | 行为 |
| :--- | :---: | :---: | :--- |
| `/workspace-switch` | ❌ | ✅ | 弹出工作区选择器 Sheet |
| `/workspace-add` | ❌ | ✅ | 弹出添加工作区弹窗 |
| `/workspace-delete` | ❌ | ✅ | 弹出工作区选择器（含删除确认） |

#### 5.2.4 应用命令（移动端新增）

| 命令 | 桌面端 | 移动端 | 行为 |
| :--- | :---: | :---: | :--- |
| `/toggle-theme` | ❌ | ✅ | 切换主题（亮/暗） |

### 5.3 命令面板分组展示

移动端命令面板按四个分组展示，所有 19 个命令均可通过搜索过滤：

```
┌─────────────────────────────────────┐
│  ─── 拖动条 ───                     │
│  🔍 搜索命令...                     │
│                                     │
│  ── 会话（9 个，全部来自桌面端）──── │
│  💬 /new            创建新会话      │
│  💬 /switch         切换会话        │
│  💬 /rename         重命名当前会话   │
│  💬 /export         导出会话记录    │
│  💬 /rollback       回滚消息        │
│  💬 /pause          暂停当前会话    │
│  💬 /resume         恢复当前会话    │
│  💬 /cancel         取消当前操作    │
│  💬 /help           显示帮助        │
│                                     │
│  ── 面板（6 个，移动端新增）──────── │
│  📂 /files          打开文件面板    │
│  🧩 /skills         技能管理        │
│  🧠 /memory         记忆管理        │
│  📊 /dashboard      仪表盘          │
│  ⚙️ /settings       设置            │
│  💻 /terminal       终端            │
│                                     │
│  ── 工作区（3 个，移动端新增）────── │
│  📁 /workspace-switch  切换工作区    │
│  ➕ /workspace-add     添加工作区    │
│  🗑 /workspace-delete  删除工作区    │
│                                     │
│  ── 应用（1 个，移动端新增）──────── │
│  🌙 /toggle-theme   切换主题        │
└─────────────────────────────────────┘
```

### 5.4 与桌面端命令面板的关系

| | 桌面端 | 移动端 |
| :--- | :--- | :--- |
| 触发方式 | 输入 `/` 键 | 点击 `[/]` 按钮 |
| UI 容器 | 输入框上方浮层（520px 宽） | 底部 Sheet（半屏/全屏） |
| 交互方式 | 键盘输入过滤 + ↑↓ 导航 + Enter 选择 | 点击选择 + 可搜索过滤 |
| 命令数量 | 当前 9 个 | 19 个（9 个桌面端 + 10 个新增） |
| 命令注册 | `useCommand` 统一注册 | 同一套，完全复用 |
| 命令定义 | `id` + `name` + `description` + `placeholder` | 扩展 `group` + `mobileLabel` |
| 分组 | 无（平铺列表） | 有（4 个分组） |

> **关键设计决策**：桌面端 9 个命令在移动端**全部保留**，不做任何删减。移动端额外新增 10 个命令（6 个面板 + 3 个工作区 + 1 个主题），覆盖桌面端通过左侧栏和右侧面板 Tab 直达的功能。桌面端用户习惯用快捷键（Ctrl+K）、全局快捷键（Alt+Y、Esc）和 UI 点击操作面板/工作区，因此不需要面板类和工作区类命令；移动端没有键盘和侧栏，这些命令成为必需品。

---

## 六、`[/]` 按钮设计

### 6.1 视觉规格

```
┌─────────────────────────────────────────┐
│                                         │
│  ┌──────┬──────────────────────┬──────┐ │
│  │ [/]  │ 输入消息...           │ 发送 │ │
│  └──────┴──────────────────────┴──────┘ │
│     ↑                                    │
│    40×40px 圆角方块                      │
│    主题色背景 + 白色文字                  │
│    与发送按钮视觉呼应                     │
└─────────────────────────────────────────┘
```

| 属性 | 值 |
| :--- | :--- |
| 尺寸 | 40×40px |
| 圆角 | 8px |
| 背景 | `var(--color-accent)` 主题色 |
| 文字 | 白色 `/` 字符，字号 18px，字重 600 |
| 反馈 | 点击 scale(0.95) → scale(1.0)，200ms ease-out |
| 位置 | 输入框左侧，间距 8px |
| 长按 | 预留，未来可扩展为语音输入 |

### 6.2 输入框整合

`MobileInputBar` 组件整合 `[/]` 按钮 + 输入框 + 发送按钮 + 底部信息栏：

```
MobileInputBar.vue
├── 输入行
│   ├── [/] 按钮（command-button）
│   ├── 输入框（复用 InputArea 的输入逻辑）
│   │   ├── 文本输入
│   │   ├── @ 提及（未来）
│   │   ├── 多行自动扩展
│   │   └── 大段粘贴折叠（与桌面端一致）
│   └── 发送/停止按钮
│       ├── idle 状态：发送图标
│       └── streaming 状态：停止图标
└── 底部信息栏（复用 InputArea 的 input-footer 逻辑，去版本号，字号 11px）
    ├── Context token 用量
    ├── 会话累计 Tokens（输入 ↑ + 输出 ↓）
    ├── FPS 帧率
    └── 工作区目录（超出省略）
```

桌面端 `InputArea` 的 `/` 触发命令面板逻辑在移动端**不生效**，改为由 `[/]` 按钮触发。输入框本身只负责文本输入和发送。

底部信息栏与桌面端/VSCode 端**完全一致**，直接复用 `InputArea` 中 `input-footer` 的模板逻辑和数据源。移动端窄屏下文字缩小至 12px，超出宽度时横向滚动。

---

## 七、模式检测

### 7.1 检测逻辑

在 `usePlatform.ts` 中扩展：

```typescript
function detectMode(): 'browser' | 'vscode' | 'mobile' {
  const params = new URLSearchParams(window.location.search)
  const mode = params.get('mode')
  
  if (mode === 'vscode') return 'vscode'
  if (mode === 'mobile') return 'mobile'
  
  // 自动检测：屏幕宽度 < 768px 且非 VSCode 模式 → 自动走 MobileLayout
  if (window.innerWidth < 768) return 'mobile'
  
  return 'browser'
}
```

### 7.2 访问方式

| 访问方式 | URL | 布局 |
| :--- | :--- | :--- |
| 桌面浏览器 | `http://localhost:8080/` | BrowserLayout |
| 平板/手机浏览器 | `http://localhost:8080/` | MobileLayout（自动检测） |
| 强制桌面模式 | `http://localhost:8080/?mode=browser` | BrowserLayout |
| VSCode Webview | `http://localhost:8080/?mode=vscode` | VscodeLayout |
| 强制移动端模式 | `http://localhost:8080/?mode=mobile` | MobileLayout |

### 7.3 App.vue 布局切换

扩展 `App.vue` 模板中的条件渲染：

```
<template>
  <VscodeLayout v-if="layoutMode === 'vscode'" />
  <MobileLayout v-else-if="layoutMode === 'mobile'" />
  <BrowserLayout v-else />
</template>
```

---

## 八、移动端特有适配

### 8.1 安全区域

适配刘海屏（notch）和底部横条（home indicator）：

```css
.mobile-layout {
  padding-top: env(safe-area-inset-top, 0px);
  padding-bottom: env(safe-area-inset-bottom, 0px);
  padding-left: env(safe-area-inset-left, 0px);
  padding-right: env(safe-area-inset-right, 0px);
}

.mobile-input-bar {
  /* 底部额外留出安全区域 */
  padding-bottom: calc(12px + env(safe-area-inset-bottom, 0px));
}

.mobile-command-sheet {
  /* Sheet 底部圆角时也考虑安全区域 */
  padding-bottom: env(safe-area-inset-bottom, 0px);
}
```

### 8.2 动态视口高度

使用 `dvh`（dynamic viewport height）替代 `100vh`，避免移动浏览器地址栏展开/收起时布局抖动：

```css
.mobile-layout {
  height: 100dvh;
}
```

### 8.3 触摸目标

所有可点击元素遵循移动端最小触摸目标标准：

| 平台 | 最小触摸目标 |
| :--- | :--- |
| Apple HIG | 44×44px |
| Material Design | 48×48px |
| 本项目采用 | 44×44px（含 padding） |

```css
.touch-target {
  min-width: 44px;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
}
```

### 8.4 键盘处理

输入框聚焦时，键盘弹起后的处理：

- 消息列表自动滚动到底部
- 输入区跟随键盘上移（使用 `visualViewport` API）
- 命令面板和选择器在键盘弹起时自动收起

```typescript
// visualViewport 监听
window.visualViewport?.addEventListener('resize', () => {
  const keyboardHeight = window.innerHeight - (window.visualViewport?.height ?? 0)
  // 调整输入区位置
})
```

### 8.5 手势支持

| 手势 | 作用 |
| :--- | :--- |
| 右滑（面板内） | 关闭右侧面板，返回聊天 |
| 下滑（Sheet） | 关闭底部 Sheet |
| 左滑（列表项） | 露出操作按钮（删除、重命名） |
| 长按（列表项） | 进入编辑模式（未来） |

### 8.6 横向滚动 Tab

面板内的二级 Tab（Files/Skills/Memory/Dashboard/Settings/Terminal）在窄屏下支持横向滚动：

```css
.panel-tabs {
  display: flex;
  overflow-x: auto;
  scroll-behavior: smooth;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;  /* 隐藏滚动条 */
}

.panel-tabs::-webkit-scrollbar {
  display: none;
}

.panel-tab {
  flex-shrink: 0;
  padding: 8px 16px;
  white-space: nowrap;
}
```

---

## 九、新增/修改文件清单

### 9.1 新增文件

```
web/src/
├── layouts/
│   └── MobileLayout.vue              ← 移动端布局壳
│
├── components/
│   ├── mobile/
│   │   ├── MobileCommandSheet.vue    ← 底部命令面板 Sheet
│   │   ├── MobilePanelDrawer.vue     ← 右侧滑入面板容器
│   │   ├── MobileInputBar.vue        ← 输入栏（[/] + 输入框 + 发送）
│   │   ├── MobileWorkspacePicker.vue ← 工作区选择器 Sheet
│   │   └── MobileSessionPicker.vue   ← 会话选择器 Sheet
│   │
│   └── layout/
│       ├── WorkspaceList.vue         ← 从 AppSidebar 抽出（共享）
│       └── SessionList.vue           ← 从 AppSidebar 抽出（共享）
```

### 9.2 修改文件

| 文件 | 改动 |
| :--- | :--- |
| `App.vue` | 增加 `MobileLayout` 条件分支 |
| `composables/usePlatform.ts` | 增加 `mobile` 模式检测 + 自动检测逻辑 |
| `composables/useCommand.ts` | 命令接口扩展 `group` + `mobileLabel` 字段 |
| `components/layout/AppSidebar.vue` | 抽离 `WorkspaceList` + `SessionList` 为独立子组件 |
| `components/layout/StatusBar.vue` | 增加 `compact` prop（移动端只读模式） |
| `components/modal/ApprovalModal.vue` | 增加 `variant` prop（`modal` / `sheet`） |
| `stores/ui.ts` | 增加 `layoutMode` 状态字段 |

---

## 十、实施计划

| 阶段 | 内容 | 预估 |
| :--- | :--- | :--- |
| **Phase 1** | 重构 `AppSidebar.vue`：抽离 `WorkspaceList.vue` + `SessionList.vue` 子组件 | 小 |
| **Phase 2** | 扩展 `useCommand.ts`：增加 `group` + `mobileLabel` 字段 | 小 |
| **Phase 3** | 新建 `MobileCommandSheet.vue`：底部 Sheet 命令面板 | 中 |
| **Phase 4** | 新建 `MobileWorkspacePicker.vue` + `MobileSessionPicker.vue`：选择器 Sheet | 小 |
| **Phase 5** | 新建 `MobilePanelDrawer.vue`：右侧滑入面板容器 | 中 |
| **Phase 6** | 新建 `MobileInputBar.vue`：输入栏（`[/]` 按钮 + 输入框 + 发送） | 中 |
| **Phase 7** | 新建 `MobileLayout.vue`：组装所有移动端组件 | 中 |
| **Phase 8** | 修改 `usePlatform.ts`：增加 `mobile` 模式检测 | 小 |
| **Phase 9** | 修改 `App.vue`：增加 `MobileLayout` 分支 | 小 |
| **Phase 10** | 改造 `ApprovalModal.vue`：增加 Sheet 模式 | 小 |
| **Phase 11** | 移动端样式适配：安全区域、dvh、触摸优化、键盘处理 | 中 |
| **Phase 12** | 手势支持：右滑关闭面板、下滑关闭 Sheet、左滑操作 | 中 |

---

## 十一、关键设计决策总结

| 决策 | 方案 | 理由 |
| :--- | :--- | :--- |
| 默认界面 | 极简聊天 + `[/]` 按钮 | 移动端屏幕宝贵，默认只展示聊天 |
| 功能触达 | 命令面板驱动 | 与 VS Code Command Palette 哲学一致，功能完整不阉割 |
| 面板展示 | 右侧滑入 | 与桌面端右侧面板空间映射一致，右滑返回自然 |
| 选择器 | 底部 Sheet | 拇指热区，单手可操作，下滑关闭 |
| 审批弹窗 | 底部 Sheet | 移动端原生体验，不遮挡过多内容 |
| 命令体系 | 复用 `useCommand`，19 个命令全覆盖 | 桌面端 9 个命令全部保留 + 移动端新增 10 个（面板/工作区/主题），桌面端 `/` 和移动端 `[/]` 共享同一套命令注册 |
| 组件复用 | 最大化复用 | ChatPanel、6 个 Panel、GlobalModals、Toast 全部直接复用 |
| 键盘处理 | `visualViewport` API | 输入区跟随键盘，消息列表自动滚动 |
| 安全区域 | `env(safe-area-inset-*)` | 适配刘海屏和底部横条 |
| 视口高度 | `100dvh` | 避免移动浏览器地址栏抖动 |
| 自动检测 | 屏幕宽度 < 768px | 同一 URL 在手机上自动切换，无需手动加参数 |
| 触摸目标 | 最小 44×44px | 符合 Apple HIG 标准 |