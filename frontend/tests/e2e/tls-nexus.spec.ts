import { expect, test } from '@playwright/test'

import { login } from './helpers'

test('nexus tls list: drawer add (save-emit) + action buttons', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('tls')

  await page.getByRole('button', { name: 'Add', exact: true }).first().click()
  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add TLS')

  const name = `tls-${Date.now()}`
  await page.getByLabel('Name').fill(name)
  await drawer.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText(name)).toBeVisible()

  const actionsCell = page.locator('.nexus-data-table__actions').first()
  await expect(actionsCell).not.toContainText('[object Object]')
  await expect(actionsCell.getByRole('button', { name: 'Edit' })).toBeVisible()
})
