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
  timeZone?: unknown
}

export interface TrafficTimeZoneOption {
  label: string
  value: string
}

export type TrafficRange = '1h' | '6h' | '12h' | '24h' | '7d' | '30d'

type TrafficBucket = {
  download: number
  upload: number
}

const defaultRange: TrafficRange = '24h'
const defaultTimeZone = 'UTC'
export const trafficTimeZoneStorageKey = 'nexus-overview-traffic-timezone'

const curatedTrafficTimeZones = [
  'UTC',
  'Europe/London',
  'Europe/Berlin',
  'Europe/Moscow',
  'Asia/Dubai',
  'Asia/Kolkata',
  'Asia/Shanghai',
  'Asia/Tokyo',
  'Australia/Sydney',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
]

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

export const isValidTrafficTimeZone = (value: unknown): value is string => {
  if (typeof value !== 'string' || value.length === 0) return false

  try {
    new Intl.DateTimeFormat('en-US', { timeZone: value }).format(0)
    return true
  } catch {
    return false
  }
}

const browserTrafficTimeZone = (): string => {
  try {
    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone
    return isValidTrafficTimeZone(timeZone) ? timeZone : defaultTimeZone
  } catch {
    return defaultTimeZone
  }
}

export const resolveTrafficTimeZone = (value?: unknown): string => {
  return isValidTrafficTimeZone(value) ? value : browserTrafficTimeZone()
}

export const loadTrafficTimeZone = (storage: Storage | undefined = typeof localStorage === 'undefined' ? undefined : localStorage): string => {
  try {
    return resolveTrafficTimeZone(storage?.getItem(trafficTimeZoneStorageKey))
  } catch {
    return browserTrafficTimeZone()
  }
}

export const persistTrafficTimeZone = (
  timeZone: string,
  storage: Storage | undefined = typeof localStorage === 'undefined' ? undefined : localStorage,
): void => {
  if (!isValidTrafficTimeZone(timeZone)) return

  try {
    storage?.setItem(trafficTimeZoneStorageKey, timeZone)
  } catch {
    // Ignore storage failures; the selected timezone still applies in memory.
  }
}

const trafficTimeZoneOffsetMinutes = (timeZone: string, atMs = Date.now()): number => {
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).formatToParts(new Date(atMs))

    const part = (type: Intl.DateTimeFormatPartTypes) => Number(parts.find(item => item.type === type)?.value)
    const hour = part('hour') % 24
    const asUtc = Date.UTC(part('year'), part('month') - 1, part('day'), hour, part('minute'), part('second'))
    const offset = Math.round((asUtc - atMs) / 60000)

    return Number.isFinite(offset) ? offset : 0
  } catch {
    return 0
  }
}

const trafficTimeZoneOffsetLabel = (timeZone: string): string => {
  const offset = trafficTimeZoneOffsetMinutes(timeZone)
  if (offset === 0) return 'UTC +0'

  const sign = offset > 0 ? '+' : '-'
  const absolute = Math.abs(offset)
  const hours = Math.floor(absolute / 60)
  const minutes = absolute % 60

  return `UTC ${sign}${minutes === 0 ? hours : `${hours}:${String(minutes).padStart(2, '0')}`}`
}

const supportedTrafficTimeZones = (): string[] => {
  try {
    return typeof Intl.supportedValuesOf === 'function'
      ? Intl.supportedValuesOf('timeZone')
      : []
  } catch {
    return []
  }
}

const defaultTrafficTimeZoneValues = (selectedTimeZone?: string): unknown[] => [
  defaultTimeZone,
  browserTrafficTimeZone(),
  selectedTimeZone,
  ...curatedTrafficTimeZones,
]

const uniqueValidTrafficTimeZones = (timeZones: readonly unknown[]): string[] => {
  const seen = new Set<string>()
  const values: string[] = []

  for (const timeZone of timeZones) {
    if (!isValidTrafficTimeZone(timeZone) || seen.has(timeZone)) continue

    seen.add(timeZone)
    values.push(timeZone)
  }

  return values
}

const createTrafficTimeZoneOption = (value: string): TrafficTimeZoneOption => ({
  value,
  label: `${value} (${trafficTimeZoneOffsetLabel(value)})`,
})

export const trafficTimeZoneOptions = (
  selectedTimeZone?: unknown,
  searchQuery?: unknown,
): TrafficTimeZoneOption[] => {
  const query = typeof searchQuery === 'string' ? searchQuery.trim().toLowerCase() : ''
  const selected = isValidTrafficTimeZone(selectedTimeZone) ? selectedTimeZone : undefined
  const supportedTimeZones = query.length > 0 ? supportedTrafficTimeZones() : []
  const source = supportedTimeZones.length > 0 ? supportedTimeZones : defaultTrafficTimeZoneValues(selected)
  const matchingSource = query.length > 0
    ? source.filter(timeZone => typeof timeZone === 'string' && timeZone.toLowerCase().includes(query))
    : source

  return uniqueValidTrafficTimeZones(matchingSource).map(createTrafficTimeZoneOption)
}

export const formatTrafficLabel = (dateTime: number, timeZone?: unknown): string => {
  const date = new Date(dateTime * 1000)
  if (Number.isNaN(date.getTime())) return String(dateTime)

  const resolvedTimeZone = resolveTrafficTimeZone(timeZone)
  try {
    const parts = new Intl.DateTimeFormat('en-CA', {
      timeZone: resolvedTimeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    }).formatToParts(date)

    const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(item => item.type === type)?.value ?? '00'

    return `${part('year')}-${part('month')}-${part('day')} ${part('hour')}:${part('minute')}`
  } catch {
    return date.toISOString()
  }
}

const summaryBucketStart = (bucket: unknown, fallback: number): number => {
  return isSelectorRecord(bucket) ? nonNegativeNumber(bucket.startTime) ?? fallback : fallback
}

const summaryBucketTraffic = (bucket: unknown, key: 'download' | 'upload'): number => {
  return isSelectorRecord(bucket) ? nonNegativeNumber(bucket[key]) ?? 0 : 0
}

export const selectTrafficSeries = (input?: TrafficSelectorInput | null): TrafficSeries => {
  const range = isTrafficRange(input?.range) ? input.range : defaultRange
  const timeZone = resolveTrafficTimeZone(input?.timeZone)
  const summary = isSelectorRecord(input?.summary) ? input.summary : undefined
  const summaryBuckets = Array.isArray(summary?.buckets) ? summary.buckets : undefined

  if (summaryBuckets?.length) {
    return {
      labels: summaryBuckets.map((bucket, index) => formatTrafficLabel(summaryBucketStart(bucket, index), timeZone)),
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
      labels: buckets.map((_, index) => formatTrafficLabel(startSec + (index * bucketSpanSec), timeZone)),
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
    labels: dateTimes.map(dateTime => formatTrafficLabel(dateTime, timeZone)),
    download: dateTimes.map((dateTime) => buckets.get(dateTime)?.download ?? 0),
    upload: dateTimes.map((dateTime) => buckets.get(dateTime)?.upload ?? 0),
    range,
  }
}
