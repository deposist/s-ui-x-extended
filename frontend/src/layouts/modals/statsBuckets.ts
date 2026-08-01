export interface StatsBucketInput {
  dateTime: number
  direction: unknown
  traffic: number
}

export interface StatsBucketSeries {
  labelSteps: number[]
  uplinkData: Array<number | null>
  downlinkData: Array<number | null>
}

/**
 * Aggregate the historical stats graph in O(rows + buckets) time.
 *
 * The API's legacy chart contract intentionally uses open intervals, emits
 * bucketCount - 1 buckets, and omits the final interval ending at `nowMs`.
 * Keep those details here so the optimized pass remains byte-for-byte
 * compatible with the former per-bucket filter/reduce implementation.
 */
export const buildStatsBucketSeries = (
  stats: readonly StatsBucketInput[],
  limitHours: number,
  nowMs: number,
  bucketCount = 360,
): StatsBucketSeries => {
  const oneStep = limitHours * 3600 * 1000 / bucketCount
  const steps: number[] = []
  for (let i = bucketCount; i >= 0; i--) {
    steps.push(nowMs - (oneStep * i))
  }

  const outputCount = Math.max(0, bucketCount - 1)
  const uplinkData = Array<number | null>(outputCount).fill(null)
  const downlinkData = Array<number | null>(outputCount).fill(null)

  if (oneStep > 0) {
    const windowStart = steps[0] ?? nowMs
    for (const stat of stats) {
      const timestamp = stat.dateTime * 1000
      const bucketIndex = Math.floor((timestamp - windowStart) / oneStep)
      if (bucketIndex < 0 || bucketIndex >= outputCount) continue

      const lowerBound = steps[bucketIndex]
      const upperBound = steps[bucketIndex + 1]
      if (lowerBound === undefined || upperBound === undefined) continue
      if (!(timestamp > lowerBound && timestamp < upperBound)) continue

      const target = stat.direction ? uplinkData : downlinkData
      target[bucketIndex] = (target[bucketIndex] ?? 0) + stat.traffic
    }
  }

  return {
    labelSteps: steps.slice(1, bucketCount),
    uplinkData,
    downlinkData,
  }
}
