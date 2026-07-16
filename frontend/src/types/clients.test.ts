import { describe, expect, it } from 'vitest'
import { randomConfigs, normalizeSudokuClientKey } from './clients'

const validKey = '01'.padEnd(64, '0') + '02'.padEnd(64, '0')

describe('Sudoku client split key form contract', () => {
  it('creates an optional Sudoku config field', () => {
    Object.defineProperty(globalThis, 'window', {
      value: { crypto: { getRandomValues: (values: Uint32Array) => values.fill(1) } },
      configurable: true,
    })
    expect(randomConfigs('alice').sudoku.key).toBe('')
  })

  it('trims and lowercases a 128-character hex key', () => {
    expect(normalizeSudokuClientKey(`  ${validKey.toUpperCase()}\n`)).toEqual({ value: validKey, valid: true })
  })

  it('allows fallback and rejects malformed or master-length values', () => {
    expect(normalizeSudokuClientKey('')).toEqual({ value: '', valid: true })
    expect(normalizeSudokuClientKey('ab'.repeat(32)).valid).toBe(false)
    expect(normalizeSudokuClientKey('z'.repeat(128)).valid).toBe(false)
  })
})
