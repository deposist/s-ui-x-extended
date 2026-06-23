import { reactive, ref } from 'vue'

// Per user spec the section header (name + stats + search) lives in the topbar.
// Rather than teleport from each view's PageHeader (fragile across route changes —
// the slot can end up empty when mount/unmount order races), the active view
// publishes its header into this shared reactive state and the topbar renders it.

// The search box is rendered by the topbar; views read this ref (via PageHeader's
// update:search) for filtering.
export const topbarSearch = ref('')

export const pageHeader = reactive<{
  title: string
  subtitle: string
  searchable: boolean
  active: boolean
}>({ title: '', subtitle: '', searchable: false, active: false })

// Owner token: only the header that currently owns the topbar may clear it, so a
// freshly-mounted page header isn't wiped by the previous page's unmount (the
// router does not guarantee new-mounts-before-old-unmounts).
let owner: symbol | null = null

export const setPageHeader = (
  id: symbol,
  next: { title: string; subtitle?: string; searchable?: boolean },
): void => {
  owner = id
  pageHeader.title = next.title
  pageHeader.subtitle = next.subtitle ?? ''
  pageHeader.searchable = next.searchable ?? false
  pageHeader.active = true
}

export const clearPageHeader = (id: symbol): void => {
  if (owner !== id) return

  owner = null
  pageHeader.active = false
  pageHeader.title = ''
  pageHeader.subtitle = ''
  pageHeader.searchable = false
  topbarSearch.value = ''
}
