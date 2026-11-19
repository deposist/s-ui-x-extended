import { devices, expect, test, type Browser, type Page } from '@playwright/test'
import { spawnSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { csrfToken, setEnglishLocale } from './helpers'

type Panel = { url: string; passwordFile: string }
type Stand = { master: Panel; slave: Panel; inboundPort: number; executable: string }

const fixture = process.env.SUI_NODES_E2E_STATE

async function openPanel(browser: Browser, panel: Panel, mobile = false) {
  const context = await browser.newContext({
    ...(mobile ? devices['iPhone 13'] : {}),
    baseURL: panel.url,
    ignoreHTTPSErrors: true,
  })
  const page = await context.newPage()
  await setEnglishLocale(page)
  await page.goto('login')
  await page.locator('input').nth(0).fill('admin')
  await page.locator('input').nth(1).fill(readFileSync(panel.passwordFile, 'utf8').trim())
  await page.locator('button[type=submit]').click()
  await page.waitForURL(panel.url)
  await page.goto('nodes')
  await openPanelRole(page)
  return { context, page }
}

async function openPanelRole(page: Page) {
  const role = page.getByRole('button', { name: 'Panel role', exact: true })
  await role.focus()
  await role.press('Enter')
  await expect(page.getByText(/^Current role:/)).toBeVisible()
}

test('HTTPS Master and mobile Slave pair, probe traffic and preview an assignment', async ({ browser }) => {
  test.skip(!fixture, 'Set SUI_NODES_E2E_STATE to a fresh two-panel HTTPS fixture.')
  test.setTimeout(240_000)
  const stand = JSON.parse(readFileSync(fixture!, 'utf8')) as Stand
  const master = await openPanel(browser, stand.master)
  const slave = await openPanel(browser, stand.slave, true)
  try {
    await master.page.getByRole('button', { name: 'Use this panel as Master' }).click()
    await master.page.getByRole('button', { name: 'Create pairing code' }).click()
    const invitation = await master.page.getByLabel('Copy invitation', { exact: true }).inputValue()
    expect(invitation.startsWith(`s-ui-node v1 ${stand.master.url} `)).toBe(true)
    const pasted = slave.page.getByLabel('Paste invitation')
    await pasted.fill(invitation.replace('v1 ', 'v2 '))
    await slave.page.getByRole('button', { name: 'Read invitation' }).click()
    await expect(slave.page.getByText('Paste a version 1 invitation with a valid HTTPS address.')).toBeVisible()
    await pasted.fill(invitation)
    await slave.page.getByRole('button', { name: 'Read invitation' }).click()
    await expect(slave.page.getByLabel('Master panel HTTPS address')).toHaveValue(stand.master.url)
    await slave.page.getByRole('button', { name: 'Pair as Slave' }).click()
    await expect(slave.page.getByText('Current role: Slave', { exact: true })).toBeVisible()
    expect(await slave.page.evaluate(secret => {
      const fields = [...Object.values(localStorage), ...Object.values(sessionStorage), location.href]
      return fields.some(field => field.includes(secret))
    }, invitation.split(' ').at(-1)!)).toBe(false)
    await setPublicHost(slave.page, '127.0.0.1')
    await slave.page.getByRole('link', { name: 'Create VLESS TCP inbound' }).click()
    const inbound = slave.page.getByRole('dialog').filter({ hasText: 'Add Inbound' })
    await expect(inbound.getByRole('combobox', { name: 'Type', exact: true })).toHaveValue(/^vless$/i)
    await inbound.getByLabel('Tag', { exact: true }).fill('nodes-e2e')
    await inbound.getByLabel('Address', { exact: true }).fill('127.0.0.1')
    await inbound.getByLabel('Port', { exact: true }).fill(String(stand.inboundPort))
    await inbound.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(slave.page.getByText('nodes-e2e', { exact: true })).toBeVisible()

    const row = master.page.getByRole('row').filter({ hasText: 'nodes-e2e' })
    await expect(row).toBeVisible({ timeout: 45_000 })
    await row.getByRole('button', { name: 'Clients', exact: true }).click()
    const dialog = master.page.getByRole('dialog')
    await dialog.getByRole('button', { name: 'Check connection' }).click()
    await expect(dialog.getByText(/HTTPS response · Successful/)).toBeVisible({ timeout: 70_000 })

    await master.page.goto('clients')
    await master.page.getByRole('button', { name: 'Add', exact: true }).first().click()
    const client = master.page.getByRole('dialog').filter({ hasText: 'Add Client' })
    await client.getByLabel('Name', { exact: true }).fill('nodes-e2e-client')
    await client.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(master.page.getByText('nodes-e2e-client', { exact: true })).toBeVisible()
    await master.page.goto('nodes')
    await row.getByRole('button', { name: 'Clients', exact: true }).click()
    const assignment = dialog.getByRole('row').filter({ hasText: 'nodes-e2e-client' })
    await assignment.getByRole('checkbox').check()
    await dialog.getByRole('button', { name: 'Add (1)', exact: true }).click()
    await expect(dialog.getByText('1 changes, 0 skipped')).toBeVisible()
    await expect(assignment.getByText('no', { exact: true })).toBeVisible()
    await dialog.getByRole('button', { name: 'Apply assignments' }).click()
    await expect(assignment.getByText('yes', { exact: true })).toBeVisible()

    await master.page.setViewportSize({ width: 390, height: 844 })
    const table = dialog.locator('.nodes-page__client-table .v-table__wrapper')
    await expect.poll(() => table.evaluate(el => el.scrollWidth - el.clientWidth)).toBeLessThanOrEqual(1)

    await master.page.setViewportSize({ width: 1280, height: 900 })
    await master.page.goto('clients')
    const clientRow = master.page.getByRole('row').filter({ hasText: 'nodes-e2e-client' })
    await clientRow.getByRole('button', { name: 'Delivery' }).click()
    await master.page.getByRole('tab', { name: 'Hiddify' }).click()
    const subscription = await master.page.getByLabel('Subscription URL', { exact: true }).and(master.page.locator('input[value]:not([value*=format])')).inputValue()
    await expect.poll(async () => {
      const response = await master.page.request.get(subscription, { timeout: 3000 })
      if (!response.ok()) return false
      const body = await response.text()
      const decoded = body.includes('://') ? body : Buffer.from(body, 'base64').toString()
      const link = decoded.split(/\s+/).find(item => item.startsWith('vless://'))
      if (!link || new URL(link).port !== String(stand.inboundPort)) return false
      return spawnSync(stand.executable, ['check-vless'], { input: link, timeout: 6000, windowsHide: true }).status === 0
    }, { timeout: 55_000 }).toBe(true)
    await master.page.goto('clients')
    await clientRow.getByRole('button', { name: 'Diagnose' }).click()
    await expect(master.page.getByRole('heading', { name: 'Node assignments' })).toBeVisible()
    await expect(master.page.getByRole('dialog').getByText('Link published', { exact: true })).toBeVisible()

    await master.page.goto('nodes')
    await row.getByRole('button', { name: 'Clients', exact: true }).click()
    await slave.page.goto('nodes')
    await openPanelRole(slave.page)
    await setPublicHost(slave.page, '127.0.0.2')
    await expect(row.getByText('127.0.0.2', { exact: true })).toBeVisible({ timeout: 45_000 })
    await dialog.getByRole('button', { name: 'Check connection' }).click()
    await expect(dialog.getByText(/Proxy connection · Proxy connection failed/)).toBeVisible({ timeout: 70_000 })
    await expect(row.getByText('Online', { exact: true })).toBeVisible()
    const list = await master.page.request.get('api/nodes')
    const nodeUUID = ((await list.json()).obj as { uuid: string; paired: boolean }[]).find(node => node.paired)!.uuid
    const blocked = await slave.page.request.post(`api/nodes/${nodeUUID}/probe`, { data: { inboundId: 1 }, headers: { 'X-CSRF-Token': await csrfToken(slave.page) } })
    expect(blocked.status()).toBe(403)
    const disconnected = await master.page.request.delete(`api/nodes/${nodeUUID}`, { headers: { 'X-CSRF-Token': await csrfToken(master.page) } })
    expect((await disconnected.json()).success).toBe(true)
    await expect.poll(async () => {
      const list = await master.page.request.get('api/nodes')
      return ((await list.json()).obj as { uuid: string }[]).some(node => node.uuid === nodeUUID)
    }, { timeout: 50_000 }).toBe(false)
  } finally {
    await Promise.all([master.context.close(), slave.context.close()])
  }
})

async function setPublicHost(page: Page, value: string) {
  await page.getByLabel('Address for clients').fill(value)
  const response = page.waitForResponse(reply => reply.url().endsWith('/api/nodes/panel/hostname'))
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  expect((await (await response).json()).success).toBe(true)
}
