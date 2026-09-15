import { i18n } from '@/locales'

// One row of the /api/capabilities response that tells whether a type can run in
// this binary: not compiled in (build tag absent) or not implemented on this
// platform (e.g. REDIRECT/TPROXY outside linux). The editors keep every type
// visible but disabled with the reason, so an unusable entry is explained rather
// than silently missing.
export type CapabilityRow = {
  type: string
  available: boolean
  platforms?: string[]
}

export type TypeItem = {
  title: string
  value: string
  props?: { disabled: boolean }
}

/** Normalize the optional capabilities response into typed rows. */
export function capabilityRows(entries: unknown): CapabilityRow[] {
  if (!Array.isArray(entries)) return []
  const rows: CapabilityRow[] = []
  for (const entry of entries) {
    if (!entry || typeof entry !== 'object') continue
    const row = entry as { type?: unknown; available?: unknown; platforms?: unknown }
    if (typeof row.type !== 'string') continue
    rows.push({
      type: row.type,
      available: row.available !== false,
      platforms: Array.isArray(row.platforms) ? row.platforms.filter((p): p is string => typeof p === 'string') : [],
    })
  }
  return rows
}

/** The reason a type is unusable here, or an empty string when it is usable. */
export function unavailableReason(rows: CapabilityRow[], type: string): string {
  const row = rows.find((r) => r.type === type)
  if (!row || row.available) return ''
  if (row.platforms && row.platforms.length > 0) {
    return i18n.global.t('capability.platformOnly', { platforms: row.platforms.join('/') })
  }
  return i18n.global.t('capability.notInBuild')
}

/**
 * v-select items for a type-map (`OutTypes`/`InTypes`/`EpTypes`/`SrvTypes`) where
 * unusable types stay listed, disabled, with the reason appended to the label.
 */
export function capabilityTypeItems(types: Record<string, string>, rows: CapabilityRow[]): TypeItem[] {
  return Object.entries(types).map(([label, value]) => {
    const reason = unavailableReason(rows, value)
    if (!reason) return { title: label, value }
    return { title: `${label} — ${reason}`, value, props: { disabled: true } }
  })
}