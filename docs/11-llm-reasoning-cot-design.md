# 大模型思维链（Reasoning / CoT）接入设计文档

**版本**：1.1.0

**作者**：Devo Team

**状态**：后端已实施，前端待开发

**适用范围**：backend LLM 客户端、agent loop、SSE 协议、Web/TUI 前端、SQLite 持久化

---

## 1. 背景与目标

### 1.1 什么是思维链（Chain of Thought, CoT）

大模型在回答问题前，先把推理过程显式输出（"先思考、再作答"）。目前业界有三种实现形态：

| 形态 | 说明 | 代表模型 |
|------|------|----------|
| Prompted CoT | 通过提示词触发，推理过程混在 `content` 里作为普通 token 输出 | GPT-4o、Claude 3.5 默认 |
| Trained CoT / Reasoning Models | 模型被训练成先思考再答，思考过程作为独立 token 流出现 | OpenAI o1/o3、DeepSeek-R1、Qwen3-thinking、Kimi K1 |
| API 暴露的 reasoning 字段 | 推理过程通过协议字段显式分离，与最终 `content` 分开流式返回 | OpenAI `reasoning`、DeepSeek `reasoning_content`、Anthropic `thinking` blocks |

### 1.2 当前代码现状

`internal/taskexec/llmclient/providers/openai/openai.go` 是 OpenAI 兼容协议的实现：

- 请求体 `openaiChatRequest`（第 433 行）只有 `model/messages/tools/stream/tool_choice`，**没有 `reasoning_effort` 之类的采样参数**。
- 流式 delta 结构 `openaiStreamDelta`（第 409 行）只解析 `content` 和 `tool_calls`，**`reasoning_content` / `reasoning` 字段被直接丢弃**。
- 非流式响应 `openaiRespMessage`（第 477 行）同样只取 `content` 和 `tool_calls`。
- `StreamEvent`（`llmclient/client.go:19`）和 `EventBus` 事件类型（`streaming_token`）也没有 reasoning 通道。
- `Message` 结构体（`core/session/session.go:54`）没有 `Reasoning` 字段，思考过程无法持久化。
- `MessageModel`（`storage/sqlite/models.go:40`）没有 `Reasoning` 列。
- 前端 `SSEEventType`（`web/src/types/sse.ts`）和 `Message` 接口（`web/src/types/message.ts`）都没有 reasoning 概念。

### 1.3 设计目标

1. **协议层**：OpenAI 兼容协议上接收 `reasoning_content`（DeepSeek 风格）和 `reasoning`（OpenAI o1 风格）两种字段，统一为内部 `Reasoning` 概念。
2. **流式层**：思考过程独立流式输出，不混入 `streaming_token`，前端能区分展示。
3. **持久化层**：思考过程作为 `Message.Reasoning` 字段持久化，支持回滚、归档、跨会话回放。
4. **配置层**：用户可在配置中选择是否启用思考、思考强度（`reasoning_effort`：low/medium/high）。
5. **前端层**：Web 和 TUI 都要展示思考过程，默认折叠，可展开查看。
6. **兼容性**：旧模型（不输出 reasoning）继续工作，无破坏性变更。

---

## 2. 影响范围分析

### 2.1 影响矩阵

| 层级 | 模块 | 影响等级 | 改动内容 |
|------|------|----------|----------|
| 协议层 | `llmclient/providers/openai` | **高** | 请求体加 `reasoning_effort`，响应 delta 加 reasoning 字段 |
| 客户端接口 | `llmclient/client.go` | **高** | `StreamEvent` 加 `Reasoning` 字段，新增 `reasoning_token` 事件类型 |
| 核心层 | `core/agentloop/state_handlers.go` | **中** | thinkingHandler 处理 reasoning token，发布新事件 |
| 核心层 | `core/session/session.go` | **中** | `Message` 加 `Reasoning` 字段 |
| 持久化 | `storage/sqlite/models.go` | **中** | `MessageModel` 加 `Reasoning` 列，需要 schema migration |
| 持久化 | `storage/sqlite/store_session.go` | **中** | 读写时处理 Reasoning 字段 |
| 接口层 | `interfaces/rest/sse_handler.go` | **低** | 仅转发事件，无逻辑变更 |
| 接口层 | `interfaces/rest/message_handler.go` | **低** | GET messages 返回时带上 reasoning 字段 |
| 接口层 | `interfaces/rest/session_handler.go` | **低** | 流式 token 转发逻辑可能需要区分 |
| 配置层 | `internal/config/config.go` | **低** | 加 `reasoning_effort`、`enable_reasoning` 配置项 |
| 配置层 | `internal/config/defaults.go` | **低** | 默认值 |
| 归档 | `core/archive/` | **中** | Markdown 归档需要区分思考与正文 |
| 压缩器 | `core/compressor/` | **低** | 思考内容是否计入 token 预算需要决策 |
| 前端 | `web/src/types/sse.ts` | **中** | 新增 `reasoning_token`、`reasoning_complete` 事件 |
| 前端 | `web/src/types/message.ts` | **中** | `Message` 加 `reasoning` 字段 |
| 前端 | `web/src/stores/chat.ts` | **高** | 加 `streamingReasoning` 状态，区分流式思考与正文 |
| 前端 | `web/src/components/chat/ThinkingIndicator.vue` | **高** | 重新设计为思考与正文双区显示 |
| 前端 | `web/src/components/chat/MessageBubble.vue` | **中** | 历史消息展示思考过程（折叠） |
| 前端 | `web/src/components/chat/VirtualMessageItem.vue` | **低** | 透传 reasoning |
| 前端 | `web/src/composables/useSSE.ts` | **低** | 仅注册新事件类型 |
| 前端 | `web/src/composables/useSession.ts` | **低** | 处理新事件回调 |
| 前端 | `web/src/test/fixtures/messages.ts` | **低** | 测试夹具更新 |
| TUI | `internal/interfaces/tui/handlers_sse.go` | **中** | 处理 `reasoning_token` 事件 |
| TUI | `internal/interfaces/tui/components/` | **中** | 思考区 UI 组件 |
| VSCode 扩展 | `vscode-extension/` | **低** | 仅是 webview 容器，自动继承 |
| Electron | `electron/` | **低** | 同上 |
| 测试 | `tests/`（独立 module） | **中** | 集成测试覆盖 reasoning 流式 |

### 2.2 关键风险

1. **Token 计费与预算**：reasoning token 在 OpenAI o1 上是计费的，但在 `usage` 里通常以 `completion_tokens_details.reasoning_tokens` 单独列出。当前 `tokenmeter` 没有这个维度，需要扩展。否则用户看到的 token 用量会"对不上"。
2. **上下文压缩**：reasoning 内容是否进入下一轮的 `messages` 上下文？DeepSeek-R1 推荐把 `reasoning_content` 喂回去（提升多轮连贯性），OpenAI o1 不推荐（会被截断）。这是个**关键策略决策**。
3. **持久化膨胀**：思考过程动辄几千 token，全部入库会让 `messages` 表和归档 Markdown 体积膨胀 3-5 倍。需要考虑是否默认持久化、是否压缩存储。
4. **回放一致性**：回滚到某条消息后，重新发送，新一轮思考可能不同。需要明确 reasoning 是否参与回滚。
5. **协议差异**：DeepSeek、OpenAI o1、Anthropic 的字段名和流式行为都不一样，统一抽象时容易遗漏边界情况（如 OpenAI o1 流式只给 summary 不给完整 reasoning）。
6. **前端性能**：思考过程高频流式（每秒几十 token），如果直接绑定到 `v-html` 上会有渲染压力，需要复用现有虚拟滚动和节流逻辑。

---

## 3. 后端设计

### 3.1 协议层扩展（`llmclient/providers/openai`）

#### 3.1.1 请求体增加 reasoning 控制

```go
type openaiChatRequest struct {
    Model          string                `json:"model"`
    Messages       []openaiMessage       `json:"messages"`
    Tools          []openaiToolDef       `json:"tools,omitempty"`
    ToolChoice     string                `json:"tool_choice,omitempty"`
    Stream         bool                  `json:"stream,omitempty"`
    StreamOptions  *openaiStreamOptions  `json:"stream_options,omitempty"`
    
    // 新增：reasoning 控制
    ReasoningEffort string `json:"reasoning_effort,omitempty"` // "low" | "medium" | "high"
    // 注：Anthropic 风格的 thinking 用 thinking={type:"enabled",budget_tokens:N} 表达
    // OpenAI o1 不接受 reasoning_effort 通过 chat completions 传，需要走 /responses 端点
    // 这里采用最通用的字段，由 provider 内部转换
}
```

**Provider 适配**：未来加 Anthropic、豆包等 provider 时，各自负责把统一的 `ReasoningEffort` 转换为协议对应字段。

#### 3.1.2 响应 delta 增加 reasoning 字段

```go
type openaiStreamDelta struct {
    Content          string           `json:"content,omitempty"`
    ReasoningContent string           `json:"reasoning_content,omitempty"` // DeepSeek-R1 风格
    Reasoning        string           `json:"reasoning,omitempty"`         // OpenAI o1 风格（部分版本）
    ToolCalls        []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiRespMessage struct {
    Role             string           `json:"role"`
    Content          string           `json:"content"`
    ReasoningContent string           `json:"reasoning_content,omitempty"`
    Reasoning        string           `json:"reasoning,omitempty"`
    ToolCalls        []openaiToolCall `json:"tool_calls,omitempty"`
}
```

解析时统一合并：

```go
func extractReasoning(delta openaiStreamDelta) string {
    if delta.ReasoningContent != "" {
        return delta.ReasoningContent
    }
    return delta.Reasoning
}
```

#### 3.1.3 非流式响应同样处理

`openaiChoice.Message` 也需要加 `ReasoningContent` / `Reasoning` 字段并在 `Complete` 中转换到 `CompleteResult.Reasoning`。

### 3.2 客户端接口扩展（`llmclient/client.go`）

```go
type CompleteResult struct {
    Text       string                 `json:"text"`
    Reasoning  string                 `json:"reasoning,omitempty"`  // 新增
    ToolCalls  []session.ToolCall     `json:"tool_calls"`
    TokenUsage *tokenmeter.TokenUsage `json:"token_usage,omitempty"`
}

type StreamEvent struct {
    Type         string                 `json:"type"`
    Token        string                 `json:"token,omitempty"`
    Reasoning    string                 `json:"reasoning,omitempty"`     // 新增：思考增量
    FullText     string                 `json:"full_text,omitempty"`
    FullReasoning string               `json:"full_reasoning,omitempty"` // 新增：累计思考
    ToolCalls    []session.ToolCall     `json:"tool_calls,omitempty"`
    FinishReason string                 `json:"finish_reason,omitempty"`
    TokenUsage   *tokenmeter.TokenUsage `json:"token_usage,omitempty"`
    Err          error                  `json:"-"`
}
```

**事件类型扩展**：在 `parseSSEStream` 中新增 `"reasoning_token"` 类型，与 `"token"` 并行下发。

```go
if reasoningChunk := extractReasoning(delta); reasoningChunk != "" {
    fullReasoningBuilder.WriteString(reasoningChunk)
    callback(llmclient.StreamEvent{
        Type:          "reasoning_token",
        Reasoning:      reasoningChunk,
        FullReasoning: fullReasoningBuilder.String(),
    })
}
```

### 3.3 Token 计量扩展（`tokenmeter`）

`tokenmeter.TokenUsage` 当前只有 `InputTokens` / `OutputTokens` / `TotalTokens` / `CachedTokens`。需要加：

```go
type TokenUsage struct {
    InputTokens      int
    OutputTokens     int
    ReasoningTokens  int   // 新增：思考 token，部分 provider 单独计费
    TotalTokens      int
    CachedTokens     int
    Source           Source
}
```

OpenAI o1 在 `usage.completion_tokens_details.reasoning_tokens` 给出，DeepSeek-R1 把思考 token 算在 `completion_tokens` 里不单独列。Provider 适配时按各自协议填。

### 3.4 Agent Loop 集成（`core/agentloop/state_handlers.go`）

`thinkingHandler` 当前只处理 `token` 和 `done`，需要扩展：

```go
func (l *Loop) thinkingHandler(ctx context.Context, lc *LoopContext) (LoopState, error) {
    lc.EventBus.Publish("loop.thinking_started", nil)

    err := l.llmClient.CompleteStream(ctx, lc.ActiveMsgs, lc.DynamicPrompt, func(evt llmclient.StreamEvent) {
        switch evt.Type {
        case "reasoning_token":
            lc.EventBus.Publish("reasoning_token", map[string]any{
                "token": evt.Reasoning,
            })
            // 累计思考到 LoopContext，供 done 时使用
            lc.ReasoningBuilder.WriteString(evt.Reasoning)
        case "token":
            lc.EventBus.Publish("streaming_token", map[string]any{
                "token": evt.Token,
            })
        case "done":
            lc.LLMResult = &llmclient.CompleteResult{
                Text:       evt.FullText,
                Reasoning:  evt.FullReasoning,  // 新增
                ToolCalls:  evt.ToolCalls,
                TokenUsage: evt.TokenUsage,
            }
            // ... 原有逻辑
            lc.EventBus.Publish("reasoning_complete", map[string]any{
                "full_reasoning": evt.FullReasoning,
            })
            lc.EventBus.Publish("streaming_complete", map[string]any{
                "tool_calls":    evt.ToolCalls,
                "finish_reason": evt.FinishReason,
            })
        case "error":
            streamErr = evt.Err
        }
    })
    // ...
}
```

`LoopContext` 新增字段：

```go
type LoopContext struct {
    // ... 既有字段
    ReasoningBuilder strings.Builder
}
```

### 3.5 Session 模型扩展（`core/session/session.go`）

```go
type Message struct {
    ID         string     `json:"id"`
    Role       Role       `json:"role"`
    Content    string     `json:"content"`
    Reasoning  string     `json:"reasoning,omitempty"`  // 新增：assistant 消息的思考过程
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
    CreatedAt  time.Time  `json:"created_at"`
}
```

### 3.6 持久化扩展（`storage/sqlite`）

#### 3.6.1 Schema 迁移

```go
type MessageModel struct {
    ID            string `gorm:"primaryKey;size:64"`
    SessionID     string `gorm:"size:64;index:idx_msg_session_id;not null"`
    Role          string `gorm:"size:16"`
    Content       string `gorm:"type:text"`
    Reasoning     string `gorm:"type:text"`  // 新增
    ToolCallsJSON string `gorm:"type:text"`
    ToolCallID    string `gorm:"size:64"`
    Seq           int    `gorm:"index:idx_msg_seq"`
    CreatedAt     time.Time
}
```

GORM 自动迁移会加列，不需要手写 migration。`fromDomain` / `ToDomain` 双向都要带上 Reasoning。

#### 3.6.2 归档 Markdown

```markdown
## Assistant · 2026-07-28 12:34:56

<details>
<summary>💭 思考过程</summary>

[reasoning content here]

</details>

[content here]
```

折叠块默认收起，避免回放时刷屏。

### 3.7 配置层扩展（`internal/config`）

```go
type Config struct {
    // ... 既有字段
    EnableReasoning   bool   `json:"enable_reasoning"`    // 默认 false
    ReasoningEffort   string `json:"reasoning_effort"`    // "low" | "medium" | "high"，默认 "medium"
    PersistReasoning  bool   `json:"persist_reasoning"`   // 是否持久化，默认 true
    FeedReasoningBack bool   `json:"feed_reasoning_back"` // 多轮对话是否把 reasoning 喂回去，默认 false
}
```

环境变量：

```
DEVO_LLM_ENABLE_REASONING=true
DEVO_LLM_REASONING_EFFORT=medium
DEVO_LLM_PERSIST_REASONING=true
DEVO_LLM_FEED_REASONING_BACK=false
```

### 3.8 REST API 扩展

#### 3.8.1 新增 SSE 事件类型

```typescript
// 新增
'reasoning_token'     // { token: string } 思考增量
'reasoning_complete'  // { full_reasoning: string } 思考结束
```

#### 3.8.2 GET /sessions/{id}/messages 响应

```json
{
  "messages": [
    {
      "id": "msg-...",
      "role": "assistant",
      "content": "最终回答内容",
      "reasoning": "思考过程全文...",
      "created_at": "2026-07-28T12:34:56Z"
    }
  ]
}
```

#### 3.8.3 POST /sessions/{id}/config

允许运行时切换 `reasoning_effort`：

```json
{ "reasoning_effort": "high" }
```

### 3.9 上下文压缩策略

`core/compressor` 需要决策：

- **方案 A（推荐）**：reasoning 不计入上下文 token 预算，压缩时直接丢弃 reasoning 字段。
- **方案 B**：reasoning 计入预算，旧消息的 reasoning 参与压缩摘要。
- **方案 C**：用户配置 `feed_reasoning_back=true` 时，把上一轮 reasoning 作为 system 消息插入，计入预算。

默认采用 A，简单且 token 消耗可预测。

---

## 4. 前端设计

### 4.1 类型扩展

#### 4.1.1 `web/src/types/sse.ts`

```typescript
export type SSEEventType =
  | 'thinking'
  | 'reasoning_token'      // 新增
  | 'reasoning_complete'    // 新增
  | 'streaming_token'
  | 'streaming_complete'
  // ... 既有类型

export interface SSEEventData {
  content?: string
  token?: string
  reasoning?: string         // 新增：思考增量 token
  fullReasoning?: string     // 新增：思考完整文本
  messageId?: string
  // ... 既有字段
}
```

#### 4.1.2 `web/src/types/message.ts`

```typescript
export interface Message {
  id: string
  sessionId: string
  role: MessageRole
  content: string
  reasoning?: string         // 新增
  timestamp: string
  tokenUsage?: TokenUsage
  toolCall?: ToolCall
}
```

### 4.2 Chat Store 扩展（`web/src/stores/chat.ts`）

新增流式思考状态：

```typescript
export const useChatStore = defineStore('chat', () => {
  const messages = ref<Message[]>([])
  const isStreaming = ref(false)
  const streamingContent = ref('')
  const streamingReasoning = ref('')        // 新增
  const streamingMessageId = ref<string | null>(null)
  const isReasoningActive = ref(false)      // 新增：当前是否处于思考阶段
  
  function appendReasoningChunk(content: string): void {
    streamingReasoning.value += content
  }
  
  function startReasoning(): void {
    isReasoningActive.value = true
    streamingReasoning.value = ''
  }
  
  function finishReasoning(): void {
    isReasoningActive.value = false
  }
  
  function finishStreaming(tokenUsage?, reasoning?: string): void {
    if (streamingContent.value) {
      const msg: Message = {
        id: streamingMessageId.value ?? generateId(),
        sessionId: '',
        role: 'assistant',
        content: streamingContent.value,
        reasoning: reasoning || streamingReasoning.value || undefined,  // 新增
        timestamp: new Date().toISOString(),
        tokenUsage,
      }
      messages.value.push(msg)
    }
    isStreaming.value = false
    isReasoningActive.value = false
    streamingContent.value = ''
    streamingReasoning.value = ''
    streamingMessageId.value = null
  }
  
  // ... 既有方法
})
```

### 4.3 SSE 事件路由（`web/src/composables/useSession.ts`）

```typescript
useSSE().onEvent('reasoning_token', (data) => {
  chatStore.appendReasoningChunk(data.token)
})

useSSE().onEvent('reasoning_complete', (data) => {
  chatStore.finishReasoning()
})

useSSE().onEvent('streaming_token', (data) => {
  if (!chatStore.isReasoningActive && !chatStore.isStreaming) {
    chatStore.startStreaming()
  }
  chatStore.appendStreamChunk(data.token)
})

useSSE().onEvent('message_complete', (data) => {
  chatStore.finishStreaming(
    data.input_tokens && data.output_tokens ? { input: data.input_tokens, output: data.output_tokens } : undefined,
    data.full_reasoning
  )
})
```

**关键**：reasoning 阶段 `isStreaming` 应保持 false（不要把思考当正文流式），完成思考后第一个 `streaming_token` 才触发 `startStreaming`。

### 4.4 UI 组件改造

#### 4.4.1 `ThinkingIndicator.vue` 重新设计

当前 `ThinkingIndicator` 只显示 `streamingContent`，需要扩展为双区结构：

```vue
<template>
  <div class="thinking-indicator">
    <!-- 思考过程区（reasoning 阶段 + 折叠历史） -->
    <div v-if="hasReasoning" class="reasoning-section">
      <div class="reasoning-header" @click="toggleReasoning">
        <span class="reasoning-icon">💭</span>
        <span class="reasoning-title">
          {{ isReasoningActive ? '正在思考...' : '思考过程' }}
        </span>
        <span v-if="isReasoningActive" class="thinking-dots">...</span>
        <span class="toggle-icon">{{ reasoningExpanded ? '▼' : '▶' }}</span>
      </div>
      <div v-show="reasoningExpanded" class="reasoning-content">
        <pre>{{ chatStore.streamingReasoning }}</pre>
      </div>
    </div>
    
    <!-- 正文流式区（content 阶段） -->
    <div class="streaming-bubble">
      <div class="bubble-header">
        <span class="bubble-role">Devo</span>
      </div>
      <div v-if="hasContent" class="bubble-content">
        <pre class="streaming-text">{{ chatStore.streamingContent }}<span class="cursor-blink">|</span></pre>
      </div>
      <div v-else-if="!isReasoningActive" class="bubble-content bubble-empty">
        <span class="empty-hint">正在思考...</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useChatStore } from '@/stores/chat'

const chatStore = useChatStore()
const reasoningExpanded = ref(false)  // 默认折叠

const hasReasoning = computed(() => chatStore.streamingReasoning.length > 0)
const hasContent = computed(() => chatStore.streamingContent.length > 0)
const isReasoningActive = computed(() => chatStore.isReasoningActive)

function toggleReasoning() {
  reasoningExpanded.value = !reasoningExpanded.value
}
</script>
```

**交互设计**：
- reasoning 阶段默认折叠，仅显示"正在思考..."加 spinner。
- 用户可点击展开实时查看思考。
- 切换到 content 阶段后，reasoning 区自动收起为可点击的"查看思考过程"摘要。

#### 4.4.2 `MessageBubble.vue` 历史消息展示

```vue
<template>
  <div class="message-bubble">
    <details v-if="message.reasoning" class="reasoning-collapse">
      <summary>💭 思考过程</summary>
      <pre class="reasoning-text">{{ message.reasoning }}</pre>
    </details>
    <div class="content">{{ message.content }}</div>
  </div>
</template>
```

用原生 `<details>` 标签，无 JS 依赖，默认折叠。

#### 4.4.3 样式建议

```css
.reasoning-section {
  margin-bottom: var(--space-sm);
  border: 1px dashed var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
  opacity: 0.85;
}

.reasoning-content {
  padding: var(--space-sm) var(--space-md);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  font-style: italic;
  max-height: 300px;
  overflow-y: auto;
}

.reasoning-text {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-mono);
}
```

### 4.5 测试夹具更新（`web/src/test/fixtures/messages.ts`）

```typescript
export const mockMessages: Message[] = [
  {
    id: 'msg-1',
    sessionId: 'session-1',
    role: 'user',
    content: '帮我分析这段代码',
    timestamp: '2026-07-28T12:00:00Z',
  },
  {
    id: 'msg-2',
    sessionId: 'session-1',
    role: 'assistant',
    content: '这段代码的功能是...',
    reasoning: '用户问的是代码分析，我需要先理解代码结构，再逐行解读...',
    timestamp: '2026-07-28T12:00:05Z',
  },
]
```

### 4.6 Mobile 适配

`MobileInputBar.vue` 不涉及，但消息列表组件需要确保 reasoning 折叠区在窄屏下也能正常滚动，建议给 `reasoning-content` 加 `max-height: 40vh`。

### 4.7 TUI 适配（`internal/interfaces/tui/`）

`handlers_sse.go` 增加 case：

```go
case "reasoning_token":
    if text, ok := msg.Data["token"].(string); ok {
        a.chatView.MessageView.AddReasoningChunk(text)
        a.statusBar.SetActivity("思考中: " + truncate(text, 40))
    }
case "reasoning_complete":
    a.chatView.MessageView.FinalizeReasoning()
```

`components/MessageView` 增加 reasoning 区，用浅色显示在正文上方。

### 4.8 VSCode 扩展与 Electron

仅是 webview 容器，自动继承前端改动，**无需独立适配**。但需要确认：
- VSCode webview 的 CSP 是否允许 `<details>` 标签 - 允许。
- Electron 的 IPC 通道是否需要透传 reasoning - 否，前端直连 SSE。

---

## 5. 兼容性策略

### 5.1 旧模型回退

| 模型 | reasoning 字段 | 行为 |
|------|----------------|------|
| GPT-4o | 不返回 | `Reasoning` 为空字符串，前端不渲染思考区 |
| DeepSeek-R1 | `reasoning_content` | 正常流式 |
| OpenAI o1 | `reasoning`（流式可能只有 summary） | 部分支持，按实际字段填充 |
| Claude 3.5（无 extended thinking） | 不返回 | 同 GPT-4o |
| Claude 4.x（开 extended thinking） | `thinking` blocks | 需 Anthropic provider 适配，本期不实现 |

### 5.2 数据迁移

- `MessageModel` 加列后，旧消息的 `Reasoning` 字段为空字符串，符合 `omitempty` 行为，前端不渲染。
- 不需要 backfill，不需要 schema 版本号。
- 回滚兼容：旧版本读取新数据库时，GORM 会忽略未知列，无破坏性。

### 5.3 配置默认值

- `enable_reasoning = false`（默认关，老用户无感知）
- `reasoning_effort = "medium"`
- `persist_reasoning = true`
- `feed_reasoning_back = false`（保守，避免上下文膨胀）

---

## 6. 实施计划

### 6.1 分阶段落地

| 阶段 | 内容 | 验收标准 |
|------|------|----------|
| P0 | 协议层 + 客户端接口扩展 | 接 DeepSeek-R1 能流式输出 reasoning，单元测试覆盖 |
| P1 | Agent loop + EventBus 集成 | SSE 能下发 `reasoning_token` 事件，浏览器能看到 |
| P2 | 持久化 + 配置 | `Message.Reasoning` 入库，重启后历史消息有思考过程 |
| P3 | Web 前端展示 | 折叠/展开交互完整，流式渲染无卡顿 |
| P4 | TUI 适配 | TUI 也能展示思考过程 |
| P5 | 配置项与文档 | 用户可配置 effort、是否启用，文档同步 |

### 6.2 测试矩阵

| 测试项 | 类型 | 关注点 |
|--------|------|--------|
| `openai_test.go` | 单元 | 解析 reasoning_content/reasoning 字段 |
| `state_handlers_test.go` | 单元 | reasoning_token 事件正确发布 |
| `store_session_test.go` | 集成 | Reasoning 字段持久化与读取 |
| `useSSE.test.ts` | 单元 | reasoning_token 事件路由 |
| `chat.test.ts` | 单元 | streamingReasoning 状态机 |
| `ThinkingIndicator.test.ts` | 单元 | 折叠交互、双区渲染 |
| E2E：DeepSeek-R1 完整对话 | 集成 | 真实模型流式 thinking + content |
| E2E：GPT-4o 回退 | 集成 | 无 reasoning 时不渲染思考区 |
| 性能：千 token reasoning 流式 | 性能 | 帧率不低于 30fps |

---

## 7. 决策结论

以下决策已确认，本期实施按此执行：

1. **本期范围**：只做后端，前端改造留到下一期。后端 SSE 事件先发布出去，前端暂时只把 reasoning 当普通 token 处理（旧客户端能"看到"思考但不区分展示），下期再做前端 UI 适配。
2. **reasoning 参与回滚**：按消息 ID 切除时，连同该消息的 reasoning 字段一起删除，不需要特殊处理。
3. **思考过程计入 token 用量**：`TokenUsage` 加 `ReasoningTokens` 维度，OpenAI o1 从 `usage.completion_tokens_details.reasoning_tokens` 读取，DeepSeek-R1 不单独列则填 0（已包含在 `OutputTokens` 中）。本期后端记录即可，Dashboard 拆分维度是前端工作，留到下期。
4. **归档到 Markdown**：默认开启，使用 `<details>` 折叠块，避免回放刷屏。用户可通过 `persist_reasoning` 配置关闭。
5. **不支持 Anthropic thinking blocks**：本期只做 OpenAI 兼容协议（覆盖 OpenAI o1 / DeepSeek-R1 / Qwen3-thinking / Kimi K1 等）。Anthropic provider 接入时再扩展。
6. **压缩策略采用方案 A**：reasoning 不计入上下文 token 预算，压缩时直接丢弃 reasoning 字段。
7. **`feed_reasoning_back` 默认 false**：多轮对话默认不把上轮 reasoning 喂回去，避免上下文膨胀。
8. **`reasoning_effort` 运行时切换**：通过 `POST /sessions/{id}/config` 即时生效，下一轮 thinking 应用新值，不清理历史消息。

### 7.1 本期实施情况

**已完成**（本期后端实施）：

- `internal/taskexec/llmclient/providers/openai/openai.go`：请求体加 `reasoning_effort`，delta 结构加 `reasoning_content` / `reasoning` 字段，`parseSSEStream` 发出 `reasoning_token` 事件，`Complete` 填充 `CompleteResult.Reasoning`，`convertUsage` 解析 `completion_tokens_details.reasoning_tokens`
- `internal/taskexec/llmclient/client.go`：`StreamEvent` 加 `Reasoning` / `FullReasoning` 字段，`CompleteResult` 加 `Reasoning` 字段
- `internal/core/tokenmeter/tokenmeter.go`：`TokenUsage` 加 `ReasoningTokens` 维度
- `internal/core/session/session.go`：`Message` 加 `Reasoning` 字段，`omitempty` 保证旧消息无破坏
- `internal/core/agentloop/loop_context.go`：`LoopContext` 加 `ReasoningBuilder strings.Builder`
- `internal/core/agentloop/state_handlers.go`：`thinkingHandler` 处理 `reasoning_token` 事件并发布到 EventBus，结束时发布 `reasoning_complete`；`token_usage` 事件附带 `reasoning_tokens`；`toolExecutingHandler` 与 `textResponseHandler` 持久化 Message 时带上 Reasoning；`message_complete` 事件附带 `full_reasoning`
- `internal/storage/sqlite/models.go`：`MessageModel` 加 `Reasoning` 列，`fromDomain`/`ToDomain` 双向处理（GORM AutoMigrate 自动加列）
- `internal/core/archive/archive.go`：新增 `AppendAssistantMessageWithReasoning` 方法，归档时用 `<details><summary>💭 思考过程</summary>` 折叠块；`renderArchive` 在 SyncArchive 时同样渲染思考；旧 `AppendAssistantMessage` 保持向后兼容
- `internal/config/config.go`：`LLMConfig` 加 `EnableReasoning` 与 `ReasoningEffort` 字段，环境变量 `DEVO_LLM_ENABLE_REASONING` 与 `DEVO_LLM_REASONING_EFFORT` 支持，`Merge` 合并项目级配置
- `internal/config/defaults.go`：加 `DefaultReasoningEffort = "medium"`
- `internal/taskexec/llmclient/providers/factory.go`：根据 `EnableReasoning` 决定是否下发 `ReasoningEffort` 给 OpenAI 客户端
- 测试覆盖：openai provider 解析（`reasoning_test.go`）、agent loop reasoning 事件发布（`reasoning_test.go`）、SQLite 持久化（`store_message_reasoning_test.go`）、归档 Markdown 渲染（`archive_reasoning_test.go`），全部通过

**SSE 新增事件类型**：

- `reasoning_token`：`{ token: string }` 思考增量
- `reasoning_complete`：`{ full_reasoning: string }` 思考结束（仅在存在思考内容时发布）

**message_complete 事件扩展字段**：

- `full_reasoning: string`（如有）
- `reasoning_tokens: int`（如有，> 0 时才附带）

**token_usage 事件扩展字段**：

- `reasoning_tokens: int`（如有，> 0 时才附带）

**REST API 变更**：

- `GET /sessions/{id}/messages` 响应中 assistant 消息带 `reasoning` 字段（`omitempty`，旧消息无此字段）

**配置示例**：

```json
{
  "llm": {
    "api_key": "sk-...",
    "base_url": "https://api.deepseek.com/v1",
    "model": "deepseek-reasoner",
    "enable_reasoning": true,
    "reasoning_effort": "medium"
  }
}
```

或环境变量：

```bash
export DEVO_LLM_ENABLE_REASONING=true
export DEVO_LLM_REASONING_EFFORT=medium
```

**未实施**（下一期前端工作）：

- Web 前端 UI 改造（ThinkingIndicator 重设计、MessageBubble 折叠展示）
- TUI 适配（reasoning_token 事件处理与展示）
- Dashboard 拆分"思考 token"维度
- Anthropic provider 接入（thinking blocks）

---

## 8. 相关文档

- [2-architecture.md](./2-architecture.md) - 整体架构
- [3-frontend-design.md](./3-frontend-design.md) - 前端设计
- [4-web-architecture.md](./4-web-architecture.md) - Web 架构
- [6-agent-loop-event-driven-refactor.md](./6-agent-loop-event-driven-refactor.md) - Agent Loop 事件驱动
- [9-performance-optimization.md](./9-performance-optimization.md) - 性能优化（流式渲染节流）
