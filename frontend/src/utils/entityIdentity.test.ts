import { describe, expect, it } from 'vitest'

import { isBlankIdentity } from './entityIdentity'

describe('isBlankIdentity', () => {
  it('rejects missing, empty, and whitespace-only identities', () => {
    expect(isBlankIdentity(undefined)).toBe(true)
    expect(isBlankIdentity(null)).toBe(true)
    expect(isBlankIdentity('')).toBe(true)
    expect(isBlankIdentity(' \t\n ')).toBe(true)
  })

  it('accepts real identities without normalizing them', () => {
    expect(isBlankIdentity('direct')).toBe(false)
    expect(isBlankIdentity(' my tag ')).toBe(false)
    expect(isBlankIdentity('0')).toBe(false)
  })
})
