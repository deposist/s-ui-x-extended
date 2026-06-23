import { describe, expect, it } from 'vitest'

import en from './en'
import ru from './ru'

// Project convention (see CLAUDE.md / redesign notes): en is the source of truth
// and ru is the second fully-maintained locale; fa/vi/zhcn/zhtw intentionally fall
// back to en for newer keys (fallbackLocale='en'). So we enforce en<->ru parity
// only - this catches a translation added to one but forgotten in the other (which
// would surface as an unexpected English string in RU).
const flatten = (obj: Record<string, unknown>, prefix = ''): string[] => {
  const out: string[] = []
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      out.push(...flatten(v as Record<string, unknown>, key))
    } else {
      out.push(key)
    }
  }
  return out
}

describe('en/ru locale key parity', () => {
  const enKeys = new Set(flatten(en as Record<string, unknown>))
  const ruKeys = new Set(flatten(ru as Record<string, unknown>))

  it('ru defines every key en defines', () => {
    const missing = [...enKeys].filter((k) => !ruKeys.has(k)).sort()
    expect(missing, `keys in en but missing from ru: ${missing.join(', ')}`).toEqual([])
  })

  it('ru does not define keys absent from en', () => {
    const extra = [...ruKeys].filter((k) => !enKeys.has(k)).sort()
    expect(extra, `keys in ru but missing from en: ${extra.join(', ')}`).toEqual([])
  })
})
