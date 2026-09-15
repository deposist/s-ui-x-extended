import { describe, it, expect } from 'vitest'

// The capability manifest (core/capabilities/protocols.json) is the single source
// of truth for which protocol types exist. This test fail-closes the other half
// of that contract: every type in the manifest must actually be editable in the
// panel, i.e. its editor dispatch must reference the type's own enum member.
// A new manifest row without an editor would otherwise ship a type nobody can
// configure — the failure mode this test exists to make impossible.
//
// Types that are deliberately not bound in a specific editor are listed in
// EXCEPTIONS with the reason, and the reason's factual basis is asserted so an
// exception cannot quietly become wrong.

const manifestModules = import.meta.glob('../../../core/capabilities/protocols.json', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const vueSources = import.meta.glob('../**/*.vue', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

// Paths are relative to this file's directory, and vite keys them by their path
// relative to the glob root ('./services.ts'), which is what sourceOf matches on.
const tsSources = import.meta.glob('../types/*.ts', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const sources: Record<string, string> = { ...tsSources, ...vueSources }

type Row = { type: string; alias?: boolean }
type Manifest = {
  inbounds: Row[]
  outbounds: Row[]
  endpoints: Row[]
  services: Row[]
}

function manifest(): Manifest {
  const entries = Object.values(manifestModules)
  if (entries.length !== 1) throw new Error('manifest must be readable from the test')
  return JSON.parse(entries[0]) as Manifest
}

/** Concatenated source of the editor components that bind a type to its editor. */
function editorSource(paths: string[]): string {
  return paths
    .map((p) => {
      const found = Object.entries(sources).find(([name]) => name.endsWith('/' + p))
      if (!found) throw new Error(`editor source ${p} not found`)
      return found[1]
    })
    .join('\n')
}

function sourceOf(path: string): string {
  const found = Object.entries(sources).find(([name]) => name.endsWith('/' + path))
  if (!found) throw new Error(`source ${path} not found`)
  return found[1]
}

/** value -> enum member name for an `export const X = { Member: 'value' }` map. */
function enumMembers(path: string, enumName: string): Record<string, string> {
  const body = new RegExp(`${enumName}\\s*=\\s*\\{([\\s\\S]*?)\\n\\}`, 'm').exec(sourceOf(path))
  if (!body) throw new Error(`enum ${enumName} not found in ${path}`)
  const members: Record<string, string> = {}
  for (const [, member, value] of body[1].matchAll(/(\w+):\s*'([^']+)'/g)) {
    members[value] = member
  }
  return members
}

type CategorySpec = {
  rows: Row[]
  enumPath: string
  enumName: string
  /** Prefixes used to reference the enum in the editor templates. */
  refPrefixes: string[]
  editors: string[]
  /** Types with no direct binding, each with the reason it is still editable. */
  exceptions: Record<string, string>
}

function categories(): Record<string, CategorySpec> {
  const m = manifest()
  return {
    inbounds: {
      rows: m.inbounds,
      enumPath: 'inbounds.ts',
      enumName: 'InTypes',
      refPrefixes: ['inTypes', 'InTypes'],
      editors: ['layouts/modals/Inbound.vue', 'components/nexus/drawers/InboundDrawer.vue'],
      exceptions: {
        // Legacy alias kept for migrated databases: the migration rewrites it, and
        // the inbound picker must not offer it as a user-selectable type.
        shadowsocks16: 'alias row: created only by migration, never chosen in the UI',
      },
    },
    outbounds: {
      rows: m.outbounds,
      enumPath: 'outbounds.ts',
      enumName: 'OutTypes',
      refPrefixes: ['outTypes', 'OutTypes'],
      editors: ['layouts/modals/Outbound.vue', 'components/nexus/drawers/OutboundDrawer.vue'],
      exceptions: {},
    },
    endpoints: {
      rows: m.endpoints,
      enumPath: 'endpoints.ts',
      enumName: 'EpTypes',
      refPrefixes: ['epTypes', 'EpTypes'],
      editors: ['layouts/modals/Endpoint.vue'],
      exceptions: {},
    },
    services: {
      rows: m.services,
      enumPath: 'services.ts',
      enumName: 'SrvTypes',
      refPrefixes: ['srvTypes', 'SrvTypes'],
      editors: ['layouts/modals/Service.vue', 'components/nexus/drawers/ServiceDrawer.vue'],
      exceptions: {
        // ResolvedServiceOptions is ListenOptions only, so the always-rendered
        // shared Listen section already covers every field it has.
        resolved: 'composed of the shared Listen section; the service has no other fields',
      },
    },
  }
}

describe('every manifest type is editable in the panel', () => {
  // Each editor component is checked on its own, not as a union: the panel ships
  // two editor surfaces (the nexus drawer and the legacy modal) and an operator on
  // either one must be able to edit every type.
  for (const [category, spec] of Object.entries(categories())) {
    for (const editor of spec.editors) {
      it(`binds every ${category} type in ${editor}`, () => {
        const members = enumMembers(spec.enumPath, spec.enumName)
        const source = editorSource([editor])
        const unbound: string[] = []

        for (const row of spec.rows) {
          if (row.alias) continue
          if (spec.exceptions[row.type]) continue
          const member = members[row.type]
          if (!member) {
            unbound.push(`${row.type} (no ${spec.enumName} member)`)
            continue
          }
          const referenced = spec.refPrefixes.some((prefix) =>
            new RegExp(`${prefix}\\.${member}\\b`).test(source),
          )
          if (!referenced) unbound.push(`${row.type} (${spec.enumName}.${member} unused)`)
        }

        expect(
          unbound,
          `${editor} cannot edit these ${category} types declared in protocols.json: ${unbound.join(', ')}. ` +
            'Add the editor binding, or add a documented exception with a reason.',
        ).toEqual([])
      })
    }
  }

  it('keeps the documented exceptions factually true', () => {
    const m = manifest()
    // shadowsocks16 must still be an alias row, otherwise the exception above would
    // hide a genuinely missing editor.
    const ss16 = m.inbounds.find((i) => i.type === 'shadowsocks16')
    expect(ss16?.alias, 'shadowsocks16 must stay an alias row').toBe(true)

    // resolved must still be renderable by the shared Listen section: if it ever
    // gains fields, the exception has to be revisited.
    expect(sourceOf('services.ts')).toMatch(/export interface Resolved extends SrvBasics \{\}/)
  })
})