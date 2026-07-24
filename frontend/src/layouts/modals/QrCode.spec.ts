import { describe, expect, it } from 'vitest'
import source from './QrCode.vue?raw'

describe('QR subscription availability check', () => {
  it('describes the action as a best-effort request rather than HTTP success', () => {
    expect(source).toContain("$t('delivery.testBestEffort')")
    expect(source).not.toContain("$t('delivery.testOk')")
  })
})
