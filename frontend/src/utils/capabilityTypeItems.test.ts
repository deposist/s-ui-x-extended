import { beforeAll, describe, it, expect } from 'vitest'
import { loadLocaleMessages } from '@/locales'
import { capabilityRows, capabilityTypeItems, unavailableReason } from './capabilityTypeItems'

// The app loads locale messages asynchronously before mounting; the unit test has
// to do the same, otherwise i18n returns the raw key and the labels below would
// assert nothing about what a user actually reads.
beforeAll(async () => {
  await loadLocaleMessages('en')
})

const types = { Wireguard: 'wireguard', OpenVPNClient: 'openvpn-client', Redirect: 'redirect' }

describe('editor type picker explains unavailable types', () => {
  it('lists every type and disables only the unusable ones', () => {
    const rows = capabilityRows([
      { type: 'wireguard', available: true },
      { type: 'openvpn-client', available: false },
    ])
    const items = capabilityTypeItems(types, rows)

    expect(items.map((i) => i.value)).toEqual(['wireguard', 'openvpn-client', 'redirect'])
    expect(items.find((i) => i.value === 'wireguard')?.props).toBeUndefined()
    expect(items.find((i) => i.value === 'openvpn-client')?.props).toEqual({ disabled: true })
  })

  it('names the platform when the type is only unimplemented here', () => {
    const rows = capabilityRows([{ type: 'redirect', available: false, platforms: ['linux', 'darwin'] }])
    const item = capabilityTypeItems(types, rows).find((i) => i.value === 'redirect')

    expect(item?.title).toContain('linux/darwin')
    expect(item?.props).toEqual({ disabled: true })
  })

  it('falls back to the build reason when no platform list is reported', () => {
    const rows = capabilityRows([{ type: 'openvpn-client', available: false }])
    expect(unavailableReason(rows, 'openvpn-client')).toBeTruthy()
    expect(capabilityTypeItems(types, rows).find((i) => i.value === 'openvpn-client')?.title).not.toContain('linux')
  })

  it('treats a missing or malformed capabilities response as "everything available"', () => {
    for (const payload of [undefined, null, [], 'nonsense', [{ available: false }]]) {
      const rows = capabilityRows(payload)
      expect(unavailableReason(rows, 'redirect')).toBe('')
      expect(capabilityTypeItems(types, rows).every((i) => i.props === undefined)).toBe(true)
    }
  })
})