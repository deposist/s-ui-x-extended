import { describe, it, expect } from 'vitest'
import { OutTypes, createOutbound, type Call } from './outbounds'
import { outboundGroupCapabilities } from './capabilities'

describe('outbound group types', () => {
  it('exports all four group mode types', () => {
    expect(OutTypes.Selector).toBe('selector')
    expect(OutTypes.URLTest).toBe('urltest')
    expect(OutTypes.Fallback).toBe('fallback')
    expect(OutTypes.Failover).toBe('failover')
  })

  it('capability manifest declares all four group modes with sessionRecovery=false', () => {
    const byType = new Map(outboundGroupCapabilities.map((g) => [g.type, g]))
    for (const groupType of ['selector', 'urltest', 'fallback', 'failover']) {
      expect(byType.has(groupType), `missing group capability ${groupType}`).toBe(true)
    }
    for (const group of outboundGroupCapabilities) {
      expect(group.sessionRecovery, `${group.type} must not claim session recovery`).toBe(false)
    }
  })

  it('panel failover is assembled as selector', () => {
    const failover = outboundGroupCapabilities.find((g) => g.type === 'failover')
    expect(failover).toBeDefined()
    expect(failover!.assembledAs).toBe('selector')
    expect(failover!.panelManaged).toBe(true)
  })

  it('creates a call outbound with join_link required by the core', () => {
    const call = createOutbound(OutTypes.Call, { id: 0, tag: 'call' }) as Call

    expect(call.type).toBe('call')
    expect(call.platform).toBe('dion')
    expect(call.join_link).toBe('')
    expect(call.read_buffer).toBe(32768)
    expect(call.cookies).toBeUndefined()
  })
})
