import { isSelectorRecord, nonNegativeNumber } from './selectorUtils'

export interface TrafficSeries {
  labels: string[]
  download: number[]
  upload: number[]
  range: '24h' | '7d' | '30d' | 'realtime'
}

export interface TrafficSelectorInput {
  range?: unknown
  stats?: readonly unknown[] | null
}

type TrafficRange = TrafficSeries['range']

type TrafficBucket = {
  download: number
  upload: number
}

const defaultRange: TrafficRange = '24h'

const isTrafficRange = (value: unknown): value is TrafficRange => {
  return value === '24h' || value === '7d' || value === '30d' || value === 'realtime'
}

const trafficLabel = (dateTime: number): string => {
  const date = new Date(dateTime * 1000)
  return Number.isNaN(date.getTime()) ? String(dateTime) : date.toISOString()
}

export const selectTrafficSeries = (input?: TrafficSelectorInput | null): TrafficSeries => {
  const buckets = new Map<number, TrafficBucket>()

  for (const stat of input?.stats ?? []) {
    if (!isSelectorRecord(stat) || typeof stat.direction !== 'boolean') continue

    const dateTime = nonNegativeNumber(stat.dateTime)
    const traffic = nonNegativeNumber(stat.traffic)
    if (dateTime === undefined || traffic === undefined) continue

    const bucket = buckets.get(dateTime) ?? { download: 0, upload: 0 }
    if (stat.direction) bucket.upload += traffic
    else bucket.download += traffic
    buckets.set(dateTime, bucket)
  }

  const dateTimes = [...buckets.keys()].sort((left, right) => left - right)

  return {
    labels: dateTimes.map(trafficLabel),
    download: dateTimes.map((dateTime) => buckets.get(dateTime)?.download ?? 0),
    upload: dateTimes.map((dateTime) => buckets.get(dateTime)?.upload ?? 0),
    range: isTrafficRange(input?.range) ? input.range : defaultRange,
  }
}
