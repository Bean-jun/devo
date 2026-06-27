# Devo Web 前端测试文档

**版本**：1.3.0

**定位**：本文档定义 Devo Web 前端（Vue 3 + Vite + TypeScript）的测试策略、测试架构和测试用例清单。测试分为两层：**Vitest** 负责单元测试和组件测试，**Playwright** 负责端到端（E2E）测试。

---

## 1. 测试策略

### 1.1 测试金字塔

```
           ┌──────────┐
           │   E2E    │  Playwright · 浏览器真实环境 · 少量关键路径
           │   ~15%   │
           ├──────────┤
           │  组件测试 │  Vitest + @vue/test-utils · 组件隔离 · 覆盖核心交互
           │   ~35%   │
           ├──────────┤
           │  单元测试 │  Vitest · 纯函数/Store/Composable · 覆盖所有逻辑
           │   ~50%   │
           └──────────┘
```

### 1.2 测试原则

| 原则 | 说明 |
| :--- | :--- |
| **可重复** | 每次运行结果一致，不依赖外部状态（网络、数据库、时间） |
| **隔离** | 每个测试用例独立，不共享可变状态，不依赖执行顺序 |
| **快速** | 单元测试 < 10ms/用例，组件测试 < 100ms/用例，整体 < 30s |
| **可读** | 测试名描述行为，使用 AAA 模式（Arrange / Act / Assert） |
| **Mock 边界** | 只 Mock 外部依赖（API、SSE、浏览器 API），不 Mock 内部模块 |
| **覆盖率** | 行覆盖率 ≥ 80%，分支覆盖率 ≥ 70%，核心模块 ≥ 90% |

### 1.3 测试职责划分

| 层级 | 框架 | 测试什么 | 不测试什么 |
| :--- | :--- | :--- | :--- |
| 单元测试 | Vitest | 纯函数、类型转换、格式化、Store 逻辑、Composable 独立逻辑 | DOM 渲染、组件交互 |
| 组件测试 | Vitest + @vue/test-utils | 组件渲染、Props/Emits、事件处理、条件渲染、v-model | 真实网络请求、跨组件协作 |
| E2E 测试 | Playwright | 完整用户流程、SSE 事件流、多组件协作、真实浏览器行为 | 内部实现细节 |

---

## 2. 测试环境配置

### 2.1 Vitest 配置

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  test: {
    // 测试环境：jsdom 模拟浏览器 DOM
    environment: 'jsdom',

    // 全局变量：无需在每个测试文件中 import { describe, it, expect }
    globals: true,

    // Setup 文件：全局 Mock、扩展匹配器
    setupFiles: ['./src/test/setup.ts'],

    // 覆盖率配置
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      reportsDirectory: './coverage',
      include: ['src/**/*.{ts,vue}'],
      exclude: [
        'src/main.ts',
        'src/types/**',
        'src/test/**',
        '**/*.d.ts',
      ],
      thresholds: {
        lines: 80,
        branches: 70,
        functions: 80,
        statements: 80,
      },
    },

    // 别名（与 vite.config.ts 保持一致）
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
    },

    // CSS 处理：测试中忽略 CSS 导入
    css: {
      modules: {
        classNameStrategy: 'non-scoped',
      },
    },
  },
})
```

### 2.2 全局 Setup 文件

```typescript
// src/test/setup.ts
import { vi } from 'vitest'

// 1. Mock 浏览器 API（jsdom 不提供的）
// EventSource Mock
class MockEventSource {
  static instances: MockEventSource[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  url: string
  readyState: number = 0 // CONNECTING

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
  }

  close() {
    this.readyState = 2 // CLOSED
  }

  // 测试辅助：模拟收到消息
  dispatchEvent(type: string, data: string) {
    if (type === 'open' && this.onopen) this.onopen()
    if (type === 'message' && this.onmessage) {
      this.onmessage(new MessageEvent('message', { data }))
    }
    if (type === 'error' && this.onerror) this.onerror()
  }

  static resetAll() {
    MockEventSource.instances = []
  }
}

// 将 Mock 挂载到全局
globalThis.EventSource = MockEventSource as any

// 2. Mock fetch
globalThis.fetch = vi.fn()

// 3. Mock matchMedia（jsdom 不提供）
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// 4. Mock ResizeObserver
globalThis.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// 5. Mock IntersectionObserver
globalThis.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
  root: null,
  rootMargin: '',
  thresholds: [],
}))

// 6. Mock scrollTo
Element.prototype.scrollTo = vi.fn() as any
Element.prototype.scrollIntoView = vi.fn() as any
```

### 2.3 Playwright 配置

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // 测试目录
  testDir: './e2e',

  // 全局超时
  timeout: 30_000,

  // 期望超时
  expect: {
    timeout: 5_000,
  },

  // 失败重试（本地开发不重试，保持快速反馈）
  retries: 0,

  // 并行执行
  fullyParallel: true,
  workers: 1,

  // 报告器
  reporter: [
    ['html', { outputFolder: './playwright-report' }],
    ['json', { outputFile: './test-results/results.json' }],
  ],

  use: {
    // 基础 URL（指向 devo --web 启动的服务）
    baseURL: 'http://localhost:8080',

    // 截图：仅在失败时
    screenshot: 'only-on-failure',

    // 视频：仅在失败时保留
    video: 'retain-on-failure',

    // 追踪：仅在首次重试时
    trace: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    // macOS 仅在有 macOS 执行环境时启用
    // {
    //   name: 'webkit',
    //   use: { ...devices['Desktop Safari'] },
    // },
  ],

  // 启动前启动 Devo 服务
  webServer: {
    command: 'cd .. && go run ./cmd/devo --web --port 8080',
    port: 8080,
    timeout: 15_000,
    reuseExistingServer: true,
  },
})
```

### 2.4 package.json 测试脚本

```json
{
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage",
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui",
    "test:e2e:report": "playwright show-report",
    "test:all": "npm run test && npm run test:e2e"
  }
}
```

---

## 3. 测试目录结构

```
web/
├── src/
│   ├── test/
│   │   ├── setup.ts                    # Vitest 全局 Setup
│   │   ├── mocks/
│   │   │   ├── api.ts                  # API Mock 工厂
│   │   │   ├── sse.ts                  # SSE Mock 工厂
│   │   │   └── stores.ts              # Store Mock 工厂
│   │   └── fixtures/
│   │       ├── sessions.ts            # 会话测试数据
│   │       ├── messages.ts            # 消息测试数据
│   │       ├── tools.ts               # 工具调用测试数据
│   │       ├── skills.ts             # 技能测试数据（v1.1.0 新增）
│   │       ├── memory.ts             # 记忆测试数据（v1.1.0 新增）
│   │       ├── mcp.ts                # MCP 工具测试数据（v1.1.0 新增）
│   │       ├── dashboard.ts          # 仪表盘测试数据（v1.1.0 新增）
│   │       └── workspaces.ts         # 工作区测试数据（v1.2.0 新增）
│   │
│   ├── utils/
│   │   ├── markdown.test.ts           # Markdown 渲染工具测试
│   │   ├── formatters.test.ts         # 格式化工具测试
│   │   └── constants.test.ts          # 常量测试
│   │
│   ├── stores/
│   │   ├── session.test.ts            # 会话 Store 测试
│   │   ├── chat.test.ts               # 聊天 Store 测试
│   │   ├── approval.test.ts           # 审批 Store 测试
│   │   ├── command.test.ts            # 命令 Store 测试
│   │   ├── ui.test.ts                 # UI Store 测试
│   │   ├── skills.test.ts            # 技能 Store 测试（v1.1.0 新增）
│   │   ├── memory.test.ts            # 记忆 Store 测试（v1.1.0 新增）
│   │   ├── dashboard.test.ts         # 仪表盘 Store 测试（v1.1.0 新增）
│   │   ├── settings.test.ts          # 设置 Store 测试（v1.1.0 新增）
│   │   └── mcp.test.ts               # MCP Store 测试（v1.1.0 新增）
│   │
│   ├── composables/
│   │   ├── useApi.test.ts             # API 封装测试
│   │   ├── useSSE.test.ts             # SSE 封装测试
│   │   ├── useSession.test.ts         # 会话操作测试
│   │   ├── useCommand.test.ts         # 命令面板逻辑测试
│   │   ├── useKeyboard.test.ts        # 键盘快捷键测试
│   │   ├── useAutoScroll.test.ts      # 自动滚动测试
│   │   ├── usePlatform.test.ts       # 平台检测测试（v1.1.0 新增）
│   │   ├── useSkills.test.ts         # 技能操作测试（v1.1.0 新增）
│   │   ├── useMemory.test.ts         # 记忆操作测试（v1.1.0 新增）
│   │   └── useMcp.test.ts            # MCP 工具操作测试（v1.1.0 新增）
│   │
│   └── components/
│       ├── layout/
│       │   ├── AppSidebar.test.ts     # 左侧栏（含 workspace 选择器 + 会话列表）（v1.2.0 更新）
│       │   ├── AppHeader.test.ts      # 顶部栏
│       │   ├── VscodeLayout.test.ts   # VSCode 布局（v1.2.0 更新）
│       │   ├── BrowserLayout.test.ts  # 浏览器布局（三栏）（v1.2.0 更新）
│       │   ├── RightPanel.test.ts     # 右侧面板 Tab 切换框架（v1.2.0 新增）
│       │   ├── GlobalModals.test.ts   # 全局弹窗复用组件（v1.2.0 新增）
│       │   ├── StatusBar.test.ts
│       │   └── ToastContainer.test.ts
│       ├── chat/
│       │   ├── InputArea.test.ts
│       │   ├── MessageBubble.test.ts
│       │   ├── ToolCallCard.test.ts
│       │   ├── ThinkingIndicator.test.ts
│       │   └── MessageList.test.ts
│       ├── command/
│       │   └── CommandPalette.test.ts
│       ├── modal/
│       │   ├── ApprovalModal.test.ts
│       │   ├── SessionPicker.test.ts
│       │   ├── RollbackPicker.test.ts
│       │   ├── HelpPanel.test.ts
│       │   ├── SkillInstallDialog.test.ts   # 技能安装弹窗（v1.1.0 新增）
│       │   └── MemoryEditDialog.test.ts     # 记忆编辑弹窗（v1.1.0 新增）
│       ├── panels/                              ← 【v1.2.0】替代原 views/，新增面板测试
│       │   ├── files/
│       │   │   └── FilesPanel.test.ts           # 文件树面板（v1.2.0 新增）
│       │   ├── skills/
│       │   │   ├── SkillsPanel.test.ts          # 技能管理面板（全局 + 项目两层）（v1.2.0 新增）
│       │   │   ├── SkillDetailPanel.test.ts     # 技能详情面板（v1.2.0 新增）
│       │   │   └── components/
│       │   │       ├── SkillCard.test.ts        # 技能卡片
│       │   │       ├── SkillInstallDialog.test.ts # 安装新技能弹窗
│       │   │       └── SkillSolidifyPanel.test.ts # 固化审批面板（v1.2.0 新增）
│       │   ├── memory/
│       │   │   ├── MemoryPanel.test.ts          # 记忆管理面板（全局 + 项目两层）（v1.2.0 新增）
│       │   │   └── components/
│       │   │       ├── MemoryCard.test.ts       # 记忆卡片
│       │   │       └── MemoryEditDialog.test.ts # 编辑记忆弹窗
│       │   ├── dashboard/
│       │   │   └── DashboardPanel.test.ts       # Token 用量仪表盘（v1.2.0 新增）
│       │   ├── settings/
│       │   │   ├── SettingsPanel.test.ts        # 设置面板（子 Tab：项目 / 全局）（v1.2.0 新增）
│       │   │   ├── ApprovalPolicyPanel.test.ts  # 审批策略配置（v1.2.0 新增）
│       │   │   └── McpSettingsPanel.test.ts     # MCP 工具管理（v1.2.0 新增）
│       │   └── terminal/
│       │       └── TerminalPanel.test.ts        # 终端面板（v1.2.0 新增）
│       ├── views/
│       │   └── ChatView.test.ts              # 聊天视图（v1.2.0 更新）
│       └── shared/
│           ├── DiffView.test.ts
│           ├── CodeBlock.test.ts
│           ├── MarkdownView.test.ts
│           └── Spinner.test.ts
│
├── e2e/
│   ├── fixtures/
│   │   └── test-data.ts               # E2E 测试数据
│   ├── helpers/
│   │   ├── api.ts                      # E2E API 辅助函数
│   │   └── assertions.ts              # 自定义断言
│   ├── session.spec.ts                # 会话管理 E2E
│   ├── chat.spec.ts                   # 聊天对话 E2E
│   ├── approval.spec.ts              # 审批流程 E2E
│   ├── command-palette.spec.ts        # 命令面板 E2E
│   ├── rollback.spec.ts              # 消息回滚 E2E
│   ├── mode-routing.spec.ts          # 模式分流 E2E（v1.2.0 更新：改为面板切换测试）
│   ├── skills.spec.ts                # 技能管理 E2E（v1.2.0 更新）
│   ├── memory.spec.ts                # 记忆管理 E2E（v1.2.0 更新）
│   ├── mcp-tools.spec.ts             # MCP 工具管理 E2E（v1.2.0 更新）
│   ├── dashboard.spec.ts             # 仪表盘 E2E（v1.2.0 更新）
│   ├── right-panel.spec.ts           # 右侧面板 Tab 切换 E2E（v1.2.0 新增）
│   ├── workspace-scope.spec.ts       # 全局/工作区双模式 E2E（v1.2.0 新增）
│   └── sidebar.spec.ts               # 左侧栏 workspace 选择 + 会话列表 E2E（v1.2.0 新增）
│
├── vitest.config.ts
├── playwright.config.ts
└── package.json
```

---

## 4. 测试编写规范

### 4.1 命名规范

```
测试文件：<模块名>.test.ts           # 例：session.test.ts, useApi.test.ts
E2E 文件：<功能名>.spec.ts           # 例：chat.spec.ts, approval.spec.ts

测试套件：describe('<模块名>', ...)
测试用例：it('should <行为描述> when <条件>', ...)

示例：
  describe('SessionStore', () => {
    it('should create a session with default values', ...)
    it('should return the current session via getter', ...)
    it('should throw an error when creating a session with duplicate name', ...)
  })
```

### 4.2 AAA 模式

```typescript
import { describe, it, expect, beforeEach } from 'vitest'

describe('Counter', () => {
  // Arrange：准备测试数据
  let counter: number

  beforeEach(() => {
    counter = 0
  })

  it('should increment by 1', () => {
    // Act：执行被测操作
    counter++

    // Assert：验证结果
    expect(counter).toBe(1)
  })
})
```

### 4.3 Store 测试模式

```typescript
// stores/session.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '@/stores/session'

describe('SessionStore', () => {
  beforeEach(() => {
    // 每个测试前重置 Pinia
    setActivePinia(createPinia())

    // Mock API 调用
    vi.stubGlobal('fetch', vi.fn())
  })

  describe('createSession', () => {
    it('should create a session and set it as current', async () => {
      const store = useSessionStore()

      const mockResponse = {
        id: 'sess-001',
        title: 'My Project',
        state: 'idle',
        created_at: '2026-01-01T00:00:00Z',
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      } as Response)

      await store.createSession('My Project')

      expect(store.currentSession).toMatchObject(mockResponse)
      expect(store.sessions).toHaveLength(1)
    })

    it('should handle API error gracefully', async () => {
      const store = useSessionStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      } as Response)

      await expect(store.createSession('Test')).rejects.toThrow()
      expect(store.currentSession).toBeNull()
    })
  })

  describe('switchSession', () => {
    it('should update currentSession when switching to existing session', async () => {
      const store = useSessionStore()

      store.sessions = [
        { id: 'sess-1', title: 'Session 1', state: 'idle' },
        { id: 'sess-2', title: 'Session 2', state: 'idle' },
      ] as any

      store.switchSession('sess-2')

      expect(store.currentSession?.id).toBe('sess-2')
    })

    it('should not switch to a non-existent session', () => {
      const store = useSessionStore()

      store.sessions = [{ id: 'sess-1', title: 'Session 1', state: 'idle' }] as any
      store.currentSession = store.sessions[0]

      store.switchSession('sess-999')

      expect(store.currentSession?.id).toBe('sess-1')
    })
  })
})
```

### 4.4 Composable 测试模式

```typescript
// composables/useSSE.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useSSE } from '@/composables/useSSE'

describe('useSSE', () => {
  beforeEach(() => {
    MockEventSource.resetAll()
  })

  it('should create an EventSource connection', () => {
    const { connect } = useSSE()

    connect('sess-001')

    const instances = MockEventSource.instances
    expect(instances).toHaveLength(1)
    expect(instances[0].url).toContain('sess-001')
  })

  it('should call onMessage handler when event received', () => {
    const { connect, onEvent } = useSSE()
    const handler = vi.fn()

    onEvent('message_chunk', handler)
    connect('sess-001')

    const instance = MockEventSource.instances[0]
    instance.dispatchEvent('message', 'event: message_chunk\ndata: {"content":"Hello"}')

    expect(handler).toHaveBeenCalledWith({ content: 'Hello' })
  })

  it('should auto-reconnect on error', () => {
    vi.useFakeTimers()

    const { connect } = useSSE()
    connect('sess-001')

    const instance = MockEventSource.instances[0]
    instance.dispatchEvent('error', '')

    // 第一个实例关闭
    expect(instance.readyState).toBe(2)

    // 等待重连延迟
    vi.advanceTimersByTime(2000)

    // 第二个实例创建
    expect(MockEventSource.instances).toHaveLength(2)

    vi.useRealTimers()
  })

  it('should disconnect without reconnect', () => {
    const { connect, disconnect } = useSSE()

    connect('sess-001')
    disconnect()

    const instance = MockEventSource.instances[0]
    instance.dispatchEvent('error', '')

    // 断开后不重连
    expect(MockEventSource.instances).toHaveLength(1)
  })
})
```

### 4.5 组件测试模式

```typescript
// components/chat/InputArea.test.ts
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import InputArea from '@/components/chat/InputArea.vue'

describe('InputArea', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render textarea and send button', () => {
    const wrapper = mount(InputArea)

    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="发送"]').exists()).toBe(true)
  })

  it('should emit send event with text content on Enter', async () => {
    const wrapper = mount(InputArea)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hello, world!')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeTruthy()
    expect(wrapper.emitted('send')![0]).toEqual(['Hello, world!'])
  })

  it('should not emit send event on Shift+Enter', async () => {
    const wrapper = mount(InputArea)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hello')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: true })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should clear textarea after sending', async () => {
    const wrapper = mount(InputArea)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Test message')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect((textarea.element as HTMLTextAreaElement).value).toBe('')
  })

  it('should show stop button when isProcessing is true', async () => {
    const wrapper = mount(InputArea, {
      props: {
        isProcessing: true,
      },
    })

    expect(wrapper.find('button[aria-label="停止"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="发送"]').exists()).toBe(false)
  })

  it('should disable input when isDisabled is true', () => {
    const wrapper = mount(InputArea, {
      props: {
        isDisabled: true,
      },
    })

    const textarea = wrapper.find('textarea')
    expect(textarea.attributes('disabled')).toBeDefined()
  })

  it('should show character count', async () => {
    const wrapper = mount(InputArea)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hi')

    const counter = wrapper.find('[data-test="char-count"]')
    expect(counter.text()).toContain('2')
  })

  it('should open command palette on /', async () => {
    const wrapper = mount(InputArea)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('/')
    await textarea.trigger('keydown', { key: '/' })

    expect(wrapper.emitted('open-command')).toBeTruthy()
  })

  it('should not send empty message', async () => {
    const wrapper = mount(InputArea)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('   ')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeFalsy()
  })
})
```

### 4.6 Playwright E2E 测试模式

```typescript
// e2e/chat.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Chat Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // 等待页面加载完成
    await page.waitForSelector('[data-test="chat-panel"]')
  })

  test('should send a message and receive a response', async ({ page }) => {
    // 输入消息
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Hello, AI!')
    await input.press('Enter')

    // 用户消息出现在消息列表中
    const userMessage = page.locator('[data-test="message-bubble"]').filter({ hasText: 'Hello, AI!' })
    await expect(userMessage).toBeVisible()

    // 等待助手回复（SSE 流式）
    const assistantMessage = page.locator('[data-test="message-bubble"].assistant')
    await expect(assistantMessage).toBeVisible({ timeout: 30_000 })

    // 输入框应被清空
    await expect(input).toHaveValue('')
  })

  test('should display thinking indicator during processing', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Write a function')
    await input.press('Enter')

    // 思考指示器出现
    const thinkingIndicator = page.locator('[data-test="thinking-indicator"]')
    await expect(thinkingIndicator).toBeVisible({ timeout: 5_000 })

    // 等待完成
    await page.waitForSelector('[data-test="thinking-indicator"]', {
      state: 'detached',
      timeout: 30_000,
    })
  })

  test('should display tool call card', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Create a file called test.txt')
    await input.press('Enter')

    // 工具调用卡片出现
    const toolCard = page.locator('[data-test="tool-call-card"]')
    await expect(toolCard).toBeVisible({ timeout: 30_000 })

    // 工具名称显示
    await expect(toolCard.locator('[data-test="tool-name"]')).toContainText('write_file')
  })

  test('should stop processing on stop button click', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Write a long story')
    await input.press('Enter')

    // 等待处理开始
    await page.waitForSelector('[data-test="stop-button"]', { timeout: 5_000 })

    // 点击停止
    await page.locator('[data-test="stop-button"]').click()

    // 状态栏显示空闲
    const status = page.locator('[data-test="session-status"]')
    await expect(status).toContainText('空闲', { timeout: 5_000 })
  })

  test('should format code blocks with syntax highlighting', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Show me a Go hello world example')
    await input.press('Enter')

    // 等待响应中出现代码块
    const codeBlock = page.locator('[data-test="code-block"]')
    await expect(codeBlock).toBeVisible({ timeout: 30_000 })

    // 代码块内部有语法高亮 span
    const highlightedSpan = codeBlock.locator('.hljs-keyword')
    await expect(highlightedSpan.first()).toBeVisible()
  })
})
```

```typescript
// e2e/command-palette.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Command Palette', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]')
  })

  test('should open on / key', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('/')
    await input.press('Enter')

    const palette = page.locator('[data-test="command-palette"]')
    await expect(palette).toBeVisible()
  })

  test('should open on Ctrl+K shortcut', async ({ page }) => {
    await page.keyboard.press('Control+k')

    const palette = page.locator('[data-test="command-palette"]')
    await expect(palette).toBeVisible()
  })

  test('should filter commands by query', async ({ page }) => {
    await page.keyboard.press('Control+k')

    const palette = page.locator('[data-test="command-palette"]')
    const input = palette.locator('input')

    await input.fill('new')

    const items = palette.locator('[data-test="command-item"]')
    await expect(items).toHaveCount(1)
    await expect(items.first()).toContainText('new')
  })

  test('should create a new session via /new command', async ({ page }) => {
    await page.keyboard.press('Control+k')

    const palette = page.locator('[data-test="command-palette"]')
    const input = palette.locator('input')

    await input.fill('/new test-project')
    await input.press('Enter')

    // 状态栏应显示新会话名称
    const status = page.locator('[data-test="session-name"]')
    await expect(status).toContainText('test-project', { timeout: 5_000 })
  })

  test('should close on Escape', async ({ page }) => {
    await page.keyboard.press('Control+k')

    const palette = page.locator('[data-test="command-palette"]')
    await expect(palette).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(palette).not.toBeVisible()
  })
})
```

```typescript
// e2e/approval.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Approval Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]')
  })

  test('should show approval modal for high-risk operation', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    // 审批弹窗出现
    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 10_000 })

    // 显示风险等级
    await expect(modal.locator('[data-test="risk-level"]')).toBeVisible()
  })

  test('should execute action on approve click', async ({ page }) => {
    // 触发审批
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 10_000 })

    // 点击批准
    await modal.locator('[data-test="approve-button"]').click()

    // 弹窗关闭
    await expect(modal).not.toBeVisible({ timeout: 5_000 })

    // 工具调用卡片显示批准状态
    const toolCard = page.locator('[data-test="tool-call-card"]')
    await expect(toolCard.locator('[data-test="approval-badge"]')).toContainText('已批准')
  })

  test('should reject action on reject click', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 10_000 })

    // 点击拒绝
    await modal.locator('[data-test="reject-button"]').click()

    // 弹窗关闭
    await expect(modal).not.toBeVisible({ timeout: 5_000 })

    // 工具调用卡片显示拒绝状态
    const toolCard = page.locator('[data-test="tool-call-card"]')
    await expect(toolCard.locator('[data-test="approval-badge"]')).toContainText('已拒绝')
  })

  test('should approve on Y key', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 10_000 })

    await page.keyboard.press('y')

    await expect(modal).not.toBeVisible({ timeout: 5_000 })
  })
})
```

### 4.7 侧边栏组件测试 — AppSidebar

#### 4.7.1 工作区名称过长 — 文本截断

```typescript
// components/layout/AppSidebar.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AppSidebar from '@/components/layout/AppSidebar.vue'

describe('AppSidebar — 长文本截断', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('工作区名称过长', () => {
    it('should truncate workspace name with ellipsis when name is too long', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const uiStore = useUiStore()
      uiStore.workspaceList = [
        {
          id: '/home/user/projects/this-is-an-extremely-long-workspace-name-that-exceeds-sidebar',
          name: 'this-is-an-extremely-long-workspace-name-that-exceeds-sidebar-width',
          path: '/home/user/projects/this-is-an-extremely-long-workspace-name-that-exceeds-sidebar',
        },
      ]

      const nameEl = wrapper.find('.workspace-name')
      expect(nameEl.exists()).toBe(true)

      const computedStyle = getComputedStyle(nameEl.element)
      expect(computedStyle.overflow).toBe('hidden')
      expect(computedStyle.textOverflow).toBe('ellipsis')
      expect(computedStyle.whiteSpace).toBe('nowrap')
    })

    it('should still show full path in small text below workspace name', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const uiStore = useUiStore()
      const longPath = '/home/user/projects/very-very-long-workspace-directory-path'
      uiStore.workspaceList = [
        { id: longPath, name: 'long-project', path: longPath },
      ]

      const pathEl = wrapper.find('.workspace-path')
      expect(pathEl.exists()).toBe(true)
      expect(pathEl.text()).toBe(longPath)
      // 路径也应有截断
      const pathStyle = getComputedStyle(pathEl.element)
      expect(pathStyle.textOverflow).toBe('ellipsis')
    })

    it('should display full workspace name in tooltip (title attribute)', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const uiStore = useUiStore()
      const longName = 'a-very-long-workspace-name-for-testing'
      uiStore.workspaceList = [
        { id: '/test', name: longName, path: '/test/project' },
      ]

      const item = wrapper.find('.workspace-item')
      // 父容器或子元素应有 title 属性
      const nameEl = item.find('.workspace-name')
      expect(nameEl.attributes('title') || item.attributes('title')).toBeTruthy()
    })
  })

  describe('会话名称过长', () => {
    it('should truncate session title with ellipsis when title is too long', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const sessionStore = useSessionStore()
      sessionStore.sessions = [
        {
          id: 'sess-1',
          title: '这是一个非常非常非常长的会话标题用来测试侧边栏截断效果是否正常显示省略号',
          state: 'idle',
          createdAt: '2026-01-01T00:00:00Z',
          lastActiveAt: '2026-01-01T00:00:00Z',
          messageCount: 0,
          workingDirectory: '/test',
        } as any,
      ]

      const titleEl = wrapper.find('.session-title')
      expect(titleEl.exists()).toBe(true)

      const style = getComputedStyle(titleEl.element)
      expect(style.overflow).toBe('hidden')
      expect(style.textOverflow).toBe('ellipsis')
      expect(style.whiteSpace).toBe('nowrap')
    })

    it('should show full session title on hover via title attribute', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const sessionStore = useSessionStore()
      const longTitle = '修复用户登录页面在Safari浏览器下Cookie无法正确设置的Bug'
      sessionStore.sessions = [
        {
          id: 'sess-1',
          title: longTitle,
          state: 'idle',
          createdAt: '2026-01-01T00:00:00Z',
          lastActiveAt: '2026-01-01T00:00:00Z',
          messageCount: 0,
          workingDirectory: '/test',
        } as any,
      ]

      const item = wrapper.find('.session-item')
      expect(item.attributes('title')).toBe(longTitle)
    })

    it('should handle empty session title gracefully', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const sessionStore = useSessionStore()
      sessionStore.sessions = [
        {
          id: 'sess-1',
          title: '',
          state: 'idle',
          createdAt: '2026-01-01T00:00:00Z',
          lastActiveAt: '2026-01-01T00:00:00Z',
          messageCount: 0,
          workingDirectory: '/test',
        } as any,
      ]

      const titleEl = wrapper.find('.session-title')
      expect(titleEl.exists()).toBe(true)
      // 空标题不应崩溃，显示占位文本
      expect(titleEl.text()).toBeDefined()
    })
  })
})
```

#### 4.7.2 列表条目过多 — 滚动与定位

```typescript
describe('AppSidebar — 列表溢出滚动', () => {
  function generateWorkspaces(count: number) {
    const list = []
    for (let i = 1; i <= count; i++) {
      const name = `project-${String(i).padStart(3, '0')}`
      list.push({
        id: `/home/user/${name}`,
        name,
        path: `/home/user/projects/${name}`,
      })
    }
    return list
  }

  function generateSessions(count: number) {
    const list = []
    for (let i = 1; i <= count; i++) {
      list.push({
        id: `sess-${String(i).padStart(3, '0')}`,
        title: `会话 #${i} — 讨论关于项目架构和性能优化的问题`,
        state: 'idle' as const,
        createdAt: `2026-01-${String(i).padStart(2, '0')}T00:00:00Z`,
        lastActiveAt: `2026-01-${String(i).padStart(2, '0')}T00:00:00Z`,
        messageCount: i * 5,
        workingDirectory: '/test',
      })
    }
    return list
  }

  describe('工作区列表溢出', () => {
    it('should make workspace list scrollable when items exceed viewport', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const uiStore = useUiStore()
      uiStore.workspaceList = generateWorkspaces(50)

      const section = wrapper.find('.sidebar-section')
      const sectionEl = section.element as HTMLElement

      // 50 个工作区应该超出容器高度
      expect(sectionEl.scrollHeight).toBeGreaterThan(sectionEl.clientHeight)
    })

    it('should scroll to a specific workspace when selected', async () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const uiStore = useUiStore()
      const workspaces = generateWorkspaces(50)
      uiStore.workspaceList = workspaces

      // 选中第 45 个工作区
      const target = workspaces[44]
      uiStore.setActiveWorkspace(target.id)

      await wrapper.vm.$nextTick()

      const activeItem = wrapper.find('.workspace-item.active')
      expect(activeItem.exists()).toBe(true)

      // 验证 active 元素在可视区域内
      const itemEl = activeItem.element as HTMLElement
      const parentEl = itemEl.parentElement!
      const itemTop = itemEl.offsetTop
      const itemBottom = itemTop + itemEl.offsetHeight
      const parentScrollTop = parentEl.scrollTop
      const parentHeight = parentEl.clientHeight

      // 选中项应在可视区域内
      expect(itemBottom).toBeGreaterThan(parentScrollTop)
      expect(itemTop).toBeLessThan(parentScrollTop + parentHeight)
    })

    it('should scroll to active workspace on mount', async () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const uiStore = useUiStore()
      const workspaces = generateWorkspaces(50)
      uiStore.workspaceList = workspaces
      uiStore.setActiveWorkspace(workspaces[30].id)

      await wrapper.vm.$nextTick()

      const activeItem = wrapper.find('.workspace-item.active')
      const itemEl = activeItem.element as HTMLElement
      const parentEl = itemEl.parentElement!

      // active 项应在可视区域内（scrollIntoView 已调用）
      expect(itemEl.offsetTop).toBeGreaterThanOrEqual(parentEl.scrollTop)
    })
  })

  describe('会话列表溢出', () => {
    it('should make session list scrollable when many sessions exist', () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const sessionStore = useSessionStore()
      sessionStore.sessions = generateSessions(100) as any

      const sessionsList = wrapper.find('.sessions-list')
      const listEl = sessionsList.element as HTMLElement

      // 100 个会话应该超出容器高度
      expect(listEl.scrollHeight).toBeGreaterThan(listEl.clientHeight)
    })

    it('should scroll to active session when selected', async () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const sessionStore = useSessionStore()
      const sessions = generateSessions(80) as any
      sessionStore.sessions = sessions
      sessionStore.currentSession = sessions[60]

      await wrapper.vm.$nextTick()

      const activeItem = wrapper.find('.session-item.active')
      expect(activeItem.exists()).toBe(true)

      const itemEl = activeItem.element as HTMLElement
      const parentEl = itemEl.parentElement!
      const itemTop = itemEl.offsetTop
      const parentScrollTop = parentEl.scrollTop
      const parentHeight = parentEl.clientHeight

      expect(itemTop).toBeGreaterThanOrEqual(parentScrollTop)
      expect(itemTop).toBeLessThan(parentScrollTop + parentHeight)
    })

    it('should scroll to newly created session at bottom of list', async () => {
      const wrapper = mount(AppSidebar, {
        props: { collapsed: false },
      })

      const sessionStore = useSessionStore()
      sessionStore.sessions = generateSessions(30) as any

      const newSession = {
        id: 'sess-new',
        title: '新创建的会话',
        state: 'idle' as const,
        createdAt: '2026-01-31T00:00:00Z',
        lastActiveAt: '2026-01-31T00:00:00Z',
        messageCount: 0,
        workingDirectory: '/test',
      } as any
      sessionStore.sessions = [...sessionStore.sessions, newSession]
      sessionStore.currentSession = newSession

      await wrapper.vm.$nextTick()

      const activeItem = wrapper.find('.session-item.active')
      const itemEl = activeItem.element as HTMLElement
      const parentEl = itemEl.parentElement!

      // 新会话应在可视区域底部
      expect(itemEl.offsetTop).toBeGreaterThanOrEqual(parentEl.scrollTop)
    })
  })
})
```

#### 4.7.3 折叠模式下图标列表溢出

```typescript
describe('AppSidebar — 折叠模式溢出', () => {
  it('should make collapsed icon list scrollable with many workspaces', () => {
    const wrapper = mount(AppSidebar, {
      props: { collapsed: true },
    })

    const uiStore = useUiStore()
    const workspaces = []
    for (let i = 1; i <= 30; i++) {
      workspaces.push({
        id: `/home/user/project-${i}`,
        name: `project-${i}`,
        path: `/home/user/project-${i}`,
      })
    }
    uiStore.workspaceList = workspaces
    const sessionStore = useSessionStore()
    const sessions = []
    for (let i = 1; i <= 30; i++) {
      sessions.push({
        id: `sess-${i}`,
        title: `会话 ${i}`,
        state: 'idle',
        createdAt: '2026-01-01T00:00:00Z',
        lastActiveAt: '2026-01-01T00:00:00Z',
        messageCount: 0,
        workingDirectory: '/test',
      } as any)
    }
    sessionStore.sessions = sessions

    const iconsContainer = wrapper.find('.collapsed-icons')
    const containerEl = iconsContainer.element as HTMLElement

    // 60 个图标应超出容器高度，显示滚动条
    expect(containerEl.scrollHeight).toBeGreaterThan(containerEl.clientHeight)
  })

  it('should show active workspace icon in collapsed mode', async () => {
    const wrapper = mount(AppSidebar, {
      props: { collapsed: true },
    })

    const uiStore = useUiStore()
    const workspaces = []
    for (let i = 1; i <= 20; i++) {
      workspaces.push({
        id: `/home/user/project-${i}`,
        name: `project-${i}`,
        path: `/home/user/project-${i}`,
      })
    }
    uiStore.workspaceList = workspaces
    uiStore.setActiveWorkspace('/home/user/project-15')

    await wrapper.vm.$nextTick()

    const activeIcon = wrapper.find('.collapsed-icon-btn.active')
    expect(activeIcon.exists()).toBe(true)
  })
})
```

---

## 5. 测试数据与 Mock 工厂

### 5.1 API Mock 工厂

```typescript
// src/test/mocks/api.ts
import { vi } from 'vitest'
import type { Session, Message, ToolCall, Approval } from '@/types'

export interface MockApiOptions {
  shouldFail?: boolean
  errorStatus?: number
  delay?: number
}

export function createMockFetch(overrides: Partial<MockApiOptions> = {}) {
  const options: MockApiOptions = {
    shouldFail: false,
    errorStatus: 500,
    delay: 0,
    ...overrides,
  }

  return vi.fn().mockImplementation(async (url: string, init?: RequestInit) => {
    if (options.delay) {
      await new Promise(r => setTimeout(r, options.delay))
    }

    if (options.shouldFail) {
      return {
        ok: false,
        status: options.errorStatus,
        statusText: 'Mock Error',
        json: () => Promise.resolve({ error: 'Mock error' }),
      } as Response
    }

    // 根据 URL 返回不同的 Mock 数据
    if (url.includes('/sessions') && init?.method === 'POST') {
      return createMockResponse({ id: 'sess-new', title: 'New Session', state: 'idle' })
    }

    return createMockResponse({})
  })
}

function createMockResponse(data: unknown): Response {
  return {
    ok: true,
    status: 200,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  } as Response
}
```

### 5.2 测试数据 Fixtures

```typescript
// src/test/fixtures/sessions.ts
import type { Session } from '@/types'

export const mockSession: Session = {
  id: 'sess-001',
  title: 'Test Project',
  state: 'idle',
  workingDirectory: '/tmp/test',
  createdAt: '2026-01-01T00:00:00Z',
  lastActiveAt: '2026-01-01T12:00:00Z',
  messageCount: 5,
  tokenUsage: { input: 500, output: 200 },
  trustLevel: 'session_trust',
}

export const mockSessions: Session[] = [
  mockSession,
  {
    id: 'sess-002',
    title: 'Another Project',
    state: 'archived',
    workingDirectory: '/tmp/another',
    createdAt: '2026-01-02T00:00:00Z',
    lastActiveAt: '2026-01-03T00:00:00Z',
    messageCount: 20,
    tokenUsage: { input: 3000, output: 1500 },
    trustLevel: 'full_trust',
  },
  {
    id: 'sess-003',
    title: 'Processing Project',
    state: 'processing',
    workingDirectory: '/tmp/processing',
    createdAt: '2026-01-04T00:00:00Z',
    lastActiveAt: '2026-01-04T12:00:00Z',
    messageCount: 8,
    tokenUsage: { input: 800, output: 400 },
    trustLevel: 'always_ask',
  },
]

// src/test/fixtures/messages.ts
import type { Message, ToolCall } from '@/types'

export const mockUserMessage: Message = {
  id: 'msg-001',
  sessionId: 'sess-001',
  role: 'user',
  content: 'Hello, can you help me write a function?',
  timestamp: '2026-01-01T12:00:00Z',
}

export const mockAssistantMessage: Message = {
  id: 'msg-002',
  sessionId: 'sess-001',
  role: 'assistant',
  content: 'Sure! Here is a function:\n\n```typescript\nfunction hello() {\n  return "Hello";\n}\n```',
  timestamp: '2026-01-01T12:00:05Z',
  tokenUsage: { input: 50, output: 30 },
}

export const mockSystemMessage: Message = {
  id: 'msg-003',
  sessionId: 'sess-001',
  role: 'system',
  content: 'Session created',
  timestamp: '2026-01-01T12:00:00Z',
}

export const mockToolCallMessage: Message = {
  id: 'msg-004',
  sessionId: 'sess-001',
  role: 'tool',
  content: 'File written successfully',
  timestamp: '2026-01-01T12:00:03Z',
  toolCall: {
    id: 'tool-001',
    name: 'write_file',
    parameters: { path: '/tmp/test/file.txt', content: 'Hello' },
    result: { success: true, bytesWritten: 5 },
    status: 'success',
    duration: 234,
  },
}

// src/test/fixtures/tools.ts
import type { ToolCall } from '@/types'

export const mockToolCallPending: ToolCall = {
  id: 'tool-001',
  name: 'write_file',
  parameters: {
    path: '/tmp/test/file.txt',
    content: 'Hello, world!',
  },
  status: 'pending',
  riskLevel: 'medium',
}

export const mockToolCallSuccess: ToolCall = {
  ...mockToolCallPending,
  status: 'success',
  result: {
    success: true,
    bytesWritten: 13,
    path: '/tmp/test/file.txt',
  },
  duration: 234,
}

export const mockToolCallFailed: ToolCall = {
  ...mockToolCallPending,
  status: 'failed',
  result: {
    success: false,
    error: 'Permission denied',
  },
  duration: 50,
}

export const mockToolCallRejected: ToolCall = {
  ...mockToolCallPending,
  status: 'rejected',
}
```

---

## 6. 完整测试用例清单

### 6.1 单元测试 — Utils（8 个用例）

| 编号 | 模块 | 测试用例 | 类型 |
| :--- | :--- | :--- | :--- |
| U-01 | formatters | should format bytes to human readable string | 工具函数 |
| U-02 | formatters | should format duration in milliseconds | 工具函数 |
| U-03 | formatters | should format token count with thousand separators | 工具函数 |
| U-04 | formatters | should format relative time | 工具函数 |
| U-05 | formatters | should truncate long text with ellipsis | 工具函数 |
| U-06 | markdown | should parse basic Markdown to HTML | 工具函数 |
| U-07 | markdown | should highlight code blocks with correct language | 工具函数 |
| U-08 | markdown | should render tables correctly | 工具函数 |

### 6.2 单元测试 — Stores（35 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| S-01 | session | should create a session and set as current |
| S-02 | session | should handle API error on create |
| S-03 | session | should fetch all sessions |
| S-04 | session | should switch to an existing session |
| S-05 | session | should not switch to non-existent session |
| S-06 | session | should rename current session |
| S-07 | session | should archive current session |
| S-08 | session | should get session status via getter |
| C-01 | chat | should append user message |
| C-02 | chat | should append assistant message |
| C-03 | chat | should append system message |
| C-04 | chat | should append tool call message |
| C-05 | chat | should update streaming message content |
| C-06 | chat | should mark streaming as complete |
| C-07 | chat | should clear all messages |
| C-08 | chat | should rollback to specific message index |
| C-09 | chat | should not rollback beyond message count |
| C-10 | chat | should get last message via getter |
| A-01 | approval | should set current approval |
| A-02 | approval | should clear approval after approve |
| A-03 | approval | should clear approval after reject |
| A-04 | approval | should add to approval history |
| A-05 | approval | should set trust level |
| A-06 | approval | should detect pending approval via getter |
| CM-01 | command | should open command palette |
| CM-02 | command | should close command palette |
| CM-03 | command | should filter commands by query |
| CM-04 | command | should select command by index |
| CM-05 | command | should get filtered commands via getter |
| U-01 | ui | should add toast message |
| U-02 | ui | should remove toast by ID |
| U-03 | ui | should auto-remove toast after duration |
| U-04 | ui | should set connection status |
| U-05 | ui | should set active modal |

### 6.3 单元测试 — Composables（18 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| CP-01 | useApi | should make GET request and return data |
| CP-02 | useApi | should make POST request with body |
| CP-03 | useApi | should handle HTTP error |
| CP-04 | useApi | should handle network error |
| CP-05 | useApi | should handle timeout |
| CP-06 | useSSE | should connect to session events |
| CP-07 | useSSE | should parse SSE event stream |
| CP-08 | useSSE | should call handler for specific event type |
| CP-09 | useSSE | should auto-reconnect on error |
| CP-10 | useSSE | should not reconnect after disconnect |
| CP-11 | useSSE | should reconnect with exponential backoff |
| CP-12 | useSession | should create session and update store |
| CP-13 | useSession | should switch session and update store |
| CP-14 | useKeyboard | should register shortcut handler |
| CP-15 | useKeyboard | should trigger handler on key combination |
| CP-16 | useKeyboard | should cleanup on unmount |
| CP-17 | useAutoScroll | should auto-scroll to bottom on new message |
| CP-18 | useAutoScroll | should not scroll when user scrolled up |

### 6.4 组件测试（30 个用例）

| 编号 | 组件 | 测试用例 |
| :--- | :--- | :--- |
| CT-01 | StatusBar | should render session name |
| CT-02 | StatusBar | should render session status indicator |
| CT-03 | StatusBar | should render token usage |
| CT-04 | StatusBar | should render connection status |
| CT-05 | StatusBar | should show correct color for each status |
| CT-06 | InputArea | should render textarea and send button |
| CT-07 | InputArea | should emit send on Enter |
| CT-08 | InputArea | should not emit on Shift+Enter |
| CT-09 | InputArea | should clear after send |
| CT-10 | InputArea | should show stop button when processing |
| CT-11 | InputArea | should disable input when disabled |
| CT-12 | InputArea | should show character count |
| CT-13 | InputArea | should open command palette on / |
| CT-14 | MessageBubble | should render user message with correct alignment |
| CT-15 | MessageBubble | should render assistant message with Markdown |
| CT-16 | MessageBubble | should render system message with correct style |
| CT-17 | MessageBubble | should show timestamp on hover |
| CT-18 | ToolCallCard | should render tool name and parameters |
| CT-19 | ToolCallCard | should show success status |
| CT-20 | ToolCallCard | should show failed status |
| CT-21 | ToolCallCard | should collapse/expand parameters |
| CT-22 | ThinkingIndicator | should render when thinking |
| CT-23 | ThinkingIndicator | should not render when not thinking |
| CT-24 | CommandPalette | should render command list |
| CT-25 | CommandPalette | should filter by query |
| CT-26 | CommandPalette | should emit select on Enter |
| CT-27 | CommandPalette | should emit close on Escape |
| CT-28 | ApprovalModal | should render risk level badge |
| CT-29 | ApprovalModal | should emit approve on button click |
| CT-30 | ApprovalModal | should emit reject on button click |

### 6.5 E2E 测试（20 个用例）

| 编号 | 测试文件 | 测试用例 |
| :--- | :--- | :--- |
| E-01 | session | should load page and auto-create session |
| E-02 | session | should create new session via /new command |
| E-03 | session | should switch between sessions via /switch command |
| E-04 | session | should rename session via /rename command |
| E-05 | session | should archive session via /archive command |
| E-06 | chat | should send message and receive streaming response |
| E-07 | chat | should display thinking indicator during processing |
| E-08 | chat | should display tool call card |
| E-09 | chat | should render code blocks with syntax highlighting |
| E-10 | chat | should stop processing on stop button click |
| E-11 | chat | should clear screen on /clear command |
| E-12 | approval | should show approval modal for high-risk operation |
| E-13 | approval | should execute on approve click |
| E-14 | approval | should cancel on reject click |
| E-15 | approval | should approve on Y key press |
| E-16 | approval | should reject on N key press |
| E-17 | command-palette | should open on Ctrl+K |
| E-18 | command-palette | should filter commands by input |
| E-19 | command-palette | should close on Escape |
| E-20 | rollback | should rollback messages via /rollback command |

### 6.6 Phase 1 扩展 — Stores（UI Store 扩展）（5 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| U-06 | ui | should detect VSCode mode from URL parameter `?mode=vscode` |
| U-07 | ui | should detect browser mode by default |
| U-08 | ui | should set activeWorkspace and switch between global/workspace mode |
| U-09 | ui | should set activeRightTab and toggle rightPanelVisible |
| U-10 | ui | should toggle sidebarCollapsed state |

### 6.7 Phase 2 — Stores（Skills）（11 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| SK-01 | skills | should fetch all skills |
| SK-02 | skills | should filter by scope (global / workspace) |
| SK-03 | skills | should return globalSkills via getter |
| SK-04 | skills | should return workspaceSkills via getter |
| SK-05 | skills | should enable skill |
| SK-06 | skills | should disable skill |
| SK-07 | skills | should get skill detail by name |
| SK-08 | skills | should install new skill |
| SK-09 | skills | should handle install error |
| SK-10 | skills | should handle `skill_solidified` SSE event |
| SK-11 | skills | should filter by selectedFilter (all/global/workspace) |

### 6.8 Phase 3 — Stores（Memory + MCP）（13 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| ME-01 | memory | should fetch all memories |
| ME-02 | memory | should filter by scope (global / workspace) |
| ME-03 | memory | should return globalMemories via getter |
| ME-04 | memory | should return workspaceMemories via getter |
| ME-05 | memory | should add new memory |
| ME-06 | memory | should update existing memory |
| ME-07 | memory | should delete memory |
| ME-08 | memory | should handle `memory_updated` SSE event |
| ME-09 | memory | should handle API error on add |
| MC-01 | mcp | should fetch all MCP tools |
| MC-02 | mcp | should show server connection status |
| MC-03 | mcp | should filter tools by server |
| MC-04 | mcp | should handle `mcp_tool_discovered` SSE event |
| MC-05 | mcp | should show `trust_note` for each tool |

### 6.9 Phase 4 — Stores（Dashboard + Settings）（9 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| DB-01 | dashboard | should fetch token usage stats |
| DB-02 | dashboard | should show today's usage |
| DB-03 | dashboard | should show monthly usage |
| DB-04 | dashboard | should show per-session breakdown |
| DB-05 | dashboard | should show per-model breakdown |
| ST-01 | settings | should fetch approval policy |
| ST-02 | settings | should update approval policy for operation type |
| ST-03 | settings | should set trust level |
| ST-04 | settings | should fetch project settings |

### 6.10 Phase 2~3 — Composables（11 个用例）

| 编号 | 模块 | 测试用例 |
| :--- | :--- | :--- |
| CP-19 | usePlatform | should detect VSCode mode |
| CP-20 | usePlatform | should detect browser mode |
| CP-21 | usePlatform | should return correct layout component |
| CP-22 | useSkills | should fetch and return skills |
| CP-23 | useSkills | should toggle skill state |
| CP-24 | useSkills | should install skill |
| CP-25 | useMemory | should fetch memories |
| CP-26 | useMemory | should add memory |
| CP-27 | useMemory | should delete memory |
| CP-28 | useMemory | should handle update conflict |
| CP-29 | useMcp | should fetch MCP tools |
| CP-30 | useMcp | should group tools by server |

### 6.11 Phase 1 扩展 — 组件测试（Layouts）（14 个用例）

| 编号 | 组件 | 测试用例 |
| :--- | :--- | :--- |
| CT-31 | VscodeLayout | should render ChatView |
| CT-32 | VscodeLayout | should render GlobalModals |
| CT-33 | VscodeLayout | should not render AppSidebar |
| CT-34 | BrowserLayout | should render AppSidebar |
| CT-35 | BrowserLayout | should render RightPanel |
| CT-36 | BrowserLayout | should render `<router-view />` (ChatView) |
| CT-37 | BrowserLayout | should render GlobalModals |
| CT-38 | AppSidebar | should render global entry (non-selectable) |
| CT-39 | AppSidebar | should render workspace list |
| CT-40 | AppSidebar | should render session list for active workspace |
| CT-41 | AppSidebar | should highlight active workspace |
| CT-42 | AppSidebar | should emit workspace-select on click |
| CT-43 | AppHeader | should render session title |
| CT-44 | AppHeader | should render theme toggle button |

### 6.12 Phase 2 — 组件测试（RightPanel + 技能管理面板）（14 个用例）

| 编号 | 组件 | 测试用例 |
| :--- | :--- | :--- |
| CT-45 | RightPanel | should render Tab bar |
| CT-46 | RightPanel | should switch active Tab on click |
| CT-47 | RightPanel | should show correct Tabs in global mode (3 tabs) |
| CT-48 | RightPanel | should show correct Tabs in workspace mode (6 tabs) |
| CT-49 | RightPanel | should hide panel when rightPanelVisible is false |
| CT-50 | RightPanel | should support drag-to-resize width |
| CT-51 | ChatView | should render ChatPanel |
| CT-52 | SkillsPanel | should render skill cards grouped by scope |
| CT-53 | SkillsPanel | should filter by scope selector (all/global/workspace) |
| CT-54 | SkillsPanel | should show empty state |
| CT-55 | SkillsPanel | should show loading state |
| CT-56 | SkillDetailPanel | should render skill details |
| CT-57 | SkillDetailPanel | should render SKILL.md preview |
| CT-58 | SkillDetailPanel | should handle not found |

### 6.13 Phase 3 — 组件测试（记忆 + MCP + 文件面板）（18 个用例）

| 编号 | 组件 | 测试用例 |
| :--- | :--- | :--- |
| CT-59 | MemoryPanel | should render memory list grouped by scope |
| CT-60 | MemoryPanel | should filter by scope selector (all/global/workspace) |
| CT-61 | MemoryPanel | should open edit dialog on add |
| CT-62 | MemoryPanel | should show empty state |
| CT-63 | MemoryCard | should render memory key and value |
| CT-64 | MemoryCard | should emit edit on click |
| CT-65 | MemoryCard | should emit delete on click |
| CT-66 | MemoryEditDialog | should render form fields |
| CT-67 | MemoryEditDialog | should emit save on confirm |
| CT-68 | MemoryEditDialog | should emit close on cancel |
| CT-69 | McpSettingsPanel | should render server list |
| CT-70 | McpSettingsPanel | should show connection status indicator |
| CT-71 | McpSettingsPanel | should render tool list |
| CT-72 | McpSettingsPanel | should show trust_note |
| CT-73 | FilesPanel | should render file tree |
| CT-74 | FilesPanel | should expand/collapse folders |
| CT-75 | FilesPanel | should show file content on click |
| CT-76 | FilesPanel | should show empty state when no workspace |

### 6.14 Phase 4~5 — 组件测试（仪表盘 + 设置 + 终端面板）（16 个用例）

| 编号 | 组件 | 测试用例 |
| :--- | :--- | :--- |
| CT-77 | DashboardPanel | should render usage summary |
| CT-78 | DashboardPanel | should render per-session chart |
| CT-79 | DashboardPanel | should render per-model chart |
| CT-80 | DashboardPanel | should show loading state |
| CT-81 | SettingsPanel | should render sub-tabs (项目设置 / 全局设置) |
| CT-82 | SettingsPanel | should switch between sub-tabs |
| CT-83 | SettingsPanel | should show project settings form in workspace mode |
| CT-84 | ApprovalPolicyPanel | should render policy list |
| CT-85 | ApprovalPolicyPanel | should allow policy change |
| CT-86 | ApprovalPolicyPanel | should show trust level selector |
| CT-87 | SkillCard | should render skill name and scope badge |
| CT-88 | SkillCard | should show toggle state |
| CT-89 | SkillInstallDialog | should render install form |
| CT-90 | SkillInstallDialog | should emit install on confirm |
| CT-91 | SkillInstallDialog | should emit close on cancel |
| CT-92 | TerminalPanel | should render terminal area (仅工作区模式) |

### 6.15 Phase 1 扩展 ~ Phase 5 — E2E 测试（26 个用例）

| 编号 | 测试文件 | 测试用例 |
| :--- | :--- | :--- |
| E-21 | mode-routing | should show browser layout on default URL |
| E-22 | mode-routing | should show VSCode layout with `?mode=vscode` |
| E-23 | mode-routing | should show chat as main content in both modes |
| E-24 | right-panel | should show 3 tabs in global mode (无 workspace) |
| E-25 | right-panel | should show 6 tabs in workspace mode (选中 workspace) |
| E-26 | right-panel | should switch between tabs without leaving chat |
| E-27 | right-panel | should maintain panel state across tab switches |
| E-28 | right-panel | should support drag-to-resize right panel width |
| E-29 | skills | should view skills list in SkillsPanel |
| E-30 | skills | should filter skills by scope (global/workspace/all) |
| E-31 | skills | should view skill detail in SkillDetailPanel |
| E-32 | skills | should install new skill via dialog |
| E-33 | memory | should view memory list in MemoryPanel |
| E-34 | memory | should add new memory |
| E-35 | memory | should edit existing memory |
| E-36 | memory | should delete memory |
| E-37 | mcp-tools | should view MCP tool list in McpSettingsPanel |
| E-38 | mcp-tools | should show connection status per server |
| E-39 | mcp-tools | should refresh on `mcp_tool_discovered` event |
| E-40 | dashboard | should view token usage dashboard |
| E-41 | dashboard | should show per-session breakdown |
| E-42 | workspace-scope | should select workspace from sidebar |
| E-43 | workspace-scope | should show session list for selected workspace |
| E-44 | workspace-scope | should configure global settings without workspace |
| E-45 | workspace-scope | should switch between global and workspace mode |
| E-46 | sidebar | should display workspace list in left sidebar |

**总计**：250 个测试用例

| 分类 | Phase 1 已有 | Phase 1 扩展 ~ Phase 5 新增 | 合计 |
| :--- | :--- | :--- | :--- |
| Utils | 8 | — | 8 |
| Stores | 35 | 39 | 74 |
| Composables | 18 | 12 | 30 |
| 组件测试 | 30 | 62 | 92 |
| E2E | 20 | 26 | 46 |
| **合计** | **111** | **139** | **250** |

---

## 7. 实现计划

| 编号 | 任务 | 内容 | 依赖 |
| :--- | :--- | :--- | :--- |
| TEST-01 | 配置脚手架 | install vitest, @vue/test-utils, jsdom, playwright；编写 vitest.config.ts 和 playwright.config.ts | WEB-01 |
| TEST-02 | 全局 Setup | 编写 `src/test/setup.ts`（Mock EventSource、fetch、浏览器 API） | TEST-01 |
| TEST-03 | Mock 工厂 + Fixtures | 编写 API Mock、SSE Mock、测试数据 fixtures | TEST-02 |
| TEST-04 | Utils 单元测试 | formatters、markdown 工具函数测试（8 个用例） | WEB-02 |
| TEST-05 | Stores 单元测试 | 5 个 Store 的完整测试（35 个用例） | WEB-05 |
| TEST-06 | Composables 单元测试 | 6 个 Composable 的完整测试（18 个用例） | WEB-04 |
| TEST-07 | 组件测试 — 布局 | StatusBar、ToastContainer、ToastItem | WEB-06, WEB-17 |
| TEST-08 | 组件测试 — 聊天 | InputArea、MessageBubble、ToolCallCard、ThinkingIndicator、MessageList | WEB-07~12 |
| TEST-09 | 组件测试 — 面板 | CommandPalette、ApprovalModal、SessionPicker、RollbackPicker、HelpPanel | WEB-13~16 |
| TEST-10 | 组件测试 — 共享 | DiffView、CodeBlock、MarkdownView、Spinner | WEB-09 |
| TEST-11 | E2E — 会话管理 | session.spec.ts（5 个用例） | WEB-15 |
| TEST-12 | E2E — 聊天对话 | chat.spec.ts（6 个用例） | WEB-07~12 |
| TEST-13 | E2E — 审批流程 | approval.spec.ts（5 个用例） | WEB-14 |
| TEST-14 | E2E — 命令面板 | command-palette.spec.ts（3 个用例） | WEB-13 |
| TEST-15 | E2E — 回滚 | rollback.spec.ts（1 个用例） | WEB-16 |
| TEST-16 | CI/CD 配置 | GitHub Actions workflow 配置 | TEST-11~15 |

### Phase 1 扩展 ~ Phase 5 新增任务（v1.2.0）

| 编号 | 任务 | 内容 | 依赖 |
| :--- | :--- | :--- | :--- |
| TEST-17 | 新 Fixtures | 编写 skills、memory、mcp、dashboard、workspaces 测试数据 | WEB-30, WEB-35, WEB-39, WEB-42 |
| TEST-18 | 新 Stores 测试 | skills、memory、dashboard、settings、mcp 共 5 个 Store（39 个用例，含 scope getter） | WEB-30, WEB-35, WEB-39, WEB-42 |
| TEST-19 | 新 Composables 测试 | usePlatform、useSkills、useMemory、useMcp（12 个用例） | WEB-24, WEB-30, WEB-35, WEB-39 |
| TEST-20 | 组件测试 — 布局 | VscodeLayout、BrowserLayout、AppSidebar、AppHeader、RightPanel、GlobalModals（14 个用例） | WEB-25~29 |
| TEST-21 | 组件测试 — 技能面板 | SkillsPanel、SkillDetailPanel、SkillCard、SkillInstallDialog、SkillSolidifyPanel（14 个用例） | WEB-37 |
| TEST-22 | 组件测试 — 记忆 + MCP + 文件面板 | MemoryPanel、MemoryCard、MemoryEditDialog、McpSettingsPanel、FilesPanel（18 个用例） | WEB-38, WEB-36, WEB-43 |
| TEST-23 | 组件测试 — 仪表盘 + 设置 + 终端面板 | DashboardPanel、SettingsPanel、ApprovalPolicyPanel、TerminalPanel（16 个用例） | WEB-39~42 |
| TEST-24 | E2E — 模式分流 | mode-routing.spec.ts（3 个用例） | WEB-32 |
| TEST-25 | E2E — 右侧面板 | right-panel.spec.ts（5 个用例，含 Tab 切换、宽高拖拽、全局/工作区 Tab 差异） | WEB-29 |
| TEST-26 | E2E — 技能管理 | skills.spec.ts（4 个用例，含 scope 筛选） | WEB-37 |
| TEST-27 | E2E — 记忆管理 | memory.spec.ts（4 个用例，含 scope 筛选） | WEB-38 |
| TEST-28 | E2E — MCP 工具 | mcp-tools.spec.ts（3 个用例） | WEB-43 |
| TEST-29 | E2E — 仪表盘 | dashboard.spec.ts（2 个用例） | WEB-39 |
| TEST-30 | E2E — 全局/工作区模式 | workspace-scope.spec.ts（4 个用例） | WEB-28, WEB-31 |
| TEST-31 | E2E — 左侧栏 | sidebar.spec.ts（1 个用例） | WEB-28 |

---

## 9. 验收标准

### Phase 1 验收（v1.0.0）✅ 已完成

1. **`npm test` 全部通过**：所有 Vitest 测试用例（111 个）零失败
2. **覆盖率达标**：行覆盖率 ≥ 80%，分支覆盖率 ≥ 70%，核心模块（stores、composables）≥ 90%
3. **`npm run test:e2e` 全部通过**：所有 Playwright 测试用例（20 个）在 Chromium 和 Firefox 上零失败
4. **测试可重复**：连续运行 3 次，结果一致
5. **测试速度**：`npm test` 在 30 秒内完成，`npm run test:e2e` 在 5 分钟内完成
6. **CI 通过**：GitHub Actions 上 unit + e2e 两个 job 全部通过
7. **测试隔离**：任意单独运行一个测试文件，结果与全量运行一致
8. **Mock 完整**：单元测试和组件测试不依赖真实后端服务

### Phase 1 扩展 ~ Phase 5 验收（v1.2.0）

1. **`npm test` 全部通过**：所有 Vitest 测试用例（250 个）零失败
2. **覆盖率达标**：行覆盖率 ≥ 80%，分支覆盖率 ≥ 70%，新增模块（skills、memory、mcp、dashboard、settings、panels）≥ 85%
3. **`npm run test:e2e` 全部通过**：所有 Playwright 测试用例（46 个）在 Chromium 和 Firefox 上零失败
4. **模式分流测试**：VSCode 模式与浏览器模式布局切换正确，功能无回归
5. **三栏面板测试**：浏览器模式下右侧面板 Tab 切换不离开聊天上下文，全局模式显示 3 个 Tab，工作区模式显示 6 个 Tab
6. **scope 分层测试**：Skills 和 Memory 面板支持 global/workspace/all 三种 scope 筛选
7. **全局/工作区双模式**：无 workspace 时可配置全局设置，选择 workspace 后显示项目面板
8. **左侧栏测试**：workspace 列表和会话列表正确显示，切换 workspace 更新会话列表
9. **新 Store 完整**：skills、memory、dashboard、settings、mcp 五个 Store 的 API 调用、状态变更、SSE 事件处理、scope getter 均覆盖
10. **测试速度**：`npm test` 在 60 秒内完成，`npm run test:e2e` 在 10 分钟内完成
11. **Mock 完整**：新 API 端点均有 Mock 支持，不依赖真实后端服务