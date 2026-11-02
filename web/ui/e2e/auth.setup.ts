import fs from 'node:fs'
import path from 'node:path'
import { test } from '@playwright/test'
import { login } from './helpers'

test('authenticate', async ({ page }) => {
  await login(page)
  const statePath = process.env.SAVEANY_E2E_AUTH_STATE
  if (!statePath) {
    throw new Error('SAVEANY_E2E_AUTH_STATE is not set')
  }
  fs.mkdirSync(path.dirname(statePath), { recursive: true })
  await page.context().storageState({ path: statePath })
})
