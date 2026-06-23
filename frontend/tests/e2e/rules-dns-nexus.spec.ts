import { expect, test } from '@playwright/test'

import { login } from './helpers'

// The Rules/DNS entity forms now render via FormShell (drawer in Nexus). Their
// views still mount those modals closed, so verify the pages render (the shell
// sidebar stays present => no blank-page crash from a closed FormShell drawer).
test('nexus rules and dns pages render without blanking', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)

  await page.goto('rules')
  await expect(page.getByRole('link', { name: 'Inbounds' })).toBeVisible()

  await page.goto('dns')
  await expect(page.getByRole('link', { name: 'Inbounds' })).toBeVisible()
})
