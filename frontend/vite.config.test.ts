import { describe, expect, it } from 'vitest'
import config from './vite.config.mts'

const output = config.build?.rollupOptions?.output

if (!output || Array.isArray(output)) {
  throw new Error('expected single Rollup output config')
}

describe('vite asset names', () => {
  it('prefixes emitted assets before hashes so Go embed keeps them', () => {
    expect(output.entryFileNames).toBe('assets/entry-[hash].js')
    expect(output.chunkFileNames).toBe('assets/chunk-[hash].js')
    expect(output.assetFileNames?.({ names: ['app.css'] } as any)).toBe('assets/style-[hash].css')
    expect(output.assetFileNames?.({ names: ['_icon.svg'] } as any)).toBe('assets/asset-[name][extname]')
  })
})
