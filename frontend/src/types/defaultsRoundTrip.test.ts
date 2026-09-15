import { beforeAll, describe, it, expect } from 'vitest'
import { webcrypto } from 'node:crypto'
import { InTypes, createInbound } from './inbounds'
import { OutTypes, createOutbound } from './outbounds'
import { EpTypes, createEndpoint } from './endpoints'
import { SrvTypes, createSrv } from './services'
import { DnsTypes, createDnsServer } from './dns'
import { ProviderTypes, createProvider } from './providers'

// Editors create an object, the panel serializes it into the config blob, and the
// editor re-opens it from that JSON. Anything that does not survive
// JSON.stringify -> JSON.parse therefore becomes a field the operator sets and
// loses: `undefined` inside a container, NaN/Infinity from a parse, a Date, a
// Map/Set, a class instance, a function. This gate runs that round trip over the
// defaults of every editor type, so a new editor type cannot ship with defaults
// that silently drop fields.
//
// It also states the two value classes that truthiness-based editors get wrong:
// `false` and `0` must stay themselves, never disappear.

beforeAll(() => {
  // The editor helpers generate credentials through the browser crypto API. The
  // node test environment has no `window`, so provide the same webcrypto surface
  // rather than skipping those defaults.
  const globals = globalThis as Record<string, unknown>
  if (!globals.window) {
    globals.window = { crypto: webcrypto }
  }
})

function roundTrip(value: unknown): unknown {
  return JSON.parse(JSON.stringify(value))
}

/** Paths whose value is undefined/NaN/Infinity or a non-plain container. */
function unserializablePaths(value: unknown, path = '$'): string[] {
  const found: string[] = []
  const walk = (node: unknown, at: string) => {
    if (node === undefined) {
      found.push(`${at} is undefined`)
      return
    }
    if (typeof node === 'number' && !Number.isFinite(node)) {
      found.push(`${at} is ${node}`)
      return
    }
    if (typeof node === 'function' || typeof node === 'symbol' || typeof node === 'bigint') {
      found.push(`${at} is a ${typeof node}`)
      return
    }
    if (node instanceof Date || node instanceof Map || node instanceof Set) {
      found.push(`${at} is not a plain JSON value`)
      return
    }
    if (Array.isArray(node)) {
      node.forEach((item, index) => walk(item, `${at}[${index}]`))
      return
    }
    if (node && typeof node === 'object') {
      for (const [key, child] of Object.entries(node as unknown as Record<string, unknown>)) {
        walk(child, `${at}.${key}`)
      }
    }
  }
  walk(value, path)
  return found
}

type Case = { category: string; type: string; build: () => Record<string, unknown> }

function cases(): Case[] {
  const all: Case[] = []
  for (const type of Object.values(InTypes)) {
    all.push({ category: 'inbound', type, build: () => createInbound(type as never) as unknown as Record<string, unknown> })
  }
  for (const type of Object.values(OutTypes)) {
    all.push({ category: 'outbound', type, build: () => createOutbound(type) as unknown as Record<string, unknown> })
  }
  for (const type of Object.values(EpTypes)) {
    all.push({ category: 'endpoint', type, build: () => createEndpoint(type) as unknown as Record<string, unknown> })
  }
  for (const type of Object.values(SrvTypes)) {
    all.push({ category: 'service', type, build: () => createSrv(type) as unknown as Record<string, unknown> })
  }
  for (const type of Object.values(DnsTypes)) {
    all.push({ category: 'dns-server', type, build: () => createDnsServer(type) as unknown as Record<string, unknown> })
  }
  for (const type of Object.values(ProviderTypes)) {
    all.push({ category: 'provider', type, build: () => createProvider(type) as unknown as Record<string, unknown> })
  }
  return all
}

describe('editor defaults survive the save/load round trip', () => {
  it('has a case for every editor type', () => {
    const built = cases()
    expect(built.length).toBeGreaterThan(50)
    const duplicates = built
      .map((c) => `${c.category}:${c.type}`)
      .filter((key, index, array) => array.indexOf(key) !== index)
    expect(duplicates, 'duplicate type keys across categories').toEqual([])
  })

  for (const { category, type, build } of cases()) {
    it(`${category} ${type}`, () => {
      const created = build()

      // A type with no defaults entry would silently produce an object without a
      // `type`, which the core rejects - assert the factory knows the type.
      expect((created as Record<string, unknown>).type, `${category} ${type} has no defaults entry`).toBe(type)

      const bad = unserializablePaths(created)
      expect(bad, `${category} ${type} defaults cannot be serialized: ${bad.join(', ')}`).toEqual([])

      const reloaded = roundTrip(created) as Record<string, unknown>
      expect(reloaded, `${category} ${type} defaults changed across the round trip`).toEqual(created)
    })
  }

  it('keeps false and 0 from being dropped as falsy values', () => {
    // A stored object with falsy-but-meaningful values must come back unchanged:
    // an editor that rebuilds its object from truthy defaults would drop these.
    const stored = {
      type: InTypes.VLESS,
      tag: 'node',
      listen: '::',
      listen_port: 0,
      udp: false,
      tls: { enabled: false, insecure: false, alpn: [] },
      users: [],
      transport: {},
    }
    const reloaded = roundTrip(createInbound(stored.type, stored)) as Record<string, unknown>

    expect(reloaded.listen_port).toBe(0)
    expect(reloaded.udp).toBe(false)
    expect(reloaded.tls).toMatchObject({ enabled: false, insecure: false, alpn: [] })
    expect(reloaded.users).toEqual([])
    expect(reloaded.transport).toEqual({})
  })

  it('keeps nested lists and secrets intact across the round trip', () => {
    const stored = {
      type: OutTypes.VMess,
      tag: 'out',
      server: 'example.com',
      server_port: 443,
      uuid: '11111111-1111-4111-8111-111111111111',
      alter_id: 0,
      security: 'auto',
      tls: { enabled: true, server_name: 'example.com', reality: { enabled: false, public_key: '', short_id: '' } },
      transport: { type: 'ws', path: '/ws', headers: { Host: 'example.com' } },
    }
    const reloaded = roundTrip(createOutbound(stored.type, stored)) as Record<string, unknown>

    expect(reloaded).toMatchObject(stored)
    expect((reloaded.tls as unknown as Record<string, unknown>).reality).toMatchObject({ enabled: false })
    expect((reloaded.transport as unknown as Record<string, unknown>).headers).toEqual({ Host: 'example.com' })
  })
})