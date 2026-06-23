import { expect, test } from '@playwright/test'

import { login } from './helpers'

// Verifies the Failover outbound editor renders in the NEXUS drawer (the default
// UI). Before the fix it rendered nothing here and left server/port visible.
// The failover engine itself is covered by Go unit/integration tests.
test('nexus outbounds: failover editor renders in the nexus drawer', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('outbounds')

  await page.getByRole('button', { name: 'Add', exact: true }).first().click()
  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Outbound')

  // Open the Type select and pick Failover.
  await drawer.locator('.v-select').filter({ hasText: 'Type' }).first().click()
  await page.getByRole('option', { name: 'Failover', exact: true }).click()

  // The dedicated Failover editor must now render (this is what was missing in
  // the nexus drawer before the fix).
  await expect(drawer).toContainText('Failover Group')
  await expect(drawer).toContainText('Member outbounds (priority order)')
  await expect(drawer).toContainText('Probe target (domain or IP)')
  await expect(drawer.getByRole('button', { name: 'Add member' })).toBeVisible()

  // Server / port fields must be hidden for a group (NoServer includes Failover).
  await expect(drawer).not.toContainText('Server Address')
})
