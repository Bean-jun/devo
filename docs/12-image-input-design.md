# 图像理解（Base64 输入）接入设计文档

**版本**：1.0.0

**作者**：Devo Team

**状态**：方案设计

**适用范围**：backend LLM 客户端、消息结构、Web/TUI 前端、图像压缩、持久化

---

## 1. 背景与目标

### 1.1 当前 API 现状

消息发送接口为 `POST /api/v1/sessions/{id}/messages`（定义于 [handler.go](../internal/interfaces/rest/handler.go) 第 84 行），当前请求/响应结构如下：

**请求体**（[message_handler.go](../internal/interfaces/rest/message_handler.go) 第 14-16 行）：
```go
type postMessageRequest struct {
    Content string `json:"content"`
}
```

**响应体**（[message_handler.go](../internal/interfaces/rest/message_handler.go) 第 55-61 行）：
```go
type messageItem struct {
    ID         string             `json:"id"`
    Role       string             `json:"role"`
    Content    string             `json:"content"`
    CreatedAt  string             `json:"created_at"`
    ToolCalls  []session.ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string             `json:"tool_call_id,omitempty"`
}
```

**后端调用链**：`PostMessage` → `h.loop.ProcessMessage(ctx, id, content)` → 将 `content`（纯字符串）构建为 `session.Message` 存入消息列表。

**本次改动**：在 `postMessageRequest` 中新增 `images` 字段接收 Base64 图像数组，`messageItem` 新增 `content_parts` 字段返回多模态内容，`ProcessMessage` 签名扩展以支持图像。

### 1.2 API 改动汇总

| 接口 | 改动项 | 说明 |
|------|--------|------|
| `POST /api/v1/sessions/{id}/messages` | 请求体加 `images` 字段 | `{"content":"text", "images":["base64..."]}` |
| `GET /api/v1/sessions/{id}/messages` | 响应体加 `content_parts` 字段 | 返回多模态内容数组 |
| `GET /api/v1/sessions/{id}/events` (SSE) | 无改动 | 流式响应不涉及图像内容 |

### 1.3 什么是图像理解

基于 Qwen 系列视觉模型（Qwen3-VL、Qwen-VL 等），将图像以 Base64 编码形式传入大模型，模型能够理解图像内容并返回文字描述、分析、OCR 提取等结果。支持的功能包括：

- 图像内容描述与分类
- 文字提取与信息识别（OCR）
- 复杂视觉问题解答（数学、物理等题目）
- 文档/PDF 解析
- 物体定位（2D/3D 边界框）
- 根据视觉设计生成代码

### 1.4 当前代码现状

`internal/taskexec/llmclient/providers/openai/openai.go` 是 OpenAI 兼容协议的实现：

- `session.Message`（`core/session/session.go:54`）的 `Content` 字段是 `string` 类型，**不支持多模态内容数组**。
- `convertMessages` 函数直接将 `msg.Content` 作为字符串赋值给 `openaiMessage.Content`（`interface{}` 类型），**未处理图像内容**。
- 前端未提供图像上传/粘贴入口。
- 无图像压缩/预处理逻辑。

### 1.5 设计目标

1. **消息结构扩展**：`Message.Content` 支持多模态内容（文本 + 图像），兼容 OpenAI 兼容协议的 content 数组格式。
2. **Base64 编码**：前端将图像以 Base64 传递给后端，后端负责压缩后拼接 Data URL 格式传入 LLM。
3. **图像压缩**：后端对图像进行智能压缩（照片→JPG、截图/文字→PNG），长边限制 2048px，减少传输体积。
4. **前端支持**：Web 端支持粘贴/拖拽/选择图片，TUI 端酌情支持本地图片路径。
5. **持久化**：图像内容以 Base64 形式存入消息，支持回滚、归档、跨会话回放。
6. **兼容性**：纯文本消息继续正常工作，无破坏性变更。

---

## 2. API 协议说明

### 2.1 Qwen 视觉模型 OpenAI 兼容格式

根据文档，OpenAI 兼容接口的图像消息格式如下：

```json
{
  "model": "qwen3.7-plus",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/jpeg;base64,/9j/4AAQSkZJRg..."
          }
        },
        {
          "type": "text",
          "text": "描述这张图片中的内容"
        }
      ]
    }
  ]
}
```

### 2.2 图像限制（Qwen3-VL 系列）

| 限制项 | 值 |
|--------|-----|
| 单次请求最大图像数（Base64） | 250 张 |
| 原始文件最大 | 20MB |
| 编码后 Data URI 字符串最大 | 20MB |
| 支持格式 | BMP、JPEG、PNG、TIFF、WEBP、HEIC |
| 高分辨率模式 | `vl_high_resolution_images=true` 可启用 |

---

## 3. 影响范围分析

### 3.1 影响矩阵

| 层级 | 模块 | 影响等级 | 改动内容 |
|------|------|----------|----------|
| 核心层 | `core/session/session.go` | **高** | `Message.Content` 从 `string` 改为支持数组的 `ContentPart` 类型 |
| 协议层 | `llmclient/providers/openai/openai.go` | **高** | `convertMessages` 支持多模态 content 数组 |
| 新增 | `internal/taskexec/imageproc/` | **高** | 新增图像压缩/预处理模块 |
| 接口层 | `interfaces/rest/message_handler.go` | **中** | 接收请求中的 Base64 图像数据 |
| 接口层 | `interfaces/rest/sse_handler.go` | **低** | 仅转发事件，无逻辑变更 |
| 持久化 | `storage/sqlite/models.go` | **中** | `MessageModel.Content` 存储结构调整 |
| 持久化 | `storage/sqlite/store_session.go` | **中** | 读写时处理图像内容 |
| 归档 | `core/archive/` | **中** | Markdown 归档时图像的处理方式 |
| 压缩器 | `core/compressor/` | **低** | 图像 token 计入 context 预算 |
| 前端 | `web/src/types/message.ts` | **高** | `Message` 接口加 `images` 字段 |
| 前端 | `web/src/components/chat/InputArea.vue` | **高** | 加粘贴/拖拽/选择图片功能 |
| 前端 | `web/src/components/chat/MessageBubble.vue` | **中** | 展示用户消息中的图片缩略图 |
| 前端 | `web/src/stores/chat.ts` | **中** | 发送请求时携带图像数据 |
| TUI | `internal/interfaces/tui/components/inputarea.go` | **低** | 可选支持本地图片路径 |
| 测试 | `tests/` | **中** | 集成测试覆盖图像输入 |

### 3.2 关键风险

1. **Base64 体积膨胀**：Base64 编码使数据量增大约 33%，一张 15MB 的 PNG 编码后约 20MB，刚好触及 API 上限。必须在后端压缩。
2. **持久化膨胀**：图像 Base64 数据存入 SQLite 会极大增加数据库体积和查询开销。需要考虑是否单独存储或压缩存储。
3. **上下文 Token 预算**：图像会转换为大量 visual token（默认一张图约 1280-2560 tokens），需纳入上下文压缩的 token 预算计算。
4. **压缩影响 OCR 精度**：JPG 压缩会产生伪影，对文字识别场景不友好，需要区分场景选择压缩格式。
5. **前端内存**：大图 Base64 在前端内存中占用大量空间，可能影响前端性能。

---

## 4. 后端设计

### 4.1 消息结构扩展（`core/session/session.go`）

`Message.Content` 从 `string` 扩展为 `ContentPart` 数组：

```go
// ContentPart 表示消息内容的一部分，可以是文本或图像
type ContentPart struct {
    Type     string    `json:"type"`               // "text" 或 "image_url"
    Text     string    `json:"text,omitempty"`      // 文本内容
    ImageURL *ImageURL `json:"image_url,omitempty"` // 图像 URL（Base64 Data URL）
}

// ImageURL 表示图像 URL 信息
type ImageURL struct {
    URL    string `json:"url"`              // Base64 Data URL
    Detail string `json:"detail,omitempty"` // 可选：auto/low/high
}

// Message 结构体调整
type Message struct {
    ID         string        `json:"id"`
    Role       Role          `json:"role"`
    Content    string        `json:"content"`             // 向后兼容：纯文本快捷字段
    ContentParts []ContentPart `json:"content_parts,omitempty"` // 多模态内容（优先级高于 Content）
    Reasoning  string        `json:"reasoning,omitempty"`
    ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
    ToolCallID string        `json:"tool_call_id,omitempty"`
    CreatedAt  time.Time     `json:"created_at"`
}

// HasImages 判断消息是否包含图像
func (m *Message) HasImages() bool {
    for _, part := range m.ContentParts {
        if part.Type == "image_url" && part.ImageURL != nil {
            return true
        }
    }
    return false
}
```

**兼容性策略**：
- `Content` 字段保留，纯文本消息继续使用 `Content`
- `ContentParts` 非空时，优先使用 `ContentParts` 构建 OpenAI 请求
- `Content` 和 `ContentParts` 互斥，消息中只使用其中一个

### 4.2 协议层适配（`llmclient/providers/openai/openai.go`）

`convertMessages` 函数增加多模态内容处理：

```go
func convertMessages(messages []session.Message, systemPrompt string) []openaiMessage {
    openaiMsgs := make([]openaiMessage, 0, len(messages)+1)

    if systemPrompt != "" {
        openaiMsgs = append(openaiMsgs, openaiMessage{
            Role:    "system",
            Content: systemPrompt,
        })
    }

    for _, msg := range messages {
        om := openaiMessage{
            Role:       string(msg.Role),
            ToolCallID: msg.ToolCallID,
        }

        // 多模态内容（图像+文本）
        if len(msg.ContentParts) > 0 {
            parts := make([]openaiContentPart, 0, len(msg.ContentParts))
            for _, cp := range msg.ContentParts {
                switch cp.Type {
                case "text":
                    parts = append(parts, openaiContentPart{
                        Type: "text",
                        Text: cp.Text,
                    })
                case "image_url":
                    parts = append(parts, openaiContentPart{
                        Type: "image_url",
                        ImageURL: &openaiImageURL{
                            URL:    cp.ImageURL.URL,
                            Detail: cp.ImageURL.Detail,
                        },
                    })
                }
            }
            om.Content = parts
        } else if msg.Content != "" {
            om.Content = msg.Content
        }

        // ... tool_calls 处理保持不变
        openaiMsgs = append(openaiMsgs, om)
    }
    return openaiMsgs
}

// 新增：OpenAI 多模态内容部件类型
type openaiContentPart struct {
    Type     string          `json:"type"`
    Text     string          `json:"text,omitempty"`
    ImageURL *openaiImageURL `json:"image_url,omitempty"`
}

type openaiImageURL struct {
    URL    string `json:"url"`
    Detail string `json:"detail,omitempty"`
}
```

### 4.3 图像预处理模块（`internal/taskexec/imageproc/`）

新增 `internal/taskexec/imageproc/compressor.go`，负责图像压缩。

#### 4.3.1 压缩策略

```
输入: Base64 编码的原始图像
  │
  ├─ 1. 解码 Base64 → 图像数据
  │
  ├─ 2. 判断图像类型
  │     ├─ 照片/场景 → JPG(quality=85)
  │     └─ 截图/文字 → PNG
  │
  ├─ 3. 尺寸缩放
  │     ├─ 长边 > 2048px → 缩放至 2048px
  │     └─ 长边 ≤ 2048px → 保持原尺寸
  │
  ├─ 4. 小图跳过（< 500KB 原始数据）
  │
  └─ 5. 重新编码 Base64 → 返回 Data URL
```

#### 4.3.2 模块接口

```go
package imageproc

// CompressOptions 压缩选项
type CompressOptions struct {
    MaxLongSide  int    // 长边最大像素，默认 2048
    JPEGQuality  int    // JPG 质量 1-100，默认 85
    ForceJPEG    bool   // 强制转 JPG（忽略截图判断）
    SkipSmall    bool   // 跳过大图压缩，默认 true
    SmallThreshold int  // 小图阈值（字节），默认 500KB
}

// CompressResult 压缩结果
type CompressResult struct {
    DataURL      string // 压缩后的 Base64 Data URL（如 data:image/jpeg;base64,xxx）
    OriginalSize int    // 原始字节数
    CompressedSize int  // 压缩后字节数
    Format       string // 输出格式（jpeg/png）
    Width        int    // 输出宽度
    Height       int    // 输出高度
}

// Compress 压缩图像
func Compress(base64Data string, opts *CompressOptions) (*CompressResult, error)

// IsImage 判断 Base64 字符串是否为有效图像
func IsImage(base64Data string) bool
```

#### 4.3.3 截图 vs 照片判断逻辑

```go
// 判断依据：
// 1. 图像尺寸：截图通常宽高比接近屏幕比例（16:9、16:10、4:3）
// 2. 颜色分布：截图通常色块大、边缘锐利
// 3. 简化方案：默认全部转 JPG，用户可强制指定保持 PNG
```

**简化方案（推荐）**：第一版统一转 JPG（quality=85），因为：
- 视觉理解场景对画质不敏感
- JPG 压缩率远高于 PNG
- 除非用户明确需要 OCR 场景，否则 JPG 足够

### 4.4 请求处理流程

```
前端                                   后端                                    Qwen API
 │                                      │                                        │
 │  POST /api/v1/sessions/{id}/messages │                                        │
 │  {                                   │                                        │
 │    "content": "描述图片",              │                                        │
 │    "images": ["base64.."]            │                                        │
 │  }                                   │                                        │
 │ ───────────────────────────────────► │                                        │
 │                                      │  1. PostMessage 解析请求体              │
 │                                      │     postMessageRequest {               │
 │                                      │       Content string                   │
 │                                      │       Images  []string  // 新增         │
 │                                      │     }                                  │
 │                                      │  2. 每张图调用 imageproc.Compress()     │
 │                                      │  3. 构建 session.Message：              │
 │                                      │     Content: "描述图片"                  │
 │                                      │     ContentParts: [                     │
 │                                      │       {type:image_url, ...},            │
 │                                      │       {type:text, ...}                  │
 │                                      │     ]                                  │
 │                                      │  4. 调用 l.ProcessMessage(ctx,id,msg)   │
 │                                      │     (ProcessMessage 签名扩展)            │
 │                                      │  5. agentloop 存入 session.Messages     │
 │                                      │  6. 调用 LLM（convertMessages 构建数组） │
 │                                      │  ────────────────────────────────────► │
 │                                      │  ◄──────────────────────────────────── │
 │  ◄───────────────────────────────── │                                        │
 │  SSE streaming response              │                                        │
```

### 4.5 持久化

`MessageModel.Content` 字段存储策略：

- 纯文本消息：`Content` 字段存储文本，`ContentParts` 为空
- 带图像消息：`Content` 字段存储文本，`ContentParts` 以 JSON 形式存入新字段 `content_parts TEXT`
- 图像 Base64 数据存入 `content_parts` JSON 中

**存储膨胀风险**：如果图像 Base64 数据较大，建议：
- 短期方案：直接存入 SQLite（接受膨胀）
- 长期方案：图像数据单独存储到文件系统，数据库中只存文件路径引用

---

## 5. 前端设计

### 5.1 Web 端

#### 5.1.1 图像输入方式

- **粘贴**：Ctrl+V / Cmd+V 粘贴剪贴板中的图像
- **拖拽**：从文件管理器拖拽图片到输入区
- **选择**：点击输入区附件按钮，选择本地图片文件

#### 5.1.2 用户消息展示

用户消息气泡中展示图片缩略图（限制最大宽度），点击可查看原图。

#### 5.1.3 API 请求格式

```typescript
// POST /api/v1/sessions/{id}/messages
{
  "content": "描述这张图片",
  "images": ["base64_encoded_image_data"]  // 不含 data:xxx;base64, 前缀，后端自动拼接
}
```

#### 5.1.4 API 响应格式（GET /api/v1/sessions/{id}/messages）

```json
{
  "messages": [
    {
      "id": "msg-xxx",
      "role": "user",
      "content": "描述这张图片",
      "content_parts": [
        {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,xxx"}},
        {"type": "text", "text": "描述这张图片"}
      ],
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### 5.2 TUI 端

TUI 端（终端界面）受限于字符终端，图像输入能力有限：
- 支持通过 `--image` 参数传入本地图片路径
- 输入框中输入图片文件路径（如 `/path/to/image.png`）

---

## 6. 配置项

在 `config.LLMConfig` 中新增：

```go
type LLMConfig struct {
    // ... 现有字段

    // 图像相关
    VLHighResolutionImages bool   `json:"vl_high_resolution_images"` // 启用高分辨率图像模式
    ImageMaxLongSide       int    `json:"image_max_long_side"`       // 图像长边最大像素，默认 2048
    ImageJPEGQuality       int    `json:"image_jpeg_quality"`        // JPG 压缩质量，默认 85
}
```

---

## 7. 实施计划

### Phase 1：后端核心（消息结构 + 协议层 + 压缩）

1. 扩展 `session.Message` 支持 `ContentParts`（`core/session/session.go`）
2. 适配 `convertMessages` 构建多模态 content 数组（`llmclient/providers/openai/openai.go`）
3. 实现 `imageproc` 压缩模块（新文件 `internal/taskexec/imageproc/compressor.go`）
4. 修改 `message_handler.go`：
   - `postMessageRequest` 新增 `Images []string` 字段
   - `messageItem` 新增 `ContentParts` 字段
   - `PostMessage` 中调用 `imageproc.Compress()` 处理图像
5. 扩展 `ProcessMessage` 签名：`content string` → `msg session.Message`（`agentloop/loop.go`）

### Phase 2：持久化

1. SQLite `MessageModel` 增加 `content_parts` 字段
2. 读写适配

### Phase 3：前端

1. Web 端图像输入（粘贴/拖拽/选择）
2. Web 端消息展示（缩略图）
3. TUI 端图像文件路径输入

### Phase 4：测试

1. 图像压缩单元测试
2. 多模态消息转换测试
3. 端到端集成测试

---

## 8. 参考

- [Qwen 视觉模型文档](https://platform.qianwenai.com/docs)
- OpenAI 兼容 API 协议参考：`internal/taskexec/llmclient/providers/openai/openai.go`