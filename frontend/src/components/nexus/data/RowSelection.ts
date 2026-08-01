import { computed, ref, watch } from 'vue'

export type RowKey = string | number

// Headless row-selection state for NexusDataTable. Pure data structure (a Set)
// with no DOM access, so it unit-tests without a component harness. `pageKeys`
// is a getter so select-all/indeterminate always reflect the current page.
export const useRowSelection = (pageKeys: () => readonly RowKey[]) => {
  const selected = ref<Set<RowKey>>(new Set())

  const isSelected = (key: RowKey): boolean => selected.value.has(key)

  const replace = (next: Set<RowKey>): void => {
    selected.value = next
  }
  watch(() => pageKeys(), keys => {
    const visible = new Set(keys)
    const next = new Set([...selected.value].filter(key => visible.has(key)))
    if (next.size !== selected.value.size) replace(next)
  })

  const toggle = (key: RowKey): void => {
    const next = new Set(selected.value)

    if (next.has(key)) next.delete(key)
    else next.add(key)

    replace(next)
  }

  const clear = (): void => replace(new Set())

  const selectedKeys = computed<RowKey[]>(() => [...selected.value])
  const count = computed<number>(() => selected.value.size)

  const allSelected = computed<boolean>(() => {
    const keys = pageKeys()

    return keys.length > 0 && keys.every(key => selected.value.has(key))
  })

  const indeterminate = computed<boolean>(() => {
    const keys = pageKeys()
    return keys.some(key => selected.value.has(key)) && !allSelected.value
  })

  // Selection is scoped to the currently visible page; changing page/filter
  // prunes keys that can no longer be acted on.
  const toggleAll = (): void => {
    const keys = pageKeys()
    const next = new Set(selected.value)

    if (allSelected.value) keys.forEach(key => next.delete(key))
    else keys.forEach(key => next.add(key))

    replace(next)
  }

  return {
    selectedKeys,
    count,
    allSelected,
    indeterminate,
    isSelected,
    toggle,
    toggleAll,
    clear,
  }
}
