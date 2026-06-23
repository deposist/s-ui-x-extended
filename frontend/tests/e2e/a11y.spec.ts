import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { login, setEnglishLocale, writeJSONArtifact } from './helpers'

test.setTimeout(90_000)

type AxeViolation = {
  id: string
  impact: string | null
  nodes: unknown[]
}

type AxeResult = {
  violations?: AxeViolation[]
}

const criticalViolationSummaries = (name: string, result: AxeResult): string[] =>
  (result.violations ?? [])
    .filter(violation => violation.impact === 'critical')
    .map(violation => `${name}:${violation.id}:${violation.nodes.length}`)

test('axe baseline for login and authenticated pages', async ({ page }) => {
  await setEnglishLocale(page)

  const results: Record<string, unknown> = {}
  const criticalFailures: string[] = []
  await page.goto('login')
  const loginResult = await new AxeBuilder({ page }).analyze()
  results.login = loginResult
  criticalFailures.push(...criticalViolationSummaries('login', loginResult))

  await login(page)
  for (const [name, route] of [
    ['dashboard', ''],
    ['migrate-xui', 'migrate-xui'],
    ['settings', 'settings'],
    ['rules', 'rules'],
    ['dns', 'dns'],
    ['audit', 'audit'],
  ] as const) {
    await page.goto(route)
    await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined)
    try {
      const result = await new AxeBuilder({ page }).analyze()
      results[name] = result
      criticalFailures.push(...criticalViolationSummaries(name, result))
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      results[name] = {
        error: message,
        url: page.url(),
      }
      criticalFailures.push(`${name}:axe-error:${message}`)
    }
  }

  writeJSONArtifact('a11y/axe-results.json', results)
  expect(criticalFailures, 'critical axe violations').toEqual([])
})
