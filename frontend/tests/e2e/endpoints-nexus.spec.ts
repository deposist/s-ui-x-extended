import { expect, test } from '@playwright/test'

import { login } from './helpers'

// Verifies the FormShell path: the Endpoint modal renders as a right-side
// drawer (role=dialog) in Nexus mode, opened from the new list's Add button.
// NOTE: actually saving a WireGuard/Warp/Tailscale endpoint needs the sing-box
// core / external connectivity (api/keypairs key-gen, Cloudflare, auth keys),
// which the test stack disables (SUI_DISABLE_CORE/XUI_DISABLE_REMOTE) — the same
// in Classic, so it is an env limitation, not a redesign regression.
test('nexus endpoints list opens the FormShell drawer', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('endpoints')

  await page.getByRole('button', { name: 'Add', exact: true }).first().click()
  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Endpoint')
})
