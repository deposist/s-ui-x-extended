import {
  isSelectorRecord,
  nonNegativeNumber,
  plainText,
  plainTextList,
} from './selectorUtils'

export interface SystemStatus {
  ipv4: string[]
  ipv6: string[]
  appVersion: string
  bootTime: number
  uptimeSec: number
  singboxRunning: boolean
  singboxVersion?: string
  singboxAlloc?: number
  singboxUptimeSec?: number
}

const seconds = (value: unknown): number => {
  const number = nonNegativeNumber(value)
  return number === undefined ? 0 : Math.floor(number)
}

export const selectSystemStatus = (payload?: unknown, nowSec?: number): SystemStatus => {
  const status = isSelectorRecord(payload) ? payload : {}
  const sys = isSelectorRecord(status.sys) ? status.sys : {}
  const sbd = isSelectorRecord(status.sbd) ? status.sbd : {}
  const sbdStats = isSelectorRecord(sbd.stats) ? sbd.stats : {}
  const bootTime = seconds(sys.bootTime)
  const currentTime = seconds(nowSec)
  const singboxVersion = plainText(sbd.version)
  const singboxAlloc = nonNegativeNumber(sbdStats.Alloc)
  const singboxUptimeSec = nonNegativeNumber(sbdStats.Uptime)

  const selected: SystemStatus = {
    ipv4: plainTextList(sys.ipv4),
    ipv6: plainTextList(sys.ipv6),
    appVersion: plainText(sys.appVersion) ?? '',
    bootTime,
    uptimeSec: bootTime > 0 && currentTime > bootTime ? currentTime - bootTime : 0,
    singboxRunning: sbd.running === true,
  }

  if (singboxVersion) selected.singboxVersion = singboxVersion
  if (singboxAlloc !== undefined) selected.singboxAlloc = singboxAlloc
  if (singboxUptimeSec !== undefined) selected.singboxUptimeSec = singboxUptimeSec

  return selected
}
