import { expect, test } from '@playwright/test'

import { login } from './helpers'

// The Clients page is the most feature-rich list. Verify it renders (not blank)
// in Nexus mode and that its Client form opens as a FormShell drawer.
test('nexus clients list renders and opens the client drawer', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('clients')

  const add = page.getByRole('button', { name: 'Add', exact: true }).first()
  await expect(add).toBeVisible()
  await add.click()

  // Scope to the opened Client drawer: other EntityDrawer-based modals on this
  // page (bulk add/edit) also expose role="dialog" while closed.
  const drawer = page.getByRole('dialog').filter({ hasText: 'Add Client' })
  await expect(drawer).toContainText('Add Client')
  await expect(drawer.getByRole('button', { name: 'Save', exact: true })).toBeVisible()
})
