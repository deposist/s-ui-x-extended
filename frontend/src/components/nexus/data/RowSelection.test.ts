import { describe, expect, it } from 'vitest'

import { type RowKey, useRowSelection } from './RowSelection'

const page: RowKey[] = [1, 2, 3]

describe('useRowSelection', () => {
  it('starts empty', () => {
    const sel = useRowSelection(() => page)

    expect(sel.count.value).toBe(0)
    expect(sel.allSelected.value).toBe(false)
    expect(sel.indeterminate.value).toBe(false)
  })

  it('toggles a single key on and off', () => {
    const sel = useRowSelection(() => page)

    sel.toggle(2)
    expect(sel.isSelected(2)).toBe(true)
    expect(sel.selectedKeys.value).toEqual([2])

    sel.toggle(2)
    expect(sel.isSelected(2)).toBe(false)
    expect(sel.count.value).toBe(0)
  })

  it('is indeterminate on a partial page selection', () => {
    const sel = useRowSelection(() => page)

    sel.toggle(1)

    expect(sel.indeterminate.value).toBe(true)
    expect(sel.allSelected.value).toBe(false)
  })

  it('selects and clears the whole page with toggleAll', () => {
    const sel = useRowSelection(() => page)

    sel.toggleAll()
    expect(sel.allSelected.value).toBe(true)
    expect(sel.indeterminate.value).toBe(false)
    expect(sel.selectedKeys.value).toEqual([1, 2, 3])

    sel.toggleAll()
    expect(sel.count.value).toBe(0)
  })

  it('preserves off-page selections when toggling the page', () => {
    let keys: RowKey[] = [1, 2]
    const sel = useRowSelection(() => keys)

    sel.toggle(99) // off-page selection
    sel.toggleAll() // select page [1,2]
    expect(sel.selectedKeys.value.sort()).toEqual([1, 2, 99])

    keys = [1, 2]
    sel.toggleAll() // deselect page [1,2], keep 99
    expect(sel.selectedKeys.value).toEqual([99])
  })

  it('clear() removes every selection', () => {
    const sel = useRowSelection(() => page)

    sel.toggleAll()
    sel.clear()

    expect(sel.count.value).toBe(0)
    expect(sel.allSelected.value).toBe(false)
  })
})
