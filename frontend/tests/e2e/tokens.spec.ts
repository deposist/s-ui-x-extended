import { expect, test } from '@playwright/test'
import { csrfToken, login } from './helpers'

test('API token can be created, disabled, enabled, listed masked, and deleted', async ({ page }) => {
  await login(page)
  const token = await csrfToken(page)

  const addResponse = await page.request.post('api/addToken', {
    headers: { 'X-CSRF-Token': token },
    form: {
      desc: 'phase6-e2e-token',
      expiry: '1',
      scope: 'read',
    },
  })
  const addBody = await addResponse.json()
  expect(addBody.success).toBe(true)
  const plainToken = addBody.obj as string
  expect(plainToken).toMatch(/^[A-Za-z0-9]{32}$/)

  const listBody = await (await page.request.get('api/tokens')).json()
  expect(listBody.success).toBe(true)
  const created = listBody.obj.find((item: any) => item.desc === 'phase6-e2e-token')
  expect(created).toBeTruthy()
  expect(created.token).toBeUndefined()
  expect(created.tokenPrefix).toBe(plainToken.slice(0, 8))

  for (const enabled of [false, true]) {
    const response = await page.request.post('api/setTokenEnabled', {
      headers: { 'X-CSRF-Token': token },
      form: { id: String(created.id), enabled: String(enabled) },
    })
    const body = await response.json()
    expect(body.success).toBe(true)
  }

  const deleteResponse = await page.request.post('api/deleteToken', {
    headers: { 'X-CSRF-Token': token },
    form: { id: String(created.id) },
  })
  const deleteBody = await deleteResponse.json()
  expect(deleteBody.success).toBe(true)
})
