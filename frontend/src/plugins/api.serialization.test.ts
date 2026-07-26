import { describe, expect, it, vi } from 'vitest'

vi.mock('@/store/csrf', () => ({
  clearCSRFToken: vi.fn(),
  getCSRFToken: vi.fn(async () => 'test-csrf-token'),
}))

vi.mock('@/plugins/base-url', () => ({
  getBaseUrl: () => '/',
}))

import api from '@/plugins/api'

// Capture what the instance would actually put on the wire, without sending it.
const serializeBody = async (payload: object): Promise<string> => {
  let body = ''
  await api.post('api/probe', payload, {
    adapter: async (config: any) => {
      body = String(config.data ?? '')
      return { data: { success: true, msg: '', obj: null }, status: 200, statusText: 'OK', headers: {}, config }
    },
  } as any)
  return body
}

// This instance posts as application/x-www-form-urlencoded, which has no way to
// express a nested array. Anything nested is flattened into indexed keys that the
// Go handlers do not read, so the request arrives looking empty. Callers with
// structured payloads must JSON-encode them into a single "data" field.
describe('api instance form encoding', () => {
  it('flattens a nested payload instead of sending JSON', async () => {
    const body = await serializeBody({ sources: [{ tag: 'geosite-ru', url: 'https://example.test/a.srs' }] })

    expect(body).not.toContain('{')
    // The shape that silently loses the payload for a Go handler reading
    // c.PostForm("data").
    expect(body).toContain('sources%5B0%5D%5Btag%5D')
    expect(new URLSearchParams(body).get('data')).toBeNull()
  })

  it('preserves a structured payload when it is JSON-encoded into data', async () => {
    const sources = [{ tag: 'geosite-ru', url: 'https://example.test/a.srs' }]
    const body = await serializeBody({ data: JSON.stringify({ sources }) })

    const decoded = new URLSearchParams(body).get('data')
    expect(decoded).not.toBeNull()
    expect(JSON.parse(decoded as string)).toEqual({ sources })
  })
})
