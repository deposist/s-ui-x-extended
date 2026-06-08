import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 45_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  workers: 1,
  outputDir: '../tests/baseline/phase6/playwright/test-results',
  reporter: [
    ['list'],
    ['junit', { outputFile: '../tests/baseline/phase6/playwright.junit.xml' }],
    ['html', { outputFolder: '../tests/baseline/phase6/playwright/html', open: 'never' }],
  ],
  use: {
    baseURL: process.env.SUI_E2E_BASE_URL ?? 'http://127.0.0.1:3000/app/',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: process.env.SUI_E2E_SKIP_WEB_SERVER === '1' ? undefined : {
    command: 'node ../tests/e2e/run-server.js',
    url: process.env.SUI_E2E_BASE_URL ?? 'http://127.0.0.1:3000/app/login',
    reuseExistingServer: true,
    timeout: 180_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
