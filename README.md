# Devo

**Devo**（Developer + Evolution）是一个以会话为核心、对话驱动的自主编码 AI 代理。它在你本地运行，直接访问文件系统和 Shell，通过 LLM 驱动完成编码任务。

---

## 特性

- **对话驱动** — 自然语言描述任务，AI 自主规划、编码、修复
- **Web 控制中心** — 三栏面板布局，聊天主场不离开，右侧面板承载文件/技能/记忆/仪表盘/设置/终端/MCP/后台进程
- **三模式运行** — 浏览器完整功能、VSCode Webview 极简聊天、移动端触摸优化，同一份代码
- **多工作区管理** — 多项目切换，工作区与后端目录同步
- **审批门控** — 按操作风险分级（高/中/低/无），文件编辑带 diff 对比，支持 YOLO 自动批准
- **长期记忆 + 技能进化** — 记住偏好和项目经验，从对话中提炼 Skill 指令集，跨会话复用（全局 + 项目两层级）
- **上下文压缩** — 长对话自动压缩摘要，突破上下文窗口限制
- **消息回滚** — 回滚到任意历史位置重来，不影响文件系统
- **MCP 扩展** — 支持 MCP 协议动态接入外部工具
- **跨平台** — Linux、macOS、Windows 全支持

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
│  工具集 · Python 统一沙箱 · LLM 客户端 · Token 计量 │
└──────────────────────┬───────────────────────────┘
                       │ Go 接口
┌──────────────────────▼───────────────────────────┐
│                核心层 (Core)                       │
│  Agent Loop · 审批门控 · 上下文压缩 · 消息回滚     │
│  长期记忆 · Skills 管理 · 会话存档 · 工作区管理    │
│  并发隔离 · 崩溃恢复 · MCP 客户端                  │
└──────────────────────────────────────────────────┘
```

**部署模型**：本地核心（必需）运行在开发者电脑上，直接访问本地文件系统。可选团队服务仅做统计上报和远程审批中转，**不上传代码或对话内容**。

---

## 快速开始

### 前置依赖

- Go 1.25+
- Node.js 18+（Web 前端构建）
- Python 3.8+（命令执行沙箱）

### 配置

在项目根目录 `.devo/` 或 `~/.devo/` 下创建 `config.json`：

```json
{
  "llm": {
    "api_key": "sk-your-key-here",
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o"
  }
}
```

也支持环境变量：`DEVO_LLM_API_KEY`、`DEVO_LLM_BASE_URL`、`DEVO_LLM_MODEL`、`DEVO_DB_PATH`、`DEVO_LOG_PATH`。

### 构建

```bash
make build          # 构建后端 + Web 前端 + VS Code 插件
make build-web      # 仅构建 Web 前端
make build-go       # 仅构建 Go 后端
```

### 启动

```bash
devo -web                              # Web 模式（http://localhost:8080）
devo -web -port 9090                   # 指定端口
devo -web -workspace /path/to/project  # 指定工作目录
devo -tui                              # TUI 模式
```

### 快速试用

```bash
# 创建会话
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"working_directory": "/path/to/your/project"}'

# 发送消息
curl -X POST http://localhost:8080/api/v1/sessions/{id}/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "帮我重构这个模块"}'
```

完整 API 文档见 [架构设计](docs/2-architecture.md#41-rest-api)。

---

## 项目结构

```
Devo/
├── cmd/                    ← 入口程序
├── internal/               ← 内部包
│   ├── core/               ← 核心层（会话/审批/记忆/技能/Agent Loop）
│   ├── task/               ← 任务处理层（工具集/沙箱/LLM 客户端）
│   └── interfaces/         ← 接口层
│       ├── rest/           ← REST API + SSE
│       └── tui/            ← TUI 终端界面
├── web/                    ← Web 前端（Vue 3 + TypeScript）
│   └── src/
│       ├── layouts/        ← BrowserLayout / VscodeLayout / MobileLayout
│       ├── components/     ← 聊天/布局/弹窗/命令/移动端/编辑器
│       ├── panels/         ← 面板（files/skills/memory/dashboard/settings/terminal/mcp/background）
│       ├── stores/         ← Pinia 状态管理（9 个 Store）
│       ├── composables/    ← 可复用逻辑
│       ├── types/          ← 类型定义
│       └── styles/         ← CSS 变量/基础样式/动画
├── docs/                   ← 设计文档
│   ├── 1-PRD.md            ← 产品需求文档
│   ├── 2-architecture.md   ← 完整架构设计
│   ├── 3-frontend-design.md← 前端设计方案
│   └── 4-web-architecture.md← Web 架构文档
└── .devo/                  ← 运行时数据
    ├── config.json         ← 全局配置
    ├── sessions/           ← 会话存档
    ├── skills/             ← Skills 指令集（project/ + global/）
    ├── memory/             ← 长期记忆
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