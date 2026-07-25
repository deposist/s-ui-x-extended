import { describe, expect, it } from 'vitest'

import type { UiPalette } from './palette'
import { resolveNexusThemeName } from './nexusTheme'

describe('resolveNexusThemeName', () => {
  it.each([
    ['technical', 'technicalDark', 'technicalLight'],
    ['navy', 'nexusDark', 'nexusLight'],
    ['emerald', 'emeraldDark', 'emeraldLight'],
    ['dracula', 'draculaDark', 'draculaLight'],
  ] as const)('maps %s to its dark and light themes', (palette, dark, light) => {
    expect(resolveNexusThemeName('dark', true, palette)).toBe(dark)
    expect(resolveNexusThemeName('light', false, palette)).toBe(light)
  })

  it.each([
    ['technical', 'technicalDark', 'technicalLight'],
    ['navy', 'nexusDark', 'nexusLight'],
    ['emerald', 'emeraldDark', 'emeraldLight'],
    ['dracula', 'draculaDark', 'draculaLight'],
  ] as const)('uses system darkness for %s', (palette, dark, light) => {
    expect(resolveNexusThemeName('system', true, palette)).toBe(dark)
    expect(resolveNexusThemeName('system', false, palette)).toBe(light)
  })

  it('falls back to the technical palette for exhaustive runtime safety', () => {
    expect(resolveNexusThemeName('dark', true, 'unknown' as UiPalette)).toBe('technicalDark')
  })
})
