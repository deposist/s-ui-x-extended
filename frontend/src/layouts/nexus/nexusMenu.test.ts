import { describe, expect, it } from 'vitest'

import { nexusMenu, nexusSingBoxSettingsPaths } from './nexusMenu'

describe('nexus sing-box settings route parity', () => {
  it('keeps Nexus navigation wired to shared sing-box editor surfaces', () => {
    expect(nexusSingBoxSettingsPaths).toEqual(expect.arrayContaining([
      '/inbounds',
      '/outbounds',
      '/endpoints',
      '/services',
      '/tls',
      '/basics',
      '/rules',
      '/dns',
    ]))
  })

  it('keeps each Nexus menu path unique', () => {
    const paths = nexusMenu.map(item => item.path)

    expect(new Set(paths).size).toBe(paths.length)
  })
})
