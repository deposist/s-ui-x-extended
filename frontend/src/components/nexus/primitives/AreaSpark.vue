<template>
  <div
    class="nexus-area-spark"
    :aria-label="ariaLabel"
    :role="ariaLabel ? 'img' : undefined"
  >
    <Line v-if="hasValues" :data="chartData" :options="chartOptions" />
    <div v-else class="nexus-area-spark__empty" aria-hidden="true" />
  </div>
</template>

<script lang="ts" setup>
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
} from 'vue'
import { Line } from 'vue-chartjs'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
  type Chart,
  type ChartData,
  type ChartOptions,
  type ScriptableContext,
} from 'chart.js'

type SparkPoint = number | null

const props = withDefaults(defineProps<{
  values?: SparkPoint[]
  labels?: string[]
  ariaLabel?: string
}>(), {
  values: () => [],
  labels: () => [],
})

ChartJS.register(
  Tooltip,
  Filler,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
)

let motionQuery: MediaQueryList | undefined
const prefersReducedMotion = ref(false)

function syncMotionPreference(event?: MediaQueryListEvent): void {
  prefersReducedMotion.value = event?.matches ?? motionQuery?.matches ?? false
}

if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  syncMotionPreference()
}

onMounted(() => {
  motionQuery?.addEventListener('change', syncMotionPreference)
})

onBeforeUnmount(() => {
  motionQuery?.removeEventListener('change', syncMotionPreference)
})

function resolveToken(chart: Chart, token: string, fallback: string): string {
  if (typeof window === 'undefined')
    return fallback

  return getComputedStyle(chart.canvas).getPropertyValue(token).trim() || fallback
}

function withOpacity(color: string, opacity: number): string {
  const hex = color.match(/^#([\da-f]{3}|[\da-f]{6})$/i)?.[1]

  if (!hex)
    return opacity === 0 ? 'transparent' : color

  const expandedHex = hex.length === 3
    ? hex.split('').map(value => `${value}${value}`).join('')
    : hex
  const red = Number.parseInt(expandedHex.slice(0, 2), 16)
  const green = Number.parseInt(expandedHex.slice(2, 4), 16)
  const blue = Number.parseInt(expandedHex.slice(4, 6), 16)

  return `rgba(${red}, ${green}, ${blue}, ${opacity})`
}

function areaGradient(context: ScriptableContext<'line'>): CanvasGradient | string {
  const color = resolveToken(context.chart, '--nexus-chart-1', '#36d0c4')
  const { chartArea, ctx } = context.chart

  if (!chartArea)
    return withOpacity(color, 0.26)

  const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
  gradient.addColorStop(0, withOpacity(color, 0.34))
  gradient.addColorStop(1, withOpacity(color, 0))
  return gradient
}

const labels = computed(() => Array.from(
  { length: props.values.length },
  (_, index) => props.labels[index] ?? '',
))

const hasValues = computed(() => props.values.some(
  value => typeof value === 'number' && Number.isFinite(value),
))

const chartData = computed<ChartData<'line', SparkPoint[], string>>(() => ({
  labels: labels.value,
  datasets: [
    {
      data: [...props.values],
      backgroundColor: areaGradient,
      borderColor: context => resolveToken(context.chart, '--nexus-chart-1', '#36d0c4'),
      borderWidth: 1.5,
      fill: true,
      pointHitRadius: 0,
      pointHoverRadius: 0,
      pointRadius: 0,
      tension: 0.36,
    },
  ],
}))

const chartOptions = computed<ChartOptions<'line'>>(() => ({
  animation: prefersReducedMotion.value ? false : { duration: 240 },
  maintainAspectRatio: false,
  responsive: true,
  plugins: {
    tooltip: {
      enabled: false,
    },
  },
  scales: {
    x: {
      display: false,
    },
    y: {
      beginAtZero: true,
      display: false,
    },
  },
}))
</script>

<style scoped>
.nexus-area-spark {
  block-size: 48px;
  min-block-size: 40px;
  min-inline-size: 0;
  position: relative;
}

.nexus-area-spark :deep(canvas) {
  display: block;
}

.nexus-area-spark__empty {
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--nexus-chart-1) 22%, transparent),
      transparent
    );
  border-block-end: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-sm);
  block-size: 100%;
}
</style>
