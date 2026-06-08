import { isSelectorRecord, nonNegativeNumber, plainText } from './selectors/selectorUtils'

export interface OverviewCapacityMetric {
  current?: number
  total?: number
  percent?: number
}

export interface OverviewStatusMetrics {
  cpuPercent?: number
  memory: OverviewCapacityMetric
  disk: OverviewCapacityMetric
}

export interface NetworkStatusSample {
  download: number
  upload: number
  capturedAt: number
}

export interface NetworkTrafficRate {
  downloadBps: number
  uploadBps: number
}

const capacityMetric = (payload: unknown): OverviewCapacityMetric => {
  const metric = isSelectorRecord(payload) ? payload : {}
  const current = nonNegativeNumber(metric.current)
  const total = nonNegativeNumber(metric.total)
  const selected: OverviewCapacityMetric = {}

  if (current !== undefined) selected.current = current
  if (total !== undefined) selected.total = total
  if (current !== undefined && total && total > 0) {
    selected.percent = Math.min(100, (current / total) * 100)
  }

  return selected
}

export const payloadItems = (payload: unknown): unknown[] => {
  return Array.isArray(payload) ? payload : []
}

export const auditEventsFromPayload = (payload: unknown): unknown[] => {
  const result = isSelectorRecord(payload) ? payload : {}
  return payloadItems(result.events)
}

export const overviewInboundTags = (
  inbounds?: readonly unknown[] | null,
): string[] => {
  const tags = new Set<string>()

  for (const inbound of inbounds ?? []) {
    if (!isSelectorRecord(inbound) || inbound.enable === false) continue

    const tag = plainText(inbound.tag)
    if (tag) tags.add(tag)
  }

  return [...tags]
}

export const overviewStatusMetrics = (payload?: unknown): OverviewStatusMetrics => {
  const status = isSelectorRecord(payload) ? payload : {}
  const cpuPercent = nonNegativeNumber(status.cpu)
  const metrics: OverviewStatusMetrics = {
    memory: capacityMetric(status.mem),
    disk: capacityMetric(status.dsk),
  }

  if (cpuPercent !== undefined) metrics.cpuPercent = Math.min(100, cpuPercent)
  return metrics
}

export const overviewStatusNetworkSample = (
  payload?: unknown,
  capturedAt = Date.now(),
): NetworkStatusSample | undefined => {
  const status = isSelectorRecord(payload) ? payload : {}
  const net = isSelectorRecord(status.net) ? status.net : {}
  const download = nonNegativeNumber(net.recv)
  const upload = nonNegativeNumber(net.sent)

  if (download === undefined || upload === undefined) return

  return {
    download,
    upload,
    capturedAt,
  }
}

export const networkRateFromSamples = (
  previous?: NetworkStatusSample,
  current?: NetworkStatusSample,
): NetworkTrafficRate | undefined => {
  if (!previous || !current) return

  const elapsedSeconds = (current.capturedAt - previous.capturedAt) / 1000
  if (elapsedSeconds <= 0) return

  const download = current.download - previous.download
  const upload = current.upload - previous.upload
  if (download < 0 || upload < 0) return

  return {
    downloadBps: download / elapsedSeconds,
    uploadBps: upload / elapsedSeconds,
  }
}
