import { describe, expect, it } from 'vitest'
import source from './Stats.vue?raw'

describe('Stats polling lifecycle', () => {
  it('does not create a second polling interval while one is active', () => {
    expect(source).toContain('if (this.intervalId === 0) {')
  })

  it('stops polling and resets its interval ID when hidden or unmounted', () => {
    expect(source).toContain('beforeUnmount() {')
    expect(source).toContain('this.stopPolling()')
    expect(source).toContain('this.intervalId = 0')
  })
})
