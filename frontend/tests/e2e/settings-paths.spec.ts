import { expect, test } from '@playwright/test'
import { csrfToken, login } from './helpers'

test('settings path conflict is rejected by the local panel', async ({ page }) => {
  await login(page)
  const token = await csrfToken(page)
  const settingsResponse = await page.request.get('api/settings')
  const settingsBody = await settingsResponse.json()
  expect(settingsBody.success).toBe(true)

  const payload = {
    ...settingsBody.obj,
    subPath: '/phase6-conflict/',
    subJsonPath: '/phase6-conflict/',
    subClashPath: '/phase6-clash/',
  }

  const response = await page.request.post('api/save', {
    headers: { 'X-CSRF-Token': token },
    form: {
      object: 'settings',
      action: 'set',
      data: JSON.stringify(payload),
    },
  })
  const body = await response.json()

  expect(body.success).toBe(false)
  expect(String(body.msg).toLowerCase()).toMatch(/path|conflict|subscription|duplicate/)
})
