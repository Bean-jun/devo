import { test, expect } from '@playwright/test'

test.describe('Rollback', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
  })

  test('should open rollback picker and select a message', async ({ page }) => {
    // Send a few messages first
    const input = page.locator('[data-test="message-input"]')

    await input.fill('Message 1')
    await input.press('Enter')
    await page.waitForTimeout(1000)

    await input.fill('Message 2')
    await input.press('Enter')
    await page.waitForTimeout(1000)

    // Open command palette and select rollback
    await page.keyboard.press('Control+k')
    const palette = page.locator('[data-test="command-palette"]')
    await expect(palette).toBeVisible()

    const rollbackInput = palette.locator('input')
    await rollbackInput.fill('/rollback')
    await rollbackInput.press('Enter')

    // Rollback picker should be visible
    const rollbackPicker = page.locator('.rollback-picker')
    await expect(rollbackPicker).toBeVisible({ timeout: 5_000 })

    // Messages should be listed
    const timelineItems = rollbackPicker.locator('.timeline-item')
    const count = await timelineItems.count()
    expect(count).toBeGreaterThanOrEqual(2)
  })

  test('should close rollback picker on Escape', async ({ page }) => {
    await page.keyboard.press('Control+k')
    const palette = page.locator('[data-test="command-palette"]')
    await palette.locator('input').fill('/rollback')
    await palette.locator('input').press('Enter')

    const rollbackPicker = page.locator('.rollback-picker')
    await expect(rollbackPicker).toBeVisible({ timeout: 5_000 })

    await page.keyboard.press('Escape')
    await expect(rollbackPicker).not.toBeVisible({ timeout: 3_000 })
  })
})