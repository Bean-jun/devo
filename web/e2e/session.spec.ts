import { test, expect } from '@playwright/test'

test.describe('Session Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
  })

  test('should display default session name in status bar', async ({ page }) => {
    const statusBar = page.locator('.statusbar')
    await expect(statusBar).toBeVisible()
    await expect(statusBar.locator('.session-name')).toBeVisible()
  })

  test('should create a new session via button', async ({ page }) => {
    await page.keyboard.press('Control+k')
    const palette = page.locator('[data-test="command-palette"]')
    await expect(palette).toBeVisible()

    const input = palette.locator('input')
    await input.fill('/new E2E Test Session')
    await input.press('Enter')

    const statusBar = page.locator('.statusbar')
    await expect(statusBar.locator('.session-name')).toContainText('E2E Test Session', { timeout: 10_000 })
  })

  test('should switch between sessions', async ({ page }) => {
    // Create first session
    await page.keyboard.press('Control+k')
    let palette = page.locator('[data-test="command-palette"]')
    await palette.locator('input').fill('/new First Session')
    await palette.locator('input').press('Enter')
    await page.waitForTimeout(2000)

    // Create second session
    await page.keyboard.press('Control+k')
    palette = page.locator('[data-test="command-palette"]')
    await palette.locator('input').fill('/new Second Session')
    await palette.locator('input').press('Enter')
    await page.waitForTimeout(2000)

    // Open session picker and switch to first
    await page.keyboard.press('Control+k')
    palette = page.locator('[data-test="command-palette"]')
    await palette.locator('input').fill('/sessions')
    await palette.locator('input').press('Enter')

    const sessionPicker = page.locator('.session-picker')
    await expect(sessionPicker).toBeVisible({ timeout: 5_000 })

    const firstItem = sessionPicker.locator('.picker-item').first()
    await firstItem.click()

    await expect(sessionPicker).not.toBeVisible({ timeout: 5_000 })
  })

  test('should display token usage in status bar', async ({ page }) => {
    const statusBar = page.locator('.statusbar')
    const tokenUsage = statusBar.locator('.token-usage')
    await expect(tokenUsage).toBeVisible()
    await expect(tokenUsage).toContainText('Tokens:')
  })

  test('should display connection status indicator', async ({ page }) => {
    const statusBar = page.locator('.statusbar')
    const connectionStatus = statusBar.locator('.connection-status')
    await expect(connectionStatus).toBeVisible()
  })
})