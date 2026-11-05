import { expect, type Page } from '@playwright/test'

export async function login(page: Page, password = 'test-admin'): Promise<void> {
  await page.goto('/login')
  await page.locator('#login-password').fill(password)
  await page.locator('form').getByRole('button').click()
  await expect(page).toHaveURL(/\/$/)
}
