<template>
  <div
    class="nexus-area-series"
    :aria-label="ariaLabel"
    :role="ariaLabel ? 'img' : undefined"
  >
    <Line v-if="hasValues" :data="chartData" :options="chartOptions" />
    <div v-else class="nexus-area-series__empty" aria-hidden="true" />
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
  type TooltipModel,
} from 'chart.js'

type AreaPoint = number | null

interface Series {
  label: string
  values: AreaPoint[]
}

interface SeriesColor {
  token: string
  fallback: string
}

const props = withDefaults(defineProps<{
  labels?: string[]
  series?: Series[]
  ariaLabel?: string
  valueFormatter?: (value: number) => string
}>(), {
  labels: () => [],
  series: () => [],
})

const SERIES_COLORS: SeriesColor[] = [
  { token: '--nexus-chart-1', fallback: '#36d0c4' },
  { token: '--nexus-chart-2', fallback: '#5ec8ff' },
  { token: '--nexus-chart-3', fallback: '#a78bfa' },
  { token: '--nexus-chart-4', fallback: '#fbbf24' },
  { token: '--nexus-chart-5', fallback: '#fb7185' },
  { token: '--nexus-chart-6', fallback: '#4ade80' },
]

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

function colorFor(index: number): SeriesColor {
  return SERIES_COLORS[index % SERIES_COLORS.length]
}

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

function areaGradient(
  context: ScriptableContext<'line'>,
  color: SeriesColor,
): CanvasGradient | string {
  const seriesColor = resolveToken(context.chart, color.token, color.fallback)
  const { chartArea, ctx } = context.chart

  if (!chartArea)
    return withOpacity(seriesColor, 0.22)

  const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
  gradient.addColorStop(0, withOpacity(seriesColor, 0.3))
  gradient.addColorStop(1, withOpacity(seriesColor, 0))
  return gradient
}

function formatValue(value: number): string {
  return props.valueFormatter?.(value) ?? value.toLocaleString()
}

function formatTickValue(value: string | number): string {
  const numberValue = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(numberValue) ? formatValue(numberValue) : String(value)
}

function tooltipValue(rawValue: unknown, fallback: string): string {
  return typeof rawValue === 'number' && Number.isFinite(rawValue)
    ? formatValue(rawValue)
    : fallback
}

function findTooltip(chart: Chart): HTMLElement | undefined {
  const host = chart.canvas.closest('.nexus-area-series')

  if (!(host instanceof HTMLElement))
    return

  let tooltip = host.querySelector<HTMLElement>('.nexus-area-series__tooltip')

  if (!tooltip) {
    tooltip = document.createElement('div')
    tooltip.className = 'nexus-area-series__tooltip'
    tooltip.hidden = true
    host.append(tooltip)
  }

  return tooltip
}

function renderTooltip(chart: Chart, model: TooltipModel<'line'>): void {
  const tooltip = findTooltip(chart)

  if (!tooltip)
    return

  if (model.opacity === 0) {
    tooltip.hidden = true
    return
  }

  tooltip.replaceChildren()

  if (model.title.length > 0) {
    const title = document.createElement('span')
    title.className = 'nexus-area-series__tooltip-title'
    title.textContent = model.title.join(' ')
    tooltip.append(title)
  }

  model.dataPoints.forEach(point => {
    const row = document.createElement('span')
    row.className = 'nexus-area-series__tooltip-row'

    const swatch = document.createElement('span')
    swatch.className = 'nexus-area-series__tooltip-swatch'
    swatch.style.backgroundColor = `var(${colorFor(point.datasetIndex).token})`
    row.append(swatch)

    const label = document.createElement('span')
    label.className = 'nexus-area-series__tooltip-label'
    label.textContent = point.dataset.label ?? ''
    row.append(label)

    const value = document.createElement('strong')
    value.className = 'nexus-area-series__tooltip-value'
    value.textContent = tooltipValue(point.raw, point.formattedValue)
    row.append(value)

    tooltip.append(row)
  })

  tooltip.hidden = false
  tooltip.style.left = `${chart.canvas.offsetLeft + model.caretX}px`
  tooltip.style.top = `${chart.canvas.offsetTop + model.caretY}px`
}

const labels = computed(() => {
  const pointCount = Math.max(
    props.labels.length,
    ...props.series.map(series => series.values.length),
  )

  return Array.from({ length: pointCount }, (_, index) => props.labels[index] ?? '')
})

const hasValues = computed(() => props.series.some(series => series.values.some(
  value => typeof value === 'number' && Number.isFinite(value),
)))

const chartData = computed<ChartData<'line', AreaPoint[], string>>(() => ({
  labels: labels.value,
  datasets: props.series.map((series, index) => {
    const color = colorFor(index)

    return {
      label: series.label,
      data: [...series.values],
      backgroundColor: context => areaGradient(context, color),
      borderColor: context => resolveToken(context.chart, color.token, color.fallback),
      borderWidth: 2,
      fill: true,
      pointBackgroundColor: context => resolveToken(context.chart, color.token, color.fallback),
      pointBorderWidth: 0,
      pointHoverRadius: 4,
      pointRadius: 0,
      tension: 0.34,
    }
  }),
}))

const chartOptions = computed<ChartOptions<'line'>>(() => ({
  animation: prefersReducedMotion.value ? false : { duration: 280 },
  interaction: {
    intersect: false,
    mode: 'index',
  },
  maintainAspectRatio: false,
  responsive: true,
  plugins: {
    tooltip: {
      enabled: false,
      external: context => renderTooltip(context.chart, context.tooltip),
    },
  },
  scales: {
    x: {
      border: {
        display: false,
      },
      grid: {
        display: false,
      },
      ticks: {
        maxRotation: 0,
      },
    },
    y: {
      beginAtZero: true,
      border: {
        display: false,
      },
      grid: {
        color: context => resolveToken(
          context.chart,
          '--nexus-border',
          'rgba(139, 205, 214, 0.18)',
        ),
      },
      ticks: {
        callback: value => formatTickValue(value),
        maxTicksLimit: 5,
      },
    },
  },
}))
</script>

<style scoped>
.nexus-area-series {
  block-size: 100%;
  min-block-size: 240px;
  min-inline-size: 0;
  position: relative;
}

.nexus-area-series :deep(canvas) {
  display: block;
}

.nexus-area-series__empty {
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--nexus-chart-1) 16%, transparent),
      transparent 72%
    );
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  block-size: 100%;
  min-block-size: inherit;
}

.nexus-area-series :deep(.nexus-area-series__tooltip) {
  background: color-mix(in srgb, var(--nexus-surface-2) 94%, transparent);
  border: 1px solid var(--nexus-border-strong);
  border-radius: var(--nexus-radius-sm);
  box-shadow: 0 12px 26px color-mix(in srgb, var(--nexus-surface-0) 44%, transparent);
  color: rgb(var(--v-theme-on-surface) / 92%);
  display: grid;
  font-size: 0.76rem;
  gap: var(--nexus-gap-1);
  inline-size: max-content;
  line-height: 1.3;
  max-inline-size: min(240px, calc(100% - var(--nexus-gap-3)));
  padding: var(--nexus-gap-2);
  pointer-events: none;
  position: absolute;
  transform: translate(-50%, calc(-100% - var(--nexus-gap-2)));
  z-index: 1;
}

.nexus-area-series :deep(.nexus-area-series__tooltip[hidden]) {
  display: none;
}

.nexus-area-series :deep(.nexus-area-series__tooltip-title) {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-weight: 600;
  letter-spacing: 0;
  overflow-wrap: anywhere;
}

.nexus-area-series :deep(.nexus-area-series__tooltip-row) {
  align-items: center;
  display: grid;
  gap: var(--nexus-gap-1);
  grid-template-columns: 8px minmax(0, 1fr) auto;
  letter-spacing: 0;
  min-inline-size: 0;
}

.nexus-area-series :deep(.nexus-area-series__tooltip-swatch) {
  block-size: 8px;
  border-radius: 50%;
  inline-size: 8px;
}

.nexus-area-series :deep(.nexus-area-series__tooltip-label) {
  min-inline-size: 0;
  overflow-wrap: anywhere;
}

.nexus-area-series :deep(.nexus-area-series__tooltip-value) {
  font-weight: 650;
  letter-spacing: 0;
}
</style>
