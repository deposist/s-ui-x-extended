<script lang="ts" setup>
import { HumanReadable } from '@/plugins/utils'
import { computed } from 'vue'

const props = defineProps({
  tilesData: <any>{},
  type: String
})

interface GaugeData {
  percent: number
  current: string
  currentUnit: string
  total: string
  totalUnit: string
  hasTotal: boolean
}

const emptyGauge = (): GaugeData => ({
  percent: 0,
  current: '-',
  currentUnit: '',
  total: '',
  totalUnit: '',
  hasTotal: false,
})

const data = computed<GaugeData>(() => {
  const d = props.tilesData
  if (!d.mem && !d.cpu) return emptyGauge()
  switch (props.type) {
    case 'g-cpu': {
      const value = Math.ceil(d.cpu)
      return { percent: d.cpu, current: String(value), currentUnit: '%', total: '', totalUnit: '', hasTotal: false }
    }
    case 'g-mem':
      return gaugeData(d.mem)
    case 'g-dsk':
      return gaugeData(d.dsk)
    case 'g-swp':
      return gaugeData(d.swp)
  }
  return emptyGauge()
})

const gaugeData = (d: any): GaugeData => {
  if (!d) return emptyGauge()
  const curr = HumanReadable.sizeFormat(d.current, 0).split(' ')
  const total = HumanReadable.sizeFormat(d.total, 0).split(' ')
  const currentUnit = curr[1] === total[1] ? '' : (curr[1] ?? '')
  return {
    percent: Math.ceil((d.current * 100) / d.total),
    current: curr[0] ?? '0',
    currentUnit,
    total: total[0] ?? '0',
    totalUnit: total[1] ?? '',
    hasTotal: true,
  }
}

const cssTransformRotateValue = computed(() => {
  const percentageAsFraction = data.value.percent / 100
  const halfPercentage = percentageAsFraction / 2

  return `${halfPercentage}turn`
})

const gaugeColor = computed(() => {
  if (data.value.percent > 90) return 'error'
  if (data.value.percent > 70) return 'warning'
  return 'info'
})
</script>

<template>
  <div class="gauge__outer">
    <div class="gauge__inner">
      <div
        class="gauge__fill"
        :style="{
          transform: `rotate(${cssTransformRotateValue})`,
          background: `rgb(var(--v-theme-${gaugeColor}))`
          }">
      </div>
      <div class="gauge__cover">
        <span dir="ltr">
          {{ data.current }}<sup>{{ data.currentUnit || '\u00a0' }}</sup>
          <template v-if="data.hasTotal">/{{ data.total }}<sup>{{ data.totalUnit || '\u00a0' }}</sup></template>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gauge__outer {
  width: 100%;
  max-width: 250px;
}

.gauge__inner {
  width: 100%;
  height: 0;
  padding-bottom: 50%;
  background: rgb(var(--v-theme-surface));
  position: relative;
  border-top-left-radius: 100% 200%;
  border-top-right-radius: 100% 200%;
  overflow: hidden;
}

.gauge__fill {
  position: absolute;
  top: 100%;
  left: 0;
  width: inherit;
  height: 100%;
  background: rgb(var(--v-theme-primary));
  transform-origin: center top;
  transform: rotate(0turn);
  transition: transform 0.2s ease-out;
}

.gauge__cover {
  width: 75%;
  height: 150%;
  background: rgb(var(--v-theme-background));
  position: absolute;
  top: 25%;
  left: 50%;
  transform: translateX(-50%);
  border-radius: 50%;

  /* Text */
  display: flex;
  align-items: center;
  justify-content: center;
  padding-bottom: 25%;
  box-sizing: border-box;
  font-family: 'Lexend', sans-serif;
  font-weight: bold;
  font-size: 32px;
}

sup {
  font-size: 16px;
}
</style>
