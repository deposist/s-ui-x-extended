import { afterEach, describe, expect, it, vi } from 'vitest'

import { isNexusEnabled } from './featureGate'

describe('isNexusEnabled', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('defaults to enabled when no feature gate is defined', () => {
    vi.stubEnv('VITE_ENABLE_NEXUS', undefined)

    expect(isNexusEnabled()).toBe(true)
  })

  it.each(['false', '0'])('disables Nexus for %s', value => {
    vi.stubEnv('VITE_ENABLE_NEXUS', value)

    expect(isNexusEnabled()).toBe(false)
  })

  it('keeps Nexus enabled for an unknown value', () => {
    vi.stubEnv('VITE_ENABLE_NEXUS', 'preview')

    expect(isNexusEnabled()).toBe(true)
  })
})
