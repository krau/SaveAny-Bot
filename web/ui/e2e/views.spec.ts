import { expect, test } from '@playwright/test'

const views: Array<[string, string]> = [
  ['/', 'Overview'],
  ['/guide', 'Guide'],
  ['/tasks', 'Tasks'],
  ['/storages', 'Storages'],
  ['/users', 'Users'],
  ['/telegram', 'Telegram'],
  ['/settings', 'Settings'],
  ['/system', 'System'],
]

test('loads every console view without errors', async ({ page }) => {
  const failures: string[] = []
  page.on('pageerror', (error) => failures.push(`pageerror: ${error.message}`))
  page.on('response', (response) => {
    if (response.status() >= 500) {
      failures.push(`${response.status()} ${response.request().method()} ${response.url()}`)
    }
  })

  for (const [route, heading] of views) {
    await page.goto(route)
    await expect(page.getByRole('heading', { name: heading }).first()).toBeVisible()
  }
  expect(failures).toEqual([])
})
