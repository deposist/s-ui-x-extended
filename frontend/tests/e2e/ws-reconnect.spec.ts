import { test } from '@playwright/test'

// XFAIL: пункт 32 реестра; degraded fallback пока не делает healing reconnect.
test.fixme('websocket returns from offline/degraded state back to connected', async () => {})
