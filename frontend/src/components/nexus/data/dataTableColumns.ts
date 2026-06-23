export type SortDirection = 'asc' | 'desc'

export type SortableValue = string | number | boolean | null | undefined

export interface Column<T = Record<string, unknown>> {
  key: string
  // i18n key the table resolves for the header label.
  labelKey?: string
  // Literal label for headers that should not go through i18n.
  label?: string
  sortable?: boolean
  align?: 'start' | 'center' | 'end'
  width?: string
  // Accessor used for sorting when the sort value is not a plain item[key].
  sortValue?: (item: T) => SortableValue
}

export interface SortState {
  key: string
  direction: SortDirection
}

const compareValues = (a: SortableValue, b: SortableValue): number => {
  if (a == null && b == null) return 0
  if (a == null) return -1
  if (b == null) return 1
  if (typeof a === 'number' && typeof b === 'number') return a - b

  return String(a).localeCompare(String(b), undefined, {
    numeric: true,
    sensitivity: 'base',
  })
}

// Pure, side-effect-free sort: returns a new array and never mutates the input.
// A null sort returns an untouched copy so callers can rely on referential change.
export const sortItems = <T extends Record<string, unknown>>(
  items: readonly T[],
  sort: SortState | null,
  columns: readonly Column<T>[] = [],
): T[] => {
  const result = [...items]

  if (!sort) return result

  const column = columns.find(candidate => candidate.key === sort.key)
  const accessor = column?.sortValue ?? ((item: T) => item[sort.key] as SortableValue)
  const factor = sort.direction === 'asc' ? 1 : -1

  return result.sort((a, b) => factor * compareValues(accessor(a), accessor(b)))
}

// Cycle a header: unsorted -> asc -> desc -> unsorted (returns the next state).
export const nextSortState = (current: SortState | null, key: string): SortState | null => {
  if (!current || current.key !== key) return { key, direction: 'asc' }
  if (current.direction === 'asc') return { key, direction: 'desc' }

  return null
}
