import { afterEach, describe, expect, it, vi } from 'vitest'

import RandomUtil from './randomUtil'

const mockUint64Samples = (...samples: bigint[]) => {
  const queue = [...samples]
  const getRandomValues = vi.fn((values: Uint32Array) => {
    const sample = queue.shift()
    if (sample === undefined) {
      throw new Error('No mocked random sample remains')
    }

    values[0] = Number(sample >> 32n)
    values[1] = Number(sample & 0xffffffffn)
    return values
  })

  vi.stubGlobal('window', { crypto: { getRandomValues } })
  return getRandomValues
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('RandomUtil integer sampling', () => {
  it('maps samples to both inclusive boundaries', () => {
    mockUint64Samples(0n, 6n)

    expect(RandomUtil.randomIntRange(-3, 3)).toBe(-3)
    expect(RandomUtil.randomIntRange(-3, 3)).toBe(3)
  })

  it('retries a rejected sample before accepting', () => {
    const getRandomValues = mockUint64Samples((1n << 64n) - 1n, 9n)

    expect(RandomUtil.randomIntRange(0, 9)).toBe(9)
    expect(getRandomValues).toHaveBeenCalledTimes(2)
  })

  it('supports swapped bounds and returns singleton ranges without sampling', () => {
    const getRandomValues = mockUint64Samples(4n)

    expect(RandomUtil.randomIntRange(9, 5)).toBe(9)
    expect(RandomUtil.randomIntRange(42, 42)).toBe(42)
    expect(getRandomValues).toHaveBeenCalledOnce()
  })

  it('normalizes invalid bounds to safe-integer endpoints', () => {
    mockUint64Samples(0n)

    expect(RandomUtil.randomIntRange(Number.NaN, Number.POSITIVE_INFINITY))
      .toBe(Number.MIN_SAFE_INTEGER)
  })

  it('covers the full safe-integer span without overflowing its width', () => {
    const upperBoundaryOffset = BigInt(Number.MAX_SAFE_INTEGER) - BigInt(Number.MIN_SAFE_INTEGER)
    mockUint64Samples(upperBoundaryOffset)

    expect(RandomUtil.randomIntRange(Number.MIN_SAFE_INTEGER, Number.MAX_SAFE_INTEGER))
      .toBe(Number.MAX_SAFE_INTEGER)
  })
})

describe('RandomUtil inclusive callers', () => {
  it('keeps sequence indices within their alphabets', () => {
    mockUint64Samples(61n, 62n, 35n, 36n)

    expect(RandomUtil.randomSeq(1)).toBe('Z')
    expect(RandomUtil.randomSeq(1)).toBe('0')
    expect(RandomUtil.randomLowerAndNum(1)).toBe('z')
    expect(RandomUtil.randomLowerAndNum(1)).toBe('0')
  })

  it('uses 255 as the inclusive maximum byte value for short IDs', () => {
    const samples = Array.from({ length: 23 }, () => [0n, 255n, 0n]).flat()
    mockUint64Samples(...samples)
    const randomInt = vi.spyOn(RandomUtil, 'randomInt')

    const shortIds = RandomUtil.randomShortId()

    expect(shortIds[0]).toBe('')
    expect(shortIds.slice(1)).toEqual(new Array(23).fill('ff'))
    expect(randomInt).toHaveBeenCalledWith(255)
    expect(randomInt).not.toHaveBeenCalledWith(256)
  })
})
