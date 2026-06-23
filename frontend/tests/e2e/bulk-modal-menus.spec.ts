import { expect, test, type Page } from '@playwright/test'

import { login } from './helpers'

// Regression: the client bulk modals (Add/Edit Bulk) are FormShell drawers, not
// raw v-dialogs. A v-select menu inside a v-dialog here did NOT close on
// click-away (two menus could stack open); inside the drawer it closes. Guard
// the drawer behaviour so a future revert to v-dialog is caught.
const openMenus = (page: Page) =>
  page.locator('.v-overlay--active .v-list[role="listbox"]').count()

test('Edit Bulk drawer select menu closes on click-away', async ({ page }) => {
  test.setTimeout(90_000)
  await login(page)
  await page.goto('clients')

  await page.locator('.nexus-toolbar').getByRole('button', { name: 'Action' }).click()
  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: 'Edit Bulk' }).click()

  const drawer = page.getByRole('dialog').filter({ hasText: 'Edit Bulk' })
  await expect(drawer).toBeVisible()

  await drawer.locator('.v-select').nth(0).click()
  await page.waitForTimeout(400)
  const afterOpen = await openMenus(page)

  await drawer.locator('.nexus-drawer__title').click({ force: true })
  await page.waitForTimeout(500)
  const afterClickAway = await openMenus(page)

  expect(afterClickAway, `menus still open after click-away (afterOpen=${afterOpen})`).toBe(0)
})

test('Add Client drawer select menu closes on click-away', async ({ page }) => {
  test.setTimeout(90_000)
  await login(page)
  await page.goto('clients')

  await page.locator('.nexus-toolbar').getByRole('button', { name: 'Add', exact: true }).click()
  const drawer = page.getByRole('dialog').filter({ hasText: 'Add Client' })
  await expect(drawer).toBeVisible()

  await drawer.locator('.v-select').first().click()
  await page.waitForTimeout(400)
  const afterOpen = await openMenus(page)

  await drawer.getByText('Add Client', { exact: true }).click({ force: true })
  await page.waitForTimeout(500)
  const afterClickAway = await openMenus(page)

  expect(afterClickAway, `drawer menus still open after click-away (afterOpen=${afterOpen})`).toBe(0)
})
