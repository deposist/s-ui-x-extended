import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('getBaseUrl', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.unstubAllGlobals()
    vi.stubGlobal('window', {})
  })

  const stubBaseMeta = (content: string) => {
    vi.stubGlobal('document', {
      querySelector: (selector: string) => selector === 'meta[name="s-ui-base-url"]'
        ? { getAttribute: () => content }
        : null,
    })
  }

  it('reads the server-rendered base URL from a meta tag', async () => {
    stubBaseMeta('/whatafuck2/')

    const { getBaseUrl } = await import('./base-url')

    expect(getBaseUrl()).toBe('/whatafuck2/')
  })

  it('normalizes missing trailing slash', async () => {
    stubBaseMeta('/panel')

    const { getBaseUrl } = await import('./base-url')

    expect(getBaseUrl()).toBe('/panel/')
  })

  it('falls back to dev base when the template placeholder is not rendered', async () => {
    stubBaseMeta('{{ .BASE_URL }}')

    const { getBaseUrl } = await import('./base-url')

    expect(getBaseUrl()).toBe('/app/')
  })
})
