import { describe, expect, it } from 'vitest'
import { displayIP, hasRawIPRows, maskRawIP } from '@/components/ipHistory'

describe('ip history masking', () => {
  it('keeps backend masks unchanged', () => {
    expect(displayIP('masked:abcdef12', false)).toBe('masked:abcdef12')
    expect(displayIP('masked', false)).toBe('masked')
  })

  it('masks raw IPv4 and IPv6 values by default', () => {
    expect(maskRawIP('198.51.100.42')).toBe('198.51.100.x')
    expect(maskRawIP('2001:db8:abcd:12::1')).toBe('2001:db8:abcd:12::/64')
  })

  it('reveals raw values only when explicitly requested', () => {
    expect(displayIP('198.51.100.42', false)).toBe('198.51.100.x')
    expect(displayIP('198.51.100.42', true)).toBe('198.51.100.42')
  })

  it('detects whether rows contain raw IP values', () => {
    expect(hasRawIPRows([{ ip: 'masked:abcdef12', firstSeen: 1, lastSeen: 1 }])).toBe(false)
    expect(hasRawIPRows([{ ip: '198.51.100.42', firstSeen: 1, lastSeen: 1 }])).toBe(true)
  })
})
