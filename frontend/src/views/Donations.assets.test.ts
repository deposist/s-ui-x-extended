import { statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

import source from './Donations.vue?raw'

const assetSize = (name: string) => statSync(fileURLToPath(new URL(`../assets/${name}`, import.meta.url))).size

describe('donation runtime image formats', () => {
  it('serves AVIF first with WebP and PNG compatibility fallbacks', () => {
    const avif = source.indexOf('support-s-ui-x.avif')
    const webp = source.indexOf('support-s-ui-x.webp')
    const png = source.indexOf('support-s-ui-x.png')

    expect(avif).toBeGreaterThan(-1)
    expect(webp).toBeGreaterThan(avif)
    expect(png).toBeGreaterThan(webp)
  })

  it('ships materially smaller modern runtime assets', () => {
    const pngBytes = assetSize('support-s-ui-x.png')
    const webpBytes = assetSize('support-s-ui-x.webp')
    const avifBytes = assetSize('support-s-ui-x.avif')

    expect(webpBytes).toBeLessThan(pngBytes * 0.5)
    expect(avifBytes).toBeLessThan(pngBytes * 0.2)
  })
})
