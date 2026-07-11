import { test, expect } from '@playwright/test'

test.describe('Mobile Layout', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 })
    await page.goto('/?mode=mobile')
    await page.waitForSelector('[data-test="mobile-layout"]', { timeout: 10_000 })
  })

  test('should render mobile layout shell', async ({ page }) => {
    const layout = page.locator('[data-test="mobile-layout"]')
    await expect(layout).toBeVisible()
  })

  test('should render status bar', async ({ page }) => {
    const statusBar = page.locator('[data-test="status-bar"]')
    await expect(statusBar).toBeVisible()
  })

  test('should render textarea and command button', async ({ page }) => {
    const textarea = page.locator('[data-test="mobile-input-textarea"]')
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')

    await expect(textarea).toBeVisible()
    await expect(cmdBtn).toBeVisible()
    await expect(cmdBtn).toContainText('/')
  })

  test('should open command sheet on / button click', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const sheet = page.locator('[data-test="command-sheet"]')
    await expect(sheet).toBeVisible({ timeout: 3_000 })
  })

  test('should close command sheet on backdrop click', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const overlay = page.locator('[data-test="command-sheet-overlay"]')
    await overlay.click({ position: { x: 10, y: 10 } })

    const sheet = page.locator('[data-test="command-sheet"]')
    await expect(sheet).not.toBeVisible({ timeout: 3_000 })
  })

  test('should filter commands by search', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const searchInput = page.locator('[data-test="command-search-input"]')
    await searchInput.fill('new')

    const items = page.locator('[data-test="command-item"]')
    await expect(items).toHaveCount(1)
    await expect(items.first()).toContainText('/new')
  })

  test('should show empty state when no commands match', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const searchInput = page.locator('[data-test="command-search-input"]')
    await searchInput.fill('zzz_nonexistent_cmd')

    const empty = page.locator('[data-test="sheet-empty"]')
    await expect(empty).toBeVisible()
  })

  test('should open panel drawer on panel command', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const filesCmd = page.locator('[data-test="command-item"]').filter({ hasText: '/files' })
    await filesCmd.click()

    const drawer = page.locator('[data-test="panel-drawer"]')
    await expect(drawer).toBeVisible({ timeout: 3_000 })
  })

  test('should close panel drawer on back button', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const filesCmd = page.locator('[data-test="command-item"]').filter({ hasText: '/files' })
    await filesCmd.click()

    const backBtn = page.locator('[data-test="drawer-back-btn"]')
    await backBtn.click()

    const drawer = page.locator('[data-test="panel-drawer"]')
    await expect(drawer).not.toBeVisible({ timeout: 3_000 })
  })

  test('should switch panel tabs', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const filesCmd = page.locator('[data-test="command-item"]').filter({ hasText: '/files' })
    await filesCmd.click()

    const skillsTab = page.locator('[data-test="drawer-tab"]').filter({ hasText: 'Skills' })
    await skillsTab.click()

    await expect(skillsTab).toHaveClass(/active/)
  })

  test('should open workspace picker on workspace-switch command', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const wsCmd = page.locator('[data-test="command-item"]').filter({ hasText: '/workspace-switch' })
    await wsCmd.click()

    const picker = page.locator('[data-test="workspace-picker"]')
    await expect(picker).toBeVisible({ timeout: 3_000 })
  })

  test('should open session picker on switch command', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const switchCmd = page.locator('[data-test="command-item"]').filter({ hasText: '/switch' })
    await switchCmd.click()

    const picker = page.locator('[data-test="session-picker"]')
    await expect(picker).toBeVisible({ timeout: 3_000 })
  })

  test('should show new session dialog on /new command', async ({ page }) => {
    const cmdBtn = page.locator('[data-test="mobile-command-btn"]')
    await cmdBtn.click()

    const newCmd = page.locator('[data-test="command-item"]').filter({ hasText: '/new' })
    await newCmd.click()

    const dialog = page.locator('[data-test="new-session-dialog"]')
    await expect(dialog).toBeVisible({ timeout: 3_000 })
  })

  test('should send message on Enter', async ({ page }) => {
    const textarea = page.locator('[data-test="mobile-input-textarea"]')
    await textarea.fill('Hello, AI!')
    await textarea.press('Enter')

    const userMessage = page.locator('[data-test="message-bubble"]')
    await expect(userMessage).toBeVisible({ timeout: 10_000 })
    await expect(userMessage).toContainText('Hello, AI!')
  })

  test('should show stop button when processing', async ({ page }) => {
    const textarea = page.locator('[data-test="mobile-input-textarea"]')
    await textarea.fill('Write a long story')
    await textarea.press('Enter')

    const stopBtn = page.locator('[data-test="mobile-stop-btn"]')
    await expect(stopBtn).toBeVisible({ timeout: 5_000 })
  })

  test('should show footer with context and tokens', async ({ page }) => {
    const footer = page.locator('[data-test="mobile-input-footer"]')
    await expect(footer).toBeVisible()
    await expect(footer).toContainText('Context')
    await expect(footer).toContainText('Tokens')
  })

  test('should show FPS counter', async ({ page }) => {
    const fps = page.locator('[data-test="fps-counter"]')
    await expect(fps).toBeVisible()
  })
})