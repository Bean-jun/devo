import { test, expect } from '@playwright/test'

test.describe('Approval Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
  })

  test('should show approval modal for high-risk operation', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    // Approval modal should appear
    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 15_000 })

    // Risk level badge should be visible
    await expect(modal.locator('[data-test="risk-level"]')).toBeVisible()
  })

  test('should execute action on approve click', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    // Wait for approval modal
    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 15_000 })

    // Click approve
    await modal.locator('[data-test="approve-button"]').click()

    // Modal should close
    await expect(modal).not.toBeVisible({ timeout: 5_000 })
  })

  test('should reject action on reject click', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 15_000 })

    // Click reject
    await modal.locator('[data-test="reject-button"]').click()

    // Modal should close
    await expect(modal).not.toBeVisible({ timeout: 5_000 })
  })

  test('should approve on Y key', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 15_000 })

    // Press Y to approve
    await page.keyboard.press('y')

    await expect(modal).not.toBeVisible({ timeout: 5_000 })
  })

  test('should reject on N key', async ({ page }) => {
    const input = page.locator('[data-test="message-input"]')
    await input.fill('Delete the file important.txt')
    await input.press('Enter')

    const modal = page.locator('[data-test="approval-modal"]')
    await expect(modal).toBeVisible({ timeout: 15_000 })

    // Press N to reject
    await page.keyboard.press('n')

    await expect(modal).not.toBeVisible({ timeout: 5_000 })
  })
})