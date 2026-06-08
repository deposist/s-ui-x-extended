import { expect, test } from '@playwright/test'
import { csrfToken, login, readServerState, setEnglishLocale } from './helpers'

test.describe('login + CSRF smoke', () => {
  test('rejects an invalid password without leaving the login page', async ({ page }) => {
    await setEnglishLocale(page)
    await page.goto('login')
    const inputs = page.locator('input')
    await inputs.nth(0).fill(readServerState().username)
    await inputs.nth(1).fill('not-the-password')
    await page.locator('button[type="submit"]').click()

    await expect(page).toHaveURL(/\/app\/login$/)
  })

  test('logs in with the generated test admin password', async ({ page }) => {
    await login(page)
    await expect(page.locator('body')).toBeVisible()
  })

  test('rejects protected POST requests without a CSRF token', async ({ page }) => {
    await login(page)

    const response = await page.request.post('api/save', {
      form: {
        object: 'settings',
        action: 'set',
        data: '{}',
      },
    })
    const body = await response.json()

    expect(response.status()).toBe(403)
    expect(body.success).toBe(false)
    expect(body.msg).toContain('CSRF')

    await expect(csrfToken(page)).resolves.toEqual(expect.any(String))
  })
})
