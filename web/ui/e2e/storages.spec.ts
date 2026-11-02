import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { expect, test } from '@playwright/test'

test('tests the seeded storage', async ({ page }) => {
  await page.goto('/storages')
  const card = page.locator('.storage-card', { hasText: 'test-local' })
  await expect(card).toBeVisible()
  await card.getByRole('button', { name: 'Test' }).click()
  await expect(page.locator('body')).toContainText('test-local is reachable')
})

test('creates a local storage', async ({ page }) => {
  await page.goto('/storages')
  const name = `ui-${Date.now()}`
  const basePath = fs.mkdtempSync(path.join(os.tmpdir(), 'saveany-ui-storage-'))

  await page.getByRole('button', { name: 'Create storage' }).first().click()
  await page.locator('#storage-name').fill(name)
  await page.locator('#storage-type').click()
  await page.getByRole('option', { name: 'Local disk' }).click()
  await page.locator('#storage-field-base_path').fill(basePath)
  await page.getByRole('dialog').getByRole('button', { name: 'Create storage' }).click()

  await expect(page.locator('.storage-card', { hasText: name })).toBeVisible()
})
