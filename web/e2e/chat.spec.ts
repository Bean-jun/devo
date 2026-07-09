import { test, expect } from '@playwright/test'

test.describe('Chat Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
  })

  test('should send a message and see it in chat', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Hello, AI!')
    await input.press('Enter')

    // User message appears in message list
    const userMessage = page.locator('[data-test="message-bubble"].role-user')
    await expect(userMessage).toBeVisible({ timeout: 10_000 })
    await expect(userMessage).toContainText('Hello, AI!')

    // Input should be cleared
    await expect(input).toHaveValue('')
  })

  test('should display thinking indicator during processing', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Write a Go function to calculate fibonacci')
    await input.press('Enter')

    // Thinking indicator should appear
    const thinkingIndicator = page.locator('[data-test="thinking-indicator"]')
    await expect(thinkingIndicator).toBeVisible({ timeout: 10_000 })

    // Wait for response to complete
    await page.waitForSelector('[data-test="thinking-indicator"]', {
      state: 'detached',
      timeout: 60_000,
    })
  })

  test('should display tool call card', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Create a file called test.txt with content "Hello World"')
    await input.press('Enter')

    // Tool call card should appear
    const toolCard = page.locator('[data-test="tool-call-card"]')
    await expect(toolCard.first()).toBeVisible({ timeout: 30_000 })

    // Tool name should be displayed
    await expect(toolCard.locator('[data-test="tool-name"]').first()).toBeVisible()
  })

  test('should stop processing on stop button click', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Write a very long story about space exploration')
    await input.press('Enter')

    // Wait for stop button to appear
    const stopButton = page.locator('[data-test="stop-button"]')
    await expect(stopButton).toBeVisible({ timeout: 5_000 })

    // Click stop
    await stopButton.click()

    // Status should return to idle
    await expect(page.locator('.status-indicator')).toContainText('空闲', { timeout: 10_000 })
  })

  test('should not send empty message', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('   ')
    await input.press('Enter')

    // Message should not appear
    const messages = page.locator('[data-test="message-bubble"]')
    const count = await messages.count()
    expect(count).toBe(0)
  })

  test('should support Shift+Enter for newline', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Line 1')
    await input.press('Shift+Enter')
    await input.press('b')
    await input.press('b')

    const value = await input.inputValue()
    expect(value).toContain('\n')
  })
})

test.describe('Tool Streaming', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
  })

  test('should display streaming output chunk by chunk', async ({ page }) => {
    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.appendToolCallMessage({
        id: 'tool-stream-001',
        name: 'search_codebase',
        parameters: { query: 'main function' },
        status: 'pending',
        riskLevel: 'low',
      })
    })

    const toolCard = page.locator('[data-test="tool-call-card"]')
    await expect(toolCard).toBeVisible({ timeout: 5_000 })
    await expect(toolCard.locator('[data-test="tool-name"]')).toContainText('search_codebase')

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.updateToolProgress('tool-stream-001', 'running')
      store.appendToolStreamChunk('tool-stream-001', '找到 3 个匹配...\n')
    })
    await expect(page.locator('[data-test="tool-streaming"]')).toContainText('找到 3 个匹配')

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.appendToolStreamChunk('tool-stream-001', '文件: main.go 第 12 行\n')
    })
    await expect(page.locator('[data-test="tool-streaming"]')).toContainText('main.go')

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.appendToolStreamChunk('tool-stream-001', '文件: utils.go 第 45 行\n')
    })
    await expect(page.locator('[data-test="tool-streaming"]')).toContainText('utils.go')

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.updateToolCallStatus('tool-stream-001', 'success', {
        success: true,
        stdout: '搜索完成，共 3 个匹配',
      })
    })
    await expect(page.locator('[data-test="tool-streaming"]')).not.toBeVisible()
    await expect(page.locator('.tool-result')).toContainText('搜索完成')
  })

  test('should show stage transitions', async ({ page }) => {
    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.appendToolCallMessage({
        id: 'tool-progress-001',
        name: 'exec_python',
        parameters: { command: 'go build ./...' },
        status: 'pending',
        riskLevel: 'medium',
      })
    })

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.updateToolProgress('tool-progress-001', 'starting')
    })
    await expect(page.locator('[data-test="tool-stage"]')).toContainText('starting')

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.updateToolProgress('tool-progress-001', 'running')
    })
    await expect(page.locator('[data-test="tool-stage"]')).toContainText('running')

    await page.evaluate(() => {
      const store = (window as any).__chatStore
      store.updateToolProgress('tool-progress-001', 'done')
    })
    await expect(page.locator('[data-test="tool-stage"]')).toContainText('done')
  })
})