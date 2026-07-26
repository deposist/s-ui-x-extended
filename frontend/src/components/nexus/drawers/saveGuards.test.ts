/**
 * Guards the "Save is blocked for a blank identity" rule in all four entity
 * drawers.
 *
 * These call each drawer's real saveBlockedReason computed with a stand-in `this`
 * rather than mounting the component: the drawers pull in the full Vuetify
 * protocol component tree, which the node test environment cannot render. The
 * computed is the single source of truth the Save button is bound to, so calling
 * it directly still covers the behaviour that regressed.
 */
import { describe, expect, it, vi } from 'vitest'

// The drawers transitively import the app router, which calls createWebHistory()
// and needs a real `window`. Stubbing the module keeps these tests in the node
// environment instead of pulling in a DOM implementation for a pure rule check.
vi.mock('@/router', () => ({ default: { push: vi.fn(), currentRoute: { value: {} } } }))

import InboundDrawer from './InboundDrawer.vue'
import OutboundDrawer from './OutboundDrawer.vue'
import ServiceDrawer from './ServiceDrawer.vue'
import TlsDrawer from './TlsDrawer.vue'

type Computed = (this: Record<string, unknown>) => string

function reasonFor(component: any, name: string, context: Record<string, unknown>): string {
  const computed = component.computed?.[name] as Computed | undefined
  if (!computed) throw new Error(`${name} computed not found`)
  // $t echoes the key so assertions read as the message key, not a translation.
  return computed.call({ $t: (key: string) => key, ...context })
}

const blankIdentities = ['', ' ', '   ', '\t', '\n', ' \t ']

describe('InboundDrawer saveBlockedReason', () => {
  const base = { OnlyTLS: [], selectedTlsTemplateCompatible: true }

  it.each(blankIdentities)('blocks Save for a blank tag (%j)', (tag) => {
    const reason = reasonFor(InboundDrawer, 'saveBlockedReason', {
      ...base,
      inbound: { tag, type: 'direct', listen_port: 443 },
    })
    expect(reason).toBe('form.cannotSave.tagRequired')
  })

  it('allows Save for a real tag', () => {
    const reason = reasonFor(InboundDrawer, 'saveBlockedReason', {
      ...base,
      inbound: { tag: 'in-1', type: 'direct', listen_port: 443 },
    })
    expect(reason).toBe('')
  })

  it('still reports the port range problem for a valid tag', () => {
    const reason = reasonFor(InboundDrawer, 'saveBlockedReason', {
      ...base,
      inbound: { tag: 'in-1', type: 'direct', listen_port: 70000 },
    })
    expect(reason).toBe('form.cannotSave.portRange')
  })
})

describe('OutboundDrawer saveBlockedReason', () => {
  it.each(blankIdentities)('blocks Save for a blank tag (%j)', (tag) => {
    const reason = reasonFor(OutboundDrawer, 'saveBlockedReason', {
      outbound: { tag, type: 'direct' },
    })
    expect(reason).toBe('form.cannotSave.tagRequired')
  })

  it('allows Save for a real tag', () => {
    const reason = reasonFor(OutboundDrawer, 'saveBlockedReason', {
      outbound: { tag: 'direct', type: 'direct' },
    })
    expect(reason).toBe('')
  })

  it('does not treat the falsy-looking tag "0" as blank', () => {
    const reason = reasonFor(OutboundDrawer, 'saveBlockedReason', {
      outbound: { tag: '0', type: 'direct' },
    })
    expect(reason).toBe('')
  })
})

describe('ServiceDrawer saveBlockedReason', () => {
  it.each(blankIdentities)('blocks Save for a blank tag (%j)', (tag) => {
    const reason = reasonFor(ServiceDrawer, 'saveBlockedReason', {
      srv: { tag, type: 'derp', listen_port: 443 },
    })
    expect(reason).toBe('form.cannotSave.tagRequired')
  })

  it('allows Save for a real tag', () => {
    const reason = reasonFor(ServiceDrawer, 'saveBlockedReason', {
      srv: { tag: 'derp-abc', type: 'derp', listen_port: 443 },
    })
    expect(reason).toBe('')
  })
})

describe('TlsDrawer saveBlockedReason', () => {
  it.each(blankIdentities)('blocks Save for a blank name (%j)', (name) => {
    const reason = reasonFor(TlsDrawer, 'saveBlockedReason', { tls: { name } })
    expect(reason).toBe('form.cannotSave.nameRequired')
  })

  it('allows Save for a real name', () => {
    const reason = reasonFor(TlsDrawer, 'saveBlockedReason', { tls: { name: 'reality-1' } })
    expect(reason).toBe('')
  })
})
