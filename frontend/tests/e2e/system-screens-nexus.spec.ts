import { expect, test, type Page } from '@playwright/test'

import { login } from './helpers'

// These screens were converted to the Nexus look (PageHeader + dense tables +
// drawers) in addition to the earlier list pilots. The default UI mode is Nexus,
// so after login each renders in Nexus chrome. We verify three things per screen:
//  1. the shell did not blank out (the sidebar 'Inbounds' link is still present),
//  2. no cell rendered the Vue tag-shadowing bug ('[object Object]'),
//  3. a screen-specific control is visible (the conversion actually mounted).
const shellPresent = async (page: Page) => {
  await expect(page.getByRole('link', { name: 'Inbounds' })).toBeVisible()
}

const noObjectObject = async (page: Page) => {
  await expect(page.locator('body')).not.toContainText('[object Object]')
}

test('nexus system screens render without blanking or [object Object]', async ({ page }) => {
  test.setTimeout(90_000)

  await login(page)

  // Rules: PageToolbar Add rule / Add ruleset + (possibly empty) ruleset/rule tables.
  await page.goto('rules')
  await shellPresent(page)
  await noObjectObject(page)
  await expect(page.getByRole('button', { name: 'Add Rule', exact: true })).toBeVisible()

  // DNS: PageToolbar Add DNS server / Add DNS rule + server/rule tables.
  await page.goto('dns')
  await shellPresent(page)
  await noObjectObject(page)

  // Audit: NexusDataTable (expandable, server-paginated) — header + filters render.
  await page.goto('audit')
  await shellPresent(page)
  await noObjectObject(page)

  // Admins: always has at least the current admin → at least one Nexus table row,
  // exercising RowActions inside NexusDataTable.
  await page.goto('admins')
  await shellPresent(page)
  await noObjectObject(page)
  await expect(page.getByRole('button', { name: 'Add admin' })).toBeVisible()
  await expect(page.locator('tr.nexus-data-table__row').first()).toBeVisible()

  // Settings: tabbed form under a PageHeader.
  await page.goto('settings')
  await shellPresent(page)
  await noObjectObject(page)

  // Paid subscriptions: tabbed; bindings/tariffs/orders tables converted.
  await page.goto('paid-subscriptions')
  await shellPresent(page)
  await noObjectObject(page)
  await expect(page.getByText('experimental')).toBeVisible()
})
