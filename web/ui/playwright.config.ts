import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from '@playwright/test'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const port = process.env.E2E_PORT || '28081'
const baseURL = process.env.E2E_BASE_URL || `http://127.0.0.1:${port}`
const bin = process.env.SAVEANY_E2E_BIN || path.join(repoRoot, 'tmp', 'saveany-bot-e2e')
const outputDir = path.join(repoRoot, 'tmp', 'e2e-web-results')
const authState = path.join(outputDir, 'auth.json')
process.env.SAVEANY_E2E_AUTH_STATE = authState

export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [['list']],
  outputDir,
  projects: [
    { name: 'setup', testMatch: /auth\.setup\.ts/ },
    {
      name: 'chromium',
      testIgnore: /auth\.setup\.ts/,
      dependencies: ['setup'],
      use: { storageState: authState },
    },
  ],
  use: {
    baseURL,
    locale: 'en-US',
    viewport: { width: 1280, height: 900 },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: `"${bin}" testenv --listen 127.0.0.1:${port} --admin-password test-admin --storage-name test-local --user-chat-id 10001`,
        url: `${baseURL}/health`,
        cwd: repoRoot,
        timeout: 120_000,
        reuseExistingServer: false,
        gracefulShutdown: { signal: 'SIGINT', timeout: 10_000 },
      },
})
