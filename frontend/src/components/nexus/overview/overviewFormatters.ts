import { i18n } from '@/locales'
import { HumanReadable } from '@/plugins/utils'

const finiteNonNegative = (value: number): number => {
  return Number.isFinite(value) ? Math.max(0, value) : 0
}

export const formatOverviewCount = (value: number): string => {
  return finiteNonNegative(value).toLocaleString()
}

export const formatOverviewSize = (value?: number): string => {
  const size = value === undefined ? 0 : finiteNonNegative(value)
  return size > 0 ? HumanReadable.sizeFormat(size) : `0 ${i18n.global.t('stats.B')}`
}

export const formatOverviewRate = (value: number): string => {
  return `${formatOverviewSize(value)}/s`
}

export const formatOverviewDuration = (value?: number): string => {
  if (value === undefined || !Number.isFinite(value) || value <= 0) return '-'
  return HumanReadable.formatSecond(value)
}

export const formatOverviewPercent = (value?: number): string => {
  if (value === undefined || !Number.isFinite(value)) return '-'
  return `${Math.min(100, Math.max(0, value)).toFixed(0)}%`
}
