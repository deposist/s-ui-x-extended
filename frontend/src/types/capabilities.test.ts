import { describe, it, expect } from 'vitest'
import {
  inboundWithUsers,
  HasInData,
  HasTls,
  MuxAvailable,
  OnlyTLS,
  outboundGroupCapabilities,
  providerTypes,
} from './capabilities'

// Frozen copies of the hand-maintained lists that lived inline in
// layouts/modals/Inbound.vue BEFORE the manifest was introduced. Proves the
// generated capabilities.ts reproduces them (set-wise; the lists are used as
// .includes() membership tests so order is irrelevant). Update deliberately when
// a later phase changes a capability.
const legacyInboundWithUsers = ['mixed', 'socks', 'http', 'shadowsocks', 'vmess', 'trojan', 'naive', 'hysteria', 'shadowtls', 'tuic', 'hysteria2', 'vless', 'anytls', 'mieru', 'trusttunnel', 'ssh', 'mtproxy']
// Phase 1 added mieru/sudoku/trusttunnel/ssh: their out_json editor is now shown
// (clientDelivery flipped to json). The other lists are unchanged from Phase 0.
const legacyHasInData = ['socks', 'http', 'mixed', 'shadowsocks', 'vmess', 'shadowtls', 'trojan', 'hysteria', 'vless', 'anytls', 'tuic', 'hysteria2', 'naive', 'mieru', 'sudoku', 'trusttunnel', 'ssh']
const legacyHasTls = ['http', 'vmess', 'trojan', 'naive', 'hysteria', 'tuic', 'hysteria2', 'vless', 'anytls', 'trusttunnel']
const legacyMuxAvailable = ['vless', 'vmess', 'trojan', 'shadowsocks']
const legacyOnlyTLS = ['hysteria', 'hysteria2', 'tuic', 'naive', 'anytls']

const asSet = (a: string[]) => [...a].sort()

// Read the shared manifest as raw text (mirrors mdiIcons.test.ts's glob approach,
// avoiding node: builtins the browser tsconfig excludes). The manifest lives
// outside frontend/, hence the climbing path.
const rawManifest = import.meta.glob('../../../core/capabilities/protocols.json', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

type InboundRow = {
  type: string
  hasUsers?: boolean
  alias?: boolean
  hasInData?: boolean
  hasTlsTemplate?: boolean
  muxAvailable?: boolean
  onlyTls?: boolean
}

type GroupRow = {
  type: string
  coreType?: string
  assembledAs?: string
  panelManaged?: boolean
  sessionRecovery: boolean
  notes?: string
}

type ProviderRow = {
  type: string
  buildTag?: string
}

describe('capabilities generated from protocols.json', () => {
  it('reproduces the legacy inline Inbound.vue lists (set-wise)', () => {
    expect(asSet(inboundWithUsers)).toEqual(asSet(legacyInboundWithUsers))
    expect(asSet(HasInData)).toEqual(asSet(legacyHasInData))
    expect(asSet(HasTls)).toEqual(asSet(legacyHasTls))
    expect(asSet(MuxAvailable)).toEqual(asSet(legacyMuxAvailable))
    expect(asSet(OnlyTLS)).toEqual(asSet(legacyOnlyTLS))
  })

  it('stays in sync with the manifest (regenerate with gen-capabilities.cjs)', () => {
    const entries = Object.values(rawManifest)
    expect(entries.length, 'manifest must be readable from the test').toBe(1)
    const inbounds = (JSON.parse(entries[0]).inbounds as InboundRow[]) ?? []
    const pick = (pred: (i: InboundRow) => boolean | undefined) =>
      inbounds.filter((i) => pred(i)).map((i) => i.type)

    expect(asSet(inboundWithUsers)).toEqual(asSet(pick((i) => i.hasUsers && !i.alias)))
    expect(asSet(HasInData)).toEqual(asSet(pick((i) => i.hasInData)))
    expect(asSet(HasTls)).toEqual(asSet(pick((i) => i.hasTlsTemplate)))
    expect(asSet(MuxAvailable)).toEqual(asSet(pick((i) => i.muxAvailable)))
    expect(asSet(OnlyTLS)).toEqual(asSet(pick((i) => i.onlyTls)))
  })

  it('keeps provider types in sync with the manifest', () => {
    const entries = Object.values(rawManifest)
    expect(entries.length, 'manifest must be readable from the test').toBe(1)
    const providers = (JSON.parse(entries[0]).providers as ProviderRow[]) ?? []
    expect(asSet(providerTypes)).toEqual(asSet(providers.map((p) => p.type)))
    expect(asSet(providerTypes)).toEqual(['inline', 'local', 'remote'])
  })

  it('documents panel-managed failover boundary in group metadata', () => {
    const byType = new Map(outboundGroupCapabilities.map((g) => [g.type, g]))

    for (const groupType of ['selector', 'urltest', 'fallback', 'failover']) {
      expect(byType.has(groupType), `missing group capability ${groupType}`).toBe(true)
    }

    const failover = byType.get('failover')!
    expect(failover.panelManaged, 'panel failover must be marked panelManaged').toBe(true)
    expect(failover.assembledAs, 'panel failover must assemble as selector').toBe('selector')
    expect(failover.sessionRecovery, 'panel failover must not claim generic session recovery').toBe(false)

    for (const groupType of ['selector', 'urltest', 'fallback']) {
      const group = byType.get(groupType)!
      expect(group.panelManaged, `core-backed group ${groupType} must not be panelManaged`).toBeFalsy()
      expect(group.sessionRecovery, `core-backed group ${groupType} must not claim session recovery`).toBe(false)
    }
  })
})
