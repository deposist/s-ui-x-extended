<template>
  <thead class="nexus-thead">
    <tr>
      <th v-if="selectable" class="nexus-thead__select">
        <v-checkbox-btn
          :aria-label="$t('table.selectAll')"
          density="compact"
          hide-details
          :indeterminate="indeterminate"
          :model-value="allSelected"
          @update:model-value="emit('toggle-all')"
        />
      </th>

      <th v-if="expandable" class="nexus-thead__expand" />

      <th
        v-for="column in columns"
        :key="column.key"
        :aria-sort="ariaSort(column)"
        :class="`nexus-thead__cell nexus-thead__cell--${column.align ?? 'start'}`"
        :data-column-key="column.key"
        :style="column.width ? { width: column.width } : undefined"
      >
        <button
          v-if="column.sortable"
          class="nexus-thead__sort"
          type="button"
          @click="emit('sort', column.key)"
        >
          <span>{{ labelFor(column) }}</span>
          <span aria-hidden="true" :class="arrowClass(column)" class="nexus-thead__arrow" />
        </button>
        <span v-else>{{ labelFor(column) }}</span>
      </th>

      <th v-if="hasActions" class="nexus-thead__actions" data-column-key="actions">{{ $t('actions.action') }}</th>
    </tr>
  </thead>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'

import type { Column, SortState } from './dataTableColumns'

const props = defineProps<{
  columns: Column[]
  sort: SortState | null
  selectable?: boolean
  expandable?: boolean
  hasActions?: boolean
  allSelected?: boolean
  indeterminate?: boolean
}>()

const emit = defineEmits<{
  sort: [key: string]
  'toggle-all': []
}>()

const { t } = useI18n()

const isSorted = (column: Column) => props.sort?.key === column.key

const labelFor = (column: Column): string => {
  if (column.label) return column.label
  if (column.labelKey) return t(column.labelKey)

  return column.key
}

const ariaSort = (column: Column): 'ascending' | 'descending' | 'none' | undefined => {
  if (!column.sortable) return undefined
  if (!isSorted(column)) return 'none'

  return props.sort?.direction === 'asc' ? 'ascending' : 'descending'
}

const arrowClass = (column: Column) => ({
  'nexus-thead__arrow--asc': isSorted(column) && props.sort?.direction === 'asc',
  'nexus-thead__arrow--desc': isSorted(column) && props.sort?.direction === 'desc',
})
</script>

<style scoped>
.nexus-thead {
  background: var(--nexus-surface-1);
}

.nexus-thead__cell,
.nexus-thead__select,
.nexus-thead__expand,
.nexus-thead__actions {
  border-block-end: 1px solid var(--nexus-border);
  color: var(--nexus-text-muted);
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.5px;
  padding: var(--nexus-gap-3);
  text-align: start;
  text-transform: uppercase;
}

.nexus-thead__cell--center { text-align: center; }
.nexus-thead__cell--end { text-align: end; }
.nexus-thead__actions { text-align: end; }
.nexus-thead__select,
.nexus-thead__expand { width: 44px; }

.nexus-thead__sort {
  align-items: center;
  /* Reset native <button> chrome — otherwise the OS default button background
   * shows as grey chips behind sortable column labels (tag/type/port). */
  background: none;
  border: 0;
  color: inherit;
  cursor: pointer;
  display: inline-flex;
  font: inherit;
  gap: var(--nexus-gap-1);
  letter-spacing: inherit;
  padding: 0;
  text-transform: inherit;
  transition: color var(--nexus-transition-fast);
}

.nexus-thead__sort:hover {
  color: var(--nexus-text-secondary);
}

.nexus-thead__arrow {
  border-inline: 4px solid transparent;
  height: 0;
  opacity: 0;
  width: 0;
}

.nexus-thead__arrow--asc {
  border-block-end: 5px solid var(--nexus-accent-primary);
  opacity: 1;
}

.nexus-thead__arrow--desc {
  border-block-start: 5px solid var(--nexus-accent-primary);
  opacity: 1;
}
</style>
