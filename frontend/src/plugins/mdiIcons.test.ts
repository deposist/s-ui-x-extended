import { describe, it, expect, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createVuetify } from 'vuetify'
import { VIcon } from 'vuetify/components'
import { mdiAccountEdit, mdiCalendar, mdiChevronRight, mdiCog, mdiShieldAlertOutline } from '@mdi/js'
import { iconPaths } from './mdiIconPaths'
import { aliases, mdi, resolveMdiIcon } from './mdiIcons'
import { auditDisplayIcons } from '@/components/nexus/overview/selectors/auditMapper'

// Read every source module as raw text via Vite's glob import (works in vitest
// and avoids node: builtins, which the browser tsconfig deliberately excludes).
const rawSources = import.meta.glob('../**/*.{vue,ts}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

// The icon infrastructure files are not icon *consumers*: mdiIconPaths.ts is the
// generated registry (holds every key by design); mdiIcons.ts and lucideIcons.ts
// both contain the 'vuetify/iconsets/mdi-svg' import path (the latter only to
// reuse that set's component type when registering the Lucide set), the generator
// script name, and doc-comment examples — none are app icon usages. Skip them;
// scan every real consumer. Test files are skipped too (this one names a
// deliberately-missing icon below).
function isInfrastructure(path: string): boolean {
  return path.endsWith('.test.ts')
    || path.endsWith('/mdiIconPaths.ts')
    || path.endsWith('/mdiIcons.ts')
    || path.endsWith('/lucideIcons.ts')
}

function iconsReferencedInSource(): Set<string> {
  const pattern = /mdi-[a-z0-9]+(?:-[a-z0-9]+)*/g
  const found = new Set<string>()
  for (const [path, content] of Object.entries(rawSources)) {
    if (isInfrastructure(path)) continue
    for (const match of content.matchAll(pattern)) found.add(match[0])
  }
  return found
}

describe('mdi icon registry (O4 SVG migration)', () => {
  it('covers every mdi-* icon referenced in source', () => {
    const referenced = [...iconsReferencedInSource()]
    expect(referenced.length).toBeGreaterThan(50) // sanity: the scan actually found icons
    const missing = referenced.filter((name) => !Object.hasOwn(iconPaths, name)).sort()
    expect(
      missing,
      `unmapped icons — regenerate with "node scripts/gen-mdi-icons.cjs": ${missing.join(', ')}`,
    ).toEqual([])
  })

  it('maps names to the matching @mdi/js path constants', () => {
    expect(iconPaths['mdi-account-edit']).toBe(mdiAccountEdit)
    expect(iconPaths['mdi-calendar']).toBe(mdiCalendar)
    expect(iconPaths['mdi-shield-alert-outline']).toBe(mdiShieldAlertOutline)
  })

  it('only contains non-empty SVG paths (never an unresolved name)', () => {
    for (const [name, path] of Object.entries(iconPaths)) {
      expect(typeof path, name).toBe('string')
      expect(path.length, name).toBeGreaterThan(0)
      expect(path.startsWith('mdi-'), name).toBe(false)
    }
  })

  it('resolves audit display icons (dynamic, data-driven) to real paths', () => {
    for (const name of auditDisplayIcons) {
      const resolved = resolveMdiIcon(name)
      expect(typeof resolved, name).toBe('string')
      expect(resolved, name).not.toBe(name) // mapped to a path, not left as the name
      expect((resolved as string).length, name).toBeGreaterThan(0)
    }
  })

  it('passes through non-mdi values unchanged (Vuetify $-aliases, raw paths, components)', () => {
    expect(resolveMdiIcon('$checkboxOn')).toBe('$checkboxOn')
    expect(resolveMdiIcon('M12,2L2,12')).toBe('M12,2L2,12')
    const component = {}
    expect(resolveMdiIcon(component)).toBe(component)
  })

  it('warns and returns the name for an unmapped mdi-* icon', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    expect(resolveMdiIcon('mdi-this-icon-does-not-exist')).toBe('mdi-this-icon-does-not-exist')
    expect(warn).toHaveBeenCalledOnce()
    warn.mockRestore()
  })

  // End-to-end: drive a real <v-icon> through Vuetify's resolver with the exact
  // icons config vuetify.ts uses, and confirm the rendered SVG carries the
  // @mdi/js path — proving useIcon -> custom "mdi" set -> VSvgIcon all wire up.
  it('renders mdi-* icons as @mdi/js SVG paths through Vuetify (SSR)', async () => {
    const vuetify = createVuetify({ icons: { defaultSet: 'mdi', aliases, sets: { mdi } } })
    const app = createSSRApp({
      render: () => h('div', [h(VIcon, { icon: 'mdi-cog' }), h(VIcon, { icon: 'mdi-chevron-right' })]),
    })
    app.use(vuetify)

    const html = await renderToString(app)
    expect(html).toContain('<path')
    expect(html).toContain(mdiCog)
    expect(html).toContain(mdiChevronRight)
    expect(html).not.toContain('mdi-cog') // the class name must not leak into output
  })
})
