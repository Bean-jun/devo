# Devo TUI 设计规范

**版本**：2.1.0
**参考**：Web 端 design_spec.md + variables.css + Claude Code 终端风格

**2.1.0 变更**：
- 消息全部左对齐，去除用户消息右对齐，统一阅读流向
- 去除常驻 HelpBar，改为输入区底部极简提示 `/? help  ^C quit`
- 命令面板增加 `↑↓` 键盘导航、`>` 选中高亮、`Enter` 执行命令
- 会话选择器增加 `↑↓` 键盘导航、`>` 选中高亮、`Enter` 切换会话

---

## 1. 设计哲学

**核心原则：终端就是终端，不模拟 GUI。**

以 Claude Code 为风格蓝本：用颜色和符号做区分，不用边框模拟网页元素。设计语言同时参考 Web 移动端（MobileLayout）的 Sheet/Overlay 导航模式。

| 约束 | 移动端 | TUI | 策略 |
|------|--------|-----|------|
| 屏幕宽度 | 窄（375-430px） | 窄（80-120列） | 单栏布局 |
| 交互方式 | 触摸（手势） | 键盘（快捷键） | 快捷键 + 命令面板 |
| 导航模式 | Sheet/抽屉 | 弹窗/覆盖层 | Overlay Stack |

**风格关键词**：极简 · 符号前缀 · 无气泡 · 终端原生 · 颜色即层次

---

## 2. 色彩体系

### 2.1 Dark 主题（默认）

| Token | 色值 | 用途 |
|-------|------|------|
| `bg-primary` | `#0d1117` | 主背景 |
| `bg-secondary` | `#161b22` | 面板底色 |
| `bg-tertiary` | `#21262d` | 输入框底色 |
| `text-primary` | `#e6edf3` | 主文字 |
| `text-secondary` | `#8b949e` | 次要文字、时间戳 |
| `text-tertiary` | `#6e7681` | 占位符 |
| `accent` | `#58a6ff` | 主强调色（蓝） |
| `success` | `#3fb950` | 成功 |
| `warning` | `#d29922` | 警告 |
| `error` | `#f85149` | 错误 |
| `border` | `#30363d` | 边框 |
| `overlay` | `#000000` | 遮罩层 |

### 2.2 Light 主题（可选）

| Token | 色值 |
|-------|------|
| `bg-primary` | `#ffffff` |
| `bg-secondary` | `#f5f5f7` |
| `bg-tertiary` | `#e8e8ed` |
| `text-primary` | `#1d1d1f` |
| `text-secondary` | `#86868b` |
| `accent` | `#0071e3` |
| `border` | `#d2d2d7` |

### 2.3 状态颜色

| 状态 | 颜色 | 色值 |
|------|------|------|
| `idle` | 绿 | `#3fb950` |
| `thinking` / `processing` / `tool_executing` | 蓝 | `#58a6ff` |
| `awaiting_approval` | 黄 | `#d29922` |
| `paused` | 灰 | `#8b949e` |
| `cancelled` / `error` | 红 | `#f85149` |
| `completed` | 绿 | `#3fb950` |

---

## 3. 布局结构

### 3.1 默认布局（单栏）

```
┌──────────────────────────────────────────────────────────┐
│  🚀 Devo  │  my-project  │  ● Processing  │  YOLO  │  ✓  │  ← StatusBar
├──────────────────────────────────────────────────────────┤
│                                                          │
│  ⏺ 帮我修复 utils.go 中的空指针问题                       │
│    14:32                                                 │
│                                                          │
│  ⏺ 我来分析一下代码...                                    │
│                                                          │
│    · 需要先读取 utils.go 找到具体位置，然后添加 nil 检查   │
│    · 使用防御性编程，在所有可能为 nil 的地方加检查         │
│                                                          │
│    ✓ read_file  utils.go  ·  156 lines  ·  0.3s         │
│    ✓ write_file  utils.go  ·  Approved  ·  1.2s         │
│    │  + if x == nil { return }                           │
│    │  - oldFunc() {                                      │
│                                                          │
│    已修复 utils.go 中的空指针问题，在 oldFunc()            │
│    开头添加了 nil 检查...                                 │
│                                                          │
│    14:32                                                 │
│                                                          │
│  session 已创建 · 2026-08-07 14:30                       │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────┐  │
│  │  /  继续优化这个函数                           [⏎] │  │  ← 输入区
│  │  Context: 12.5K  ·  Tokens: 3.2K  ·  /home/prj   │  │
│  │  /? help  ^C quit                                 │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

---

## 4. 组件详细设计

### 4.1 StatusBar

```
🚀 Devo  │  my-project  │  ● Processing  │  YOLO  │  ✓
```

- 高度 1 行，底部 `border` 色分隔线
- 背景 `bg-secondary`（`#161b22`），各段 `│` 分隔

| 段 | 内容 | 样式 |
|----|------|------|
| App 名 | `🚀 Devo` | 蓝色粗体 `#58a6ff` |
| 会话名 | `my-project` | 白色 `#e6edf3`，F2 编辑 |
| 状态 | `● Processing` | 圆点颜色随状态变化 |
| YOLO | `YOLO` | 激活时黄色背景 `#d29922` + 黑字 |
| 连接 | `✓` | 绿 `#3fb950`，断开时红 `✗` |

### 4.2 消息前缀（Claude Code 风格）

**不用气泡框，用符号前缀区分身份。所有消息左对齐，连续流式阅读。**

#### 用户消息

```
⏺ 帮我修复 utils.go 中的空指针问题
  14:32
```

- 前缀 `⏺` 蓝色 `#58a6ff`，消息内容白色 `#e6edf3`
- **左对齐**，与助手消息统一流向
- 时间戳左下角，灰色 `#8b949e`，缩进 2 格

#### 助手消息

```
⏺ 我来分析一下代码...

  · 需要先读取 utils.go 找到具体位置，然后添加 nil 检查...
  · 使用防御性编程

  ✓ read_file  utils.go  ·  156 lines  ·  0.3s
  ✓ write_file  utils.go  ·  Approved  ·  1.2s
  │  + if x == nil { return }
  │  - oldFunc() {

  已修复 utils.go 中的空指针问题，在 oldFunc() 开头添加了 nil 检查。

  14:32
```

- 前缀 `⏺` 默认色 `#e6edf3`，消息内容同色
- 左对齐，全宽
- 时间戳左下角，灰色 `#8b949e`，缩进 2 格

#### 系统消息

```
session 已创建 · 2026-08-07 14:30
```

- 灰色斜体 `#8b949e`，左对齐
- 无前缀、无装饰线

### 4.3 思考过程

```
  · 需要先读取 utils.go 找到具体位置，然后添加 nil 检查...
  · 考虑使用防御性编程，在所有可能为 nil 的地方加检查
```

- `·` 前缀，灰色斜体 `#8b949e`，缩进 2 格
- 跟随在助手消息 `⏺` 之后、正式回复之前
- 不折叠，直接展示（若内容过长可截断 + `Enter 展开全部` 提示）

### 4.4 工具调用

```
  ⏺ read_file  utils.go  ·  156 lines  ·  ✓  0.3s
  ⏺ write_file  utils.go  ·  Approved  ·  ✓  1.2s
  │  + if x == nil { return }
  │  - oldFunc() {
```

- 缩进 2 格，前缀符号表示状态：
  - `⏺` 蓝色 = 执行中
  - `✓` 绿色 = 成功
  - `✗` 红色 = 失败
  - `⏺` 黄色 = 等待审批
- 单行格式：`前缀 工具名  文件名/参数  ·  状态  ·  耗时`
- 展开的 diff/output 用 `│` 竖线缩进，终端原生 diff 风格
- **无卡片边框**，靠缩进和颜色区分层级

### 4.5 输入区

```
┌──────────────────────────────────────────────────┐
│  /  继续优化这个函数                         [⏎] │
│  Context: 12.5K  ·  Tokens: 3.2K  ·  /home/prj  │
└──────────────────────────────────────────────────┘
```

- 圆角边框 `#30363d`，底色 `bg-tertiary`（`#21262d`）
- 左侧 `/` 蓝色 `#58a6ff`（命令模式入口）
- 右侧 `[⏎]` 发送 / `[■]` 停止
- 底部页脚：左对齐 Context + Tokens，右对齐工作目录

### 4.6 HelpBar

```
^S 会话  ^N 新建  ^P 暂停  ^C 取消  ^Q 退出  ^H 帮助
```

- 高度 1 行，居中，灰色 `#8b949e`
- `^` 蓝色高亮 `#58a6ff`
- 根据状态动态显示：
  - Idle：`^S 会话  ^N 新建  ^P 暂停  ^Q 退出  ^H 帮助`
  - Processing：`^C 取消  ^P 暂停  ^H 帮助`
  - Awaiting：`Y 批准  N 拒绝  D 查看Diff`

### 4.7 审批弹窗（ApprovalModal）

```
┌──────────────────────────────────────────────────┐
│                                                  │
│   ┌──────────────────────────────────────┐       │
│   │  ⚠ Approval Required                │       │
│   │                                      │       │
│   │  Operation: write_file               │       │
│   │  Risk:      HIGH                     │       │
│   │                                      │       │
│   │  ┌ Diff ─────────────────────────┐  │       │
│   │  │ + if x == nil { return }      │  │       │
│   │  │ - oldFunc() {                 │  │       │
│   │  └───────────────────────────────┘  │       │
│   │                                      │       │
│   │  [Y] Approve  [N] Reject  [D] Diff  │       │
│   └──────────────────────────────────────┘       │
│                                                  │
└──────────────────────────────────────────────────┘
```

- 全屏半透明遮罩（黑 60%），弹窗居中，黄色边框 `#d29922`
- 风险等级：HIGH=红 `#f85149`，MEDIUM=黄 `#d29922`，LOW=绿 `#3fb950`

### 4.8 命令面板（CommandSheet，`/` 触发）

```
┌──────────────────────────────────────────────────┐
│   ┌──────────────────────────────────────┐       │
│   │  🔍 搜索命令...                       │       │
│   │  ──────────────────────────────────  │       │
│   │  SESSION                             │       │
│   │  /new        创建新会话               │       │
│   │  /switch     切换会话                 │       │
│   │  /rename     重命名会话               │       │
│   │  /export     导出会话                 │       │
│   │  /rollback   回滚到消息               │       │
│   │  ──────────────────────────────────  │       │
│   │  PANEL                               │       │
│   │  /files      文件管理                 │       │
│   │  /skills     技能管理                 │       │
│   │  /mcp        MCP 管理                │       │
│   │  /memory     记忆管理                 │       │
│   │  ──────────────────────────────────  │       │
│   │  APP                                  │       │
│   │ >/yolo       切换 YOLO 模式           │  ← 选中项
│   │  /theme      切换主题                 │       │
│   │  /help       帮助                     │       │
│   │  /quit       退出                     │       │
│   │                                      │       │
│   │       [↑↓] 导航  [Enter] 执行  [Esc]  │       │
│   └──────────────────────────────────────┘       │
└──────────────────────────────────────────────────┘
```

- 底部弹出，覆盖聊天区半屏
- 分组标签：大写蓝色 `#58a6ff`，命令名蓝色，描述灰色
- **键盘导航**：
  - `↑↓` 在命令列表中移动选中项
  - 选中项以 `>` 前缀 + 蓝色底色高亮
  - `Enter` 执行选中命令（Mock 动作为 Toast 提示）
  - `Esc` 关闭面板
- **实时搜索**：输入 `/` 后继续打字可过滤命令列表，匹配不到的命令隐藏
- 底部显示导航提示：`[↑↓] 导航  [Enter] 执行  [Esc] 关闭`

### 4.9 会话选择器（SessionPicker，Ctrl+S）

```
┌──────────────────────────────────────────────────┐
│   ┌──────────────────────────────────────┐       │
│   │  切换会话                    [+ 新建] │       │
│   │  ──────────────────────────────────  │       │
│   │  💬 修复登录Bug                       │       │
│   │     "请帮我修复登录页面的..."          │       │
│   │     12条消息 · 2小时前                │       │
│   │                                      │       │
│   │ >● 💬 当前会话                        │  ← 选中项
│   │     5条消息 · 刚刚                    │       │
│   │                                      │       │
│   │     [↑↓] 选择  [Enter] 确认  [Esc]    │       │
│   └──────────────────────────────────────┘       │
└──────────────────────────────────────────────────┘
```

- 底部弹出 Sheet，活跃会话 `●` 蓝色标记
- 显示最后消息预览 + 消息数 + 时间
- **键盘导航**：
  - `↑↓` 在会话列表中移动选中项
  - 选中项以 `>` 前缀 + 蓝色底色高亮
  - `Enter` 切换到选中会话（Mock 动作为 Toast 提示）
  - `Esc` 关闭面板

### 4.10 Toast 通知

- 右上角浮动，3 秒自动消失
- 错误：红底 `#f85149` + 白字
- 信息：蓝底 `#58a6ff` + 白字
- 成功：绿底 `#3fb950` + 白字

### 4.11 帮助面板（HelpPanel，`?`）

```
┌──────────────────────────────────────────────────┐
│  ┌──────────────────────────────────────┐        │
│  │  Help                          [Esc] │        │
│  │  ──────────────────────────────────  │        │
│  │  Navigation                          │        │
│  │  ^S    会话列表                      │        │
│  │  ^N    新建会话                      │        │
│  │                                      │        │
│  │  Chat                                │        │
│  │  Enter    发送消息                   │        │
│  │  Shift+↑  上一条消息（历史）          │        │
│  │  /        打开命令面板               │        │
│  │                                      │        │
│  │  Actions                             │        │
│  │  ^C    取消当前操作                  │        │
│  │  ^P    暂停/恢复                     │        │
│  │  ^Y    切换 YOLO 模式                │        │
│  │  ^T    切换主题                      │        │
│  │  ^Q    退出                          │        │
│  └──────────────────────────────────────┘        │
└──────────────────────────────────────────────────┘
```

---

## 5. Overlay Stack（覆盖层优先级）

按 Esc 逐层关闭，参照移动端 `closeTopOverlay()`：

```
1. ApprovalModal     ← Esc 第1次
2. InfoDialog        ← Esc 第2次
3. CommandSheet      ← Esc 第3次
4. SessionPicker     ← Esc 第4次
5. WorkspacePicker   ← Esc 第5次
6. HelpPanel         ← Esc 第6次
7. 取消当前操作/暂停  ← Esc 第7次
```

---

## 6. 快捷键映射

### 6.1 全局

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+S` | 会话选择器 |
| `Ctrl+N` | 新建会话 |
| `Ctrl+T` | 切换主题 |
| `Ctrl+Y` | 切换 YOLO |
| `Ctrl+Q` | 退出 |
| `?` | 帮助面板 |
| `Esc` | 关闭顶层覆盖层 |

### 6.2 聊天

| 快捷键 | 功能 |
|--------|------|
| `Enter` | 发送消息 |
| `Shift+Enter` | 换行 |
| `Shift+↑` / `Shift+↓` | 历史消息 |
| `/` | 命令面板 |
| `Ctrl+C` | 取消当前操作 |
| `Ctrl+P` | 暂停/恢复 |

### 6.3 会话管理

| 快捷键 | 功能 |
|--------|------|
| `F2` | 重命名会话 |
| `Ctrl+E` | 导出会话 |
| `Ctrl+R` | 回滚 |

### 6.4 审批

| 快捷键 | 功能 |
|--------|------|
| `Y` | 批准 |
| `N` | 拒绝 |
| `D` | 查看完整 Diff |

---

## 7. 消息渲染规则

| 消息类型 | 前缀 | 样式 | 对齐 |
|----------|------|------|------|
| 用户消息 | `⏺` 蓝色 | 白色文字 | 左对齐 |
| 助手消息 | `⏺` 默认色 | 白色文字 | 左对齐 |
| 系统消息 | 无 | 灰色斜体 `#8b949e` | 左对齐 |
| 思考过程 | `·` 灰色 | 灰色斜体 `#8b949e` | 缩进 2 格 |
| 工具调用-执行中 | `⏺` 蓝色 | 缩进 2 格 | 左对齐 |
| 工具调用-成功 | `✓` 绿色 | 缩进 2 格 | 左对齐 |
| 工具调用-失败 | `✗` 红色 | 缩进 2 格 | 左对齐 |
| 工具调用-等待 | `⏺` 黄色 | 缩进 2 格 | 左对齐 |
| 工具 diff 展开 | `│` 灰色 | 缩进 4 格 | 左对齐 |

---

## 8. Go 实现参考

### 8.1 主题 Token

```go
package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
    Name             string
    BgPrimary        lipgloss.Color
    BgSecondary      lipgloss.Color
    BgTertiary       lipgloss.Color
    TextPrimary      lipgloss.Color
    TextSecondary    lipgloss.Color
    TextTertiary     lipgloss.Color
    Accent           lipgloss.Color
    Success          lipgloss.Color
    Warning          lipgloss.Color
    Error            lipgloss.Color
    Border           lipgloss.Color
    Overlay          lipgloss.Color
}

var Dark = Theme{
    Name:          "dark",
    BgPrimary:     "#0d1117",
    BgSecondary:   "#161b22",
    BgTertiary:    "#21262d",
    TextPrimary:   "#e6edf3",
    TextSecondary: "#8b949e",
    TextTertiary:  "#6e7681",
    Accent:        "#58a6ff",
    Success:       "#3fb950",
    Warning:       "#d29922",
    Error:         "#f85149",
    Border:        "#30363d",
    Overlay:       "#000000",
}

var Light = Theme{
    Name:          "light",
    BgPrimary:     "#ffffff",
    BgSecondary:   "#f5f5f7",
    BgTertiary:    "#e8e8ed",
    TextPrimary:   "#1d1d1f",
    TextSecondary: "#86868b",
    TextTertiary:  "#aeaeb2",
    Accent:        "#0071e3",
    Success:       "#34c759",
    Warning:       "#ff9500",
    Error:         "#ff3b30",
    Border:        "#d2d2d7",
    Overlay:       "#000000",
}
```

### 8.2 消息前缀样式

```go
var (
    UserPrefix = lipgloss.NewStyle().
        Foreground(ColorAccent).Bold(true)

    AssistantPrefix = lipgloss.NewStyle().
        Foreground(ColorTextPrimary).Bold(true)

    ThinkingPrefix = lipgloss.NewStyle().
        Foreground(ColorTextSecondary).Italic(true)

    ToolExecuting = lipgloss.NewStyle().
        Foreground(ColorAccent).
        Padding(0, 0, 0, 2)

    ToolSuccess = lipgloss.NewStyle().
        Foreground(ColorSuccess).
        Padding(0, 0, 0, 2)

    ToolError = lipgloss.NewStyle().
        Foreground(ColorError).
        Padding(0, 0, 0, 2)

    ToolPending = lipgloss.NewStyle().
        Foreground(ColorWarning).
        Padding(0, 0, 0, 2)

    DiffLine = lipgloss.NewStyle().
        Foreground(ColorBorder).
        Padding(0, 0, 0, 4)

    SystemNotice = lipgloss.NewStyle().
        Foreground(ColorTextSecondary).Italic(true)

    Timestamp = lipgloss.NewStyle().
        Foreground(ColorTextSecondary)
)
```

### 8.3 消息渲染伪代码

```go
func (m *MessageView) renderMessage(msg Message) string {
    switch msg.Role {
    case "user":
        return renderUser(msg)
    case "assistant":
        return renderAssistant(msg)
    case "system":
        return renderSystem(msg)
    }
    return ""
}

func renderUser(msg Message) string {
    prefix := UserPrefix.Render("⏺")
    line := prefix + " " + msg.Content
    ts := Timestamp.Render("  " + msg.Time)
    // 左对齐，统一流向
    return line + "\n" + ts
}

func renderAssistant(msg Message) string {
    var b strings.Builder
    b.WriteString(AssistantPrefix.Render("⏺") + " " + msg.Content + "\n")

    // 思考过程
    if msg.Thinking != "" {
        for _, line := range strings.Split(msg.Thinking, "\n") {
            b.WriteString(ThinkingPrefix.Render("  · ") + line + "\n")
        }
    }

    // 工具调用
    for _, tc := range msg.ToolCalls {
        b.WriteString(renderToolCall(tc))
    }

    b.WriteString(Timestamp.Render(msg.Time))
    return b.String()
}

func renderToolCall(tc ToolCall) string {
    prefix := toolPrefix(tc.Status) // ⏺ / ✓ / ✗ / ⏺
    line := fmt.Sprintf("  %s %s  %s  ·  %s  ·  %s",
        prefix, tc.Name, tc.Summary, tc.Status, tc.Duration)
    if tc.Expanded && tc.Diff != "" {
        for _, dl := range strings.Split(tc.Diff, "\n") {
            line += "\n" + DiffLine.Render("│  " + dl)
        }
    }
    return line
}
```

---

## 9. 性能设计

### 9.1 问题

对话轮数多（100+ 条消息）后，每次 `renderContent()` 全量重建导致滚动卡顿：

- `glamour.Render()` 在每条 assistant 消息上重复调用（Markdown 渲染是最大瓶颈）
- 每次 streaming chunk 都触发全量渲染（每秒几十次）
- 所有消息字符串拼接成一个大字符串传给 viewport

### 9.2 三层优化策略

#### 第一层：渲染缓存（核心）

```go
type MessageViewport struct {
    // 原有字段 ...
    renderedCache  []string   // 每条消息的渲染结果缓存
    dirtyFrom      int        // -1 = 全部干净，>=0 = 从该索引起需重渲染
    cachedContent  string     // 最终拼接后的完整内容缓存
    toolCardsCache string     // 工具卡片区缓存
    streamingCache string     // streaming 区缓存
}
```

**策略**：每条消息渲染一次后缓存，仅重新渲染变化的。

| 操作 | 脏标记 | 行为 |
|------|--------|------|
| 新增消息 | `dirtyFrom = -1` | `renderedCache` 追加一条，`cachedContent += newLine` |
| 更新工具卡片 | `dirtyFrom = 工具卡片起始索引` | 只重建后半部分 |
| Streaming chunk | 不走 `renderContent()` | 快速路径：只更新 `streamingCache` |
| 切换展开/折叠 | `dirtyFrom = 该消息索引` | 只重建该消息起的内容 |
| 切换主题 | `dirtyFrom = 0` | 全量重建（极少触发） |

```go
func (m *MessageViewport) renderContent() {
    if m.dirtyFrom < 0 && !m.StreamingActive {
        return // 全量缓存有效，跳过
    }

    if m.dirtyFrom >= 0 {
        // 增量重建：只从 dirtyFrom 开始重建
        m.renderedCache = m.renderedCache[:m.dirtyFrom]
        for i := m.dirtyFrom; i < len(m.Messages); i++ {
            m.renderedCache = append(m.renderedCache, m.renderMessage(i))
        }
        m.dirtyFrom = -1
    }

    m.toolCardsCache = m.renderToolCards()
    m.streamingCache = ""
    if m.StreamingActive {
        m.streamingCache = m.renderStreamingContent(m.StreamingBuffer.String())
    }

    var parts []string
    parts = append(parts, m.renderedCache...)
    if m.toolCardsCache != "" {
        parts = append(parts, m.toolCardsCache)
    }
    if m.streamingCache != "" {
        parts = append(parts, m.streamingCache)
    }
    m.cachedContent = strings.Join(parts, "\n")
    m.viewport.SetContent(m.cachedContent)
}
```

#### 第二层：Streaming 快速路径

Streaming 是最频繁的更新（每秒几十次），**必须绕过全量渲染**：

```go
func (m *MessageViewport) AddStreamingChunk(text string) {
    m.StreamingActive = true
    m.StreamingBuffer.WriteString(text)

    // 快速路径：只更新 streaming 部分，不触发 renderContent()
    m.streamingCache = m.renderStreamingContent(m.StreamingBuffer.String())
    content := m.cachedContent + "\n" + m.streamingCache
    m.viewport.SetContent(content)
    m.viewport.GotoBottom()
}
```

#### 第三层：消息数量上限

```go
const (
    MaxRenderMessages = 200  // 最多渲染最近 200 条
    MinRenderMessages = 50   // 最少保留 50 条
)
```

超出上限时，`renderedCache` 丢弃最旧的消息，在 viewport 顶部显示提示：

```
... 以上省略 42 条消息 ...
```

数据仍在 `Messages` 切片中，只是不渲染。

### 9.3 效果预估

| 场景 | 优化前 | 优化后 |
|------|--------|--------|
| 100 条消息，新增 1 条 | 重建 101 条 + glamour × 101 | 追加 1 条 + glamour × 1 |
| 100 条消息，streaming 1000 次 | 全量渲染 × 1000 | 只更新 streaming 区域 |
| 100 条消息，更新 1 个 tool card | 全量渲染全部 | 只重建 tool card 区域 |
| 500 条消息，滚动 | 渲染 500 条，必卡 | 只渲染最近 200 条 |
| 切换主题 | 全量渲染 | 全量渲染（极少触发，可接受） |

### 9.4 注意事项

- `glamour.Render()` 结果必须缓存，同一消息绝不重复渲染
- 窗口 resize 时设置 `dirtyFrom = 0`（宽度变化影响换行）
- 缓存内容不含 ANSI 转义序列干扰（lipgloss 自动处理）
- `viewport.SetContent()` 调用次数不影响性能，影响性能的是传入的字符串大小
```