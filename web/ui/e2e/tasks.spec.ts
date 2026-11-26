import http, { type Server } from 'node:http'
import type { AddressInfo } from 'node:net'
import { expect, test } from '@playwright/test'

let server: Server | undefined
let fileURL = ''

test.beforeAll(async () => {
  server = http.createServer((_request, response) => {
    response.writeHead(200, { 'Content-Type': 'text/plain' })
    response.end('SaveAny-Bot UI E2E\n')
  })
  const listening = Promise.withResolvers<void>()
  server.listen(0, '127.0.0.1', listening.resolve)
  await listening.promise
  const address = server.address() as AddressInfo
  fileURL = `http://127.0.0.1:${address.port}/ui-e2e.txt`
})

test.afterAll(async () => {
  if (server) {
    const closed = Promise.withResolvers<void>()
    server.close(() => closed.resolve())
    await closed.promise
  }
})

test('creates a direct-links task from the UI', async ({ page }) => {
  await page.goto('/tasks')
  const target = `ui-${Date.now()}`

  await page.getByRole('button', { name: 'Create task' }).first().click()
  await page.locator('#task-type').click()
  await page.getByRole('option', { name: 'Direct link' }).click()
  await page.locator('#task-field-urls').fill(fileURL)
  await page.locator('#task-path').fill(target)
  await page.getByRole('dialog').getByRole('button', { name: 'Create task' }).click()

  const row = page.locator('.task-table tbody tr', { hasText: target })
  await expect(row).toBeVisible()
  await expect(row).toContainText('Completed')
})

test('uploads a local file from the UI', async ({ page }) => {
  await page.goto('/tasks')
  const target = `ui-upload-${Date.now()}`

  await page.getByRole('button', { name: 'Create task' }).first().click()
  await page.locator('#task-type').click()
  await page.getByRole('option', { name: 'Local upload' }).click()
  await page.locator('#task-upload-file').setInputFiles({
    name: 'browser-upload.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('SaveAny-Bot browser upload\n'),
  })
  await page.locator('#task-path').fill(target)
  await page.getByRole('dialog').getByRole('button', { name: 'Create task' }).click()

  const row = page.locator('.task-table tbody tr', { hasText: target })
  await expect(row).toBeVisible()
  await expect(row).toContainText('Completed')
})

test('confirms before discarding an in-progress upload', async ({ page }) => {
  await page.route('**/api/v1/uploads?*', () => {})
  await page.goto('/tasks')
  await page.getByRole('button', { name: 'Create task' }).first().click()
  const form = page.getByRole('dialog', { name: 'Create task' })
  await form.locator('#task-type').click()
  await page.getByRole('option', { name: 'Local upload' }).click()
  await form.locator('#task-upload-file').setInputFiles({
    name: 'discard.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('in progress'),
  })
  const sent = page.waitForRequest((req) => req.url().includes('/api/v1/uploads?'))
  await form.getByRole('button', { name: 'Create task' }).click()
  await sent
  const progress = form.getByRole('status')
  await expect(progress).toBeVisible()

  await form.getByRole('button', { name: 'Close' }).click()
  const discard = page.getByRole('alertdialog', { name: 'Discard this upload?' })
  await expect(discard).toBeVisible()
  await discard.getByRole('button', { name: 'Cancel' }).click()
  await expect(discard).not.toBeVisible()
  await expect(progress).toBeVisible()

  await form.getByRole('button', { name: 'Cancel' }).click()
  const aborted = page.waitForEvent('requestfailed', {
    predicate: (req) => req.url().includes('/api/v1/uploads?'),
  })
  await discard.getByRole('button', { name: 'Discard upload' }).click()
  await aborted
  await expect(form).not.toBeVisible()
})
