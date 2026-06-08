import { expect, test } from '@playwright/test'
import { readServerState, writeJSONArtifact } from './helpers'

test('backend admin responses include security headers', async ({ request }) => {
  const state = readServerState()
  const response = await request.get(state.backendURL)
  const headers = response.headers()

  writeJSONArtifact('security-headers/headers.json', headers)

  expect(headers['content-security-policy']).toContain("frame-ancestors 'none'")
  expect(headers['x-frame-options']).toBe('DENY')
  expect(headers['x-content-type-options']).toBe('nosniff')
  expect(headers['referrer-policy']).toBe('strict-origin-when-cross-origin')
  if (state.backendURL.startsWith('https://')) {
    expect(headers['strict-transport-security']).toContain('max-age=')
  }
})
