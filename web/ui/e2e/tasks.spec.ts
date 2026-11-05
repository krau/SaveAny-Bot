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
