import { test, expect } from '@playwright/test'

const API_BASE = 'http://localhost:8081'

async function getActiveSession(request: any): Promise<{ id: string; count: number }> {
  const resp = await request.get(`${API_BASE}/api/v1/sessions`)
  const json = await resp.json()
  const session = json.sessions?.[0]
  return { id: session.id, count: session.message_count }
}

async function getUserMessageCount(request: any, sessionId: string): Promise<number> {
  const resp = await request.get(`${API_BASE}/api/v1/sessions/${sessionId}/messages`)
  const json = await resp.json()
  return (json.messages || []).filter((m: any) => m.role === 'user').length
}

async function getLastUserMessage(request: any, sessionId: string): Promise<string> {
  const resp = await request.get(`${API_BASE}/api/v1/sessions/${sessionId}/messages`)
  const json = await resp.json()
  const userMsgs = (json.messages || []).filter((m: any) => m.role === 'user')
  const last = userMsgs[userMsgs.length - 1]
  return String(last?.content || last?.text || '')
}

async function waitForNewUserMessage(
  request: any,
  sessionId: string,
  countBefore: number,
  timeoutMs = 10_000,
): Promise<void> {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const count = await getUserMessageCount(request, sessionId)
    if (count > countBefore) return
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`Timed out waiting for user message count to exceed ${countBefore}`)
}

async function pasteLargeText(page: any, text: string): Promise<void> {
  await page.evaluate((t) => {
    const dt = new DataTransfer()
    dt.setData('text/plain', t)
    const ev = new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true })
    document.querySelector('[data-test="message-input"]')!.dispatchEvent(ev)
  }, text)
}

test.describe.serial('Input Area Paste Behavior', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForSelector('[data-test="chat-panel"]', { timeout: 10_000 })
    await page.waitForSelector('[data-test="message-input"]', { timeout: 5_000 })
  })

  test('should fold large paste into a chip', async ({ page }) => {
    const editor = page.locator('[data-test="message-input"]')
    await editor.click()

    const largeText = Array.from({ length: 50 }, (_, i) => `line ${i + 1}`).join('\n')
    await pasteLargeText(page, largeText)

    const chip = editor.locator('.paste-chip')
    await expect(chip).toBeVisible({ timeout: 5_000 })
    await expect(chip).toContainText('Pasted text #1')
    await expect(chip).toContainText('+50 lines')
  })

  test('should send full paste content on Enter when chip is present', async ({ page, request }) => {
    const session = await getActiveSession(request)
    const userCountBefore = await getUserMessageCount(request, session.id)

    const editor = page.locator('[data-test="message-input"]')
    await editor.click()

    const largeText = Array.from({ length: 22 }, (_, i) => `pasted line ${i + 1}`).join('\n')
    await pasteLargeText(page, largeText)

    await editor.locator('.paste-chip').waitFor({ timeout: 5_000 })
    await editor.press('Enter')

    await waitForNewUserMessage(request, session.id, userCountBefore)

    const lastUserText = await getLastUserMessage(request, session.id)
    expect(lastUserText).toContain('pasted line 1')
    expect(lastUserText).toContain('pasted line 22')
    expect(lastUserText).not.toContain('Pasted text #')
  })

  test('should not fold small paste below threshold', async ({ page }) => {
    const editor = page.locator('[data-test="message-input"]')
    await editor.click()

    await editor.fill('short text\nsecond line')

    await expect(editor.locator('.paste-chip')).toHaveCount(0, { timeout: 2_000 })
    const text = await editor.innerText()
    expect(text).toContain('short text')
    expect(text).toContain('second line')
  })

  test('should delete whole chip on backspace at boundary', async ({ page }) => {
    const editor = page.locator('[data-test="message-input"]')
    await editor.click()

    const largeText = Array.from({ length: 10 }, (_, i) => `line ${i + 1}`).join('\n')
    await pasteLargeText(page, largeText)

    const chip = editor.locator('.paste-chip')
    await chip.waitFor({ timeout: 5_000 })

    // Move cursor to end then backspace
    await page.evaluate(() => {
      const ed = document.querySelector('[data-test="message-input"]') as HTMLElement
      const range = document.createRange()
      range.selectNodeContents(ed)
      range.collapse(false)
      const sel = window.getSelection()!
      sel.removeAllRanges()
      sel.addRange(range)
    })
    await editor.press('Backspace')

    await expect(editor.locator('.paste-chip')).toHaveCount(0, { timeout: 2_000 })
  })

  test('should allow typing text before chip and send both', async ({ page, request }) => {
    const session = await getActiveSession(request)
    const userCountBefore = await getUserMessageCount(request, session.id)

    const editor = page.locator('[data-test="message-input"]')
    await editor.click()

    await editor.type('prefix text ')

    const largeText = Array.from({ length: 8 }, (_, i) => `line ${i + 1}`).join('\n')
    await pasteLargeText(page, largeText)

    const chip = editor.locator('.paste-chip')
    await chip.waitFor({ timeout: 5_000 })
    await editor.press('Enter')

    await waitForNewUserMessage(request, session.id, userCountBefore)

    const lastUserText = await getLastUserMessage(request, session.id)
    expect(lastUserText).toContain('prefix text')
    expect(lastUserText).toContain('line 1')
    expect(lastUserText).toContain('line 8')
    expect(lastUserText).not.toContain('Pasted text #')
  })

  test('should insert newline on Shift+Enter', async ({ page, request }) => {
    const session = await getActiveSession(request)
    const userCountBefore = await getUserMessageCount(request, session.id)

    const editor = page.locator('[data-test="message-input"]')
    await editor.click()

    await editor.type('first')
    await editor.press('Shift+Enter')
    await editor.type('second')
    await editor.press('Enter')

    await waitForNewUserMessage(request, session.id, userCountBefore)

    const lastUserText = await getLastUserMessage(request, session.id)
    expect(lastUserText).toContain('first')
    expect(lastUserText).toContain('second')
    expect(lastUserText.split('\n').length).toBeGreaterThan(1)
  })
})
