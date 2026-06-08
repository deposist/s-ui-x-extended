import { expect, test } from '@playwright/test'
import { login } from './helpers'

test('dashboard renders metric content without a blank main view', async ({ page }) => {
  await login(page)
  await page.goto('')

  await expect(page.locator('body')).toBeVisible()
  await expect(page.locator('body')).toContainText(/Usage & Counts|Logs|Sing-Box Error/i)
  const metricSurfaceCount = await page.locator('canvas, svg, .v-progress-circular, .v-progress-linear, .v-card').count()
  expect(metricSurfaceCount).toBeGreaterThan(0)
})
