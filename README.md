# Devo

**Devo**（Developer + Evolution）是一个以会话为核心、对话驱动的自主编码 AI 代理。它在你的本地开发环境中运行，直接访问文件系统和 Shell，通过 LLM 驱动完成编码任务。

---

## 特性

- **对话驱动** — 用自然语言描述任务，AI 自主规划、编码、修复
- **Web 控制中心** — 三栏面板布局，聊天主场 + 文件/技能/记忆/仪表盘/设置/终端面板
- **多工作区管理** — 多项目切换，工作区与后端目录同步，支持删除确认
- **文件浏览与预览** — 文件树浏览、代码/图片预览（5MB 上限、150+ 扩展名白名单）
- **审批门控** — 按操作风险分级（高/中/低），文件编辑带 diff 对比，命令执行需确认
- **精确编辑** — 支持基于 diff 的精确文件编辑，修改范围一目了然
- **长期记忆** — 记住你的偏好和项目经验，跨会话积累（全局 + 项目两层级）
- **技能进化** — Skills 管理器，自动从完成的任务中提炼最佳实践
- **上下文压缩** — 长对话自动压缩摘要，突破上下文窗口限制
- **会话存档** — 每次会话自动生成 Markdown 存档，可版本管理、可分享
- **消息回滚** — 回滚到历史任意位置重来，不影响文件系统
- **工具调用上限** — 防止失控循环消耗 Token，预算可控
- **崩溃恢复** — 异常中断后自动清理，不遗留僵尸进程
- **并发会话隔离** — 多个任务独立运行，互不干扰
- **MCP 扩展** — 支持 MCP 协议动态接入外部工具
- **双模式** — 浏览器完整控制中心 + VSCode Webview 极简聊天窗口，同一份代码
- **跨平台** — 支持 Linux、macOS、Windows

---

## 架构概览

```
┌──────────────────────────────────────────────────┐
│                    UI 层                          │
│  ┌────────────┐ ┌────────┐ ┌─────────────────┐   │
│  │ Web 前端    │ │  TUI   │ │ VS Code 插件     │   │
│  │ (Vue 3)    │ │ (终端)  │ │ (Webview 复用)  │   │
│  └────────────┘ └────────┘ └─────────────────┘   │
└──────────────────────┬───────────────────────────┘
                       │ HTTP (REST) + SSE
┌──────────────────────▼───────────────────────────┐
│                 接口层 (API)                       │
│  REST API · SSE 事件流 · 审批桥接                  │
└──────────────────────┬───────────────────────────┘
                       │ Go 接口
┌──────────────────────▼───────────────────────────┐
│              任务处理层 (Task)                     │
│  工具集（写文件/执行命令/搜索等）                    │
│  Python 统一沙箱执行器                             │
│  LLM 客户端封装 · Token 计量器                     │
└──────────────────────┬───────────────────────────┘
                       │ Go 接口
┌──────────────────────▼───────────────────────────┐
│                核心层 (Core)                       │
│  会话生命周期 · 对话驱动 Agent Loop                │
│  审批门控（含超时/风险分级）                        │
│  上下文压缩 · 事件总线 · 消息回滚                   │
│  长期记忆管理器 · 经验固化器                        │
│  Skills 管理器 · agents.md 加载器                  │
│  会话存档 · 并发会话隔离                            │
│  Token 计量协调 · 崩溃恢复                         │
│  工作区管理 · 文件系统操作                          │
└──────────────────────────────────────────────────┘
```

### 三层分离

| 层次 | 职责 |
| :--- | :--- |
| **核心层** | 会话管理、代理循环、审批门控、上下文压缩、长期记忆、技能管理、工作区管理、崩溃恢复 |
| **任务处理层** | 工具集（文件读写、命令执行、代码搜索）、MCP 客户端、Python 统一执行器、LLM 客户端封装 |
| **接口层** | REST API + SSE 事件流，供前端实时交互 |

### 前端 Web 应用

- **技术栈**：Vue 3 + TypeScript + Pinia + Vite，零 UI 框架依赖
- **布局**：三栏面板（左侧栏 + 聊天主场 + 右侧面板），支持拖拽调整宽度
- **状态管理**：8 个 Pinia Store（ui/session/chat/approval/command/skills/memory/mcp）
- **SSE 事件流**：14+ 事件类型实时推送，App.vue 统一分发
- **模式分流**：同一份代码支持浏览器（`?mode=vscode` 切换）和 VSCode Webview

### 部署模型

- **本地核心（必需）** — 运行在开发者个人电脑上，直接访问本地文件系统和 Shell
- **可选团队服务** — 轻量中心服务，仅做统计上报聚合和远程审批中转，**不上传代码或对话内容**

---

## 快速开始

### 前置依赖

- Go 1.25+
- Node.js 18+（用于 Web 前端构建）
- Python 3.8+（用于命令执行沙箱）

### 配置

Devo 需要 LLM API 密钥才能运行。在项目根目录 `.devo/` 或 `~/.devo/` 下创建 `config.json`：

```json
{
  "llm": {
    "api_key": "sk-your-key-here",
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o"
  }
}
```

也可以通过环境变量配置：

| 环境变量 | 说明 |
| :--- | :--- |
| `DEVO_LLM_API_KEY` | LLM API 密钥（必需） |
| `DEVO_LLM_BASE_URL` | LLM API 地址 |
| `DEVO_LLM_MODEL` | 模型名称 |
| `DEVO_DB_PATH` | 数据库文件路径（默认 `~/.devo/devo.db`） |
| `DEVO_LOG_PATH` | 日志文件路径 |

### 构建

```bash
# 构建后端 + Web 前端 + VS Code 插件
make build

# 或分步构建
make build-web   # 构建 Web 前端（输出到 web/dist/）
make build-go    # 构建 Go 后端
```

### 启动服务

```bash
# Web 模式（浏览器访问 http://localhost:8080）
.\build\devo.exe -web

# TUI 模式（终端内交互）
.\build\devo.exe -tui

# 指定端口
.\build\devo.exe -web -port 9090

# 指定工作目录
.\build\devo.exe -web -workspace /path/to/your/project
```

### 创建会话

```bash
# 创建新会话，指定工作目录
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"working_directory": "/path/to/your/project"}'
```

### 发送消息

```bash
# 发送任务消息
curl -X POST http://localhost:8080/api/v1/sessions/{id}/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "帮我重构这个模块"}'
```

---

## API 概览

### 工作区管理

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/current-workspace` | 获取当前工作目录 |
| `POST` | `/api/v1/current-workspace` | 切换当前工作目录 |
| `GET` | `/api/v1/workspace` | 获取工作区列表 |
| `DELETE` | `/api/v1/workspace?path=xxx` | 删除工作区及其所有会话 |

### 会话管理

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions` | 创建会话 |
| `GET` | `/api/v1/sessions` | 列出会话（支持 `?project=` 过滤） |
| `GET` | `/api/v1/sessions/{id}` | 获取会话详情 |
| `PUT` | `/api/v1/sessions/{id}` | 重命名会话 |
| `DELETE` | `/api/v1/sessions/{id}` | 删除会话 |
| `PUT` | `/api/v1/sessions/{id}/config` | 更新会话配置 |
| `PUT` | `/api/v1/sessions/{id}/trust` | 设置信任级别 |
| `PUT` | `/api/v1/sessions/{id}/approval-policy` | 设置审批策略 |

### 消息与对话

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions/{id}/messages` | 发送消息 |
| `GET` | `/api/v1/sessions/{id}/messages` | 获取消息历史 |
| `GET` | `/api/v1/sessions/{id}/events` | SSE 事件流 |

### 会话控制

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions/{id}/pause` | 暂停会话 |
| `POST` | `/api/v1/sessions/{id}/resume` | 恢复会话 |
| `POST` | `/api/v1/sessions/{id}/cancel` | 取消当前循环 |
| `POST` | `/api/v1/sessions/{id}/complete` | 完成会话 |
| `POST` | `/api/v1/sessions/{id}/archive` | 归档会话 |

### 审批

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/sessions/{id}/approve/{approval_id}` | 响应审批 |

### 文件与回滚

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/sessions/{id}/files?path=xxx` | 获取文件列表/内容 |
| `POST` | `/api/v1/sessions/{id}/rollback` | 回滚到指定消息 |

### 记忆（Memory）

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/sessions/{id}/memory` | 查看记忆 |
| `POST` | `/api/v1/sessions/{id}/memory` | 添加/更新记忆 |
| `DELETE` | `/api/v1/sessions/{id}/memory/{memory_id}` | 删除记忆 |

### 技能（Skills）

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/skills` | 获取技能列表 |
| `POST` | `/api/v1/sessions/{id}/skills` | 设置会话技能 |
| `POST` | `/api/v1/skills/install` | 安装技能 |
| `POST` | `/api/v1/sessions/{id}/solidify` | 经验固化 |

### MCP

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/mcp/tools` | 获取 MCP 工具列表 |

### 用量统计

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/sessions/{id}/usage` | 会话 Token 消耗统计 |
| `GET` | `/api/v1/usage/stats` | 全局用量统计 |

### 存档

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/sessions/{id}/archive` | 下载会话存档 |
| `POST` | `/api/v1/sessions/{id}/sync-archive` | 同步会话存档 |

### 版本

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/version` | 获取版本信息 |

完整 API 文档见 [docs/2-architecture.md](docs/2-architecture.md#41-rest-api)。

---

## 会话状态

```
Idle → Processing → AwaitingApproval → Processing → Idle
  ↓         ↓              ↓
Paused ← ──┴── → Completed → Archived
```

| 状态 | 说明 |
| :--- | :--- |
| `Idle` | 等待用户输入 |
| `Processing` | 代理循环运行中 |
| `AwaitingApproval` | 等待用户审批 |
| `Paused` | 暂停（主动或离线自动） |
| `Completed` | 已完成 |
| `Archived` | 归档，不可再交互 |

---

## 安全设计

- **路径安全** — 所有文件操作强制限定在工作目录内，杜绝目录穿越
- **审批门控** — 高风险操作（命令执行）始终需确认；文件编辑自动附带 diff
- **命令过滤** — 平台感知黑名单，拦截破坏性命令和高危管道组合
- **Python 中介执行** — 命令通过环境变量传递，杜绝代码注入
- **文件预览限制** — 客户端 5MB 上限 + 150+ 扩展名白名单，不预览不明文件
- **工具调用上限** — 防止死循环无限消耗 Token
- **崩溃安全** — 自动清理孤儿进程，插入恢复告知
- **工作目录约束** — 创建会话时必须提供合法路径，所有操作在该范围内进行

---

## 平台支持

| 能力 | Linux / macOS | Windows |
| :--- | :--- | :--- |
| 路径安全 | ✅ | ✅ |
| Python 中介执行 | ✅ (Python 3.8+) | ✅ (Python 3.8+) |
| 资源限制 (CPU/内存) | ✅ (`setrlimit`) | ❌ (计划中) |
| 命令黑名单 | ✅ (Unix Shell) | ✅ (PowerShell) |
| 执行超时 | ✅ | ✅ |

---

## 项目结构

```
Devo/
├── cmd/                    ← 入口程序
├── internal/               ← 内部包（核心/任务/接口层）
│   ├── core/               ← 核心层
│   ├── task/               ← 任务处理层
│   └── interfaces/         ← 接口层
│       ├── rest/           ← REST API + SSE
│       └── tui/            ← TUI 终端界面
├── web/                    ← Web 前端（Vue 3 + TypeScript）
│   └── src/
│       ├── layouts/        ← 布局组件（BrowserLayout/VscodeLayout）
│       ├── components/     ← 共享组件（聊天/布局/弹窗/命令）
│       ├── panels/         ← 右侧面板（文件/技能/记忆/仪表盘/设置/终端）
│       ├── stores/         ← Pinia 状态管理（8 个 Store）
│       ├── composables/    ← 可复用逻辑（SSE/API/主题/命令等）
│       ├── types/          ← TypeScript 类型定义
│       └── styles/         ← CSS 变量/基础样式/动画
├── docs/                   ← 设计文档
│   ├── 1-PRD.md            ← 产品需求文档
│   ├── 2-architecture.md   ← 完整架构设计
│   ├── 3-frontend-design.md← 前端设计方案
│   └── 4-web-architecture.md← Web 架构文档
└── .devo/                  ← 运行时数据
    ├── config.json         ← 全局配置
    ├── sessions/           ← 会话存档 (Markdown)
    ├── skills/             ← Skills 指令集
    │   ├── project/        ← 项目内 Skills
    │   └── global/         ← 全局 Skills
    ├── memory/             ← 长期记忆
    │   ├── user-*.md       ← 用户记忆
    │   └── projects/       ← 项目记忆（路径哈希分片）
    └── mcp_servers.json    ← MCP 服务器配置
```

---

## 深入阅读

- [PRD 需求文档](docs/1-PRD.md) — 产品需求与典型工作流
- [完整架构设计](docs/2-architecture.md) — 系统详细设计文档
- [前端设计方案](docs/3-frontend-design.md) — Web 前端 UI/UX 设计
- [Web 架构文档](docs/4-web-architecture.md) — Web 前端工程架构
- [Agent Loop 事件驱动重构](docs/6-agent-loop-event-driven-refactor.md)

## License

[MIT](LICENSE)