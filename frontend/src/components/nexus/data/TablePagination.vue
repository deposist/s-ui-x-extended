<template>
  <div class="nexus-pagination">
    <div class="nexus-pagination__per-page">
      <span class="nexus-pagination__label">{{ $t('table.rowsPerPage') }}</span>
      <v-select
        density="compact"
        hide-details
        :items="perPageOptions"
        :model-value="itemsPerPage"
        variant="outlined"
        @update:model-value="emit('update:itemsPerPage', $event)"
      />
    </div>

    <span class="nexus-pagination__range">
      {{ $t('table.showingRange', { from, to, total }) }}
    </span>

    <div class="nexus-pagination__nav">
      <v-btn
        :aria-label="$t('audit.previous')"
        density="comfortable"
        :disabled="page <= 1"
        icon="lucide:chevron-left"
        size="small"
        variant="text"
        @click="emit('update:page', page - 1)"
      />
      <v-btn
        :aria-label="$t('audit.next')"
        density="comfortable"
        :disabled="page >= pageCount"
        icon="lucide:chevron-right"
        size="small"
        variant="text"
        @click="emit('update:page', page + 1)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{
  total: number
  page: number
  itemsPerPage: number
}>()

const emit = defineEmits<{
  'update:page': [value: number]
  'update:itemsPerPage': [value: number]
}>()

// Always include the active value so a persisted size outside the preset list
// (an older "items-per-page") still selects cleanly instead of blanking.
const perPageOptions = computed(() =>
  [...new Set([10, 25, 50, 100, props.itemsPerPage])].sort((a, b) => a - b),
)

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.itemsPerPage)))
const from = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.itemsPerPage + 1))
const to = computed(() => Math.min(props.page * props.itemsPerPage, props.total))
</script>

<style scoped>
.nexus-pagination {
  align-items: center;
  color: var(--nexus-text-secondary);
  display: flex;
  flex-wrap: wrap;
  font-size: 0.8rem;
  gap: var(--nexus-gap-4);
  justify-content: flex-end;
  padding: var(--nexus-gap-2) var(--nexus-gap-3);
}

.nexus-pagination__per-page {
  align-items: center;
  display: flex;
  gap: var(--nexus-gap-2);
}

.nexus-pagination__per-page :deep(.v-field) {
  min-width: 84px;
}

.nexus-pagination__nav {
  display: flex;
  gap: var(--nexus-gap-1);
}
</style>
