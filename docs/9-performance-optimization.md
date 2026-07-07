# Devo 性能优化与对话导航面板方案

**版本**：1.0.0  
**状态**：方案设计  

---

## 一、问题概述

无论是 TUI 还是 Web 端，当对话轮数增多（超过 50 轮）后，页面出现明显卡顿。主要表现：

- 流式输出时打字速度变慢，甚至卡住
- 上下滚动时掉帧
- 面板展开/折叠响应迟钝
- 输入框输入时有延迟感

根本原因：**每次状态更新都对全量消息进行完整重绘/重渲染，消息量越大，性能衰减越严重。**

---

## 二、TUI 端性能分析与优化

### 2.1 当前问题

#### 问题 1：每次 SSE 事件都完整重绘所有消息

[messageview.go](../../internal/interfaces/tui/components/messageview.go) 中 `renderContent()` 方法在每次收到 SSE 事件（流式 token、工具调用、进度更新等）时都会重新遍历**所有历史消息**，逐一重新渲染，然后把整个内容字符串塞给 `viewport.SetContent()`。

**数据流**：

```
SSE event → AddStreamingChunk → renderContent() → 遍历所有消息 → lipgloss 样式计算 + glamour Markdown → viewport.SetContent(全量文本)
```

**影响**：流式输出时每毫秒级收到一个 token，每次都执行全量渲染 100+ 条消息，lipgloss 的 ANSI 样式计算和 glamour 的 Markdown 解析都是 CPU 密集型操作，累积延迟显著。

#### 问题 2：每 250ms 定时器触发强制全量刷新

[app.go](../../internal/interfaces/tui/app.go) 中每个 `TickMsg` 都会调用 `a.chatView.MessageView.Refresh()` → `renderContent()`，即使用户没有操作也全量重绘一次。spinner 动画只需要更新 spinner 那一行，完全不需要全量重绘所有消息。

```go
case messages.TickMsg:
    a.chatView.MessageView.SpinnerFrame++
    a.chatView.MessageView.Refresh()  // ← 每 250ms 全量重绘
```

#### 问题 3：glamour Markdown 渲染无缓存

[messageview.go](../../internal/interfaces/tui/components/messageview.go) 中每个 assistant 消息每次渲染都走 `mdRenderer.Render()`，即使消息内容从未改变，也重新解析渲染一遍 Markdown。

```go
func (m *MessageViewport) renderAssistantMessage(msg types.Message) string {
    // 每次调用都重新渲染，无缓存
    if m.mdRenderer != nil {
        rendered, err := m.mdRenderer.Render(msg.Content)
        ...
    }
}
```

#### 问题 4：上下文压缩后前端未清理消息

[handlers_sse.go](../../internal/interfaces/tui/handlers_sse.go) 收到 `context_compressed` 后只追加了一条系统通知，然后调用 `loadMessagesCmd` 重载，但 TUI 端 `MessageViewport.Messages` 数组仍然保留所有历史消息在内存中，并没有真正移除被压缩的消息，文本量持续增长。

#### 问题 5：`SetSize` 每次重建 glamour TermRenderer

[messageview.go](../../internal/interfaces/tui/components/messageview.go) 中 `SetSize` 每次调用都重新创建 `glamour.TermRenderer`，窗口大小调整时频繁创建开销大。

---

### 2.2 优化方案

#### 方案 A：增量渲染 + 缓存（核心优化）

在 `MessageViewport` 中维护一个渲染缓存：

```go
type MessageViewport struct {
    // ... 现有字段
    renderedCache map[string]string  // msgID → 渲染后的文本
    dirty         bool               // 标记是否需要全量重绘
}
```

**逻辑**：
- 新增消息时，只渲染新消息并追加到缓存，不触及已有缓存
- 流式更新时，只重新渲染最后一条（流式消息），不重绘历史消息
- 折叠/展开工具卡片时，只重新渲染工具卡片区域
- 仅当 `dirty = true`（如窗口大小变化）时才全量重绘
- `renderContent()` 直接从缓存拼接内容，不再逐个渲染

**效果**：流式输出时从 O(n) 降低到 O(1)，n 为消息总数。

#### 方案 B：定时器只更新 spinner 帧

将 spinner 帧的更新从全量刷新中分离出来：
- 定时器回调只更新 `SpinnerFrame` 计数器
- 渲染时，如果 `SpinnerFrame` 变化但内容没变，只在 viewport 中做局部的 spinner 字符替换（或直接跳过，因为概率上差异很小）
- 或者使用独立的 bubbletea spinner 组件，不耦合到 MessageViewport

#### 方案 C：glamour 渲染结果缓存

为每个 assistant 消息缓存 Markdown 渲染结果：

```go
type messageCache struct {
    content     string
    rendered    string
}
var assistantCache map[string]messageCache
```

- 渲染前先查缓存，内容相同直接返回
- 仅在消息内容变化时重新渲染

#### 方案 D：上下文压缩后同步清理前端消息

收到 `context_compressed` 事件后：
- 重新从 API 拉取消息列表
- 用新列表**替换**（不是追加）`MessageViewport.Messages`
- 清空渲染缓存，触发一次全量重绘
- 真实减少内存中的消息数量

#### 方案 E：glamour TermRenderer 复用

`SetSize` 时检查当前 `mdRenderer` 的配置，仅在宽度变化时才重建，避免每次都创建新实例。

---

### 2.3 TUI 优化优先级

| 优先级 | 方案 | 预期收益 | 复杂度 |
|--------|------|---------|--------|
| P0 | 方案 A：增量渲染 + 缓存 | 流式输出卡顿基本消除 | 中 |
| P0 | 方案 C：Markdown 渲染缓存 | 历史消息渲染零开销 | 低 |
| P1 | 方案 B：定时器优化 | 空闲时 CPU 占用降低 | 低 |
| P1 | 方案 D：压缩后清理前端 | 长对话内存稳定 | 低 |
| P2 | 方案 E：Renderer 复用 | 窗口调整更流畅 | 低 |

---

## 三、Web 端性能分析与优化

### 3.1 当前问题

#### 问题 1：流式 token 更新触发全量重新计算

[MessageList.vue](../../web/src/components/chat/MessageList.vue) 中每收到一个 `streaming_token`，`chatStore.streamingContent` 更新 → `watch` 触发 → 整个列表重新计算 `groupedMessages` → 所有 `MessageBubble` 组件重新渲染。

```
streaming_token → chatStore.streamingContent 更新 → 
  watch 触发 → groupedMessages 重新计算 → 
  所有 MessageBubble 重新渲染 → marked.parse + highlight.js 重新执行
```

**影响**：Vue 的响应式系统虽然高效，但大列表（100+ 条消息）下，每个 computed 重新计算、每个组件重新 diff 的开销累积，尤其是流式输出时每秒几十次更新。

#### 问题 2：每个 assistant 消息每次重新计算 Markdown

[MessageBubble.vue](../../web/src/components/chat/MessageBubble.vue) 中 `renderedContent` 是 `computed` 属性，每次父组件重新渲染，所有已渲染消息的 `marked.parse` + `highlight.js` 语法高亮都会重新执行一遍。

```ts
const renderedContent = computed(() => {
  if (props.message.role === 'assistant') {
    return renderMarkdown(props.message.content)  // ← 每次重新执行
  }
  return props.message.content
})
```

**影响**：`highlight.js` 的语法高亮是 CPU 密集型操作，尤其在代码块很多时，每条消息重新高亮一次，100 条消息就是 100 次高亮计算。

#### 问题 3：缺少虚拟滚动，所有消息都在 DOM 树中

即使消息完全滚出可视区域，仍然保留完整 DOM 节点。百轮对话后，加上工具调用卡片、diff 展示、代码块高亮，DOM 节点数很容易过千，滚动时浏览器需要持续重排整个列表。

**影响**：滚动性能差，内存占用高，layout/reflow 开销大。

#### 问题 4：每次新增消息都触发自动滚动

[MessageList.vue](../../web/src/components/chat/MessageList.vue) 中每次 `chatStore.messages.length` 变化都 `scrollToBottom`，强迫浏览器计算最新滚动高度并重绘。

```ts
watch(
  () => [chatStore.messages.length, chatStore.streamingContent],
  () => {
    nextTick(() => {
      requestAnimationFrame(() => {
        props.scrollToBottom(false)  // ← 每次消息变化都触发
      })
    })
  },
  { deep: false }
)
```

#### 问题 5：groupedMessages 每次重新计算

`groupedMessages` 的 computed 将连续 tool 消息分组，但每次消息变化都重新遍历整个数组，即使只是新增了一条消息。

---

### 3.2 优化方案

#### 方案 A：虚拟滚动（Virtual Scroll）

**核心思路**：只渲染可视区域内的消息，滚动时动态挂载/卸载 DOM 节点。

**技术选型**：
- 推荐使用 `vue-virtual-scroller`（Vue 3 官方推荐）
- 或自研简单实现，按消息高度预估总高度

**实现要点**：
- 每条消息估算高度（用户消息 ≈ 60px，assistant 消息 ≈ 200px，工具卡片 ≈ 80px）
- 只渲染可视区域 ± 缓冲区（上下各 3-5 条）的消息
- 滚动时动态更新渲染范围
- 兼容 `scrollToBottom` 和 `scrollToMessage` 功能

**效果**：DOM 节点数从 O(n) 降低到 O(可视区域大小)，无论多少轮对话，DOM 节点数恒定。

#### 方案 B：Markdown 渲染缓存

**核心思路**：缓存 `content → renderedHTML` 映射，内容不变直接返回缓存。

**实现方案**：

```ts
// 在 chatStore 或独立模块中维护
const markdownCache = new Map<string, string>()

export function renderMarkdownCached(content: string): string {
  if (markdownCache.has(content)) {
    return markdownCache.get(content)!
  }
  const result = renderMarkdown(content)
  markdownCache.set(content, result)
  return result
}
```

**注意事项**：
- 缓存上限：使用 LRU 策略，最多缓存 500 条
- 流式内容不停变化，流式输出期间不缓存（或使用 debounce 缓存）

#### 方案 C：Memo 优化 + 减少不必要重渲染

**核心思路**：利用 Vue 3 的 `v-memo` 或 `shallowRef` 减少子组件重渲染。

**实现要点**：
- `MessageBubble` 组件使用 `v-memo="[message.content, message.role]"` 缓存
- 仅当消息内容或角色变化时才重新渲染
- `ToolCallCard` 同样使用 memo，仅当 `toolCall.status` 变化时重新渲染

```html
<MessageBubble
  v-for="msg in visibleMessages"
  :key="msg.id"
  :message="msg"
  v-memo="[msg.content, msg.role]"
/>
```

#### 方案 D：流式内容独立渲染

**核心思路**：将 `streamingContent` 从 `messages` 数组中分离出来，作为独立的最后一条渲染。

**当前问题**：`streamingContent` 变化时，`chatStore.messages` 数组本身没变，但 `watch` 仍然触发全量重新计算。

**优化方案**：
- `MessageList` 中最后一条不渲染 `messages` 中的最后一条，而是渲染 `streamingContent`（如果存在）
- `streamingContent` 变化只触发这一个节点的更新，不触发全列表重算
- `finishStreaming` 时，将流式内容追加到 `messages` 数组末尾，触发一次正常的列表更新

#### 方案 E：groupedMessages 增量计算

**核心思路**：不在每次更新时完全重新计算分组，而是增量更新。

**实现方案**：
- 维护 `groupedMessages` 的缓存
- 仅当新消息是 `tool` 类型时，检查是否需要修改分组
- 其他情况直接追加到结果数组末尾

#### 方案 F：自动滚动优化

**核心思路**：使用 `requestAnimationFrame` 的 debounce 和条件判断。

**优化方案**：
- 仅在用户处于底部（`isNearBottom()`）时才自动滚动
- 流式输出时使用 `throttle`（每 100ms 最多滚动一次），而不是每次 token 都滚动
- 使用 `scrollTop` 直接设置而非 `scrollTo`，减少动画开销

---

### 3.3 Web 端优化优先级

| 优先级 | 方案 | 预期收益 | 复杂度 |
|--------|------|---------|--------|
| P0 | 方案 A：虚拟滚动 | 滚动性能质的飞跃，DOM 恒定 | 高 |
| P0 | 方案 B：Markdown 渲染缓存 | 消除重复计算，CPU 大幅降低 | 低 |
| P0 | 方案 D：流式内容独立渲染 | 消除流式输出时的全量重算 | 中 |
| P1 | 方案 C：Memo 优化 | 减少非必要重渲染 | 低 |
| P1 | 方案 F：自动滚动优化 | 流式输出滚动更流畅 | 低 |
| P2 | 方案 E：增量分组计算 | 长列表分组计算更快 | 中 |

---

## 四、新功能：右侧悬浮对话导航面板

### 4.1 功能描述

在 Web 端聊天界面右侧添加一个**悬浮对话导航面板**，用于快速浏览和定位对话中的关键消息。

**核心交互**：
- 用户每发送一条消息，面板中自动添加一个条目（显示消息摘要）
- 点击面板中的某一条目，消息列表自动滚动到该条消息的位置
- 当前查看的消息在面板中高亮显示
- 面板可折叠/展开，不影响主聊天区域

### 4.2 组件设计

#### 组件树

```
ChatPanel (现有)
├── MessageList (现有)
│   └── MessageBubble (现有)
│       └── [data-message-id]  ← 新增锚点标记
│
└── FloatingNavPanel (新增)          ← 右侧悬浮面板
    ├── NavHeader                    ← 面板标题 + 折叠按钮
    ├── NavList (可滚动)             ← 条目标列表
    │   ├── NavItem (用户消息)       ← 选中高亮
    │   ├── NavItem (用户消息)
    │   └── NavItem (用户消息)
    └── NavFooter                    ← 底部提示
```

#### 数据类型

```ts
interface NavItem {
  id: string          // 对应消息的 message.id
  index: number       // 在 messages 数组中的位置
  summary: string     // 消息摘要（截取前 40 个字符）
  timestamp: string   // 消息时间
  isActive: boolean   // 是否当前选中
}
```

#### 组件位置与样式

```
┌──────────────────────────────────────────────┐
│  StatusBar                                    │
├─────────────────────┬────────────────────────┤
│                     │  ┌──────────────────┐  │
│   主聊天区域        │  │ 对话导航面板      │  │
│   (MessageList)     │  │                  │  │
│                     │  │ ● 帮我写一个...   │  │
│   [用户消息 1] ◄────┼──│ ● 分析这段代码... │  │
│   [助手回复 1]      │  │ ○ 运行测试...     │  │
│                     │  │ ○ 修复 bug...     │  │
│   [用户消息 2] ◄────┼──│                  │  │
│   [助手回复 2]      │  │                  │  │
│                     │  └──────────────────┘  │
│   [用户消息 3] ◄────┼──                      │
│   ...               │                        │
│                     │                        │
├─────────────────────┴────────────────────────┤
│  InputArea                                    │
└──────────────────────────────────────────────┘
```

**样式规格**：
- 面板宽度：约 200px（可拖拽调整）
- 位置：右侧悬浮，半透明背景 + 模糊效果
- 默认折叠为小图标，hover 展开
- 面板内列表可滚动，与主列表独立

### 4.3 核心实现逻辑

#### 4.3.1 条目生成

```ts
// 在 chatStore 中新增 computed
const navItems = computed<NavItem[]>(() => {
  return chatStore.messages
    .filter(msg => msg.role === 'user')
    .map((msg, idx) => ({
      id: msg.id,
      index: idx,
      summary: msg.content.slice(0, 40) + (msg.content.length > 40 ? '...' : ''),
      timestamp: msg.timestamp,
      isActive: false,
    }))
})
```

#### 4.3.2 点击滚动定位

```ts
function scrollToMessage(messageId: string): void {
  // 1. 在消息列表中找到对应 DOM 元素
  const targetEl = document.querySelector(`[data-message-id="${messageId}"]`)
  if (!targetEl) return

  // 2. 滚动到目标位置
  targetEl.scrollIntoView({ behavior: 'smooth', block: 'center' })

  // 3. 高亮闪烁效果
  targetEl.classList.add('message-highlight')
  setTimeout(() => targetEl.classList.remove('message-highlight'), 2000)

  // 4. 更新活跃条目
  setActiveNavItem(messageId)
}
```

#### 4.3.3 滚动联动（反向同步）

当用户手动滚动消息列表时，根据当前可视区域中最靠近顶部的用户消息，更新面板中的高亮条目：

```ts
function onMessageListScroll(): void {
  const container = containerRef.value
  if (!container) return

  const userMessages = container.querySelectorAll('[data-message-id][data-role="user"]')
  let closestId = ''

  userMessages.forEach(el => {
    const rect = el.getBoundingClientRect()
    const containerRect = container.getBoundingClientRect()
    if (rect.top >= containerRect.top && rect.top <= containerRect.top + containerRect.height / 2) {
      closestId = el.getAttribute('data-message-id') || ''
    }
  })

  if (closestId) {
    setActiveNavItem(closestId)
  }
}
```

#### 4.3.4 面板折叠/展开

- 默认状态：折叠为 40px 宽的竖条，显示图标
- hover 或点击展开：显示完整面板（200px 宽）
- 点击聊天区域或面板外部：自动折叠
- 面板内点击条目后：保持展开 2 秒后自动折叠

### 4.4 需要改动的文件

| 文件 | 改动内容 |
|------|---------|
| `web/src/components/chat/FloatingNavPanel.vue` | **新增**：右侧悬浮导航面板组件 |
| `web/src/components/chat/MessageList.vue` | 每条消息添加 `data-message-id` 和 `data-role` 属性 |
| `web/src/components/chat/ChatPanel.vue` | 引入 `FloatingNavPanel`，管理面板展开/折叠状态 |
| `web/src/stores/chat.ts` | 新增 `navItems` computed、`activeNavItemId` 状态 |
| `web/src/composables/useAutoScroll.ts` | 新增 `scrollToMessage` 方法 |
| `web/src/utils/constants.ts` | 新增导航面板相关常量（宽度、延迟等） |

### 4.5 组件详细设计

#### FloatingNavPanel.vue 结构

```html
<template>
  <div class="floating-nav" :class="{ expanded: isExpanded }" @mouseenter="expand" @mouseleave="startCollapseTimer">
    <!-- 折叠状态：只显示图标 -->
    <div v-if="!isExpanded" class="nav-collapsed">
      <span class="nav-icon">📑</span>
      <span class="nav-count">{{ navItems.length }}</span>
    </div>

    <!-- 展开状态：显示完整面板 -->
    <template v-else>
      <div class="nav-header">
        <span class="nav-title">对话导航</span>
        <button class="nav-collapse-btn" @click="collapse">✕</button>
      </div>

      <div class="nav-list">
        <div
          v-for="item in navItems"
          :key="item.id"
          class="nav-item"
          :class="{ active: item.id === activeNavItemId }"
          @click="scrollToMessage(item.id)"
        >
          <span class="nav-item-index">{{ item.index + 1 }}</span>
          <span class="nav-item-summary">{{ item.summary }}</span>
        </div>
      </div>

      <div class="nav-footer">
        <span>共 {{ navItems.length }} 条消息</span>
      </div>
    </template>
  </div>
</template>
```

#### 样式规范

```css
.floating-nav {
  position: fixed;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 100;
  transition: width var(--transition-normal) ease;
  background: rgba(30, 30, 30, 0.85);
  backdrop-filter: blur(12px);
  border: 1px solid var(--color-border);
  border-right: none;
  border-radius: var(--radius-lg) 0 0 var(--radius-lg);
  overflow: hidden;
}

.floating-nav.expanded {
  width: 220px;
}

.floating-nav:not(.expanded) {
  width: 44px;
}

.nav-list {
  max-height: 400px;
  overflow-y: auto;
  padding: var(--space-sm);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs) var(--space-sm);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.nav-item:hover {
  background: var(--color-bg-hover);
}

.nav-item.active {
  background: var(--color-accent);
  color: var(--color-text-inverse);
}

.nav-item-summary {
  font-size: var(--font-size-xs);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
```

### 4.6 与虚拟滚动的兼容性

如果消息列表使用了虚拟滚动，`scrollToMessage` 需要调整：

1. **确保目标消息在渲染范围内**：调用虚拟滚动组件提供的 `scrollToItem(index)` 方法
2. **锚点标记**：即使消息未渲染，也需要预留占位高度，让浏览器能正确计算滚动位置
3. **高亮闪烁**：等待滚动完成 + 消息渲染后，再添加高亮效果

```ts
async function scrollToMessage(index: number): Promise<void> {
  // 1. 虚拟滚动：滚动到指定索引
  await virtualScroller.scrollToItem(index)

  // 2. 等待渲染完成
  await nextTick()
  await new Promise(resolve => requestAnimationFrame(resolve))

  // 3. 高亮闪烁
  const targetEl = document.querySelector(`[data-message-index="${index}"]`)
  if (targetEl) {
    targetEl.classList.add('message-highlight')
    setTimeout(() => targetEl.classList.remove('message-highlight'), 2000)
  }
}
```

---

## 五、实施计划

### Phase 1：Web 端核心性能优化（P0）

| 编号 | 任务 | 内容 | 预估 | 依赖 |
|------|------|------|------|------|
| PERF-01 | Markdown 渲染缓存 | 在 `chatStore` 中实现 `content → renderedHTML` 缓存 | 0.5h | — |
| PERF-02 | 流式内容独立渲染 | 将 `streamingContent` 从消息列表计算中分离 | 1h | PERF-01 |
| PERF-03 | MessageBubble memo 优化 | 添加 `v-memo` 减少非必要重渲染 | 0.5h | PERF-02 |
| PERF-04 | 自动滚动优化 | 流式输出时 throttle 滚动 | 0.5h | — |

### Phase 2：Web 端虚拟滚动（P0）

| 编号 | 任务 | 内容 | 预估 | 依赖 |
|------|------|------|------|------|
| PERF-05 | 虚拟滚动引入 | 调研 `vue-virtual-scroller` 或自研实现 | 1h | — |
| PERF-06 | 虚拟滚动集成 | 将 `MessageList` 改造为虚拟滚动列表 | 3h | PERF-05 |
| PERF-07 | 虚拟滚动测试 | 验证滚动定位、自动滚动、高亮等交互 | 1h | PERF-06 |

### Phase 3：右侧悬浮对话导航面板（新功能）

| 编号 | 任务 | 内容 | 预估 | 依赖 |
|------|------|------|------|------|
| FEAT-01 | chatStore 扩展 | 新增 `navItems` computed、`activeNavItemId` | 0.5h | — |
| FEAT-02 | FloatingNavPanel 组件 | 创建悬浮面板组件，实现折叠/展开/条目渲染 | 2h | FEAT-01 |
| FEAT-03 | 消息锚点标记 | MessageList 中每条消息添加 `data-message-id` | 0.5h | — |
| FEAT-04 | 滚动联动 | 点击条目 → 滚动到消息；手动滚动 → 更新高亮 | 1.5h | FEAT-02, FEAT-03 |
| FEAT-05 | ChatPanel 集成 | 引入 FloatingNavPanel，管理展开/折叠状态 | 1h | FEAT-02 |
| FEAT-06 | 面板样式与动画 | 完善 CSS 样式、hover 效果、过渡动画 | 1h | FEAT-05 |
| FEAT-07 | 虚拟滚动兼容 | 确保导航面板与虚拟滚动协同工作 | 1h | PERF-06, FEAT-04 |
| FEAT-08 | 组件测试 | FloatingNavPanel 单元测试 + 交互测试 | 1h | FEAT-06 |

### Phase 4：TUI 端性能优化（P0/P1）

| 编号 | 任务 | 内容 | 预估 | 依赖 |
|------|------|------|------|------|
| TUI-PERF-01 | MessageView 渲染缓存 | 实现增量渲染 + 消息缓存 | 2h | — |
| TUI-PERF-02 | Markdown 渲染缓存 | 为 assistant 消息缓存 glamour 渲染结果 | 0.5h | TUI-PERF-01 |
| TUI-PERF-03 | 定时器优化 | 分离 spinner 帧更新与全量刷新 | 0.5h | — |
| TUI-PERF-04 | 压缩后清理前端 | 压缩后重新加载消息列表替换 | 0.5h | — |
| TUI-PERF-05 | 集成测试 | 验证压缩后消息量减少、流式输出不卡顿 | 1h | TUI-PERF-01~04 |

### 总预估

| Phase | 内容 | 预估 |
|-------|------|------|
| Phase 1 | Web 端核心优化 | 2.5h |
| Phase 2 | Web 端虚拟滚动 | 5h |
| Phase 3 | 悬浮导航面板 | 9h |
| Phase 4 | TUI 端优化 | 4.5h |
| **总计** | | **21h** |

---

## 六、预期效果

| 指标 | 优化前 | 优化后 |
|------|--------|--------|
| 流式输出延迟 | 随消息数线性增长，100 轮后 > 200ms/token | 恒定 ~5ms/token |
| DOM 节点数（Web） | 100 轮 ≈ 2000+ 节点 | 恒定 ~200 节点 |
| 滚动帧率（Web） | 100 轮后 < 30fps | 恒定 60fps |
| TUI 渲染 CPU 占用 | 空闲时 10-15%（250ms 全量刷新） | 空闲时 < 1% |
| 消息导航定位 | 手动滚动查找 | 一键跳转 |
| 内存占用（Web） | 100 轮 ≈ 50MB+ | 100 轮 ≈ 15MB |

---

## 七、风险与注意事项

1. **虚拟滚动与消息高度**：消息高度不固定（代码块、diff 等），需要动态测量或合理估算，否则会导致滚动位置跳动
2. **缓存失效**：消息内容变化时必须正确清除缓存，否则会显示过期内容
3. **流式输出与虚拟滚动**：流式输出时消息高度持续变化，需要特殊处理滚动位置
4. **导航面板与主列表同步**：确保双向联动正确，不会出现死循环
5. **TUI 增量渲染**：lipgloss 样式计算后可能会改变字符串长度，需要精确计算缓存偏移量
6. **灰度发布**：建议先在开发环境充分测试，再逐步推广到用户