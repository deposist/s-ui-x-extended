<template>
  <span aria-hidden="true" class="nexus-page-header-anchor" />
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, watch } from 'vue'

import { clearPageHeader, setPageHeader, topbarSearch } from './pageHeaderPortal'

// Controller component: renders nothing visible. It publishes the section header
// (title + stats subtitle + searchable flag) into shared state so the topbar can
// render it, and relays the topbar's search box back to the view via update:search.
const props = withDefaults(defineProps<{
  title: string
  subtitle?: string
  search?: string
  searchable?: boolean
  debounce?: number
}>(), {
  searchable: false,
  search: '',
  debounce: 250,
})

const emit = defineEmits<{
  'update:search': [value: string]
}>()

const id = Symbol('page-header')

watch(
  () => [props.title, props.subtitle, props.searchable] as const,
  () => setPageHeader(id, { title: props.title, subtitle: props.subtitle, searchable: props.searchable }),
  { immediate: true },
)

let timer: ReturnType<typeof setTimeout> | undefined

watch(topbarSearch, (value) => {
  clearTimeout(timer)
  timer = setTimeout(() => emit('update:search', value), props.debounce)
})

onMounted(() => {
  topbarSearch.value = props.search ?? ''
})

onBeforeUnmount(() => {
  clearTimeout(timer)
  clearPageHeader(id)
})
</script>

<style scoped>
.nexus-page-header-anchor {
  display: none;
}
</style>
