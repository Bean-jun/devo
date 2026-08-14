# Changelog

## [0.1.1] - 2026-08-14

### 🚀 新功能

- **模型切换（TUI）** — 终端界面支持 `/model` 命令动态切换 LLM 模型
- **版本更新检查** — 新增版本更新检查功能，启动时自动检测新版本
- **响应式密度布局** — Web 界面支持紧凑/舒适/标准三种密度模式，适配不同屏幕尺寸
- **后台进程输出面板** — 新增后台进程运行状态的实时输出面板
- **YOLO 信任级别同步** — YOLO 自动批准模式与信任级别配置同步
- **回滚优化** — 消息回滚功能稳定性改进

### 🐛 Bug 修复

- 修复 TUI 中 `/model` 切换模型不生效的问题
- 修复单元测试中 `TempDir` 清理错误
- 修复 `approval_test.go` 测试用例
- 修复 Web 端测试用例
- 修复多项 Go 测试稳定性问题

### 🔧 重构

- **CLI 架构迁移** — 将 CLI 代码迁移至 `internal/cli` 并引入 Cobra 框架，提升命令行参数解析和扩展性

### 📝 文档

- 更新 README 文档
- 新增 CHANGELOG.md
- 新增 LLM 配置引导与多模型管理设计方案文档

---

## [v0.1.0] - 2026-08-11

### 🎉 首个正式发布

**Devo**（Developer + Evolution）是一个以会话为核心、对话驱动的自主编码 AI 代理。它在你本地运行，直接访问文件系统和 Shell，通过 LLM 驱动完成编码任务。

### ✨ 核心特性

- **对话驱动开发** — 自然语言描述需求，AI 自主规划、编码、运行、修复，直到搞定
- **多端运行** — 浏览器 Web 控制中心、终端 TUI、VS Code 插件、Electron 桌面应用、移动端触摸优化，同一份代码
- **Agent Loop 自主循环** — AI 写完代码自己跑，报错了自己看、自己改，循环迭代直到跑通
- **审批门控** — 按操作风险分级（高/中/低/无），文件编辑带 diff 对比，支持 YOLO 自动批准模式
- **长期记忆 + 技能进化** — 记住偏好和项目经验，从对话中提炼 Skill 指令集，跨会话复用（全局 + 项目两层级）
- **上下文压缩** — 长对话自动压缩摘要，突破上下文窗口限制
- **消息回滚** — 回滚到任意历史位置重来，不影响文件系统
- **MCP 协议扩展** — 支持 MCP 协议动态接入外部工具，工作区级别隔离
- **多工作区管理** — 多项目自由切换，工作区与后端目录同步
- **思维链推理** — 支持 LLM Chain-of-Thought（CoT）推理模式，可配置推理强度
- **图像理解** — 支持 Base64 图片输入的多模态能力，含图像压缩预处理
- **后台进程管理** — 阻塞模式实时流式输出，Prompt Cache 监控
- **结构化日志** — 支持日志级别和链路追踪

### 📦 构建产物

| 平台 | 文件 |
|------|------|
| Linux x86_64 | `devo-linux-amd64` |
| macOS x86_64 | `devo-darwin-amd64` |
| macOS ARM64 | `devo-darwin-arm64` |
| Windows x86_64 | `devo-windows-amd64.exe` |
| VS Code 插件 | `devo-0.1.0.vsix` |
| Linux Desktop (AppImage) | `devo-desktop-0.1.0.AppImage` |
| Linux Desktop (deb) | `devo-desktop_0.1.0_amd64.deb` |

### 🛠 快速开始

**前置依赖：** Go 1.25+ · Node.js 18+ · Python 3.8+

**配置：** 在 `.devo/config.json` 或 `~/.devo/config.json` 中配置 LLM：

```json
{
  "llm": {
    "api_key": "sk-your-key-here",
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o"
  }
}
```

也支持环境变量：`DEVO_LLM_API_KEY`、`DEVO_LLM_BASE_URL`、`DEVO_LLM_MODEL`

**运行：**

```bash
# 下载对应平台的二进制文件
./devo-linux-amd64

# 然后打开浏览器访问 http://localhost:8080
```

### 🔧 从源码构建

```bash
git clone https://github.com/your-org/devo.git
cd devo
make build-go        # 构建 Go 后端
cd web && npm ci && npm run build  # 构建前端
```

---

[Unreleased]: https://github.com/your-org/devo/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/your-org/devo/releases/tag/v0.1.0