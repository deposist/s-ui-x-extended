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
  await expect(drawer.locator('.v-card-subtitle').filter({ hasText: /^Cookies$/ })).toBeVisible()
  await expect(drawer.locator('.v-card-subtitle').filter({ hasText: /^Listen$/ })).toHaveCount(0)
  await expect(fieldWithText(drawer, 'Address')).toHaveCount(0)
  await expect(fieldWithText(drawer, 'Port')).toHaveCount(0)
})

test('trusttunnel inbound shows inbound controls only', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('inbounds')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Inbound')

  await pickType(page, drawer, 'TrustTunnel')
  await expect(fieldWithText(drawer, 'Network')).toBeVisible()
  await expect(fieldWithText(drawer, 'Congestion controller')).toBeVisible()
  await expect(fieldWithText(drawer, 'CWND')).toBeVisible()
  await expect(drawer.getByRole('checkbox', { name: 'QUIC', exact: true })).toHaveCount(0)
  await expect(drawer.getByRole('checkbox', { name: 'Health check', exact: true })).toHaveCount(0)
  await expect(drawer.locator('.v-card-subtitle').filter({ hasText: /^Multiplex$/ })).toHaveCount(0)
})

test('trojan inbound keeps fallback controls and hides outbound controls', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('inbounds')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Inbound')

  await pickType(page, drawer, 'Trojan')
  await expect(fieldWithText(drawer, 'Password')).toHaveCount(0)
  await expect(fieldWithText(drawer, 'Network')).toHaveCount(0)
  await expect(fieldWithText(drawer, 'Fallback server')).toBeVisible()
  await expect(fieldWithText(drawer, 'Fallback port')).toBeVisible()
  await expect(drawer.locator('.v-alert').filter({ hasText: 'Fallback redirects' })).toBeVisible()
})

test('sudoku inbound save sends flat HTTP mask fields', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('inbounds')
  await page.getByRole('button', { name: 'Add', exact: true }).first().click()

  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Inbound')

  await pickType(page, drawer, 'Sudoku')
  await expect(drawer.getByRole('button', { name: 'Apply inbound recommendations', exact: true })).toBeVisible()
  await drawer.getByRole('button', { name: 'Apply inbound recommendations', exact: true }).click()
  const tag = `sudoku-e2e-${Date.now()}`
  await fieldWithText(drawer, 'Tag').locator('input').fill(tag)
  await fieldWithText(drawer, 'Path root').locator('input').fill('/sudoku-e2e')
  await fieldWithText(drawer, 'Fallback').locator('input').fill('127.0.0.1:8080')
  await drawer.getByRole('checkbox', { name: 'Disable HTTP mask', exact: true }).check()

  const saveButton = drawer.getByRole('button', { name: 'Save', exact: true })
  await expect(saveButton).toBeEnabled()
  const saveRequestPromise = page.waitForRequest((request) => (
    request.method() === 'POST'
    && request.url().endsWith('/api/save')
    && new URLSearchParams(request.postData() ?? '').get('object') === 'inbounds'
  ))
  await saveButton.click()
  const saveRequest = await saveRequestPromise
  const form = new URLSearchParams(saveRequest.postData() ?? '')
  const data = JSON.parse(form.get('data') ?? '{}') as Record<string, unknown>

  expect(data).toMatchObject({
    tag,
    type: 'sudoku',
    aead_method: 'chacha20-poly1305',
    padding_min: 10,
    padding_max: 30,
    handshake_timeout: 5,
    enable_pure_downlink: true,
    http_mask_mode: 'legacy',
    fallback: '127.0.0.1:8080',
    path_root: '/sudoku-e2e',
    disable_http_mask: true,
  })
  expect(Object.hasOwn(data, 'http_mask')).toBe(false)
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
