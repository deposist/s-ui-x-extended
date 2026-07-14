import { expect, test } from '@playwright/test'

import { login } from './helpers'

test('websocket returns from offline/degraded state back to connected', async ({ page }) => {
  await page.addInitScript(() => {
    const NativeWebSocket = window.WebSocket
    const nativeSetTimeout = window.setTimeout

    class RecoveringWebSocket {
      onopen: ((event?: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onclose: ((event?: CloseEvent) => void) | null = null
      onerror: ((event?: Event) => void) | null = null
      private closed = false

      constructor(url: string | URL, protocols?: string | string[]) {
        const target = String(url)
        if (!target.includes('/api/realtime/ws')) {
          return new NativeWebSocket(url, protocols) as any
        }

        const attempt = Number(sessionStorage.getItem('ws-recovery-attempt') || '0') + 1
        sessionStorage.setItem('ws-recovery-attempt', String(attempt))
        nativeSetTimeout(() => {
          if (this.closed) return
          if (attempt === 1) {
            this.onerror?.(new Event('error'))
            return
          }
          this.onopen?.(new Event('open'))
        }, 25)
      }

      close() {
        if (this.closed) return
        this.closed = true
        nativeSetTimeout(() => this.onclose?.(new CloseEvent('close')), 0)
      }
    }

    Object.defineProperty(window, 'WebSocket', {
      configurable: true,
      value: RecoveringWebSocket,
    })
  })

  await login(page)
  const status = page.locator('.nexus-server-status')
  await expect(status).toBeVisible()
  await expect.poll(() => page.evaluate(() => Number(sessionStorage.getItem('ws-recovery-attempt') || '0'))).toBeGreaterThanOrEqual(2)
  await expect(status).toHaveClass(/nexus-server-status--success/, { timeout: 15_000 })
  await expect(status).toContainText('Online')
})
