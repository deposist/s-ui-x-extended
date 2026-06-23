import { expect, test, type Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

import { phase6Dir, readServerState } from './helpers'

const screenshotDir = path.join(phase6Dir, 'admin-smoke')

const unique = () => `${Date.now()}-${Math.floor(Math.random() * 10000)}`

// Classic renders each admin as a rounded card; Nexus renders them as dense
// table rows. Match both so the same assertions cover either mode.
const userCards = (page: Page) => page.locator('.v-card.rounded-xl, tr.nexus-data-table__row')
const userCard = (page: Page, username: string) => userCards(page).filter({ hasText: username })

const disableNotificationPointerEvents = async (page: Page) => {
  await page.addStyleTag({
    content: 'ol[aria-label="Notifications"], ol[aria-label="Notifications"] * { pointer-events: none !important; }',
  })
}

const login = async (
  page: Page,
  username = readServerState().username,
  password = readServerState().password,
  initialMode: 'classic' | 'nexus' = 'classic',
) => {
  await page.addInitScript(({ mode }) => {
    window.localStorage.setItem('locale', 'en')
    window.localStorage.setItem('sui:ui:mode', mode)
  }, { mode: initialMode })

  await page.goto('login', { waitUntil: 'domcontentloaded' })
  await disableNotificationPointerEvents(page)
  const inputs = page.locator('input')
  await inputs.nth(0).fill(username)
  await inputs.nth(1).fill(password)
  await page.locator('button[type="submit"]').click()

  await expect.poll(async () => {
    const response = await page.request.get('api/settings')
    const body = await response.json().catch(() => ({ success: false }))

    return body.success === true
  }).toBe(true)
}

const openAdmins = async (page: Page, mode: 'classic' | 'nexus') => {
  await page.goto('admins', { waitUntil: 'domcontentloaded' })
  await disableNotificationPointerEvents(page)
  await expect(page.getByRole('button', { name: 'Add admin' })).toBeVisible()

  const desiredLabel = mode === 'classic' ? 'Switch to Nexus mode' : 'Switch to Classic mode'
  const oppositeLabel = mode === 'classic' ? 'Switch to Classic mode' : 'Switch to Nexus mode'
  const desired = page.getByRole('button', { name: desiredLabel })
  if (await desired.count() === 0) {
    await page.getByRole('button', { name: oppositeLabel }).click()
  }
  await expect(desired).toBeVisible()
}

const assertSelfDeleteHidden = async (page: Page) => {
  const selfCard = userCard(page, readServerState().username)

  await expect(selfCard).toHaveCount(1)
  await expect(selfCard.getByRole('button', { name: 'Delete admin' })).toHaveCount(0)
}

const submitAdd = async (page: Page, username: string, password: string, currentPass: string) => {
  await page.getByRole('button', { name: 'Add admin' }).click()
  await expect(page.getByRole('dialog')).toContainText('Add admin')
  await page.getByLabel('Current Password').fill(currentPass)
  await page.getByLabel('New Username').fill(username)
  await page.getByLabel('New Password').fill(password)
  await page.getByLabel('Confirm Password').fill(password)
  await page.getByRole('button', { name: 'Add', exact: true }).click()
}

const addAdmin = async (page: Page, username: string, password: string, screenshotName: string) => {
  await submitAdd(page, username, password, readServerState().password)
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(userCard(page, username)).toBeVisible()
  await expect(userCard(page, username).getByRole('button', { name: 'Delete admin' })).toHaveCount(1)
  await page.screenshot({ path: path.join(screenshotDir, screenshotName), fullPage: true })
}

const assertWrongAddRejected = async (page: Page, username: string, password: string) => {
  await submitAdd(page, username, password, 'wrong-current-password')
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('button', { name: 'Close' }).click()
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(userCard(page, username)).toHaveCount(0)
}

const deleteAdmin = async (page: Page, username: string, currentPass: string) => {
  const card = userCard(page, username)

  await expect(card).toBeVisible()
  await expect(card.getByRole('button', { name: 'Delete admin' })).toHaveCount(1)
  await card.getByRole('button', { name: 'Delete admin' }).click()
  await expect(page.getByRole('dialog')).toContainText(`Delete admin ${username}`)
  await page.getByLabel('Current Password').fill(currentPass)
  await page.getByRole('button', { name: 'Delete', exact: true }).click()
}

const deleteAdminSuccessfully = async (page: Page, username: string) => {
  await deleteAdmin(page, username, readServerState().password)
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(userCard(page, username)).toHaveCount(0)
}

const assertWrongDeleteRejected = async (page: Page, username: string) => {
  await deleteAdmin(page, username, 'wrong-current-password')
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(userCard(page, username)).toBeVisible()
  await page.getByLabel('Current Password').fill(readServerState().password)
  await page.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(userCard(page, username)).toHaveCount(0)
}

const cleanupSmokeUsers = async (page: Page) => {
  await openAdmins(page, 'classic')
  const texts = await userCards(page).allTextContents()
  const usernames = texts
    .map(text => text.split('Last login')[0].trim())
    .filter(username => username && username !== readServerState().username)
    .filter(username => username.includes('-smoke-') || username.includes('-wrong-'))

  for (const username of usernames) {
    if (await userCard(page, username).count() > 0) {
      await deleteAdminSuccessfully(page, username)
    }
  }
}

test('creates and deletes admins in classic and nexus', async ({ browser, page }) => {
  test.setTimeout(90_000)

  fs.rmSync(screenshotDir, { recursive: true, force: true })
  fs.mkdirSync(screenshotDir, { recursive: true })

  await login(page)
  await cleanupSmokeUsers(page)

  for (const mode of ['classic', 'nexus'] as const) {
    await openAdmins(page, mode)
    await assertSelfDeleteHidden(page)

    await assertWrongAddRejected(page, `${mode}-wrong-${unique()}`, 'smoke-admin-pass-123')

    const username = `${mode}-smoke-${unique()}`
    await addAdmin(page, username, 'smoke-admin-pass-123', `${mode}-created.png`)
    await assertWrongDeleteRejected(page, username)
  }

  const cookieUsername = `cookie-smoke-${unique()}`
  const cookiePassword = 'cookie-smoke-pass-123'
  await openAdmins(page, 'classic')
  await addAdmin(page, cookieUsername, cookiePassword, 'cookie-created.png')

  const targetContext = await browser.newContext({ baseURL: readServerState().baseURL })
  const targetPage = await targetContext.newPage()
  await login(targetPage, cookieUsername, cookiePassword)
  const beforeDelete = await targetPage.request.get('api/settings')
  expect((await beforeDelete.json()).success).toBe(true)

  await openAdmins(page, 'classic')
  await deleteAdminSuccessfully(page, cookieUsername)

  const afterDelete = await targetPage.request.get('api/settings')
  const afterDeleteBody = await afterDelete.json().catch(() => ({ success: false }))
  expect(afterDeleteBody.success).toBe(false)

  await targetContext.close()
  await cleanupSmokeUsers(page)
})
