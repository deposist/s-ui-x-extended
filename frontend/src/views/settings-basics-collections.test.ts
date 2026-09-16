import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const settingsSource = readFileSync(fileURLToPath(new URL('./Settings.vue', import.meta.url)), 'utf8')

describe('Settings Basics tab collections', () => {
  it('renders the Additional collections section in both Basics layouts', () => {
    // The Basics tab in Settings (t6) is what users actually see: /basics is a
    // redirect to /settings?tab=basics and views/Basics.vue is never routed.
    // Both layouts (nexus grid and classic fallback) must mount the three
    // ConfigCollection editors and the shared mutators.
    const tabStart = settingsSource.indexOf('<v-window-item value="t6">')
    const tabEnd = settingsSource.indexOf('<v-window-item value="t7">')
    expect(tabStart).toBeGreaterThan(-1)
    expect(tabEnd).toBeGreaterThan(tabStart)
    const basicsTab = settingsSource.slice(tabStart, tabEnd)

    const nexusPart = basicsTab.slice(0, basicsTab.indexOf('<!-- Classic Fallback layout:'))
    const classicPart = basicsTab.slice(basicsTab.indexOf('<!-- Classic Fallback layout:'))
    for (const [label, part] of [['nexus', nexusPart], ['classic', classicPart]] as const) {
      expect(part, `${label} layout renders collections section`).toContain('basic.collections.title')
      expect(part, `${label} layout renders certificate providers`).toContain("collectionAdd('certificate_providers'")
      expect(part, `${label} layout renders http clients`).toContain("collectionAdd('http_clients'")
      expect(part, `${label} layout renders network namespaces`).toContain("collectionAdd('network_namespaces'")
      expect(part, `${label} layout mounts ConfigCollection`).toContain('<ConfigCollection')
    }
  })

  it('provides the collection mutators and imports ConfigCollection', () => {
    expect(settingsSource).toContain("import ConfigCollection from '@/components/ConfigCollection.vue'")
    expect(settingsSource).toContain('type CollectionKey')
    expect(settingsSource).toContain('const collectionAdd')
    expect(settingsSource).toContain('const collectionRemove')
  })
})
