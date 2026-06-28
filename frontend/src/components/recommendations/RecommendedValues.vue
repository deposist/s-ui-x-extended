<template>
  <v-alert
    v-if="!hideWhenEmpty || resolved.length > 0"
    class="recommended-values"
    density="compact"
    type="info"
    variant="tonal"
  >
    <div class="recommended-values__header">
      <strong>{{ title || tOrFallback('singbox.recommended', 'Recommended values') }}</strong>
      <v-btn
        v-if="showApplyAll && applicable.length > 1"
        class="ms-auto"
        color="primary"
        size="small"
        variant="tonal"
        @click="emit('apply-all', applicable)"
      >
        {{ applyLabel }}
      </v-btn>
    </div>

    <div v-if="resolved.length === 0" class="text-medium-emphasis text-caption mt-1">
      {{ emptyText }}
    </div>

    <v-list v-else bg-color="transparent" density="compact" class="recommended-values__list">
      <v-list-item
        v-for="item in resolved"
        :key="item.id || pathKey(item.path)"
        class="px-0"
      >
        <v-list-item-title>{{ item.label }}</v-list-item-title>
        <v-list-item-subtitle v-if="item.description">
          {{ item.description }}
        </v-list-item-subtitle>
        <div class="recommended-values__value text-caption mt-1">
          {{ recommendedLabel }}: <code>{{ formatValue(item.recommendedValue) }}</code>
        </div>
        <template #append>
          <v-btn
            color="primary"
            size="small"
            variant="text"
            :disabled="!item.applicable"
            @click="emit('apply', item)"
          >
            {{ applyLabel }}
          </v-btn>
        </template>
      </v-list-item>
    </v-list>
  </v-alert>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  pathToArray,
  resolveRecommendations,
  type RecommendationContext,
  type RecommendationPath,
  type RecommendationSpec,
  type ResolvedRecommendation,
} from '@/utils/recommendations'

const props = withDefaults(defineProps<{
  model: Record<string, unknown>
  specs: RecommendationSpec<Record<string, unknown>>[]
  context?: Partial<RecommendationContext<Record<string, unknown>>>
  title?: string
  applyText?: string
  emptyText?: string
  showApplyAll?: boolean
  hideWhenEmpty?: boolean
}>(), {
  showApplyAll: false,
  hideWhenEmpty: true,
  emptyText: 'No recommendations available.',
})

const emit = defineEmits<{
  apply: [spec: ResolvedRecommendation<Record<string, unknown>>]
  'apply-all': [specs: ResolvedRecommendation<Record<string, unknown>>[]]
}>()

const { t, te } = useI18n()

const recommendationContext = computed<RecommendationContext<Record<string, unknown>>>(() => ({
  ...props.context,
  model: props.model,
}))

const resolved = computed(() => resolveRecommendations(props.specs, recommendationContext.value))
const applicable = computed(() => resolved.value.filter((item) => item.applicable))

const applyLabel = computed(() => props.applyText || tFirst(['actions.apply', 'actions.set'], 'Apply'))
const recommendedLabel = computed(() => tFirst(['singbox.recommendedValue', 'singbox.recommended'], 'Recommended'))

function tOrFallback(key: string, fallback: string): string {
  return te(key) ? String(t(key)) : fallback
}

function tFirst(keys: string[], fallback: string): string {
  const key = keys.find((candidate) => te(candidate))
  return key ? String(t(key)) : fallback
}

function pathKey(path: RecommendationPath): string {
  return pathToArray(path).join('.')
}

function formatValue(value: unknown): string {
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (value == null) return ''
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}
</script>

<style scoped>
.recommended-values__header {
  align-items: center;
  display: flex;
  gap: 8px;
}

.recommended-values__list {
  padding-block: 4px 0;
}

.recommended-values__value {
  color: rgba(var(--v-theme-on-surface), .7);
  white-space: normal;
  word-break: break-word;
}
</style>
