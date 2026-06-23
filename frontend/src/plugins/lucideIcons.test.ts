import { describe, expect, it } from 'vitest'

import { iconMap } from './lucideIcons'

// Mirror of the mdi scan (mdiIcons.test.ts): the Nexus UI's visible icons use the
// `lucide:<name>` set prefix. A name not present in iconMap renders a blank <i>
// with only a DEV console.warn — invisible in production. This test fails the
// build if any `lucide:*` referenced in source is unmapped (e.g. a typo).
const rawSources = import.meta.glob('../**/*.{vue,ts}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

// lucideIcons.ts holds the map itself (and an `import.meta.glob` example would not
// apply); test files are skipped. Everything else is a real consumer.
function isInfrastructure(path: string): boolean {
  return path.endsWith('.test.ts') || path.endsWith('/lucideIcons.ts')
}

function lucideIconsReferencedInSource(): Set<string> {
  const pattern = /lucide:([a-z0-9]+(?:-[a-z0-9]+)*)/g
  const found = new Set<string>()
  for (const [path, content] of Object.entries(rawSources)) {
    if (isInfrastructure(path)) continue
    for (const match of content.matchAll(pattern)) found.add(match[1])
  }
  return found
}

describe('lucide icon set', () => {
  it('maps every lucide:* icon referenced in source', () => {
    const referenced = [...lucideIconsReferencedInSource()]
    expect(referenced.length).toBeGreaterThan(20) // sanity: the scan found icons
    const missing = referenced.filter((name) => !Object.hasOwn(iconMap, name)).sort()
    expect(
      missing,
      `unmapped lucide icons — add them to iconMap in plugins/lucideIcons.ts: ${missing.join(', ')}`,
    ).toEqual([])
  })

  it('only maps to defined Lucide components', () => {
    for (const [name, component] of Object.entries(iconMap)) {
      expect(component, name).toBeTruthy()
    }
  })
})
