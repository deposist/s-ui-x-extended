import { expect, test } from '@playwright/test'

import { login } from './helpers'

test('nexus outbounds list: drawer add, action buttons, test-all', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('outbounds')

  // Toolbar carries Add / Add Bulk / Test all.
  await expect(page.getByRole('button', { name: 'Test all' })).toBeVisible()
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Outbound')

  const tag = `obnd-${Date.now()}`
  await page.getByLabel('Tag').fill(tag)
  await drawer.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText(tag)).toBeVisible()
  await expect(page.locator('.nexus-drawer.v-navigation-drawer--active')).toHaveCount(0)

  // ACTION cell renders real icon buttons, not stringified action objects.
  const actionsCell = page.locator('.nexus-data-table__actions').first()
  await expect(actionsCell).not.toContainText('[object Object]')
  await expect(actionsCell.getByRole('button', { name: 'Edit' })).toBeVisible()
})
