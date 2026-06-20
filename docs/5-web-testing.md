# Devo Web 前端测试文档

**版本**：1.0.0

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

  // 失败重试
  retries: process.env.CI ? 2 : 0,

  // 并行执行
  fullyParallel: true,
  workers: process.env.CI ? 2 : undefined,

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
    reuseExistingServer: !process.env.CI,
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
│   │       └── tools.ts               # 工具调用测试数据
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
│   │   └── ui.test.ts                 # UI Store 测试
│   │
│   ├── composables/
│   │   ├── useApi.test.ts             # API 封装测试
│   │   ├── useSSE.test.ts             # SSE 封装测试
│   │   ├── useSession.test.ts         # 会话操作测试
│   │   ├── useCommand.test.ts         # 命令面板逻辑测试
│   │   ├── useKeyboard.test.ts        # 键盘快捷键测试
│   │   └── useAutoScroll.test.ts      # 自动滚动测试
│   │
│   └── components/
│       ├── layout/
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
│       │   └── HelpPanel.test.ts
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
│   └── keyboard.spec.ts              # 键盘快捷键 E2E
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

**总计**：111 个测试用例（单元 43 + 组件 30 + E2E 20 + Utils 8 + Composables 18 = 111 个。不对，让我重新算一下...

- Utils: 8
- Stores: 35
- Composables: 18
- 组件: 30
- E2E: 20

Total: 8 + 35 + 18 + 30 + 20 = 111 个测试用例

---

## 7. CI/CD 集成

### 7.1 GitHub Actions 配置

```yaml
# .github/workflows/web-test.yml
name: Web Tests

on:
  push:
    paths:
      - 'web/**'
  pull_request:
    paths:
      - 'web/**'

jobs:
  unit:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: web
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: npm ci
      - run: npm run test -- --coverage
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: coverage
          path: web/coverage/

  e2e:
    runs-on: ubuntu-latest
    needs: unit
    defaults:
      run:
        working-directory: web
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: web/package-lock.json
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: npm ci
      - run: npx playwright install --with-deps chromium
      - run: npm run test:e2e
      - uses: actions/upload-artifact@v4
        if: failure()
        with:
          name: playwright-report
          path: web/playwright-report/
```

### 7.2 本地开发命令

```bash
# 运行所有单元测试
npm test

# 运行单元测试（监听模式）
npm run test:watch

# 运行单个测试文件
npx vitest run src/stores/session.test.ts

# 运行覆盖率报告
npm run test:coverage

# 运行 E2E 测试
npm run test:e2e

# 运行 E2E 测试（UI 模式，可视化调试）
npm run test:e2e:ui

# 运行单个 E2E 文件
npx playwright test e2e/chat.spec.ts

# 查看 E2E 报告
npm run test:e2e:report
```

---

## 8. 实现计划

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

---

## 9. 验收标准

1. **`npm test` 全部通过**：所有 Vitest 测试用例（111 个）零失败
2. **覆盖率达标**：行覆盖率 ≥ 80%，分支覆盖率 ≥ 70%，核心模块（stores、composables）≥ 90%
3. **`npm run test:e2e` 全部通过**：所有 Playwright 测试用例（20 个）在 Chromium 和 Firefox 上零失败
4. **测试可重复**：连续运行 3 次，结果一致
5. **测试速度**：`npm test` 在 30 秒内完成，`npm run test:e2e` 在 5 分钟内完成
6. **CI 通过**：GitHub Actions 上 unit + e2e 两个 job 全部通过
7. **测试隔离**：任意单独运行一个测试文件，结果与全量运行一致
8. **Mock 完整**：单元测试和组件测试不依赖真实后端服务