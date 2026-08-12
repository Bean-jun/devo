# Devo LLM 配置引导与多模型管理设计方案

**版本**：1.1.0（2026-08-12 修订）

**定位**：将现有的"只读提示弹窗"升级为"可交互的引导配置页 + 多模型管理面板"，用户无需离开界面即可完成 LLM 配置，并支持配置多个模型、在聊天中通过命令切换。配置渠道支持 Web UI 和 CLI 两种方式。

---

## 一、现状分析

### 1.1 当前痛点

| 痛点 | 现状 |
| :--- | :--- |
| 首次配置 | 弹窗提示"去手动创建 JSON 文件"，用户需离开界面操作文件系统 |
| 配置方式 | 只能手动编辑 `~/.devo/config.json` 或 `.devo/config.json` 或设环境变量 |
| 模型数量 | 仅支持单一模型，无法切换 |
| 配置入口 | 无 UI 管理入口，配置后只能通过文件修改 |

### 1.2 现有相关代码

| 层级 | 文件 | 说明 |
| :--- | :--- | :--- |
| 后端-配置结构 | `internal/config/config.go` | `LLMConfig` 结构体，单模型 |
| 后端-配置加载 | `internal/config/config.go` → `LoadFullConfig()` | 全局 + 项目 + 环境变量合并 |
| 后端-配置状态 | `cmd/devo/app.go:L191` | `SetLLMConfigured(a.cfg.LLM.APIKey != "")` |
| 后端-配置接口 | `internal/interfaces/rest/handler.go:L128` | `GET /api/v1/config/status` |
| 后端-LLM 客户端 | `internal/taskexec/llmclient/providers/factory.go` | `NewClient()` 按单一 cfg 创建 |
| 前端-弹窗 | `web/src/components/modal/ConfigWarningDialog.vue` | 只读提示弹窗 |
| 前端-检测 | `web/src/AppController.ts:L90-L97` | `onMounted` 中 fetch 配置状态 |
| 前端-设置面板 | `web/src/panels/settings/SettingsPanel.vue` | 项目/全局设置，无模型管理 |
| 前端-UI Store | `web/src/stores/ui.ts` | `ModalType` 含 `'config-warning'` |

---

## 二、总体设计目标

```
┌─────────────────────────────────────────────────────────────┐
│                      用户旅程                                │
│                                                             │
│  首次启动 ──→ 引导页 ──→ 填写 API Key ──→ 保存 ──→ 开始聊天  │
│                  │                                          │
│                  └──→ 跳过 ──→ 稍后在设置中配置               │
│                                                             │
│  已配置用户 ──→ 设置面板 ──→ 添加/编辑/删除多个模型           │
│                  │                                          │
│                  └──→ 聊天界面 ──→ 输入 /model 切换当前模型   │
│                                                             │
│  CLI 用户 ──→ devo config models ──→ 命令行管理模型配置      │
└─────────────────────────────────────────────────────────────┘
```

### 核心原则

1. **零文件操作**：用户全程在 UI 中完成配置，不触碰文件系统
2. **向后兼容**：已有单模型配置自动迁移为多模型格式
3. **渐进式**：引导页可跳过，后续在设置面板中补配
4. **最小改动**：复用现有架构，不引入新的框架或范式

---

## 三、数据模型设计

### 3.1 后端 Go 结构体

```go
// === 新增：多模型配置 ===

// ModelConfig 单个模型配置
type ModelConfig struct {
    ID              string            `json:"id"`                        // 唯一标识，如 "openai-gpt4o"
    Name            string            `json:"name"`                      // 显示名称，如 "GPT-4o"
    Provider        string            `json:"provider,omitempty"`        // 提供商标识，如 "openai"（预留扩展）
    APIKey          string            `json:"api_key"`                   // API 密钥
    BaseURL         string            `json:"base_url"`                  // API 地址
    Model           string            `json:"model"`                     // 模型名称
    ExtraHeaders    map[string]string `json:"extra_headers,omitempty"`   // 额外请求头
    EnableReasoning bool              `json:"enable_reasoning,omitempty"`// 是否启用推理
    ReasoningEffort string            `json:"reasoning_effort,omitempty"`// 推理强度
    MaxTokens       int               `json:"max_tokens,omitempty"`      // 最大输出 token
}

// LLMConfig 升级为多模型
type LLMConfig struct {
    // 兼容旧字段（deprecated，保留用于迁移和向后兼容）
    APIKey          string            `json:"api_key,omitempty"`
    BaseURL         string            `json:"base_url,omitempty"`
    Model           string            `json:"model,omitempty"`
    ExtraHeaders    map[string]string `json:"extra_headers,omitempty"`
    EnableReasoning bool              `json:"enable_reasoning,omitempty"`
    ReasoningEffort string            `json:"reasoning_effort,omitempty"`
    MaxTokens       int               `json:"max_tokens,omitempty"`

    // 新增：多模型支持
    Models         []ModelConfig `json:"models,omitempty"`          // 模型列表
    ActiveModelID  string        `json:"active_model_id,omitempty"` // 当前激活的模型 ID
}
```

### 3.2 前端 TypeScript 类型

```typescript
// web/src/types/llm.ts（新增文件）

export interface ModelConfig {
  id: string
  name: string
  provider?: string
  api_key: string
  base_url: string
  model: string
  extra_headers?: Record<string, string>
  enable_reasoning?: boolean
  reasoning_effort?: string
  max_tokens?: number
}

export interface LLMConfigStatus {
  llm_configured: boolean
  active_model_id?: string
  models: ModelConfig[]
}

export interface OnboardRequest {
  name: string
  api_key: string
  base_url?: string
  model?: string
  enable_reasoning?: boolean
  reasoning_effort?: string
  max_tokens?: number
}
```

### 3.3 配置文件格式

```json
// ~/.devo/config.json 新格式
{
  "llm": {
    "models": [
      {
        "id": "openai-gpt4o",
        "name": "GPT-4o",
        "provider": "openai",
        "api_key": "sk-xxx",
        "base_url": "https://api.openai.com/v1",
        "model": "gpt-4o",
        "enable_reasoning": false,
        "max_tokens": 128000
      },
      {
        "id": "openai-gpt4o-mini",
        "name": "GPT-4o Mini",
        "provider": "openai",
        "api_key": "sk-xxx",
        "base_url": "https://api.openai.com/v1",
        "model": "gpt-4o-mini",
        "max_tokens": 128000
      }
    ],
    "active_model_id": "openai-gpt4o"
  }
}
```

---

## 四、后端改动

### 4.1 配置加载与迁移

**文件**：`internal/config/config.go`

在 `LoadGlobal()` 中增加迁移逻辑：

```go
func LoadGlobal() (*Config, error) {
    cfg := &Config{}
    // ... 现有读取逻辑 ...

    applyEnvOverrides(cfg)
    applyDefaults(cfg)
    migrateLegacyConfig(cfg)   // 新增：旧格式 → 新格式迁移
    return cfg, nil
}

// migrateLegacyConfig 将旧的单模型配置迁移到多模型格式
func migrateLegacyConfig(cfg *Config) {
    // 如果已有 models 数组，跳过迁移
    if len(cfg.LLM.Models) > 0 {
        return
    }
    // 如果旧字段有 api_key，自动创建默认模型条目
    if cfg.LLM.APIKey != "" {
        cfg.LLM.Models = []ModelConfig{{
            ID:              "default",
            Name:            cfg.LLM.Model,
            APIKey:          cfg.LLM.APIKey,
            BaseURL:         cfg.LLM.BaseURL,
            Model:           cfg.LLM.Model,
            ExtraHeaders:    cfg.LLM.ExtraHeaders,
            EnableReasoning: cfg.LLM.EnableReasoning,
            ReasoningEffort: cfg.LLM.ReasoningEffort,
            MaxTokens:       cfg.LLM.MaxTokens,
        }}
        cfg.LLM.ActiveModelID = "default"
    }
}
```

### 4.2 新增 API 端点

**文件**：`internal/interfaces/rest/handler.go`（路由注册）
**文件**：`internal/interfaces/rest/global_config_handler.go`（handler 实现）

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/config/status` | **增强**：返回 `llm_configured` + `models` + `active_model_id` |
| `POST` | `/api/v1/global/config/onboard` | **新增**：引导页提交初始配置 |
| `GET` | `/api/v1/global/config/models` | **新增**：获取所有模型列表 |
| `POST` | `/api/v1/global/config/models` | **新增**：添加模型 |
| `PUT` | `/api/v1/global/config/models/{id}` | **新增**：更新模型 |
| `DELETE` | `/api/v1/global/config/models/{id}` | **新增**：删除模型 |
| `PUT` | `/api/v1/global/config/models/{id}/activate` | **新增**：激活模型 |
| `POST` | `/api/v1/global/config/models/{id}/test` | **新增**：测试模型连接 |

#### 4.2.1 `GET /api/v1/config/status` 增强

```go
// 响应
{
  "llm_configured": true,
  "active_model_id": "openai-gpt4o",
  "models": [
    {
      "id": "openai-gpt4o",
      "name": "GPT-4o",
      "api_key": "sk-***",    // 脱敏显示
      "base_url": "https://api.openai.com/v1",
      "model": "gpt-4o"
    }
  ]
}
```

> 注意：`api_key` 返回时做脱敏处理，只显示前 4 位和后 4 位，中间用 `***` 替代。

#### 4.2.2 `POST /api/v1/global/config/onboard`

引导页专用，接收用户首次配置：

```go
type onboardRequest struct {
    Name            string `json:"name"`
    APIKey          string `json:"api_key"`
    BaseURL         string `json:"base_url,omitempty"`
    Model           string `json:"model,omitempty"`
    EnableReasoning bool   `json:"enable_reasoning,omitempty"`
    ReasoningEffort string `json:"reasoning_effort,omitempty"`
    MaxTokens       int    `json:"max_tokens,omitempty"`
}
```

处理逻辑：
1. 校验 `api_key` 不为空
2. 自动生成 model ID（基于 name 做 slugify）
3. 写入全局配置文件的 `llm.models` 数组
4. 设置 `llm.active_model_id`
5. 更新 Handler 的 `llmConfigured` 状态
6. 通知 LLM Client 重新初始化（或触发重载）

#### 4.2.3 模型 CRUD 端点

```go
// POST /api/v1/global/config/models - 添加模型
// PUT /api/v1/global/config/models/{id} - 更新模型
// DELETE /api/v1/global/config/models/{id} - 删除模型
// PUT /api/v1/global/config/models/{id}/activate - 激活模型
// POST /api/v1/global/config/models/{id}/test - 测试连接
```

测试连接逻辑：
```go
func (h *Handler) TestModelConnection(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    model := h.findModelByID(id)
    if model == nil {
        writeError(w, http.StatusNotFound, "model not found")
        return
    }
    // 用该模型配置创建一个临时 client，发送一个简单的 API 请求验证连通性
    err := h.testLLMConnection(model)
    if err != nil {
        writeJSON(w, http.StatusOK, map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        })
        return
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "success": true,
    })
}
```

### 4.3 LLM Client 动态切换

**文件**：`internal/taskexec/llmclient/providers/factory.go`

```go
// 新增：根据 active model 创建客户端
func NewClientFromModel(cfg *config.GlobalConfig, registry *tools.Registry) llmclient.Client {
    activeModel := cfg.GetActiveModel()  // 新增方法
    if activeModel == nil {
        return llmclient.NewMockClient()
    }
    return openai.New(openai.Config{
        LLMConfig: activeModel.ToLLMConfig(),  // 转换为兼容格式
    }, registry)
}
```

**文件**：`internal/config/config.go` 新增方法

```go
func (c *Config) GetActiveModel() *ModelConfig {
    for i := range c.LLM.Models {
        if c.LLM.Models[i].ID == c.LLM.ActiveModelID {
            return &c.LLM.Models[i]
        }
    }
    // 回退到第一个模型
    if len(c.LLM.Models) > 0 {
        return &c.LLM.Models[0]
    }
    return nil
}
```

### 4.4 模型切换通知

当用户切换模型时，需要通知正在运行的 AgentLoop 更新 LLM Client。在 Loop 上增加方法：

```go
// internal/core/agentloop/loop.go
func (l *Loop) UpdateLLMClient(client llmclient.Client) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.llm = client
}
```

### 4.5 后端改动汇总

| 文件 | 改动类型 | 说明 |
| :--- | :--- | :--- |
| `internal/config/config.go` | 修改 | 新增 `ModelConfig` 结构体，`LLMConfig` 增加 `Models`/`ActiveModelID`，增加迁移逻辑 |
| `internal/config/config.go` | 新增 | `GetActiveModel()` 方法 |
| `internal/interfaces/rest/handler.go` | 修改 | 注册新路由 |
| `internal/interfaces/rest/global_config_handler.go` | 新增 | 模型 CRUD + onboard + test handler |
| `internal/taskexec/llmclient/providers/factory.go` | 修改 | 支持从 active model 创建 client |
| `internal/core/agentloop/loop.go` | 新增 | `UpdateLLMClient()` 方法 |
| `cmd/devo/app.go` | 修改 | 适配新配置结构 |

---

## 五、CLI 配置渠道

CLI 使用标准库 `flag` 包实现，与 Web UI 操作同一份配置文件（`~/.devo/config.json`），用户无需启动后端即可管理模型。

### 5.1 命令路由

通过 `os.Args` 子命令路由实现，不引入第三方 CLI 框架：

```bash
devo config models list              # 列出所有模型
devo config models add               # 添加模型（参数式）
devo config models remove --id xxx   # 删除模型
devo config models activate --id xxx # 激活模型
devo config models test --id xxx     # 测试连接
devo config onboard                  # 交互式引导配置
```

### 5.2 实现方案

**文件**：`cmd/devo/config_models.go`（新增）

```go
// runConfigModels 处理 config models 子命令组
func runConfigModels(args []string) error {
    if len(args) == 0 {
        printModelsUsage()
        return nil
    }

    switch args[0] {
    case "list":
        return runModelsList(args[1:])
    case "add":
        return runModelsAdd(args[1:])
    case "remove":
        return runModelsRemove(args[1:])
    case "activate":
        return runModelsActivate(args[1:])
    case "test":
        return runModelsTest(args[1:])
    default:
        return fmt.Errorf("unknown subcommand: %s", args[0])
    }
}

func runModelsAdd(args []string) error {
    fs := flag.NewFlagSet("models add", flag.ExitOnError)
    name := fs.String("name", "", "模型显示名称（必填）")
    apiKey := fs.String("api-key", "", "API 密钥（必填）")
    baseURL := fs.String("base-url", "https://api.openai.com/v1", "API 地址")
    model := fs.String("model", "", "模型名称（必填）")
    maxTokens := fs.Int("max-tokens", 0, "最大输出 token")
    fs.Parse(args)

    if *name == "" || *apiKey == "" || *model == "" {
        fs.Usage()
        return fmt.Errorf("--name, --api-key, --model 为必填参数")
    }

    cfg := loadConfig()
    newModel := ModelConfig{
        ID:        slugify(*name),
        Name:      *name,
        APIKey:    *apiKey,
        BaseURL:   *baseURL,
        Model:     *model,
        MaxTokens: *maxTokens,
    }
    cfg.LLM.Models = append(cfg.LLM.Models, newModel)
    if cfg.LLM.ActiveModelID == "" {
        cfg.LLM.ActiveModelID = newModel.ID
    }
    return saveConfig(cfg)
}

func runModelsList(args []string) error {
    cfg := loadConfig()
    for _, m := range cfg.LLM.Models {
        marker := " "
        if m.ID == cfg.LLM.ActiveModelID {
            marker = "*"
        }
        fmt.Printf(" %s  %s (%s) - %s\n", marker, m.Name, m.ID, m.Model)
    }
    return nil
}

func runModelsRemove(args []string) error {
    fs := flag.NewFlagSet("models remove", flag.ExitOnError)
    id := fs.String("id", "", "模型 ID（必填）")
    fs.Parse(args)

    if *id == "" {
        return fmt.Errorf("--id 为必填参数")
    }

    cfg := loadConfig()
    models := cfg.LLM.Models
    for i, m := range models {
        if m.ID == *id {
            cfg.LLM.Models = append(models[:i], models[i+1:]...)
            if cfg.LLM.ActiveModelID == *id {
                if len(cfg.LLM.Models) > 0 {
                    cfg.LLM.ActiveModelID = cfg.LLM.Models[0].ID
                } else {
                    cfg.LLM.ActiveModelID = ""
                }
            }
            return saveConfig(cfg)
        }
    }
    return fmt.Errorf("模型 %s 不存在", *id)
}

func runModelsActivate(args []string) error {
    fs := flag.NewFlagSet("models activate", flag.ExitOnError)
    id := fs.String("id", "", "模型 ID（必填）")
    fs.Parse(args)

    if *id == "" {
        return fmt.Errorf("--id 为必填参数")
    }

    cfg := loadConfig()
    for _, m := range cfg.LLM.Models {
        if m.ID == *id {
            cfg.LLM.ActiveModelID = *id
            return saveConfig(cfg)
        }
    }
    return fmt.Errorf("模型 %s 不存在", *id)
}

func runModelsTest(args []string) error {
    fs := flag.NewFlagSet("models test", flag.ExitOnError)
    id := fs.String("id", "", "模型 ID（必填）")
    fs.Parse(args)

    if *id == "" {
        return fmt.Errorf("--id 为必填参数")
    }

    cfg := loadConfig()
    for _, m := range cfg.LLM.Models {
        if m.ID == *id {
            err := testConnection(&m)
            if err != nil {
                fmt.Printf("连接失败: %s: %v\n", m.Name, err)
                return nil
            }
            fmt.Printf("连接成功: %s\n", m.Name)
            return nil
        }
    }
    return fmt.Errorf("模型 %s 不存在", *id)
}
```

### 5.3 交互式引导

**文件**：`cmd/devo/config_onboard.go`（新增）

```bash
$ devo config onboard

  🚀 欢迎使用 Devo！请配置您的 LLM 模型。

  模型显示名称: GPT-4o
  API Key: sk-xxx
  API 地址 (默认 https://api.openai.com/v1):
  模型名称 (默认 gpt-4o):
  最大输出 Tokens (默认 128000):

  测试连接... ✅ 连接成功！

  配置已保存。当前激活模型: GPT-4o
```

该命令无参数，通过 `fmt.Scan` 交互式读取用户输入，最后调用 `runModelsAdd` 相同的保存逻辑。

### 5.4 CLI 与 Web UI 的协调

```
┌──────────────┐          ┌──────────────┐
│   CLI 命令    │──写入──→│              │
└──────────────┘          │  config.json │
                          │              │
┌──────────────┐          │  (单一数据源) │
│   Web UI     │──写入──→│              │
└──────────────┘          └──────┬───────┘
                                 │ 读取
                          ┌──────▼───────┐
                          │  后端 API    │
                          └──────────────┘
```

- CLI 直接读写配置文件，无需后端运行
- Web UI 通过 API 读写配置文件
- 后端启动时读取配置，每次 API 调用时重新读取以获取最新变更
- 并发写入场景：以最后写入为准，不做锁保护（配置文件写入开销极小）

### 5.5 CLI 改动汇总

| 文件 | 改动类型 | 说明 |
| :--- | :--- | :--- |
| `cmd/devo/config_models.go` | 新增 | `config models` 子命令组（list/add/remove/activate/test） |
| `cmd/devo/config_onboard.go` | 新增 | 交互式引导配置命令 |
| `cmd/devo/main.go` | 修改 | 注册 `config models` 和 `config onboard` 子命令路由 |

---

## 六、前端改动

### 6.1 组件架构

```
web/src/
├── components/
│   └── modal/
│       ├── ConfigWarningDialog.vue        ← 删除（或保留作为降级方案）
│       ├── OnboardingModal.vue            ← 新增：引导配置弹窗
│       └── OnboardingModalController.ts   ← 新增
│
├── components/
│   └── settings/                          ← 新增目录
│       ├── ModelEditor.vue                ← 新增：模型编辑表单
│       ├── ModelEditorController.ts       ← 新增
│       ├── ModelList.vue                  ← 新增：模型列表
│       └── ModelListController.ts         ← 新增
│
├── stores/
│   └── llmConfig.ts                       ← 新增：LLM 配置 Store
│
├── types/
│   └── llm.ts                             ← 新增：LLM 类型定义
```

### 6.2 OnboardingModal 引导页

**触发时机**：`AppController.ts` 的 `onMounted` 中检测到 `llm_configured === false`

**与现有 ConfigWarningDialog 的区别**：

| 维度 | ConfigWarningDialog（旧） | OnboardingModal（新） |
| :--- | :--- | :--- |
| 内容 | 只读提示，告诉用户去手动创建文件 | 交互式表单，可直接填写提交 |
| 操作 | 点击"我知道了"关闭 | 填写 → 测试连接 → 提交 → 自动关闭 |
| 跳过 | 关闭后下次启动仍弹出 | 可点击"稍后配置"跳过，在设置面板中补配 |

**页面布局**：

```
┌──────────────────────────────────────────┐
│  🚀 欢迎使用 Devo                         │
│  ─────────────────────────────────────── │
│                                          │
│  在开始之前，请配置您的 LLM 模型。          │
│                                          │
│  ┌─ 模型名称 ──────────────────────────┐  │
│  │  GPT-4o                      [   ] │  │
│  └────────────────────────────────────┘  │
│  ┌─ API Key ──────────────────────────┐  │
│  │  sk-...                      [👁]  │  │
│  └────────────────────────────────────┘  │
│  ┌─ API 地址 ─────────────────────────┐  │
│  │  https://api.openai.com/v1  [   ]  │  │
│  └────────────────────────────────────┘  │
│  ┌─ 模型名称（model）──────────────────┐  │
│  │  gpt-4o                      [   ]  │  │
│  └────────────────────────────────────┘  │
│                                          │
│  [ 测试连接 ]                             │
│  ✅ 连接成功！                             │
│                                          │
│  ─────────────────────────────────────── │
│  [ 稍后配置 ]              [ 完成配置 ]   │
└──────────────────────────────────────────┘
```

**交互流程**：

```
1. 用户填写表单
2. 可选：点击"测试连接"，前端调用 POST /api/v1/global/config/models/{id}/test
3. 点击"完成配置"：
   a. 前端 POST /api/v1/global/config/onboard
   b. 后端保存配置，更新 llmConfigured 状态
   c. 前端关闭弹窗，刷新配置状态
   d. 聊天界面可用
4. 点击"稍后配置"：
   a. 关闭弹窗
   b. 聊天界面仍不可用（因为无模型）
   c. 用户可以通过设置面板中的"全局设置 → 模型管理"随时配置
```

### 6.3 设置面板改造

**文件**：`web/src/panels/settings/SettingsPanel.vue`

在"全局设置" Tab 中新增"模型管理"区域：

```
┌──────────────────────────────────────────┐
│  [ 项目设置 ]  [ 全局设置 ]                │
├──────────────────────────────────────────┤
│                                          │
│  LLM 模型管理                             │
│  ┌────────────────────────────────────┐  │
│  │ ● GPT-4o                          │  │
│  │   gpt-4o · openai · 已连接    [✏] [✕]│  │
│  │                                    │  │
│  │ ○ GPT-4o Mini                     │  │
│  │   gpt-4o-mini · openai       [✏] [✕]│  │
│  │                                    │  │
│  │ ○ Claude 3.5 Sonnet               │  │
│  │   claude-3-5-sonnet · anthropic [✏] [✕]│  │
│  └────────────────────────────────────┘  │
│  [ + 添加模型 ]                           │
│                                          │
│  ─────────────────────────────────────── │
│  LLM 参数（全局默认）                      │
│  ...（现有内容保持不变）                    │
└──────────────────────────────────────────┘
```

`●` 表示当前激活的模型，`○` 表示非激活模型。点击模型行可激活。

**ModelEditor 子组件**：点击"添加模型"或"✏"编辑按钮时，展开或弹出模型编辑表单：

```
┌─ 模型名称 ──────────────────────────────┐
│  GPT-4o                           [  ] │
├─ API Key ──────────────────────────────┤
│  sk-xxx                           [👁] │
├─ API 地址 ─────────────────────────────┤
│  https://api.openai.com/v1        [  ] │
├─ 模型名称（model） ─────────────────────┤
│  gpt-4o                           [  ] │
├─ 最大输出 Tokens ───────────────────────┤
│  128000                           [  ] │
├─ 推理 ─────────────────────────────────┤
│  [✓] 启用推理  强度: [medium ▾]        │
├────────────────────────────────────────┤
│  [ 测试连接 ]    [ 取消 ]  [ 保存 ]     │
└────────────────────────────────────────┘
```

### 6.4 模型切换：`/model` 命令

模型切换通过内置命令 `/model` 实现，与现有 `/` 命令体系保持一致，无需额外 UI 组件。

**命令格式**：

```
/model                  → 显示当前模型 + 可用模型列表
/model [模型名]          → 切换为指定模型
/model [模型ID]          → 通过模型 ID 切换
```

**输入框提示**：在输入框下方字符计数区域旁显示当前模型名称，作为轻提示：

```
┌──────────────────────────────────────────────────────────┐
│  /  │ 输入消息，或按 / 使用命令...                        │
│     │                                                    │
│     │ [🖼]  123/10000 ~45 tokens  当前模型: GPT-4o  [发送]│
│     │  ↑                                ↑                │
│     │ 图片                         轻提示（不可交互）       │
└──────────────────────────────────────────────────────────┘
```

**用户输入 `/model` 后的交互**：

```
┌──────────────────────────────────────────────────┐
│  /model █                                        │
│                                                  │
│  ┌─ 可用模型 ─────────────────────────────────┐  │
│  │ ● GPT-4o (gpt-4o)                         │  │
│  │ ○ GPT-4o Mini (gpt-4o-mini)               │  │
│  │ ○ Claude 3.5 Sonnet (claude-3-5-sonnet)   │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

- 输入 `/model` 后，在输入框上方或下方展示可用模型列表（类似命令补全面板）
- `●` 表示当前激活模型，`○` 表示非激活模型
- 用户可继续输入模型名称进行过滤，或点击列表项直接切换
- 切换后显示确认提示："已切换到 GPT-4o"

**切换流程**：

1. 用户输入 `/model gpt-4o-mini`
2. 前端调用 `PUT /api/v1/global/config/models/{id}/activate`
3. 后端更新 `active_model_id`，保存配置，重建 LLM Client
4. 前端更新 `llmConfigStore`，输入框下方提示更新
5. 后续对话使用新模型

**实现说明**：

- `/model` 命令注册到现有的命令系统中（参考现有的 `/` 命令注册方式）
- 命令处理逻辑放在 `llmConfigStore` 中，复用 `activateModel()` 方法
- 无模型时（`hasModels === false`），`/model` 命令不注册，避免用户困惑

### 6.5 Store 设计

**文件**：`web/src/stores/llmConfig.ts`（新增）

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ModelConfig, LLMConfigStatus } from '@/types/llm'
import { API_BASE } from '@/utils/constants'

export const useLLMConfigStore = defineStore('llmConfig', () => {
  const configured = ref(false)
  const models = ref<ModelConfig[]>([])
  const activeModelId = ref<string | null>(null)
  const loading = ref(false)

  const activeModel = computed(() =>
    models.value.find(m => m.id === activeModelId.value) ?? null
  )

  const hasModels = computed(() => models.value.length > 0)

  async function fetchStatus() {
    loading.value = true
    try {
      const res = await fetch(`${API_BASE}/config/status`)
      const data: LLMConfigStatus = await res.json()
      configured.value = data.llm_configured
      models.value = data.models || []
      activeModelId.value = data.active_model_id || null
    } catch {
      configured.value = false
    } finally {
      loading.value = false
    }
  }

  async function onboard(config: { name: string; api_key: string; base_url?: string; model?: string }) {
    const res = await fetch(`${API_BASE}/global/config/onboard`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    })
    if (!res.ok) throw new Error('onboard failed')
    await fetchStatus()
  }

  async function addModel(model: ModelConfig) {
    const res = await fetch(`${API_BASE}/global/config/models`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(model),
    })
    if (!res.ok) throw new Error('add model failed')
    await fetchStatus()
  }

  async function updateModel(id: string, model: Partial<ModelConfig>) {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(model),
    })
    if (!res.ok) throw new Error('update model failed')
    await fetchStatus()
  }

  async function deleteModel(id: string) {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('delete model failed')
    await fetchStatus()
  }

  async function activateModel(id: string) {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}/activate`, {
      method: 'PUT',
    })
    if (!res.ok) throw new Error('activate model failed')
    activeModelId.value = id
    await fetchStatus()
  }

  async function testConnection(id: string): Promise<{ success: boolean; error?: string }> {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}/test`, {
      method: 'POST',
    })
    return res.json()
  }

  return {
    configured,
    models,
    activeModelId,
    activeModel,
    hasModels,
    loading,
    fetchStatus,
    onboard,
    addModel,
    updateModel,
    deleteModel,
    activateModel,
    testConnection,
  }
})
```

### 6.6 AppController 改动

**文件**：`web/src/AppController.ts`

```typescript
// 改动：将 config-warning 弹窗替换为 onboarding 弹窗
// 改动前：
if (!status.llm_configured) {
  uiStore.setActiveModal('config-warning')
}

// 改动后：
if (!status.llm_configured) {
  uiStore.setActiveModal('onboarding')
}
```

同时需要更新 `ModalType`：

```typescript
// web/src/stores/ui.ts
export type ModalType =
  | 'approval'
  | 'session-picker'
  | 'rollback-picker'
  | 'help'
  | 'onboarding'        // 替换 'config-warning'
  | null
```

### 6.7 GlobalModals 改动

**文件**：`web/src/components/layout/GlobalModals.vue`

将 `ConfigWarningDialog` 替换为 `OnboardingModal`。

### 6.8 前端改动汇总

| 文件 | 改动类型 | 说明 |
| :--- | :--- | :--- |
| `web/src/types/llm.ts` | 新增 | 模型相关类型定义 |
| `web/src/stores/llmConfig.ts` | 新增 | LLM 配置 Store（含 `/model` 命令处理） |
| `web/src/stores/ui.ts` | 修改 | `ModalType` 增加 `'onboarding'` |
| `web/src/components/modal/OnboardingModal.vue` | 新增 | 引导配置弹窗 |
| `web/src/components/modal/OnboardingModalController.ts` | 新增 | 引导配置逻辑 |
| `web/src/components/modal/ConfigWarningDialog.vue` | 删除/保留 | 可保留作为降级方案 |
| `web/src/components/settings/ModelList.vue` | 新增 | 模型列表组件 |
| `web/src/components/settings/ModelListController.ts` | 新增 | 模型列表逻辑 |
| `web/src/components/settings/ModelEditor.vue` | 新增 | 模型编辑表单 |
| `web/src/components/settings/ModelEditorController.ts` | 新增 | 模型编辑逻辑 |
| `web/src/components/layout/GlobalModals.vue` | 修改 | 引入 OnboardingModal |
| `web/src/AppController.ts` | 修改 | 触发 onboarding 弹窗 |
| `web/src/panels/settings/SettingsPanel.vue` | 修改 | 全局设置 Tab 增加模型管理区域 |
| `web/src/components/chat/InputArea.vue` | 修改 | 输入框下方增加当前模型名称轻提示 |
| 命令注册模块 | 修改 | 注册 `/model` 命令 |

---

## 七、交互流程

### 7.1 首次启动（无配置）

```
用户启动 Devo
    │
    ▼
前端 onMounted → GET /api/v1/config/status
    │
    ├── llm_configured === true → 正常进入聊天
    │
    └── llm_configured === false → 弹出 OnboardingModal
            │
            ├── 用户填写表单 → 测试连接（可选）→ 完成配置
            │       │
            │       ▼
            │   POST /api/v1/global/config/onboard
            │       │
            │       ▼
            │   后端保存配置 → 重建 LLM Client → 返回成功
            │       │
            │       ▼
            │   前端关闭弹窗 → 刷新状态 → 聊天可用
            │
            └── 用户点击"稍后配置"
                    │
                    ▼
                关闭弹窗 → 聊天不可用 → 用户可在设置面板中配置
```

### 7.2 模型管理（已有配置）

```
用户进入设置面板 → 全局设置 Tab
    │
    ├── 查看模型列表 → 当前激活模型高亮
    │
    ├── 点击"添加模型" → ModelEditor 展开
    │       │
    │       ▼
    │   填写表单 → 测试连接 → 保存
    │       │
    │       ▼
    │   POST /api/v1/global/config/models
    │
    ├── 点击模型行 → 激活模型
    │       │
    │       ▼
    │   PUT /api/v1/global/config/models/{id}/activate
    │
    ├── 点击编辑按钮 → ModelEditor 展开（预填数据）
    │       │
    │       ▼
    │   修改表单 → 保存
    │       │
    │       ▼
    │   PUT /api/v1/global/config/models/{id}
    │
    └── 点击删除按钮 → 确认弹窗 → 删除
            │
            ▼
        DELETE /api/v1/global/config/models/{id}
```

### 7.3 聊天中切换模型

```
用户在聊天界面输入 /model
    │
    ▼
命令补全面板展示所有可用模型，当前激活的标记 ●
    │
    ▼
用户输入模型名称或点击模型项
    │
    ▼
PUT /api/v1/global/config/models/{id}/activate
    │
    ▼
后端更新 active_model_id → 重建 LLM Client
    │
    ▼
前端更新 llmConfigStore → 输入框下方提示更新
    │
    ▼
后续对话使用新模型
```

---

## 八、实施计划

### Phase 1：后端数据模型 + 迁移（估 1.5 天）

- [ ] `internal/config/config.go`：新增 `ModelConfig` 结构体，`LLMConfig` 扩展
- [ ] `internal/config/config.go`：实现 `migrateLegacyConfig()` 迁移逻辑
- [ ] `internal/config/config.go`：实现 `GetActiveModel()` 方法
- [ ] 编写迁移逻辑的单元测试

### Phase 2：后端 API（估 2 天）

- [ ] `internal/interfaces/rest/global_config_handler.go`：实现模型 CRUD handler
- [ ] `internal/interfaces/rest/global_config_handler.go`：实现 onboard handler
- [ ] `internal/interfaces/rest/global_config_handler.go`：实现 test connection handler
- [ ] `internal/interfaces/rest/handler.go`：注册新路由
- [ ] `internal/interfaces/rest/handler.go`：增强 `GetConfigStatus` 返回模型列表
- [ ] 编写 API 集成测试

### Phase 3：LLM Client 动态切换（估 0.5 天）

- [ ] `internal/taskexec/llmclient/providers/factory.go`：支持从 active model 创建 client
- [ ] `internal/core/agentloop/loop.go`：增加 `UpdateLLMClient()` 方法
- [ ] `cmd/devo/app.go`：适配新配置结构

### Phase 4：前端 Store + 类型（估 0.5 天）

- [ ] `web/src/types/llm.ts`：新增类型定义
- [ ] `web/src/stores/llmConfig.ts`：新增 LLM 配置 Store

### Phase 5：前端 OnboardingModal（估 1 天）

- [ ] `web/src/components/modal/OnboardingModal.vue`：引导配置弹窗
- [ ] `web/src/components/modal/OnboardingModalController.ts`：逻辑
- [ ] `web/src/components/layout/GlobalModals.vue`：引入
- [ ] `web/src/AppController.ts`：触发逻辑
- [ ] `web/src/stores/ui.ts`：ModalType 更新

### Phase 6：前端设置面板 + 模型编辑器（估 1.5 天）

- [ ] `web/src/components/settings/ModelList.vue`：模型列表
- [ ] `web/src/components/settings/ModelEditor.vue`：模型编辑表单
- [ ] `web/src/panels/settings/SettingsPanel.vue`：集成模型管理区域

### Phase 7：前端 `/model` 命令（估 0.5 天）

- [ ] 在命令注册模块中注册 `/model` 命令
- [ ] 在 `llmConfigStore` 中实现命令处理逻辑（复用 `activateModel()`）
- [ ] 在 `InputArea.vue` 输入框下方增加当前模型名称轻提示
- [ ] 实现 `/model` 输入时的模型列表补全面板

### Phase 8：CLI 配置渠道（估 1 天）

- [ ] `cmd/devo/config_models.go`：实现 `config models` 子命令组
- [ ] 实现 `list`、`add`、`remove`、`activate`、`test` 子命令
- [ ] 实现 `config onboard` 交互式引导命令
- [ ] 编写 CLI 使用文档

### Phase 9：测试 + 联调（估 1 天）

- [ ] 前端单元测试
- [ ] 前后端联调
- [ ] 边界场景测试（无配置、单模型、多模型、删除激活模型等）

---

## 九、边界场景

| 场景 | 处理方式 |
| :--- | :--- |
| 旧配置文件（单模型格式） | `migrateLegacyConfig()` 自动迁移为多模型格式 |
| 删除当前激活的模型 | 自动切换到第一个可用模型；如果无模型则 `llm_configured = false` |
| 所有模型被删除 | `llm_configured = false`，弹出 OnboardingModal |
| 引导页提交失败 | 显示错误提示，表单不清空，允许重试 |
| 测试连接失败 | 显示具体错误信息，允许用户修改后重试或跳过 |
| 切换模型时正在对话 | 当前对话使用旧模型完成，后续对话使用新模型 |
| 移动端适配 | OnboardingModal 使用响应式布局，`/model` 命令在移动端同样可用 |
| API Key 脱敏 | 返回时前端只显示 `sk-***xxxx`，编辑时需重新输入（或用 `●●●●●●●●` 占位表示已设置） |
| CLI 与 Web 同时修改配置 | 以最后写入为准，后端在每次 API 调用时重新读取配置 |

---

## 十、风险与降级

| 风险 | 降级方案 |
| :--- | :--- |
| 多模型格式兼容问题 | 旧格式 `migrateLegacyConfig()` 自动转换，旧配置文件不丢失 |
| 模型切换导致 LLM Client 状态异常 | 切换时重置对话上下文，给出提示 |
| 引导页后端未就绪 | 前端 catch 错误，保留 ConfigWarningDialog 作为降级提示 |
| 模型 ID 冲突 | 保存时检查重复，提示用户修改名称 |

---

## 十一、附录：API 接口完整定义

### GET /api/v1/config/status（增强）

```
Response 200:
{
  "llm_configured": true,
  "active_model_id": "openai-gpt4o",
  "models": [
    {
      "id": "openai-gpt4o",
      "name": "GPT-4o",
      "provider": "openai",
      "api_key": "sk-***oBAz",
      "base_url": "https://api.openai.com/v1",
      "model": "gpt-4o",
      "enable_reasoning": false,
      "max_tokens": 128000
    }
  ]
}
```

### POST /api/v1/global/config/onboard

```
Request:
{
  "name": "GPT-4o",
  "api_key": "sk-xxx",
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4o",
  "enable_reasoning": false,
  "max_tokens": 128000
}

Response 200:
{
  "success": true,
  "model_id": "openai-gpt4o"
}
```

### POST /api/v1/global/config/models

```
Request:
{
  "name": "GPT-4o Mini",
  "api_key": "sk-xxx",
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4o-mini",
  "max_tokens": 128000
}

Response 200:
{
  "success": true,
  "model": { ... }
}
```

### PUT /api/v1/global/config/models/{id}

```
Request:
{
  "name": "GPT-4o Updated",
  "max_tokens": 64000
}

Response 200:
{
  "success": true,
  "model": { ... }
}
```

### DELETE /api/v1/global/config/models/{id}

```
Response 200:
{
  "success": true,
  "new_active_model_id": "openai-gpt4o-mini"
}
```

### PUT /api/v1/global/config/models/{id}/activate

```
Response 200:
{
  "success": true,
  "active_model_id": "openai-gpt4o-mini"
}
```

### POST /api/v1/global/config/models/{id}/test

```
Response 200:
{
  "success": true
}

Response 200 (失败):
{
  "success": false,
  "error": "connection timeout: dial tcp ..."
}
```