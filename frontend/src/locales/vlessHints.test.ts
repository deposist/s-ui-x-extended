import { describe, expect, it } from 'vitest'

import { vlessInboundFieldHintKeys } from '@/utils/defaultRecommendations'
import en from './en'
import fa from './fa'
import ru from './ru'
import vi from './vi'
import zhcn from './zhcn'
import zhtw from './zhtw'

const locales: Record<string, Record<string, unknown>> = { en, ru, fa, vi, zhcn, zhtw }

const getByPath = (obj: Record<string, unknown>, path: string): unknown => {
  return path.split('.').reduce<unknown>((current, segment) => {
    if (current && typeof current === 'object') return (current as Record<string, unknown>)[segment]
    return undefined
  }, obj)
}

describe('VLESS reference recommendation locale coverage', () => {
  const requiredKeys = [
    'types.vless.recommendedPreset',
    'tls.incompatibleTemplate',
    ...Object.values(vlessInboundFieldHintKeys),
  ]

  it('defines every VLESS recommendation hint in all supported locale files', () => {
    for (const [locale, messages] of Object.entries(locales)) {
      const missing = requiredKeys.filter((key) => typeof getByPath(messages, key) !== 'string' || getByPath(messages, key) === '')
      expect(missing, `${locale} is missing keys: ${missing.join(', ')}`).toEqual([])
    }
  })
})
