import { describe, expect, it } from 'vitest'

import { parseAmneziaHeader, headerRangesOverlap, validateAmnezia } from './amneziaValidation'

describe('parseAmneziaHeader', () => {
  it('returns null for absent or empty values', () => {
    expect(parseAmneziaHeader(undefined)).toBeNull()
    expect(parseAmneziaHeader(null)).toBeNull()
    expect(parseAmneziaHeader('')).toBeNull()
    expect(parseAmneziaHeader('   ')).toBeNull()
  })

  it('parses single numbers', () => {
    expect(parseAmneziaHeader(1000)).toEqual({ from: 1000, to: 1000 })
    expect(parseAmneziaHeader('12345')).toEqual({ from: 12345, to: 12345 })
    expect(parseAmneziaHeader(4294967295)).toEqual({ from: 4294967295, to: 4294967295 })
  })

  it('parses ranges', () => {
    expect(parseAmneziaHeader('1000-2000')).toEqual({ from: 1000, to: 2000 })
    expect(parseAmneziaHeader('5-5')).toEqual({ from: 5, to: 5 })
  })

  it('rejects malformed values', () => {
    expect(parseAmneziaHeader('2000-1000')).toBeUndefined()
    expect(parseAmneziaHeader('1-2-3')).toBeUndefined()
    expect(parseAmneziaHeader('abc')).toBeUndefined()
    expect(parseAmneziaHeader('10 - 20')).toBeUndefined()
    expect(parseAmneziaHeader('4294967296')).toBeUndefined()
    expect(parseAmneziaHeader(-5)).toBeUndefined()
    expect(parseAmneziaHeader(10.5)).toBeUndefined()
    expect(parseAmneziaHeader(true)).toBeUndefined()
  })
})

describe('headerRangesOverlap', () => {
  it('detects overlap and adjacency correctly', () => {
    expect(headerRangesOverlap({ from: 1000, to: 2000 }, { from: 1500, to: 2500 })).toBe(true)
    expect(headerRangesOverlap({ from: 1000, to: 2000 }, { from: 2000, to: 3000 })).toBe(true)
    expect(headerRangesOverlap({ from: 1000, to: 2000 }, { from: 2001, to: 3000 })).toBe(false)
    expect(headerRangesOverlap({ from: 5, to: 5 }, { from: 5, to: 5 })).toBe(true)
  })
})

describe('validateAmnezia', () => {
  const valid = {
    jc: 4, jmin: 40, jmax: 90,
    s1: 15, s2: 20, s3: 12, s4: 8,
    h1: '1000-1099', h2: 2000, h3: '3000-3099', h4: '4000-4099',
  }

  it('accepts a valid parameter set', () => {
    expect(validateAmnezia(valid)).toEqual({})
  })

  it('accepts absent amnezia object', () => {
    expect(validateAmnezia(undefined)).toEqual({})
    expect(validateAmnezia(null)).toEqual({})
  })

  it('flags reserved header values (legacy 1..4 defaults)', () => {
    const errors = validateAmnezia({ ...valid, h1: 1, h2: 2, h3: 3, h4: 4 })
    expect(errors.h1).toBe('headerReserved')
    expect(errors.h4).toBe('headerReserved')
  })

  it('flags a range touching reserved values', () => {
    const errors = validateAmnezia({ ...valid, h1: '4-100' })
    expect(errors.h1).toBe('headerReserved')
  })

  it('flags overlapping header ranges on both fields', () => {
    const errors = validateAmnezia({ ...valid, h1: '1000-2000', h2: '1500-2500' })
    expect(errors.h1).toBe('headerOverlap')
    expect(errors.h2).toBe('headerOverlap')
    expect(errors.h3).toBeUndefined()
  })

  it('flags malformed header text', () => {
    const errors = validateAmnezia({ ...valid, h2: '12ab' })
    expect(errors.h2).toBe('headerFormat')
  })

  it('flags jmin above jmax', () => {
    const errors = validateAmnezia({ ...valid, jmin: 90, jmax: 40 })
    expect(errors.jmin).toBe('jminAboveJmax')
    expect(errors.jmax).toBe('jminAboveJmax')
  })

  it('flags out-of-range junk values', () => {
    expect(validateAmnezia({ ...valid, jc: 129 }).jc).toBe('jcRange')
    expect(validateAmnezia({ ...valid, jc: -1 }).jc).toBe('jcRange')
    expect(validateAmnezia({ ...valid, jmax: 1281 }).jmax).toBe('junkSizeRange')
  })

  it('flags S1+56==S2 packet size collision', () => {
    const errors = validateAmnezia({ ...valid, s1: 15, s2: 71 })
    expect(errors.s1).toBe('equalPacketSizes')
    expect(errors.s2).toBe('equalPacketSizes')
  })

  it('flags init/cookie size collision (s1 vs s3)', () => {
    // 148 + 10 == 64 + 94
    const errors = validateAmnezia({ ...valid, s1: 10, s3: 94 })
    expect(errors.s1).toBe('equalPacketSizes')
    expect(errors.s3).toBe('equalPacketSizes')
  })

  it('flags out-of-range paddings', () => {
    expect(validateAmnezia({ ...valid, s4: 1281 }).s4).toBe('paddingRange')
    expect(validateAmnezia({ ...valid, s2: -1 }).s2).toBe('paddingRange')
  })

  it('accepts partial header sets', () => {
    const errors = validateAmnezia({ ...valid, h3: undefined, h4: '' })
    expect(errors).toEqual({})
  })

  const valid30 = {
    ...valid,
    s4: 12,
    header_protection_key: 'AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA=',
    content_padding_addition: '0',
    rekey_after_time: '120-180',
    rekey_timeout: 5,
    reject_after_time: '90-120',
    keepalive_timeout: '5-10',
    max_handshake_attempts: '20-30',
  }

  it('accepts valid AWG 3.0 fields', () => {
    expect(validateAmnezia(valid30)).toEqual({})
  })

  it('accepts missing timing fields (kernel falls back to defaults)', () => {
    const errors = validateAmnezia({ ...valid, s4: 12, header_protection_key: 'AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA=' })
    expect(errors).toEqual({})
  })

  it('flags malformed timing ranges', () => {
    expect(validateAmnezia({ ...valid, rekey_after_time: '2000-1000' }).rekey_after_time).toBe('timingFormat')
    expect(validateAmnezia({ ...valid, rekey_timeout: 'abc' }).rekey_timeout).toBe('timingFormat')
    expect(validateAmnezia({ ...valid, content_padding_addition: '1-2-3' }).content_padding_addition).toBe('timingFormat')
    expect(validateAmnezia({ ...valid, keepalive_timeout: 1.5 }).keepalive_timeout).toBe('timingFormat')
    expect(validateAmnezia({ ...valid, max_handshake_attempts: true }).max_handshake_attempts).toBe('timingFormat')
    expect(validateAmnezia({ ...valid, reject_after_time: '4294967296' }).reject_after_time).toBe('timingFormat')
  })

  it('flags a malformed header protection key', () => {
    expect(validateAmnezia({ ...valid, s4: 12, header_protection_key: '!!!' }).header_protection_key).toBe('keyFormat')
    expect(validateAmnezia({ ...valid, s4: 12, header_protection_key: 'c2hvcnQ=' }).header_protection_key).toBe('keyFormat')
  })

  it('flags header protection with small S paddings', () => {
    const errors = validateAmnezia({ ...valid, header_protection_key: 'AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA=' })
    expect(errors.header_protection_key).toBe('keyPadding')
  })

  it('skips wireguard-only fields for warp endpoints', () => {
    const warp = {
      jc: 4, jmin: 40, jmax: 90,
      // Legacy 2.5.x warp leftovers: reserved headers, tiny paddings and a
      // short key are all kernel-ignored on warp, so they must not error.
      s1: 1, s2: 2, s3: 3, s4: 4,
      h1: '1-100', h2: '2-200', h3: '3-300', h4: '4-400',
      header_protection_key: 'c2hvcnQ=',
    }
    expect(validateAmnezia(warp, { warp: true })).toEqual({})
  })

  it('still validates junk and timings for warp endpoints', () => {
    const errors = validateAmnezia({ jc: 129, jmin: 40, jmax: 90, rekey_after_time: '2000-1000' }, { warp: true })
    expect(errors.jc).toBe('jcRange')
    expect(errors.rekey_after_time).toBe('timingFormat')
  })
})
