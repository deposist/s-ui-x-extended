import { test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { login, setEnglishLocale, writeJSONArtifact } from './helpers'

test.setTimeout(90_000)

test('axe baseline for login and authenticated pages', async ({ page }) => {
  await setEnglishLocale(page)

  const results: Record<string, unknown> = {}
  await page.goto('login')
  results.login = await new AxeBuilder({ page }).analyze()

  await login(page)
  for (const [name, route] of [
    ['dashboard', ''],
    ['migrate-xui', 'migrate-xui'],
    ['settings', 'settings'],
    ['audit', 'audit'],
  ] as const) {
    await page.goto(route)
    await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined)
    try {
      results[name] = await new AxeBuilder({ page }).analyze()
    } catch (error) {
      results[name] = {
        error: error instanceof Error ? error.message : String(error),
        url: page.url(),
      }
    }
  }

  writeJSONArtifact('a11y/axe-results.json', results)
})
