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

```
┌──────────────────────────────────────────────────────────────┐
│                   Electron 桌面应用                           │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  main.js（Electron 主进程）                           │   │
│  │                                                      │   │
│  │  1. 分配随机端口                                      │   │
│  │  2. child_process.execFile("devo", ["-port", port])   │   │
│  │  3. 轮询等待 Go 后端就绪                               │   │
│  │  4. 创建 BrowserWindow                                │   │
│  │  5. 加载 http://localhost:{port}/                     │   │
│  │                                                      │   │
│  │  生命周期管理：                                        │   │
│  │  · 窗口关闭 → kill Go 子进程                          │   │
│  │  · Go 崩溃 → 弹窗提示重启                             │   │
│  │  · app.quit → 清理所有子进程                          │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │ spawn                              │
│  ┌──────────────────────▼───────────────────────────────┐   │
│  │  Go 后端（devo.exe 子进程）                           │   │
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

## 三、前端复用策略

### 3.1 当前模式分流逻辑

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

### 3.2 Electron 的策略：零前端改动

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

### 3.3 未来迭代：Desktop 专属布局

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

## 四、进程管理：借鉴 VSCode 扩展

Electron 的 `main.js` 与 VSCode 扩展的 `extension.ts` 在进程管理上逻辑几乎完全一致，以下逐项对照。

### 4.1 启动流程对比

| 步骤 | VSCode 扩展 | Electron |
|---|---|---|
| 触发 | 用户点击工具栏图标或命令面板 | 用户双击应用图标 |
| 端口分配 | `net.createServer().listen(0)` 获取随机端口 | 同左 |
| 启动后端 | `child_process.execFile("devo", ...)` | 同左 |
| 等待就绪 | 轮询 `GET /api/v1/sessions` | 同左 |
| 打开界面 | `createWebviewPanel()` 加载 `?mode=vscode` | `createWindow()` 加载 Browser 模式 |
| 超时处理 | 10s 超时，弹窗提示失败 | 同左 |

### 4.2 生命周期绑定

| 事件 | VSCode 扩展 | Electron |
|---|---|---|
| 启动 | `activate()` → spawn 子进程 | `app.whenReady()` → spawn 子进程 |
| 关闭 | `panel.onDidDispose()` → kill | `app.on("window-all-closed")` → kill |
| 崩溃 | `process.on("exit")` → 弹窗提示重启 | `process.on("exit")` → 弹窗提示重启 |
| 清理 | `cleanupInstance()` → kill + dispose | 同理 → kill + 清理 |

### 4.3 崩溃恢复流程

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

### 4.4 可复用代码

VSCode 扩展和 Electron main.js 均为 Node.js 环境，以下逻辑可直接复用（未来可抽取为共享包）：

| 逻辑 | 说明 |
|---|---|
| 端口分配 | `net.createServer().listen(0)` 获取空闲端口 |
| 子进程启动 | `child_process.execFile` + 参数拼接 |
| 健康检查 | `GET /api/v1/sessions` 轮询等待就绪 |
| 进程退出监听 | `process.on("exit", code => ...)` |
| 清理逻辑 | `process.kill()` + 状态重置 |

---

## 五、项目目录结构

```
devo/
├── cmd/devo/                  ← Go 后端入口（不动）
├── internal/                  ← Go 核心逻辑（不动）
├── web/                       ← Vue 前端（不动）
├── vscode-extension/          ← VSCode 扩展（不动）
├── electron/                  ← 🆕 Electron 桌面端
│   ├── main.js                ← 主进程：进程管理 + 窗口创建
│   ├── package.json           ← Electron 依赖
│   ├── resources/             ← 图标等静态资源
│   │   └── icon.png
│   └── .gitignore
├── docs/                      ← 文档（不动）
├── Makefile                   ← 添加 desktop 构建目标
└── go.mod
```

---

## 六、构建与打包

### 6.1 开发阶段

```bash
# 1. 先编译 Go 后端
make build-go    # → 输出 devo.exe

# 2. 进入 electron 目录，安装依赖
cd electron && npm install

# 3. 启动 Electron 开发模式
npm run dev      # → electron . （从 PATH 查找 devo）
```

### 6.2 生产打包

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

### 6.3 Go 二进制路径策略

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

## 七、与 `electron-go-vue` demo 的差异

之前在 `electron-go-vue` 项目中看到的 demo 是一个骨架模板，Devo 的 Electron 实现与其有本质区别：

| 对比维度 | electron-go-vue demo | Devo Electron |
|---|---|---|
| Go 后端 | 20 行 HTTP Server，返回 "Hello World" | 完整 AI Agent 系统（AgentLoop、工具集、审批、记忆、技能等） |
| Vue 前端 | 无（只有 README 中提到 Vue） | 完整的 Vue 3 应用，20+ 组件，8 个 Pinia Store |
| 进程管理 | 基础：启动 → 等待 1s → 创建窗口 | 完整：端口分配 → 健康检查轮询 → 创建窗口 → 崩溃恢复 |
| 打包 | electron-builder portable | electron-builder nsis，extraResources 内嵌 Go 二进制 |
| 模式分流 | 无 | 复用现有 Browser 模式，未来可扩展 Desktop 专属布局 |

---

## 八、实施路线图

### 第一阶段：MVP（1-2 天）

- [ ] 在 `electron/` 下新建 Electron 项目
- [ ] `main.js` 实现：端口分配 → spawn devo → 健康检查 → 创建 BrowserWindow
- [ ] 加载 `http://localhost:{port}/`（Browser 模式，零前端改动）
- [ ] 窗口关闭时 kill 子进程
- [ ] 验证：聊天、工具调用、审批全流程正常

### 第二阶段：体验打磨（2-3 天）

- [ ] 崩溃恢复：Go 子进程异常退出时弹窗提示重启
- [ ] 开发模式 vs 打包模式路径切换
- [ ] 窗口尺寸记忆（关闭时保存，打开时恢复）
- [ ] 打包脚本：`make desktop`

### 第三阶段：Desktop 独占功能（按需，后续迭代）

- [ ] 自定义 Desktop 布局（`?mode=desktop`）
- [ ] 系统托盘图标（右键菜单：显示窗口 / 退出）
- [ ] 原生通知（审批请求、任务完成）
- [ ] 全局快捷键（Ctrl+Shift+D 唤起窗口）
- [ ] 开机自启
- [ ] 自动更新（electron-updater）

---

## 九、关键设计决策总结

| 决策 | 方案 | 理由 |
|---|---|---|
| Electron 角色 | 纯壳层，不放业务逻辑 | 保持 Go 后端 + Vue 前端不变，Electron 只做进程管理 |
| 前端复用 | 复用 Browser 模式（三栏完整布局） | 零前端改动，功能最全，适合独立窗口 |
| 进程管理 | 借鉴 VSCode 扩展的成熟实现 | 逻辑一致，减少重复设计，未来可抽取共享包 |
| 启动方式 | Electron 从 PATH 或打包路径查找 devo 二进制 | 开发阶段快速迭代，打包后内嵌分发 |
| 端口策略 | 随机端口 + `net.Listen(":0")` | 与 TUI 和 VSCode 扩展保持一致，避免冲突 |
| 崩溃恢复 | 弹窗提示 + 用户确认重启 | 与 VSCode 扩展体验一致 |
| 模式分流 | 不加 `?mode` 参数 → 走 Browser 默认分支 | 现有 `usePlatform.ts` 已支持，无需改动 |
| 未来 Desktop 模式 | 预留 `?mode=desktop` 扩展点 | 现阶段不实现，迭代时再扩展 |