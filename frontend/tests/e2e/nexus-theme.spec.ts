import { expect, test } from '@playwright/test'
import { login } from './helpers'

test('nexus light theme keeps layout tokens after switching from classic', async ({ page }) => {
  await page.addInitScript(() => {
    window.localStorage.setItem('theme', 'light')
    window.localStorage.setItem('sui:ui:mode', 'classic')
    window.localStorage.setItem('sui:ui:palette', 'technical')
  })

  await login(page)
  await expect(page.locator('.nexus-shell')).toHaveCount(0)

  await page.getByRole('button', { name: 'Switch to Nexus mode' }).click()

  const shell = page.locator('.nexus-shell')
  await expect(shell).toBeVisible()
  await expect.poll(() => page.evaluate(() => document.documentElement.dataset.uiMode)).toBe('nexus')

  const layout = await shell.evaluate((element) => {
    const styles = getComputedStyle(element)
    const sidebar = document.querySelector('.nexus-sidebar')?.getBoundingClientRect()
    const topbar = document.querySelector('.nexus-topbar')?.getBoundingClientRect()
    const themeClass = Array
      .from(element.closest('[class*="v-theme--"]')?.classList ?? [])
      .find(className => className.startsWith('v-theme--'))

    return {
      gap4: styles.getPropertyValue('--nexus-gap-4').trim(),
      radiusLg: styles.getPropertyValue('--nexus-radius-lg').trim(),
      surface0: styles.getPropertyValue('--nexus-surface-0').trim(),
      themeClass,
      topbarHeight: Math.round(topbar?.height ?? 0),
      sidebarWidth: Math.round(sidebar?.width ?? 0),
    }
  })

  expect(layout).toMatchObject({
    gap4: '16px',
    radiusLg: '8px',
    surface0: '#f0f2f5',
    themeClass: 'v-theme--technicalLight',
    sidebarWidth: 240,
  })
  expect(layout.topbarHeight).toBeGreaterThanOrEqual(64)
  expect(layout.topbarHeight).toBeLessThanOrEqual(65)
})
