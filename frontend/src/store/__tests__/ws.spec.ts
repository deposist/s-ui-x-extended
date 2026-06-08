import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/plugins/httputil', () => ({
  default: { get: vi.fn() },
}))

vi.mock('@/store/modules/data', () => ({
  default: () => ({ loadData: vi.fn(), onlines: {} }),
}))

import { WsLike, WsRuntime } from '../ws'

class FakeSocket implements WsLike {
  onopen: ((event?: any) => void) | null = null
  onmessage: ((event: any) => void) | null = null
  onclose: ((event?: any) => void) | null = null
  onerror: ((event?: any) => void) | null = null
  close = vi.fn(() => {
    this.onclose?.()
  })
}

class ManualTimers {
  private nextID = 1
  timeouts: Array<{ id: number; callback: () => void; delay?: number }> = []
  intervals: Array<{ id: number; callback: () => void; delay?: number }> = []

  setTimeout = vi.fn((handler: TimerHandler, delay?: number) => {
    const callback = typeof handler === 'function' ? handler as () => void : () => undefined
    const timer = { id: this.nextID++, callback, delay }
    this.timeouts.push(timer)
    return timer.id
  }) as unknown as typeof setTimeout

  clearTimeout = vi.fn((timerID?: number) => {
    this.timeouts = this.timeouts.filter((entry) => entry.id !== timerID)
  }) as unknown as typeof clearTimeout

  setInterval = vi.fn((handler: TimerHandler, delay?: number) => {
    const callback = typeof handler === 'function' ? handler as () => void : () => undefined
    const timer = { id: this.nextID++, callback, delay }
    this.intervals.push(timer)
    return timer.id
  }) as unknown as typeof setInterval

  clearInterval = vi.fn((timerID?: number) => {
    this.intervals = this.intervals.filter((entry) => entry.id !== timerID)
  }) as unknown as typeof clearInterval

  runNextTimeout() {
    const timer = this.timeouts.shift()
    timer?.callback()
  }

  runInterval(index = 0) {
    this.intervals[index]?.callback()
  }
}

const flushPromises = async () => {
  await Promise.resolve()
  await Promise.resolve()
}

const runtimeDeps = (overrides: Partial<ConstructorParameters<typeof WsRuntime>[0]> = {}) => ({
  getToken: vi.fn(async () => 'ws-token'),
  createSocket: vi.fn(() => new FakeSocket()),
  loadData: vi.fn(),
  location: { protocol: 'http:', host: 'panel.test' },
  baseUrl: '/',
  ...overrides,
})

describe('WsRuntime regression anchors', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('connects on the happy path and dispatches parsed events', async () => {
    const socket = new FakeSocket()
    const onEvent = vi.fn()
    const onState = vi.fn()
    const deps = runtimeDeps({
      createSocket: vi.fn(() => socket),
      onEvent,
      onState,
    })
    const runtime = new WsRuntime(deps)

    await runtime.connect()
    expect(deps.createSocket).toHaveBeenCalledWith('ws://panel.test/api/realtime/ws', 'ws-token')
    expect(runtime.state).toBe('reconnecting')

    socket.onopen?.()
    expect(runtime.state).toBe('connected')
    expect(onState).toHaveBeenLastCalledWith('connected')

    socket.onmessage?.({ data: '{"type":"onlines","payload":{"alice":true}}' })
    expect(onEvent).toHaveBeenCalledWith({ type: 'onlines', payload: { alice: true } })
  })

  it('falls back to degraded polling when no websocket token is available', async () => {
    const timers = new ManualTimers()
    const deps = runtimeDeps({
      getToken: vi.fn(async () => null),
      setInterval: timers.setInterval,
      clearInterval: timers.clearInterval,
    })
    const runtime = new WsRuntime(deps)

    await runtime.connect()

    expect(runtime.state).toBe('degraded')
    expect(deps.createSocket).not.toHaveBeenCalled()
    expect(timers.setInterval).toHaveBeenCalledWith(expect.any(Function), 10000)

    timers.runInterval()
    expect(deps.loadData).toHaveBeenCalledTimes(1)
  })

  it('falls back when the socket does not open before the timeout', async () => {
    const timers = new ManualTimers()
    const socket = new FakeSocket()
    const deps = runtimeDeps({
      createSocket: vi.fn(() => socket),
      setTimeout: timers.setTimeout,
      clearTimeout: timers.clearTimeout,
      setInterval: timers.setInterval,
      clearInterval: timers.clearInterval,
    })
    const runtime = new WsRuntime(deps)

    await runtime.connect()
    expect(runtime.state).toBe('reconnecting')

    timers.runNextTimeout()

    expect(socket.close).toHaveBeenCalledTimes(1)
    expect(runtime.state).toBe('degraded')
    expect(timers.setInterval).toHaveBeenCalledWith(expect.any(Function), 10000)
  })

  it('ignores malformed messages without closing a healthy websocket', async () => {
    const socket = new FakeSocket()
    const onEvent = vi.fn()
    const runtime = new WsRuntime(runtimeDeps({
      createSocket: vi.fn(() => socket),
      onEvent,
    }))

    await runtime.connect()
    socket.onopen?.()
    socket.onmessage?.({ data: '{not-json' })

    expect(runtime.state).toBe('connected')
    expect(socket.close).not.toHaveBeenCalled()
    expect(onEvent).not.toHaveBeenCalled()
  })

  it('schedules a reconnect after a connected socket closes once', async () => {
    const timers = new ManualTimers()
    const sockets: FakeSocket[] = []
    const deps = runtimeDeps({
      createSocket: vi.fn(() => {
        const socket = new FakeSocket()
        sockets.push(socket)
        return socket
      }),
      setTimeout: timers.setTimeout,
      clearTimeout: timers.clearTimeout,
    })
    const runtime = new WsRuntime(deps)

    await runtime.connect()
    sockets[0].onopen?.()
    sockets[0].onclose?.({ code: 1006 })

    expect(runtime.state).toBe('reconnecting')
    expect(timers.setTimeout).toHaveBeenLastCalledWith(expect.any(Function), expect.any(Number))

    timers.runNextTimeout()
    await flushPromises()

    expect(deps.getToken).toHaveBeenCalledTimes(2)
    expect(deps.createSocket).toHaveBeenCalledTimes(2)
  })

  it('enters fallback after three consecutive closes before open', async () => {
    const timers = new ManualTimers()
    const sockets: FakeSocket[] = []
    const deps = runtimeDeps({
      createSocket: vi.fn(() => {
        const socket = new FakeSocket()
        sockets.push(socket)
        return socket
      }),
      setTimeout: timers.setTimeout,
      clearTimeout: timers.clearTimeout,
      setInterval: timers.setInterval,
      clearInterval: timers.clearInterval,
    })
    const runtime = new WsRuntime(deps)

    await runtime.connect()
    sockets[0].onclose?.({ code: 1006 })
    timers.runNextTimeout()
    await flushPromises()
    sockets[1].onclose?.({ code: 1006 })
    timers.runNextTimeout()
    await flushPromises()
    sockets[2].onclose?.({ code: 1006 })

    expect(runtime.state).toBe('degraded')
    expect(timers.setInterval).toHaveBeenCalledWith(expect.any(Function), 10000)
  })

  it('heals from degraded fallback after the fallback poll interval', async () => {
    vi.useFakeTimers()
    const socket = new FakeSocket()
    const deps = runtimeDeps({
      getToken: vi.fn()
        .mockResolvedValueOnce(null)
        .mockResolvedValueOnce('ws-token'),
      createSocket: vi.fn(() => socket),
    })
    const runtime = new WsRuntime(deps)

    await runtime.connect()
    expect(runtime.state).toBe('degraded')
    expect(deps.getToken).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(10000)

    expect(deps.loadData).toHaveBeenCalledTimes(1)
    expect(deps.getToken).toHaveBeenCalledTimes(2)
    expect(deps.createSocket).toHaveBeenCalledTimes(1)
    expect(runtime.state).toBe('reconnecting')

    socket.onopen?.()

    expect(runtime.state).toBe('connected')
    await vi.advanceTimersByTimeAsync(10000)
    expect(deps.loadData).toHaveBeenCalledTimes(1)
    expect(deps.createSocket).toHaveBeenCalledTimes(1)
  })
})
