import { describe, expect, it } from 'vitest'

import { buildStatsBucketSeries, type StatsBucketInput } from './statsBuckets'

const referenceBuckets = (
  stats: readonly StatsBucketInput[],
  limitHours: number,
  nowMs: number,
  bucketCount: number,
) => {
  const oneStep = limitHours * 3600 * 1000 / bucketCount
  const steps: number[] = []
  for (let i = bucketCount; i >= 0; i--) steps.push(nowMs - (oneStep * i))

  const uplinkData: Array<number | null> = []
  const downlinkData: Array<number | null> = []
  for (let i = 1; i < bucketCount; i++) {
    const up = stats
      .filter(row => row.direction && row.dateTime * 1000 < steps[i]! && row.dateTime * 1000 > steps[i - 1]!)
      .map(row => row.traffic)
    uplinkData.push(up.length > 0 ? up.reduce((sum, traffic) => sum + traffic, 0) : null)

    const down = stats
      .filter(row => !row.direction && row.dateTime * 1000 < steps[i]! && row.dateTime * 1000 > steps[i - 1]!)
      .map(row => row.traffic)
    downlinkData.push(down.length > 0 ? down.reduce((sum, traffic) => sum + traffic, 0) : null)
  }

  return { labelSteps: steps.slice(1, bucketCount), uplinkData, downlinkData }
}

describe('linear stats buckets', () => {
  it('preserves legacy strict boundaries, gaps, duplicate sums, and output order', () => {
    const nowMs = 3_600_000
    const input = Object.freeze([
      { dateTime: 1_750, direction: true, traffic: 3 },
      { dateTime: 1, direction: false, traffic: 7 },
      { dateTime: 1_750, direction: 1, traffic: 5 },
      { dateTime: 2_699, direction: false, traffic: 11 },
      { dateTime: 0, direction: true, traffic: 100 }, // exact lower boundary
      { dateTime: 1_800, direction: true, traffic: 101 }, // exact shared boundary
      { dateTime: 2_701, direction: true, traffic: 13 }, // omitted newest interval
      { dateTime: 3_600, direction: false, traffic: 103 }, // exact now boundary
    ] satisfies StatsBucketInput[])

    const got = buildStatsBucketSeries(input, 1, nowMs, 4)

    expect(got).toEqual(referenceBuckets(input, 1, nowMs, 4))
    expect(got).toEqual({
      labelSteps: [900_000, 1_800_000, 2_700_000],
      uplinkData: [null, 8, null],
      downlinkData: [7, null, 11],
    })
  })

  it('reads each input row once instead of rescanning it for every bucket', () => {
    let dateTimeReads = 0
    const rows = Array.from({ length: 2_000 }, (_, index) => ({
      get dateTime() {
        dateTimeReads++
        return 1_001 + (index % 1_998)
      },
      direction: index % 2 === 0,
      traffic: index + 1,
    }))

    buildStatsBucketSeries(rows, 1, 3_600_000, 360)

    expect(dateTimeReads).toBe(rows.length)
  })
})
