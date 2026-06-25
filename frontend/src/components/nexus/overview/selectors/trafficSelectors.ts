import { isSelectorRecord, nonNegativeNumber } from './selectorUtils'

export interface TrafficSeries {
  labels: string[]
  download: number[]
  upload: number[]
  range: TrafficRange
}

export interface TrafficSelectorInput {
  bucketCount?: number
  nowMs?: number
  range?: unknown
  stats?: readonly unknown[] | null
  summary?: unknown
}

export type TrafficRange = '1h' | '6h' | '12h' | '24h' | '7d' | '30d'

type TrafficBucket = {
  download: number
  upload: number
}

const defaultRange: TrafficRange = '24h'

export const trafficRangeHours: Record<TrafficRange, number> = {
  '1h': 1,
  '6h': 6,
  '12h': 12,
  '24h': 24,
  '7d': 24 * 7,
  '30d': 24 * 30,
}

const isTrafficRange = (value: unknown): value is TrafficRange => {
  return value === '1h'
    || value === '6h'
    || value === '12h'
    || value === '24h'
    || value === '7d'
    || value === '30d'
}

const trafficLabel = (dateTime: number): string => {
  const date = new Date(dateTime * 1000)
  return Number.isNaN(date.getTime()) ? String(dateTime) : date.toISOString()
}

const summaryBucketStart = (bucket: unknown, fallback: number): number => {
  return isSelectorRecord(bucket) ? nonNegativeNumber(bucket.startTime) ?? fallback : fallback
}

const summaryBucketTraffic = (bucket: unknown, key: 'download' | 'upload'): number => {
  return isSelectorRecord(bucket) ? nonNegativeNumber(bucket[key]) ?? 0 : 0
}

export const selectTrafficSeries = (input?: TrafficSelectorInput | null): TrafficSeries => {
  const range = isTrafficRange(input?.range) ? input.range : defaultRange
  const summary = isSelectorRecord(input?.summary) ? input.summary : undefined
  const summaryBuckets = Array.isArray(summary?.buckets) ? summary.buckets : undefined

  if (summaryBuckets?.length) {
    return {
      labels: summaryBuckets.map((bucket, index) => trafficLabel(summaryBucketStart(bucket, index))),
      download: summaryBuckets.map(bucket => summaryBucketTraffic(bucket, 'download')),
      upload: summaryBuckets.map(bucket => summaryBucketTraffic(bucket, 'upload')),
      range,
    }
  }

  const bucketCount = typeof input?.bucketCount === 'number' && Number.isFinite(input.bucketCount)
    ? Math.max(1, Math.floor(input.bucketCount))
    : undefined

  if (bucketCount !== undefined) {
    const nowMs = typeof input?.nowMs === 'number' && Number.isFinite(input.nowMs)
      ? input.nowMs
      : Date.now()
    const endSec = Math.floor(nowMs / 1000)
    const startSec = endSec - (trafficRangeHours[range] * 3600)
    const bucketSpanSec = Math.max(1, Math.ceil((endSec - startSec) / bucketCount))
    const buckets: TrafficBucket[] = Array.from(
      { length: bucketCount },
      (): TrafficBucket => ({ download: 0, upload: 0 }),
    )
    let hasStats = false

    for (const stat of input?.stats ?? []) {
      if (!isSelectorRecord(stat) || typeof stat.direction !== 'boolean') continue

      const dateTime = nonNegativeNumber(stat.dateTime)
      const traffic = nonNegativeNumber(stat.traffic)
      if (dateTime === undefined || traffic === undefined) continue
      if (dateTime < startSec || dateTime > endSec) continue

      const bucketIndex = Math.min(
        bucketCount - 1,
        Math.max(0, Math.floor((dateTime - startSec) / bucketSpanSec)),
      )
      const bucket = buckets[bucketIndex]
      if (!bucket) continue

      if (stat.direction) bucket.upload += traffic
      else bucket.download += traffic
      hasStats = true
    }

    if (!hasStats) {
      return {
        labels: [],
        download: [],
        upload: [],
        range,
      }
    }

    return {
      labels: buckets.map((_, index) => trafficLabel(startSec + (index * bucketSpanSec))),
      download: buckets.map(bucket => bucket.download),
      upload: buckets.map(bucket => bucket.upload),
      range,
    }
  }

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
    range,
  }
}
