import { describe, expect, it } from 'vitest'

import { type Column, nextSortState, sortItems } from './dataTableColumns'

interface Row extends Record<string, unknown> {
  tag: string
  port: number
  online: boolean | null
}

const rows: Row[] = [
  { tag: 'bravo', port: 443, online: true },
  { tag: 'alpha', port: 8443, online: false },
  { tag: 'charlie', port: 80, online: null },
]

describe('sortItems', () => {
  it('returns an untouched copy when sort is null', () => {
    const result = sortItems(rows, null)

    expect(result).toEqual(rows)
    expect(result).not.toBe(rows)
  })

  it('does not mutate the input array', () => {
    const snapshot = [...rows]

    sortItems(rows, { key: 'tag', direction: 'asc' })

    expect(rows).toEqual(snapshot)
  })

  it('sorts strings ascending and descending', () => {
    const asc = sortItems(rows, { key: 'tag', direction: 'asc' }).map(row => row.tag)
    const desc = sortItems(rows, { key: 'tag', direction: 'desc' }).map(row => row.tag)

    expect(asc).toEqual(['alpha', 'bravo', 'charlie'])
    expect(desc).toEqual(['charlie', 'bravo', 'alpha'])
  })

  it('sorts numbers numerically, not lexicographically', () => {
    const asc = sortItems(rows, { key: 'port', direction: 'asc' }).map(row => row.port)

    expect(asc).toEqual([80, 443, 8443])
  })

  it('orders null values first ascending', () => {
    const asc = sortItems(rows, { key: 'online', direction: 'asc' }).map(row => row.online)

    expect(asc[0]).toBeNull()
  })

  it('uses a column sortValue accessor when provided', () => {
    const columns: Column<Row>[] = [
      { key: 'clients', labelKey: 'x', sortValue: row => row.tag.length },
    ]

    const asc = sortItems(rows, { key: 'clients', direction: 'asc' }, columns).map(row => row.tag)

    expect(asc).toEqual(['bravo', 'alpha', 'charlie'])
  })
})

describe('nextSortState', () => {
  it('starts ascending for a new column', () => {
    expect(nextSortState(null, 'tag')).toEqual({ key: 'tag', direction: 'asc' })
    expect(nextSortState({ key: 'port', direction: 'desc' }, 'tag')).toEqual({ key: 'tag', direction: 'asc' })
  })

  it('cycles asc -> desc -> cleared on the same column', () => {
    expect(nextSortState({ key: 'tag', direction: 'asc' }, 'tag')).toEqual({ key: 'tag', direction: 'desc' })
    expect(nextSortState({ key: 'tag', direction: 'desc' }, 'tag')).toBeNull()
  })
})
