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

  // The rule-conditions endpoint reads only c.PostForm("data") and deliberately
  // has no fallback to flattened keys, so a deeply nested rule has to survive
  // this encoding intact or the check would validate the wrong thing.
  it('sends a nested rule to rule-conditions as a single data field', async () => {
    const rule = {
      type: 'logical',
      mode: 'and',
      rules: [
        { domain: ['a.example'] },
        // An empty nested branch must survive as {} rather than being flattened
        // away: it is exactly the shape the endpoint reports on.
        { type: 'logical', mode: 'or', rules: [{}] },
      ],
      outbound: 'direct',
    }
    const body = await serializeBody({ data: JSON.stringify({ kind: 'route', rule }) })

    const params = new URLSearchParams(body)
    expect([...params.keys()]).toEqual(['data'])
    expect(JSON.parse(params.get('data') as string)).toEqual({ kind: 'route', rule })
  })

  // The rmux multiplex protocol value must survive the same JSON-encoded data
  // path the panel uses for outbound configs (multiplex.protocol union).
  it('round-trips the rmux multiplex protocol through the data field', async () => {
    const outbound = {
      type: 'shadowsocks',
      tag: 'ss-rmux',
      multiplex: { enabled: true, protocol: 'rmux', max_connections: 8 },
    }
    const body = await serializeBody({ data: JSON.stringify({ outbound }) })

    const params = new URLSearchParams(body)
    const decoded = JSON.parse(params.get('data') as string)
    expect(decoded.outbound.multiplex).toEqual({ enabled: true, protocol: 'rmux', max_connections: 8 })
  })
})
