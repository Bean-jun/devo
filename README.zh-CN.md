# Devo

[English](README.md) | 中文

**Devo**（Developer + Evolution）是一个以会话为核心、对话驱动的自主编码 AI 代理。它在你本地运行，直接访问文件系统和 Shell，通过 LLM 驱动完成编码任务。

---

## 特性

- **对话驱动** — 自然语言描述任务，AI 自主规划、编码、运行、修复，循环迭代直到搞定
- **五端运行** — Web 控制中心、终端 TUI、VS Code 插件、Electron 桌面应用、移动端触摸优化，同一份代码
- **Web 控制中心** — 三栏面板布局，聊天主场不离开，右侧面板承载文件/技能/记忆/仪表盘/设置/终端/MCP/后台进程
- **多工作区管理** — 多项目自由切换，工作区与后端目录同步
- **审批门控** — 按操作风险分级（高/中/低/无），文件编辑带 diff 对比，支持 YOLO 自动批准模式
- **长期记忆 + 技能进化** — 记住偏好和项目经验，从对话中提炼 Skill 指令集，跨会话复用（全局 + 项目两层级）
- **上下文压缩** — 长对话自动压缩摘要，突破上下文窗口限制
- **消息回滚** — 回滚到任意历史位置重来，不影响文件系统
- **思维链推理** — 支持 LLM Chain-of-Thought（CoT）推理模式，可配置推理强度
- **图像理解** — 支持 Base64 图片输入的多模态能力，含图像压缩预处理
- **MCP 扩展** — 支持 MCP 协议动态接入外部工具，工作区级别隔离
- **后台进程管理** — 阻塞模式实时流式输出，支持 Prompt Cache 监控
- **结构化日志** — 支持日志级别和链路追踪
- **跨平台** — Linux、macOS、Windows 全支持

---

## 架构概览

| 层级 | 模块 | 职责 |
|:----:|------|------|
| **UI 层** | Web · TUI · Mobile · VS Code · Desktop | 多端用户交互入口 |
| ↓ | | |
| **接口层** | REST API · SSE 事件流 · 审批桥接 | HTTP 服务、实时推送、审批回调 |
| ↓ | | |
| **任务处理层** | 工具集 · Python 沙箱 · LLM 客户端 · MCP · 图像压缩 · 路径安全 | 工具执行、代码沙箱、模型调用 |
| ↓ | | |
| **核心层** | Agent Loop · 审批门控 · 上下文压缩 · 消息回滚 · 长期记忆 · Skills · 会话存档 · 并发隔离 · 崩溃恢复 | 智能体循环、状态管理、容错 |
| ↓ | | |
| **存储层** | SQLite 持久化 | 会话 / 消息 / 事件 存储 |

**部署模型**：本地核心（必需）运行在开发者电脑上，直接访问本地文件系统。可选团队服务仅做统计上报和远程审批中转，**不上传代码或对话内容**。

---

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go 1.25+ |
| CLI 框架 | [Cobra](https://github.com/spf13/cobra) |
| TUI 框架 | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| 前端框架 | Vue 3 + TypeScript + Vite |
| 状态管理 | Pinia |
| 路由 | Vue Router |
| Markdown 渲染 | marked + highlight.js |
| 图标库 | Phosphor Icons |
| 数据库 | SQLite (GORM) |
| LLM 协议 | OpenAI 兼容 API |
| 测试 | Vitest + Playwright (前端) / Go testing (后端) |
| 桌面端 | Electron |

---

## 快速开始

### 前置依赖

- Go 1.25+
- Node.js 22+（Web 前端构建）
- Python 3.8+（命令执行沙箱）

### 配置

Devo 支持多种方式配置 LLM，无需手动编辑配置文件：

**方式一：Web 界面配置（推荐）**

启动 Devo 后，首次使用会自动弹出配置引导对话框，或随时在右侧 **设置面板 → 全局设置** 中添加和管理模型。

**方式二：命令行配置**

```bash
# 交互式配置引导（推荐首次使用）
devo config onboard

# 直接添加模型
devo config models add --name "GPT-4o" --api-key "sk-xxx" --model "gpt-4o"

# 管理模型
devo config models list              # 列出所有模型
devo config models activate --id gpt-4o  # 激活指定模型
devo config models test --id gpt-4o      # 测试模型连接
devo config models remove --id gpt-4o    # 删除模型
```

**方式三：环境变量**

```bash
export DEVO_LLM_API_KEY="sk-your-key-here"
export DEVO_LLM_BASE_URL="https://api.openai.com/v1"
export DEVO_LLM_MODEL="gpt-4o"
```

也支持 `DEVO_DB_PATH`、`DEVO_LOG_PATH` 等环境变量覆盖默认路径。

### 构建

```bash
make build          # 构建后端 + Web 前端 + VS Code 插件 + Electron 桌面端
make build-web      # 仅构建 Web 前端
make build-go       # 仅构建 Go 后端（4 个平台二进制）
make vsix           # 仅打包 VS Code 插件
make desktop        # 仅打包 Electron 桌面应用
```

### 启动

```bash
# Web 模式（默认，自动打开浏览器）
devo

# 指定端口和工作目录
devo -web -port 9090 -workspace /path/to/project

# TUI 终端模式
devo -tui

# 查看帮助
devo --help
```

### 开发模式

```bash
make dev            # 同时启动前端 dev server + 后端（web 模式）
make dev-web        # 仅启动前端 dev server
make run-web        # 构建并运行 Web 模式
make run-tui        # 构建并运行 TUI 模式
```

### 测试

```bash
make test           # 运行全部测试（前端 + 后端）
make test-web       # 前端单元测试
make test-e2e       # 前端 E2E 测试（Playwright）
make test-go        # 后端测试
make lint           # 代码检查（前端 ESLint + 后端 go vet）
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

完整 API 文档见 [架构设计](docs/2-architecture.md)。

---

## 项目结构

```
Devo/
├── cmd/devo/              ← 程序入口（main.go）
├── internal/              ← Go 内部包
│   ├── cli/               ← CLI 命令定义与 app 启动引导
│   │   ├── commands/      ← cobra 命令（root / config）
│   │   ├── app.go         ← 应用初始化
│   │   ├── server.go      ← HTTP 服务启动
│   │   └── platform.go    ← 平台检测
│   ├── config/            ← 配置管理（全局/项目/路径）
│   ├── core/              ← 核心层
│   │   ├── agentloop/     ← Agent 循环、状态机、审批处理、崩溃恢复、回滚
│   │   ├── approval/      ← 审批门控与风险分级
│   │   ├── archive/       ← 会话 Markdown 存档
│   │   ├── compressor/    ← 上下文压缩
│   │   ├── concurrency/   ← 并发控制（路径锁）
│   │   ├── memory/        ← 长期记忆管理
│   │   ├── prompt/        ← 系统提示词组装、agents.md 加载、目录树
│   │   ├── session/       ← 会话模型、事件总线、内存存储
│   │   ├── skills/        ← Skills 管理与经验固化
│   │   └── tokenmeter/    ← Token 计量器
│   ├── interfaces/        ← 接口层
│   │   ├── rest/          ← REST API + SSE 事件流（20+ handler）
│   │   └── tui/           ← Bubble Tea TUI 界面
│   │       ├── api/       ← TUI 后端 API 客户端
│   │       ├── components/← TUI 组件（状态栏/Toast/样式）
│   │       ├── overlays/  ← TUI 覆盖层（审批/后台/命令/仪表盘/MCP 等）
│   │       ├── renderer/  ← Markdown 渲染器
│   │       └── types/     ← TUI 类型定义
│   ├── pkg/               ← 公共工具包
│   │   ├── logging/       ← 结构化日志
│   │   └── process/       ← 进程管理
│   ├── storage/           ← 存储层
│   │   └── sqlite/        ← SQLite 持久化（会话/消息/事件/回滚/SSE/用量）
│   ├── taskexec/          ← 任务处理层
│   │   ├── imageproc/     ← 图像压缩预处理
│   │   ├── llmclient/     ← LLM 客户端（OpenAI 兼容协议）
│   │   ├── mcp/           ← MCP 协议客户端管理
│   │   ├── pathsec/       ← 路径安全校验、.gitignore 解析
│   │   └── tools/         ← 工具集（读写文件/执行命令/搜索/glob/diff 等）
│   └── update/            ← 版本更新检查
├── web/                   ← Web 前端（Vue 3 + TypeScript + Vite）
│   ├── src/
│   │   ├── layouts/       ← 三种布局：BrowserLayout / VscodeLayout / MobileLayout
│   │   ├── components/    ← 聊天/布局/弹窗/命令/移动端/编辑器
│   │   ├── panels/        ← 右侧面板（files/skills/memory/dashboard/settings/terminal/mcp/background）
│   │   ├── stores/        ← Pinia 状态管理
│   │   ├── composables/   ← 可复用逻辑
│   │   ├── types/         ← TypeScript 类型定义
│   │   └── styles/        ← CSS 变量/基础样式/动画
│   └── e2e/               ← Playwright E2E 测试
├── electron/              ← Electron 桌面应用
│   ├── main.js            ← Electron 主进程
│   ├── welcome.html       ← 欢迎页
│   └── resources/         ← 平台二进制 & 图标
├── vscode-extension/      ← VS Code 插件
│   ├── dist/              ← 编译后的扩展代码
│   ├── bin/               ← 各平台后端二进制
│   └── esbuild.js         ← 扩展打包脚本
├── docs/                  ← 设计文档
│   ├── 1-PRD.md           ← 产品需求文档
│   ├── 2-architecture.md  ← 完整架构设计
│   ├── 3-cli-architecture.md ← CLI 架构设计
│   ├── 3-frontend-design.md  ← 前端设计方案
│   ├── 4-web-architecture.md ← Web 前端工程架构
│   ├── 5-web-testing.md   ← Web 测试方案
│   ├── 6-agent-loop-event-driven-refactor.md ← Agent Loop 事件驱动重构
│   ├── 7-electron-desktop-architecture.md ← Electron 桌面端架构
│   ├── 8-session-state-robustness-test.md  ← 会话状态健壮性测试
│   ├── 9-performance-optimization.md       ← 性能优化
│   ├── 10-mobile-layout-design.md          ← 移动端布局设计
│   ├── 11-llm-reasoning-cot-design.md      ← 思维链推理设计
│   ├── 12-image-input-design.md            ← 图像输入设计
│   └── 13-llm-config-onboarding-design.md  ← LLM 配置引导设计
├── build/                 ← 构建产物目录
├── .github/workflows/     ← CI/CD 工作流
│   ├── ci.yml             ← 持续集成（lint + test + build）
│   └── release.yml        ← 版本发布（多平台构建 + 打包）
├── .devo/                 ← 运行时数据
│   ├── config.json        ← 全局配置
│   └── sessions/          ← 会话存档
├── Makefile               ← 构建脚本
├── VERSION                ← 版本号
└── go.mod                 ← Go 模块定义
```

---

## 深入阅读

- [PRD 需求文档](docs/1-PRD.md) — 产品需求与典型工作流
- [完整架构设计](docs/2-architecture.md) — 系统详细设计文档
- [CLI 架构设计](docs/3-cli-architecture.md) — CLI 与 TUI 架构
- [前端设计方案](docs/3-frontend-design.md) — Web 前端 UI/UX 设计
- [Web 前端工程架构](docs/4-web-architecture.md) — Web 前端工程化方案
- [Web 测试方案](docs/5-web-testing.md) — 前端测试策略
- [Agent Loop 事件驱动重构](docs/6-agent-loop-event-driven-refactor.md) — Agent Loop 重构设计
- [Electron 桌面端架构](docs/7-electron-desktop-architecture.md) — 桌面应用架构设计
- [会话状态健壮性测试](docs/8-session-state-robustness-test.md) — 状态机测试方案
- [性能优化](docs/9-performance-optimization.md) — 性能优化方案
- [移动端布局设计](docs/10-mobile-layout-design.md) — 移动端适配方案
- [思维链推理设计](docs/11-llm-reasoning-cot-design.md) — LLM CoT 推理接入
- [图像输入设计](docs/12-image-input-design.md) — 多模态图像输入方案
- [LLM 配置引导设计](docs/13-llm-config-onboarding-design.md) — 首次配置引导方案

## License

[MIT](LICENSE)