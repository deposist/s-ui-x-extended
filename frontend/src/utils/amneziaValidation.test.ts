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
})
