import { expect, test } from '@playwright/test'

import { login } from './helpers'

// Pilot proof: the Nexus Inbounds list + right-side EntityDrawer round-trip.
// login() leaves sui:ui:mode unset, so the default (nexus) shell is exercised.
test('nexus inbounds list opens the drawer, tracks dirty, and saves', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('inbounds')

  // Toolbar Add (the empty-state CTA may also read "Add"; take the first).
  const add = page.getByRole('button', { name: 'Add', exact: true }).first()
  await expect(add).toBeVisible()
  await add.click()

  // EntityDrawer exposes an explicit role="dialog" + the add title.
  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Inbound')

  // Editing the tag flips the unsaved-changes indicator.
  const tag = `nexus-pilot-${Date.now()}`
  await page.getByLabel('Tag').fill(tag)
  await expect(drawer.getByText('Unsaved changes')).toBeVisible()

  // Save closes the drawer and the new inbound appears in the table. A Vuetify
  // temporary drawer stays in the DOM off-screen when closed, so assert on its
  // open/closed state class rather than toBeHidden.
  await drawer.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText(tag)).toBeVisible()
  await expect(page.locator('.nexus-drawer.v-navigation-drawer--active')).toHaveCount(0)

  // Regression guard: the ACTION cell renders real icon buttons, not the
  // stringified action objects ([object Object]) that a camelCase helper
  // shadowing the RowActions component would produce.
  const actionsCell = page.locator('.nexus-data-table__actions').first()
  await expect(actionsCell).not.toContainText('[object Object]')
  await expect(actionsCell.getByRole('button', { name: 'Edit' })).toBeVisible()
})
