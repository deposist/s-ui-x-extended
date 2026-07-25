import { describe, expect, it, vi } from 'vitest'

vi.mock('@/router', () => ({ default: { push: vi.fn(), currentRoute: { value: {} } } }))

import InboundDrawer from './InboundDrawer.vue'
import OutboundDrawer from './OutboundDrawer.vue'
import ServiceDrawer from './ServiceDrawer.vue'
import TlsDrawer from './TlsDrawer.vue'
type Computed = (this: Record<string, unknown>) => boolean
type DrawerOptions = { computed?: Record<string, Computed> }

function resultFor(component: unknown, name: string, context: Record<string, unknown>): boolean {
  // Vue's SFC type omits Options API internals that remain available at runtime.
  const computed = (component as DrawerOptions).computed?.[name]
  if (!computed) throw new Error(`${name} computed not found`)
  return computed.call(context)
}

const blankIdentities = ['', ' ', '   ', '\t', '\n', ' \t ']

describe('entity drawer identity guards', () => {
  it.each(blankIdentities)('blocks inbound Save for a blank tag (%j)', (tag) => {
    expect(resultFor(InboundDrawer, 'validate', {
      inbound: { tag, type: 'direct', listen_port: 443 },
      OnlyTLS: [],
      selectedTlsTemplateCompatible: true,
    })).toBe(false)
  })

  it.each(blankIdentities)('blocks outbound Save for a blank tag (%j)', (tag) => {
    expect(resultFor(OutboundDrawer, 'saveBlocked', { outbound: { tag } })).toBe(true)
  })

  it.each(blankIdentities)('blocks service Save for a blank tag (%j)', (tag) => {
    expect(resultFor(ServiceDrawer, 'saveBlocked', { srv: { tag } })).toBe(true)
  })

  it.each(blankIdentities)('blocks TLS Save for a blank name (%j)', (name) => {
    expect(resultFor(TlsDrawer, 'saveBlocked', { tls: { name } })).toBe(true)
  })

  it('allows real identities, including the tag "0"', () => {
    expect(resultFor(OutboundDrawer, 'saveBlocked', { outbound: { tag: '0' } })).toBe(false)
    expect(resultFor(ServiceDrawer, 'saveBlocked', { srv: { tag: 'service-a' } })).toBe(false)
    expect(resultFor(TlsDrawer, 'saveBlocked', { tls: { name: 'tls-a' } })).toBe(false)
  })
})
