import { describe, expect, it, vi, afterEach, beforeEach } from 'vitest'

vi.mock('axios', () => ({
  default: { get: vi.fn() },
}))

vi.mock('@/plugins/httputil', () => ({
  default: { get: vi.fn() },
}))

vi.mock('@/store/modules/data', () => ({
  default: () => ({ loadData: vi.fn(), onlines: {} }),
}))

import axios from 'axios'
import { clearCSRFToken, getCSRFToken } from './csrf'
import { reconnectDelayForRetry, WsLike, WsRuntime, wsProtocolsForToken } from './ws'

class FakeSocket implements WsLike {
  onopen: ((event?: any) => void) | null = null
  onmessage: ((event: any) => void) | null = null
  onclose: ((event?: any) => void) | null = null
  onerror: ((event?: any) => void) | null = null
  close = vi.fn(() => {
    this.onclose?.()
  })
}

describe('WsRuntime fallback', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    clearCSRFToken()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('enters degraded polling when websocket does not open within 5s', async () => {
    const loadData = vi.fn()
    const states: string[] = []
    const socket = new FakeSocket()
    const runtime = new WsRuntime({
      getToken: async () => 'token',
      createSocket: () => socket,
      loadData,
      onState: (state) => states.push(state),
      location: { protocol: 'http:', host: 'panel.test' },
      baseUrl: '/',
    })

    await runtime.connect()
    expect(runtime.state).toBe('reconnecting')

    vi.advanceTimersByTime(5000)
    expect(runtime.state).toBe('degraded')
    expect(states).toContain('degraded')

    vi.advanceTimersByTime(10000)
    expect(loadData).toHaveBeenCalledTimes(1)
  })

  it('uses native timers without illegal invocation in fallback mode', async () => {
    vi.useRealTimers()
    const loadData = vi.fn()
    const nativeSetInterval = globalThis.setInterval
    const nativeClearInterval = globalThis.clearInterval
    const timers = {
      setInterval(callback: () => void, delay?: number) {
        if (this !== globalThis) {
          throw new TypeError('Illegal invocation')
        }
        return nativeSetInterval(callback, delay)
      },
      clearInterval(timerID: ReturnType<typeof setInterval>) {
        if (this !== globalThis) {
          throw new TypeError('Illegal invocation')
        }
        nativeClearInterval(timerID)
      },
    }
    vi.stubGlobal('setInterval', timers.setInterval)
    vi.stubGlobal('clearInterval', timers.clearInterval)

    const runtime = new WsRuntime({
      getToken: async () => null,
      createSocket: () => new FakeSocket(),
      loadData,
      location: { protocol: 'http:', host: 'panel.test' },
      baseUrl: '/',
    })

    await expect(runtime.connect()).resolves.toBeUndefined()
    expect(runtime.state).toBe('degraded')
    runtime.disconnect()
    vi.unstubAllGlobals()
  })

  it('uses capped exponential reconnect backoff with jitter', () => {
    vi.spyOn(Math, 'random').mockReturnValue(0)
    expect([0, 1, 2, 3, 4, 5].map(reconnectDelayForRetry)).toEqual([
      250,
      500,
      1000,
      2000,
      4000,
      5000,
    ])

    vi.mocked(Math.random).mockReturnValue(0.5)
    expect(reconnectDelayForRetry(0)).toBe(375)
    expect(reconnectDelayForRetry(1)).toBe(625)
  })

  it('formats websocket protocols with explicit token prefix', () => {
    expect(wsProtocolsForToken('abc123')).toEqual(['sui.realtime', 'sui.token.abc123'])
  })

  it('clears csrf token after session-rotated websocket close', async () => {
    vi.mocked(axios.get)
      .mockResolvedValueOnce({ data: { obj: { token: 'csrf-1' } } })
      .mockResolvedValueOnce({ data: { obj: { token: 'csrf-2' } } })

    await expect(getCSRFToken()).resolves.toBe('csrf-1')

    const socket = new FakeSocket()
    const runtime = new WsRuntime({
      getToken: async () => 'ws-token',
      createSocket: () => socket,
      loadData: vi.fn(),
      location: { protocol: 'http:', host: 'panel.test' },
      baseUrl: '/',
    })

    await runtime.connect()
    socket.onopen?.()
    socket.onclose?.({ code: 4401, reason: 'session_rotated' })

    await expect(getCSRFToken()).resolves.toBe('csrf-2')
    expect(axios.get).toHaveBeenCalledTimes(2)
  })
})
