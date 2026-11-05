import { expect, test } from '@playwright/test'
import { login } from './helpers'

test.describe('unauthenticated', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('rejects a wrong password', async ({ page }) => {
    await page.goto('/login')
    await page.locator('#login-password').fill('wrong')
    await page.locator('form').getByRole('button').click()
    await expect(page).toHaveURL(/\/login$/)
    await expect(page.locator('#login-password')).toHaveAttribute('aria-invalid', 'true')
  })

  test('logs in and loads the overview', async ({ page }) => {
    await login(page)
    await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible()
    await page.getByRole('link', { name: 'Storages' }).click()
    await expect(page).toHaveURL(/\/storages$/)
    await expect(page.locator('body')).toContainText('test-local')
  })
})
