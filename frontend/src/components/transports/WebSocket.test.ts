import { describe, expect, it } from 'vitest'
import WebSocket from './WebSocket.vue'

describe('WebSocket Host header editing', () => {
  it.each(['new.example', ''])('preserves custom headers when Host becomes %j', value => {
    const transport = { headers: { Host: 'old.example', Authorization: 'Bearer fixture', 'X-Custom': 'keep' } }
    WebSocket.computed!.host.set.call({ $props: { transport } } as any, value)
    expect(transport.headers.Authorization).toBe('Bearer fixture')
    expect(transport.headers['X-Custom']).toBe('keep')
    if (value) expect(transport.headers.Host).toBe(value)
    else expect(transport.headers).not.toHaveProperty('Host')
  })
})
