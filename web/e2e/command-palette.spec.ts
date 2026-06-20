import { test, expect } from '@playwright/test'

test.describe('Command Palette', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
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
    const count = await items.count()
    expect(count).toBeGreaterThanOrEqual(1)

    // At least one item should contain 'new'
    const firstText = await items.first().textContent()
    expect(firstText?.toLowerCase()).toContain('new')
  })

  test('should close on Escape', async ({ page }) => {
    await page.keyboard.press('Control+k')

    const palette = page.locator('[data-test="command-palette"]')
    await expect(palette).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(palette).not.toBeVisible()
  })

  test('should show all built-in commands', async ({ page }) => {
    await page.keyboard.press('Control+k')

    const palette = page.locator('[data-test="command-palette"]')
    const items = palette.locator('[data-test="command-item"]')

    const count = await items.count()
    expect(count).toBeGreaterThanOrEqual(3)
  })
})