import { expect, test, type Page } from '@playwright/test'

import { login } from './helpers'

type LayoutMetrics = {
  bodyText: string
  clientWidth: number
  hasInternalTableScroll: boolean
  narrowReadableCells: Array<{ text: string; width: number }>
  scrollWidth: number
  topbarSearchOverlap: boolean
  visibleAlerts: string[]
}

const nexusRoutes = [
  { name: 'dashboard', path: '' },
  { name: 'clients', path: 'clients' },
  { name: 'outbounds', path: 'outbounds' },
  { name: 'rules', path: 'rules' },
  { name: 'dns', path: 'dns' },
  { name: 'settings', path: 'settings' },
] as const

const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
] as const

const readLayoutMetrics = async (page: Page): Promise<LayoutMetrics> =>
  page.evaluate(() => {
    const topbar = document.querySelector('.nexus-topbar')
    const search = topbar?.querySelector('.nexus-topbar__search')
    const append = topbar?.querySelector('.v-toolbar__append')
    const topbarSearchOverlap = Boolean(search && append && (() => {
      const searchRect = search.getBoundingClientRect()
      const appendRect = append.getBoundingClientRect()

      return searchRect.right > appendRect.left &&
        searchRect.left < appendRect.right &&
        searchRect.bottom > appendRect.top &&
        searchRect.top < appendRect.bottom
    })())
    const visibleAlerts = Array
      .from(document.querySelectorAll('[role="alert"], .Notivue__notification, .notivue__notification'))
      .map(element => (element as HTMLElement).innerText.trim())
      .filter(Boolean)
    const denseTables = Array.from(document.querySelectorAll('.nexus-dense-table'))
    const hasInternalTableScroll = denseTables.some(table => table.scrollWidth > table.clientWidth + 1)
    const narrowReadableCells = Array
      .from(document.querySelectorAll('.nexus-dense-table td, .nexus-dense-table th'))
      .map(element => ({
        text: (element as HTMLElement).innerText.trim(),
        width: Math.round(element.getBoundingClientRect().width),
      }))
      .filter(cell => cell.text.length >= 4 && cell.width > 0 && cell.width < 48)

    return {
      bodyText: document.body.innerText,
      clientWidth: document.documentElement.clientWidth,
      hasInternalTableScroll,
      narrowReadableCells,
      scrollWidth: document.documentElement.scrollWidth,
      topbarSearchOverlap,
      visibleAlerts,
    }
  })

test.describe('nexus layout polish', () => {
  for (const viewport of viewports) {
    test(`${viewport.name} has no overflow, topbar overlap, object leaks, or false info toast`, async ({ page }) => {
      test.setTimeout(90_000)

      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await page.addInitScript(() => {
        window.localStorage.setItem('sui:ui:mode', 'nexus')
        window.localStorage.setItem('theme', 'dark')
      })

      await login(page)
      await expect(page.locator('.nexus-shell')).toBeVisible()

      for (const route of nexusRoutes) {
        await test.step(route.name, async () => {
          await page.goto(route.path)
          await expect(page.locator('.nexus-shell')).toBeVisible()
          await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined)

          const metrics = await readLayoutMetrics(page)
          expect(metrics.scrollWidth, `${route.name} horizontal overflow`).toBeLessThanOrEqual(metrics.clientWidth + 1)
          expect(metrics.bodyText, `${route.name} object leak`).not.toContain('[object Object]')
          expect(metrics.topbarSearchOverlap, `${route.name} topbar search overlap`).toBe(false)
          expect(metrics.narrowReadableCells, `${route.name} squeezed readable text cells`).toEqual([])
          if (viewport.name === 'mobile' && route.name === 'outbounds') {
            expect(metrics.hasInternalTableScroll, 'outbounds mobile table scroll affordance').toBe(true)
          }
          expect(
            metrics.visibleAlerts.some(text => /Sing-Box Error/i.test(text) && /\bINFO\b/i.test(text)),
            `${route.name} false INFO error toast`,
          ).toBe(false)
        })
      }
    })
  }
})
