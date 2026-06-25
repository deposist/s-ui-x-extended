import { describe, expect, it } from 'vitest'

import { nexusMenu, nexusMenuGroups, nexusSingBoxSettingsPaths } from './nexusMenu'

describe('nexus sing-box settings route parity', () => {
  it('keeps Nexus navigation wired to shared sing-box editor surfaces', () => {
    expect(nexusSingBoxSettingsPaths).toEqual(expect.arrayContaining([
      '/inbounds',
      '/outbounds',
      '/endpoints',
      '/services',
      '/tls',
      '/rules',
      '/dns',
    ]))
  })

  it('keeps each Nexus menu path unique', () => {
    const paths = nexusMenu.map(item => item.path)

    expect(new Set(paths).size).toBe(paths.length)
  })
})

describe('nexus grouped navigation integrity', () => {
  it('derives the flat menu from the groups without dropping any entry', () => {
    const grouped = nexusMenuGroups.flatMap(group => group.items)

    expect(grouped).toEqual(nexusMenu)
  })

  it('covers every required destination across the groups', () => {
    const paths = nexusMenuGroups.flatMap(group => group.items.map(item => item.path))

    expect(paths).toEqual(expect.arrayContaining([
      '/', '/inbounds', '/clients', '/outbounds', '/endpoints', '/services',
      '/tls', '/rules', '/dns', '/telegram', '/paid-subscriptions',
      '/admins', '/audit', '/settings', '/donations',
    ]))
    expect(paths).toHaveLength(15)
  })

  it('labels every non-dashboard group with a nav.groups.* key', () => {
    const labelled = nexusMenuGroups.filter(group => group.labelKey)

    expect(labelled).toHaveLength(5)
    labelled.forEach(group => {
      expect(group.labelKey).toMatch(/^nav\.groups\./)
    })
  })

  it('leaves the first (dashboard) group without a subheader label', () => {
    expect(nexusMenuGroups[0].labelKey).toBeUndefined()
  })
})
