import { describe, expect, it } from 'vitest'
import { isPreloadError } from './preload-error'

describe('isPreloadError', () => {
  it('recognizes Firefox dynamic import failures', () => {
    expect(isPreloadError(new TypeError(
      'error loading dynamically imported module: http://example.test/app/assets/chunk.js',
    ))).toBe(true)
  })

  it('recognizes Chromium dynamic import failures', () => {
    expect(isPreloadError(new TypeError(
      'Failed to fetch dynamically imported module: http://example.test/app/assets/chunk.js',
    ))).toBe(true)
  })

  it('recognizes bundler chunk errors by name', () => {
    expect(isPreloadError({ name: 'ChunkLoadError' })).toBe(true)
  })

  it('ignores unrelated errors', () => {
    expect(isPreloadError(new Error('login failed'))).toBe(false)
  })
})
