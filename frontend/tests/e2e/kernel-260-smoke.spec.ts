import { expect, test } from '@playwright/test'

import { login } from './helpers'

// Smoke coverage for the 2.6.x UI surface:
//  - the Call protocol appears in the inbound/outbound type pickers and its
//    form renders the platform select + join link;
//  - the AmneziaWG 3.0 endpoint form renders the timing fields and no longer
//    exposes the removed 2.0 fields (J1/J2/J3/Itime).

const pickType = async (page: any, drawer: any, typeName: string) => {
  // Vuetify v3 renders v-select as a visually-hidden input inside a .v-field;
  // clicking the field control opens the menu reliably.
  await drawer.locator('.v-select .v-field').first().click()
  await page.getByRole('option', { name: typeName, exact: true }).click()
}

// Vuetify labels render inside the visible .v-field control; text nodes can
// also appear hidden inside select overlays, so assert on the field wrapper.
const fieldWithText = (drawer: any, label: string) =>
  drawer.locator('.v-field').filter({ hasText: label }).first()

test('call inbound is selectable and renders its form', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('inbounds')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Inbound')

  await pickType(page, drawer, 'Call')
  await expect(fieldWithText(drawer, 'Platform')).toBeVisible()
  await expect(fieldWithText(drawer, 'Join link')).toBeVisible()
  await expect(drawer.getByText('Cookies')).toBeVisible()
})

test('call outbound is selectable and renders its form', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('outbounds')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Outbound')

  await pickType(page, drawer, 'Call')
  await expect(fieldWithText(drawer, 'Platform')).toBeVisible()
  await expect(fieldWithText(drawer, 'Join link')).toBeVisible()
})

test('amnezia wireguard form shows AWG 3.0 fields and hides 2.0 fields', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('endpoints')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Endpoint')

  await pickType(page, drawer, 'Wireguard')

  // The Amnezia card's Enable switch is a Vuetify v-switch (checkbox input).
  const amneziaCard = drawer.locator('.v-card').filter({ hasText: 'AmneziaWG 3.0 options' })
  await amneziaCard.locator('input[type="checkbox"]').first().check()

  // AWG 3.0 timing fields are present.
  await expect(fieldWithText(drawer, 'Rekey after time (s)')).toBeVisible()
  await expect(fieldWithText(drawer, 'Max handshake attempts')).toBeVisible()
  await expect(fieldWithText(drawer, 'Content padding addition')).toBeVisible()

  // AWG 2.0-only fields are gone.
  await expect(drawer.getByText('Junk packet J1')).toHaveCount(0)
  await expect(drawer.getByText('Init interval')).toHaveCount(0)
})
