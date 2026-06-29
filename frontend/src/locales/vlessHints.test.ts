import { describe, expect, it } from 'vitest'

import {
  commonEndpointFieldHintKeys,
  commonInboundFieldHintKeys,
  commonOutboundFieldHintKeys,
  commonServiceFieldHintKeys,
  dnsRuleFieldHintKeys,
  dnsServerFieldHintKeys,
  routeRuleFieldHintKeys,
  tlsFieldHintKeys,
  vlessInboundFieldHintKeys,
} from '@/utils/defaultRecommendations'
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

describe('recommendation locale coverage', () => {
  const requiredKeys = [
    'types.vless.recommendedPreset',
    'types.inbound.recommendedPreset',
    'types.outbound.recommendedPreset',
    'types.service.recommendedPreset',
    'types.endpoint.recommendedPreset',
    'types.tls.recommendedPreset',
    'types.dns.recommendedPreset',
    'types.dnsRule.recommendedPreset',
    'types.rule.recommendedPreset',
    'tls.incompatibleTemplate',
    'setting.hint.subJsonRouteDirect',
    'setting.hint.subJsonRouteBlock',
    'setting.hint.subJsonLogLevel',
    'setting.hint.subJsonLogTimestamp',
    'setting.hint.subJsonDnsFinal',
    'setting.hint.subJsonDefaultResolver',
    'setting.hint.subJsonDnsDirect',
    'setting.hint.subJsonTunAddress',
    'setting.hint.subJsonTunMtu',
    'setting.hint.subJsonExcludePackage',
    'setting.hint.subJsonPlatformProxy',
    'setting.hint.subJsonOptions',
    'setting.hint.subClashMixedPort',
    'setting.hint.subClashAllowLan',
    'setting.hint.subClashExternalController',
    'setting.hint.subClashLogLevel',
    'setting.hint.subClashTun',
    'setting.hint.subClashDns',
    'setting.hint.subClashRules',
    'setting.hint.subClashOptions',
    ...Object.values(vlessInboundFieldHintKeys),
    ...Object.values(commonInboundFieldHintKeys),
    ...Object.values(commonOutboundFieldHintKeys),
    ...Object.values(commonServiceFieldHintKeys),
    ...Object.values(commonEndpointFieldHintKeys),
    ...Object.values(tlsFieldHintKeys),
    ...Object.values(dnsServerFieldHintKeys),
    ...Object.values(dnsRuleFieldHintKeys),
    ...Object.values(routeRuleFieldHintKeys),
  ]

  it('defines every recommendation hint in all supported locale files', () => {
    for (const [locale, messages] of Object.entries(locales)) {
      const missing = requiredKeys.filter((key) => typeof getByPath(messages, key) !== 'string' || getByPath(messages, key) === '')
      expect(missing, `${locale} is missing keys: ${missing.join(', ')}`).toEqual([])
    }
  })
})
