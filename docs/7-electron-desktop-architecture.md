# Devo Electron 桌面端架构设计

**版本**：1.0.0（2026-06-28）

**定位**：基于 Electron 的独立桌面应用，作为 Devo 四端（TUI / Web / VSCode / Desktop）之一。Electron 仅作为壳层，负责进程管理和窗口渲染，所有业务逻辑由 Go 后端提供，所有 UI 由现有 Vue 前端复用。

---

## 一、四端全景

| 端 | 入口 | 前端 | 启动方式 | 定位 |
|---|---|---|---|---|
| **TUI** | `devo --tui` | Go bubbletea TUI | 同进程 goroutine 启动 HTTP 服务 | 极客首选、SSH 远程、终端原生 |
| **Web** | `devo --web` | Vue 3 Browser 模式 | 同进程 goroutine 启动 HTTP 服务，自动打开浏览器 | 控制中心、完整功能面板 |
| **VSCode** | 扩展命令 `devo.open` | Vue 3 `?mode=vscode` 极简布局 | 扩展 spawn `devo` 子进程，webview 加载前端 | 编辑器内嵌、随叫随到 |
| **Desktop** | 双击 `Devo.exe` | Vue 3 Browser 模式 | Electron spawn `devo` 子进程，BrowserWindow 加载前端 | 独立工作台、全屏沉浸 |

**核心原则**：所有端共享同一套 Go 后端和同一套 Vue 前端，差异仅在于启动方式和 UI 布局模式。

---

## 二、架构全景图

### 2.1 启动流程概览

```
用户双击 Devo.exe
       ↓
┌──────────────────────────────────┐
│  欢迎页（本地 HTML）              │
│  · 品牌 Logo + 标题              │
│  · 「打开文件夹」按钮              │
│  · 最近打开的历史目录列表          │
│  · 用户选择工作目录               │
└──────────────┬───────────────────┘
               ↓ 用户选定目录
┌──────────────────────────────────┐
│  main.js 启动 Go 后端             │
│  1. 分配随机端口                  │
│  2. execFile("devo", [            │
│       "--port", port,             │
│       "--workspace", workspace    │
│     ])                            │
│  3. 轮询等待 Go 后端就绪           │
│  4. 创建 BrowserWindow            │
│  5. 加载 http://localhost:{port}/ │
└──────────────┬───────────────────┘
               ↓ HTTP + SSE
┌──────────────────────────────────┐
│  Go 后端（devo.exe 子进程）       │
│  · 工作目录 = 用户选择的目录       │
│  · 核心层 / 任务处理层 / 接口层    │
│  · 静态文件服务：Vue 前端 dist    │
└──────────────┬───────────────────┘
               ↓ HTTP + SSE
┌──────────────────────────────────┐
│  BrowserWindow（渲染进程）         │
│  · 加载 http://localhost:{port}/ │
│  · Vue 3 Browser 模式             │
└──────────────────────────────────┘
```

### 2.2 完整架构图

```
┌──────────────────────────────────────────────────────────────┐
│                   Electron 桌面应用                           │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  main.js（Electron 主进程）                           │   │
│  │                                                      │   │
│  │  阶段一：欢迎页                                      │   │
│  │  1. 创建 BrowserWindow，加载本地 welcome.html        │   │
│  │  2. 用户点击「打开文件夹」→ 系统原生选择对话框        │   │
│  │  3. 用户点击历史记录 → 直接使用历史路径               │   │
│  │                                                      │   │
│  │  阶段二：启动后端                                    │   │
│  │  4. 分配随机端口                                      │   │
│  │  5. execFile("devo", ["--port", p, "--workspace", w]) │   │
│  │  6. 轮询等待 Go 后端就绪                               │   │
│  │  7. BrowserWindow.loadURL(http://localhost:{port}/)   │   │
│  │                                                      │   │
│  │  生命周期管理：                                        │   │
│  │  · 欢迎页关闭 → app.quit                             │   │
│  │  · 主窗口关闭 → kill Go 子进程                        │   │
│  │  · Go 崩溃 → 弹窗提示重启                             │   │
│  │  · app.quit → 清理所有子进程                          │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │ spawn                              │
│  ┌──────────────────────▼───────────────────────────────┐   │
│  │  Go 后端（devo.exe 子进程）                           │   │
│  │  · --workspace 指定工作目录                           │   │
│  │  · 核心层：AgentLoop / 会话管理 / 审批门控            │   │
│  │  · 任务处理层：工具集 / LLM 客户端 / Python 执行器    │   │
│  │  · 接口层：REST API + SSE 事件流                      │   │
│  │  · 静态文件服务：Vue 前端 dist                        │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │ HTTP + SSE                         │
│  ┌──────────────────────▼───────────────────────────────┐   │
│  │  BrowserWindow（渲染进程）                             │   │
│  │  · 加载 http://localhost:{port}/                      │   │
│  │  · Vue 3 Browser 模式（三栏完整布局）                  │   │
│  │  · 无 ?mode 参数，走默认 Browser 分支                  │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## 三、欢迎页设计

### 3.1 设计目标

解决 Electron 启动后直接 spawn devo.exe 导致工作目录为临时路径（如 `C:\Users\bean\AppData\LocalTemp\3FkmngGh78aS4hisC23vM8ACugX`）的问题。

借鉴 VSCode 的启动体验：**先展示欢迎页，用户选择工作目录后，再启动后端**。

### 3.2 启动流程

```
用户双击 Devo.exe
       ↓
Electron 创建 BrowserWindow，加载本地 welcome.html
  （此时 Go 后端尚未启动，欢迎页完全由 Electron 独立渲染）
       ↓
用户在欢迎页选择工作目录：
  · 点击「打开文件夹」→ 系统原生 dialog.showOpenDialog
  · 点击历史记录中的路径 → 直接使用该路径
       ↓
main.js 收到工作目录路径后：
  1. 分配随机端口
  2. spawn("devo", ["--port", port, "--workspace", workspacePath])
  3. 轮询 GET /api/v1/sessions 等待就绪
  4. BrowserWindow.loadURL(http://localhost:{port}/)
       ↓
进入主界面（Vue 3 Browser 模式，三栏完整布局）
```

### 3.3 欢迎页线框设计

```
╔══════════════════════════════════════════════════════════════════════╗
║  ● ● ●  Devo                                           ─  □  ×    ║
╠══════════════════════════════════════════════════════════════════════╣
║                                                                      ║
║                                                                      ║
║                                                                      ║
║                          ┌─────────────┐                             ║
║                          │             │                             ║
║                          │    🦎        │                             ║
║                          │   LOGO       │                             ║
║                          │  (64×64)     │                             ║
║                          │             │                             ║
║                          └─────────────┘                             ║
║                                                                      ║
║                      D e v o                                         ║
║                 AI Coding Agent                                      ║
║           本地运行 · 安全可控 · 智能编码                               ║
║                                                                      ║
║                                                                      ║
║          ┌──────────────────────────────────────┐                    ║
║          │                                      │                    ║
║          │        📂   打开文件夹                 │                    ║
║          │                                      │                    ║
║          └──────────────────────────────────────┘                    ║
║                                                                      ║
║                                                                      ║
║  ┌───────────────────────────────────────────────────────────────┐   ║
║  │                                                               │   ║
║  │  ┌──────────────────────────────────────────────────────┐    │   ║
║  │  │                                                      │    │   ║
║  │  │  📁  my-react-app                               ×    │    │   ║
║  │  │      D:\Projects\my-react-app                        │    │   ║
║  │  │                                                      │    │   ║
║  │  └──────────────────────────────────────────────────────┘    │   ║
║  │                                                               │   ║
║  │  ┌──────────────────────────────────────────────────────┐    │   ║
║  │  │                                                      │    │   ║
║  │  │  📁  Devo                                      ×    │    │   ║
║  │  │      C:\Users\bean\Desktop\Devo                      │    │   ║
║  │  │                                                      │    │   ║
║  │  └──────────────────────────────────────────────────────┘    │   ║
║  │                                                               │   ║
║  │  ┌──────────────────────────────────────────────────────┐    │   ║
║  │  │                                                      │    │   ║
║  │  │  📁  backend-api                               ×    │    │   ║
║  │  │      D:\Work\backend-api                             │    │   ║
║  │  │                                                      │    │   ║
║  │  └──────────────────────────────────────────────────────┘    │   ║
║  │                                                               │   ║
║  └───────────────────────────────────────────────────────────────┘   ║
║                                                                      ║
║                                                                      ║
╚══════════════════════════════════════════════════════════════════════╝
```

### 3.4 视觉重心分布

```
         ┌──────────────────────────────────────┐
         │                                      │
         │            🦎   LOGO                 │   ← 品牌锚点
         │           D e v o                    │
         │       AI Coding Agent                │
         │                                      │
         │    ┌──────────────────────┐          │
         │    │  📂  打开文件夹       │          │   ← 唯一 CTA
         │    └──────────────────────┘          │
         │                                      │
         │  ┌──────────────────────────────┐    │
         │  │ 📁 project-1             ×   │    │
         │  │ 📁 project-2             ×   │    │   ← 历史记录
         │  │ 📁 project-3             ×   │    │
         │  └──────────────────────────────┘    │
         │                                      │
         └──────────────────────────────────────┘
              ↑                          ↑
         品牌区 40%                  历史区 60%
```

### 3.5 配色方案（深色主题）

| 用途 | 色值 | 说明 |
|---|---|---|
| 背景 | `#1b1b2f` | 深色背景，与主应用一致 |
| 卡片 | `#252540` | 历史记录卡片背景 |
| 历史项 hover | `#2f2f4a` | 悬浮时微亮 |
| 主按钮 | `#7c5cfc` | 品牌紫色 |
| 主按钮 hover | `#6a4de6` | 悬浮时加深 |
| 按钮文字 | `#ffffff` | 白色 |
| 主文字 | `#e4e4f0` | 浅灰白 |
| 次级文字 | `#7c7c9a` | 灰色 |
| 删除 × | `#5c5c78` | 默认灰色 |
| 删除 × hover | `#ff5c5c` | 悬浮变红 |
| 分割线 | `#2e2e44` | 微亮分割 |

### 3.6 字体层级

| 层级 | 规格 | 用途 |
|---|---|---|
| Logo 标题 | 24px / 700 / letter-spacing: 4px | 品牌名 |
| 副标题 | 14px / 400 / 次级色 | 产品描述 |
| 按钮文字 | 15px / 500 | 主按钮 |
| 分区标题 | 11px / 600 / 全大写 / letter-spacing: 1px | 「最近打开」标签 |
| 历史项目名 | 13px / 500 / 主文字色 | 文件夹名称 |
| 历史路径 | 12px / 次级色 | 完整路径 |

### 3.7 交互细节

| 元素 | 交互行为 |
|---|---|
| **打开文件夹** | 点击 → `dialog.showOpenDialog({ properties: ['openDirectory'] })` → 选定后 IPC 通知主进程启动 devo |
| **历史记录项** | 整行可点击，hover 时背景变亮 + × 按钮浮现（opacity 0→1, 150ms） |
| **× 删除** | hover 变红，点击后该项从列表滑出 + 淡出（300ms ease-out），同步从 localStorage 移除 |
| **空状态** | 无历史记录时，该区域显示居中灰色提示：「还没有打开过任何项目」 |
| **页面入场** | Logo 和标题先淡入（opacity 0→1, 400ms ease-out），按钮和列表随后 stagger 淡入（各延迟 100ms） |

### 3.8 技术实现

欢迎页是一个**纯静态 HTML 页面**（`electron/welcome.html`），由 Electron 的 BrowserWindow 直接加载，**不依赖 Go 后端**：

```html
<!-- electron/welcome.html -->
<div class="welcome">
  <header class="brand">
    <img class="logo" src="resources/icon.png" />
    <h1>Devo</h1>
    <p class="tagline">AI Coding Agent</p>
    <p class="subtitle">本地运行 · 安全可控 · 智能编码</p>
  </header>

  <button class="btn-primary" id="btn-open-folder">
    📂 打开文件夹
  </button>

  <section class="recent" id="section-recent">
    <h2 class="section-title">最近打开</h2>
    <ul class="recent-list" id="recent-list">
      <!-- 动态渲染历史记录 -->
    </ul>
    <p class="empty-hint" id="empty-hint" style="display:none">
      还没有打开过任何项目
    </p>
  </section>
</div>
```

历史记录通过 `localStorage` 持久化，渲染进程通过 `ipcRenderer` 与主进程通信：

| IPC 通道 | 方向 | 用途 |
|---|---|---|
| `select-folder` | 渲染 → 主 | 请求打开系统文件夹选择对话框 |
| `folder-selected` | 主 → 渲染 | 返回用户选择的文件夹路径 |
| `open-recent` | 渲染 → 主 | 用户点击历史记录，携带路径 |
| `launch-devo` | 主进程内部 | 收到路径后，启动 devo 子进程 |

### 3.9 与现有文档的差异

| 项目 | 原设计 | 新设计 |
|---|---|---|
| 启动时机 | 双击应用 → 立刻 spawn devo | 双击应用 → 欢迎页 → 用户选目录 → spawn devo |
| 工作目录 | 未指定，默认使用临时目录 | 通过 `--workspace` 参数明确传入 |
| 欢迎页 | 无 | 纯静态 HTML，独立于 Go 后端 |
| 前端改动 | 零改动 | 零改动（欢迎页是 Electron 专属，不涉及 Vue 前端） |

---

## 四、前端复用策略

### 4.1 当前模式分流逻辑

[usePlatform.ts](file:///C:/Users/bean/Desktop/Devo/web/src/composables/usePlatform.ts) 的检测逻辑：

```typescript
function detectMode(): void {
  const params = new URLSearchParams(window.location.search)
  const mode = params.get('mode')
  uiStore.setVscodeMode(mode === 'vscode')
}
```

- `?mode=vscode` → VSCode 极简布局
- 无 `?mode` 参数（或任何未知值） → Browser 完整布局

### 4.2 Electron 的策略：零前端改动

Electron 的 BrowserWindow 加载 `http://localhost:{port}/`，不带任何 `?mode` 参数，自动进入 Browser 模式（三栏完整布局），**不需要修改前端任何代码**。

```
Electron 加载:  http://localhost:52341/
                  ↓
usePlatform.detectMode()
  → params.get('mode') === null
  → isVscodeMode = false
  → isBrowserMode = true
  → 渲染 BrowserLayout（三栏面板）
```

### 4.3 未来迭代：Desktop 专属布局

当需要对 Desktop 端做差异化体验时，再新增 `?mode=desktop` 分支：

```typescript
// 未来扩展（现阶段不实现）
function detectMode(): void {
  const params = new URLSearchParams(window.location.search)
  const mode = params.get('mode')
  uiStore.setVscodeMode(mode === 'vscode')
  uiStore.setDesktopMode(mode === 'desktop')  // 新增
}
```

Desktop 专属布局可包含：
- 无边框窗口的自定义标题栏
- 系统托盘入口
- 原生通知样式
- 全局快捷键提示

**但现阶段 = 直接复用 Browser 模式，零前端改动。**

---

## 五、进程管理：借鉴 VSCode 扩展

Electron 的 `main.js` 与 VSCode 扩展的 `extension.ts` 在进程管理上逻辑几乎完全一致，以下逐项对照。

### 5.1 启动流程对比

| 步骤 | VSCode 扩展 | Electron |
|---|---|---|
| 触发 | 用户点击工具栏图标或命令面板 | 用户双击应用图标，先展示欢迎页 |
| 端口分配 | `net.createServer().listen(0)` 获取随机端口 | 同左 |
| 启动后端 | `child_process.execFile("devo", ...)` | `execFile("devo", ["--port", p, "--workspace", w])` |
| 等待就绪 | 轮询 `GET /api/v1/sessions` | 同左 |
| 打开界面 | `createWebviewPanel()` 加载 `?mode=vscode` | `createWindow()` 加载 Browser 模式 |
| 超时处理 | 10s 超时，弹窗提示失败 | 同左 |

### 5.2 生命周期绑定

| 事件 | VSCode 扩展 | Electron |
|---|---|---|
| 启动 | `activate()` → spawn 子进程 | `app.whenReady()` → 显示欢迎页 → 用户选择 → spawn 子进程 |
| 关闭 | `panel.onDidDispose()` → kill | `app.on("window-all-closed")` → kill |
| 崩溃 | `process.on("exit")` → 弹窗提示重启 | `process.on("exit")` → 弹窗提示重启 |
| 清理 | `cleanupInstance()` → kill + dispose | 同理 → kill + 清理 |

### 5.3 崩溃恢复流程

```
Go 子进程异常退出
  ↓
Electron 检测到 exit 事件（非零退出码）
  ↓
BrowserWindow 显示 crash 提示页面
  ↓
dialog.showMessageBox({
  type: 'error',
  title: 'Devo 后端已退出',
  message: '进程退出码: ${code}，是否重新启动？',
  buttons: ['重新启动', '退出']
})
  ↓
用户点击「重新启动」
  → 重新分配端口
  → 重新 spawn devo 子进程
  → 轮询等待就绪
  → BrowserWindow.loadURL(新地址)
```

### 5.4 可复用代码

VSCode 扩展和 Electron main.js 均为 Node.js 环境，以下逻辑可直接复用（未来可抽取为共享包）：

| 逻辑 | 说明 |
|---|---|
| 端口分配 | `net.createServer().listen(0)` 获取空闲端口 |
| 子进程启动 | `child_process.execFile` + 参数拼接 |
| 健康检查 | `GET /api/v1/sessions` 轮询等待就绪 |
| 进程退出监听 | `process.on("exit", code => ...)` |
| 清理逻辑 | `process.kill()` + 状态重置 |

---

## 六、项目目录结构

```
devo/
├── cmd/devo/                  ← Go 后端入口（不动）
├── internal/                  ← Go 核心逻辑（不动）
├── web/                       ← Vue 前端（不动）
├── vscode-extension/          ← VSCode 扩展（不动）
├── electron/                  ← 🆕 Electron 桌面端
│   ├── main.js                ← 主进程：欢迎页 + 进程管理 + 窗口创建
│   ├── welcome.html           ← 欢迎页（纯静态，独立于 Go 后端）
│   ├── welcome.css            ← 欢迎页样式
│   ├── welcome.js             ← 欢迎页渲染进程逻辑
│   ├── package.json           ← Electron 依赖
│   ├── resources/             ← 图标等静态资源
│   │   └── icon.png
│   └── .gitignore
├── docs/                      ← 文档（不动）
├── Makefile                   ← 添加 desktop 构建目标
└── go.mod
```

---

## 七、构建与打包

### 7.1 开发阶段

```bash
# 1. 先编译 Go 后端
make build-go    # → 输出 devo.exe

# 2. 进入 electron 目录，安装依赖
cd electron && npm install

# 3. 启动 Electron 开发模式
npm run dev      # → electron . （从 PATH 查找 devo）
```

### 7.2 生产打包

```bash
# 完整构建
make desktop     # → 编译 Go → 打包 Vue → 打包 Electron
```

Makefile 新增目标：

```makefile
desktop-build-go:
	go build -o electron/resources/bin/devo.exe

desktop: desktop-build-go build-web
	cd electron && npm run package
```

打包配置（`electron/package.json`）：

```json
{
  "build": {
    "appId": "com.devo.desktop",
    "productName": "Devo",
    "win": {
      "target": "nsis",
      "icon": "resources/icon.png"
    },
    "extraResources": [
      {
        "from": "resources/bin",
        "to": "bin",
        "filter": ["**/*"]
      }
    ]
  }
}
```

### 7.3 Go 二进制路径策略

| 环境 | 查找路径 | 说明 |
|---|---|---|
| 开发模式 | `PATH` 中的 `devo` 命令 | 用户自行安装，快速迭代 |
| 打包后 | `process.resourcesPath/bin/devo.exe` | extraResources 打包，单文件分发 |

```javascript
function getDevoPath() {
  if (app.isPackaged) {
    return path.join(process.resourcesPath, 'bin', 'devo.exe')
  }
  // 开发模式：从 PATH 查找
  return 'devo'
}
```

---

## 八、与 `electron-go-vue` demo 的差异

之前在 `electron-go-vue` 项目中看到的 demo 是一个骨架模板，Devo 的 Electron 实现与其有本质区别：

| 对比维度 | electron-go-vue demo | Devo Electron |
|---|---|---|
| Go 后端 | 20 行 HTTP Server，返回 "Hello World" | 完整 AI Agent 系统（AgentLoop、工具集、审批、记忆、技能等） |
| Vue 前端 | 无（只有 README 中提到 Vue） | 完整的 Vue 3 应用，20+ 组件，8 个 Pinia Store |
| 欢迎页 | 无 | 纯静态欢迎页，用户选择工作目录后再启动后端 |
| 进程管理 | 基础：启动 → 等待 1s → 创建窗口 | 完整：欢迎页 → 目录选择 → 端口分配 → 健康检查轮询 → 创建窗口 → 崩溃恢复 |
| 工作目录 | 使用默认临时目录 | 通过 `--workspace` 参数明确指定 |
| 打包 | electron-builder portable | electron-builder nsis，extraResources 内嵌 Go 二进制 |
| 模式分流 | 无 | 复用现有 Browser 模式，未来可扩展 Desktop 专属布局 |

---

## 九、实施路线图

### 第一阶段：MVP（1-2 天）

- [ ] 在 `electron/` 下新建 Electron 项目
- [ ] 创建 `welcome.html` 欢迎页（品牌区 + 打开文件夹 + 历史记录）
- [ ] `main.js` 实现：显示欢迎页 → 用户选择目录 → 分配端口 → spawn devo（含 `--workspace` 参数）→ 健康检查 → 创建 BrowserWindow
- [ ] 历史记录持久化（localStorage）
- [ ] 加载 `http://localhost:{port}/`（Browser 模式，零前端改动）
- [ ] 窗口关闭时 kill 子进程
- [ ] 验证：聊天、工具调用、审批全流程正常

### 第二阶段：体验打磨（2-3 天）

- [ ] 崩溃恢复：Go 子进程异常退出时弹窗提示重启
- [ ] 开发模式 vs 打包模式路径切换
- [ ] 窗口尺寸记忆（关闭时保存，打开时恢复）
- [ ] 欢迎页入场动画
- [ ] 打包脚本：`make desktop`

### 第三阶段：Desktop 独占功能（按需，后续迭代）

- [ ] 自定义 Desktop 布局（`?mode=desktop`）
- [ ] 系统托盘图标（右键菜单：显示窗口 / 退出）
- [ ] 原生通知（审批请求、任务完成）
- [ ] 全局快捷键（Ctrl+Shift+D 唤起窗口）
- [ ] 开机自启
- [ ] 自动更新（electron-updater）

---

## 十、关键设计决策总结

| 决策 | 方案 | 理由 |
|---|---|---|
| Electron 角色 | 纯壳层，不放业务逻辑 | 保持 Go 后端 + Vue 前端不变，Electron 只做进程管理 |
| 启动流程 | 欢迎页 → 用户选择目录 → 启动后端 | 避免工作目录为临时路径，与 VSCode 体验一致 |
| 工作目录 | 通过 `--workspace` 参数传入 devo | 用户明确指定，Go 后端在正确目录下运行 |
| 欢迎页 | 纯静态 HTML（welcome.html），独立于 Go 后端 | Electron 自行渲染，启动快，不依赖后端 |
| 历史记录 | localStorage 持久化 | 简单可靠，无需额外依赖 |
| 前端复用 | 复用 Browser 模式（三栏完整布局） | 零前端改动，功能最全，适合独立窗口 |
| 进程管理 | 借鉴 VSCode 扩展的成熟实现 | 逻辑一致，减少重复设计，未来可抽取共享包 |
| 启动方式 | Electron 从 PATH 或打包路径查找 devo 二进制 | 开发阶段快速迭代，打包后内嵌分发 |
| 端口策略 | 随机端口 + `net.Listen(":0")` | 与 TUI 和 VSCode 扩展保持一致，避免冲突 |
| 崩溃恢复 | 弹窗提示 + 用户确认重启 | 与 VSCode 扩展体验一致 |
| 模式分流 | 不加 `?mode` 参数 → 走 Browser 默认分支 | 现有 `usePlatform.ts` 已支持，无需改动 |
| 未来 Desktop 模式 | 预留 `?mode=desktop` 扩展点 | 现阶段不实现，迭代时再扩展 |