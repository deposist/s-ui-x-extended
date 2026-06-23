import { expect, test } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { login, readServerState } from './helpers'

const waitForFreshE2ECredentials = async () => {
  const deadline = Date.now() + 120_000
  while (Date.now() < deadline) {
    const state = readServerState()
    const passwordPath = path.join(state.dbDir, 'initial-admin.txt')
    if (state.password && fs.existsSync(passwordPath)) {
      const password = fs.readFileSync(passwordPath, 'utf8').trim()
      if (password === state.password) return
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error('Timed out waiting for fresh e2e credentials')
}

test('websocket survives repeated offline/online chaos and returns to connected', async ({ context, page }, testInfo) => {
  await page.addInitScript(() => {
    const NativeWebSocket = window.WebSocket
    const realtimeSockets: any[] = []
    ;(window as any).__SUI_FAKE_WS_ONLINE__ = true
    ;(window as any).__SUI_FAKE_WS_CLOSE_ALL__ = () => {
      for (const socket of [...realtimeSockets]) socket.close(1006, 'offline')
    }

    class FakeRealtimeWebSocket extends EventTarget {
      static CONNECTING = NativeWebSocket.CONNECTING
      static OPEN = NativeWebSocket.OPEN
      static CLOSING = NativeWebSocket.CLOSING
      static CLOSED = NativeWebSocket.CLOSED
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      readyState = NativeWebSocket.CONNECTING
      protocol = 'sui.realtime'
      url: string

      constructor(url: string | URL) {
        super()
        this.url = String(url)
        realtimeSockets.push(this)
        ;(window as any).__SUI_WS_STATE__ = 'reconnecting'
        window.setTimeout(() => {
          if ((window as any).__SUI_FAKE_WS_ONLINE__ === false) {
            this.close(1006, 'offline')
            return
          }
          this.readyState = NativeWebSocket.OPEN
          ;(window as any).__SUI_WS_STATE__ = 'reconnecting'
          const event = new Event('open')
          this.onopen?.(event)
          this.dispatchEvent(event)
          ;(window as any).__SUI_WS_STATE__ = 'connected'
        }, 0)
      }

      close(code = 1000, reason = '') {
        if (this.readyState === NativeWebSocket.CLOSED) return
        this.readyState = NativeWebSocket.CLOSED
        realtimeSockets.splice(realtimeSockets.indexOf(this), 1)
        ;(window as any).__SUI_WS_STATE__ = 'closed'
        const event = new CloseEvent('close', { code, reason })
        this.onclose?.(event)
        this.dispatchEvent(event)
      }

      send() {}
    }

    const WebSocketProxy = function (this: any, url: string | URL, protocols?: string | string[]) {
      if (String(url).includes('/api/realtime/ws')) {
        return new FakeRealtimeWebSocket(url)
      }
      return new NativeWebSocket(url, protocols)
    } as any
    WebSocketProxy.prototype = NativeWebSocket.prototype
    Object.defineProperties(WebSocketProxy, {
      CONNECTING: { value: NativeWebSocket.CONNECTING },
      OPEN: { value: NativeWebSocket.OPEN },
      CLOSING: { value: NativeWebSocket.CLOSING },
      CLOSED: { value: NativeWebSocket.CLOSED },
    })
    window.WebSocket = WebSocketProxy as typeof WebSocket
  })

  await waitForFreshE2ECredentials()
  await login(page)
  // The router-guard connect() can lose the CSRF-token race on a slow runner and
  // drop to the degraded fallback, which only retries every fallbackPollMs (10s).
  // Poll long enough to clear one fallback cycle so the initial connect is not flaky.
  await expect.poll(async () => page.evaluate(() => (window as any).__SUI_WS_STATE__), {
    timeout: 25_000,
  }).toBe('connected')
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined)

  const setFakeWsOnline = async (online: boolean) => {
    await page.evaluate((nextOnline) => {
      ;(window as any).__SUI_FAKE_WS_ONLINE__ = nextOnline
      if (!nextOnline) (window as any).__SUI_FAKE_WS_CLOSE_ALL__?.()
    }, online)
  }

  const sequence: Array<{ offline: boolean; waitMs: number }> = [
    { offline: true, waitMs: 250 },
    { offline: false, waitMs: 500 },
    { offline: true, waitMs: 750 },
    { offline: false, waitMs: 250 },
    { offline: true, waitMs: 1000 },
    { offline: false, waitMs: 500 },
    { offline: true, waitMs: 300 },
    { offline: false, waitMs: 700 },
    { offline: true, waitMs: 600 },
    { offline: false, waitMs: 1000 },
  ]

  const log: string[] = []
  for (const [index, step] of sequence.entries()) {
    if (step.offline) {
      await setFakeWsOnline(false)
      await context.setOffline(true)
    } else {
      await context.setOffline(false)
      await setFakeWsOnline(true)
    }
    log.push(`${index + 1}: offline=${step.offline} waitMs=${step.waitMs}`)
    await page.waitForTimeout(step.waitMs)
  }
  await context.setOffline(false)
  await setFakeWsOnline(true)
  await testInfo.attach('ws-reconnect-chaos-sequence.txt', {
    body: log.join('\n'),
    contentType: 'text/plain',
  })

  await expect.poll(async () => {
    return page.evaluate(() => (window as any).__SUI_WS_STATE__ ?? 'unknown')
  }, { timeout: 25_000 }).toBe('connected')
})
