export type TelegramBackupScheduleMode =
  | 'manual'
  | 'every15m'
  | 'every30m'
  | 'hourly'
  | 'every6h'
  | 'every12h'
  | 'daily3'
  | 'custom'
  | 'advanced'

export type TelegramBackupScheduleUnit = 'minutes' | 'hours'

export type TelegramBackupScheduleState = {
  mode: TelegramBackupScheduleMode
  customValue: number
  customUnit: TelegramBackupScheduleUnit
  advancedCron: string
}

export type TelegramBackupScheduleError =
  | 'customMinutesRange'
  | 'customHoursRange'
  | 'advancedCronInvalid'

const defaultCustomValue = 15
const defaultCustomUnit: TelegramBackupScheduleUnit = 'minutes'

const presetCron: Record<Exclude<TelegramBackupScheduleMode, 'manual' | 'custom' | 'advanced'>, string> = {
  every15m: '*/15 * * * *',
  every30m: '*/30 * * * *',
  hourly: '0 * * * *',
  every6h: '0 */6 * * *',
  every12h: '0 */12 * * *',
  daily3: '0 3 * * *',
}

const normalizeCronSpec = (cron: string) => cron.trim().replace(/\s+/g, ' ')

export const parseTelegramBackupSchedule = (cron: string): TelegramBackupScheduleState => {
  const trimmed = cron.trim()
  const normalized = normalizeCronSpec(cron)
  if (!normalized) {
    return {
      mode: 'manual',
      customValue: defaultCustomValue,
      customUnit: defaultCustomUnit,
      advancedCron: '',
    }
  }

  for (const [mode, spec] of Object.entries(presetCron)) {
    if (normalized === spec) {
      return {
        mode: mode as TelegramBackupScheduleMode,
        customValue: defaultCustomValue,
        customUnit: defaultCustomUnit,
        advancedCron: '',
      }
    }
  }

  const minuteMatch = normalized.match(/^\*\/([1-9]\d?) \* \* \* \*$/)
  if (minuteMatch) {
    const customValue = Number(minuteMatch[1])
    if (customValue >= 1 && customValue <= 59) {
      return {
        mode: 'custom',
        customValue,
        customUnit: 'minutes',
        advancedCron: '',
      }
    }
  }

  const hourMatch = normalized.match(/^0 \*\/([1-9]\d?) \* \* \*$/)
  if (hourMatch) {
    const customValue = Number(hourMatch[1])
    if (customValue >= 1 && customValue <= 23) {
      return {
        mode: 'custom',
        customValue,
        customUnit: 'hours',
        advancedCron: '',
      }
    }
  }

  return {
    mode: 'advanced',
    customValue: defaultCustomValue,
    customUnit: defaultCustomUnit,
    advancedCron: trimmed,
  }
}

export const serializeTelegramBackupSchedule = (state: TelegramBackupScheduleState): string => {
  if (state.mode === 'manual') return ''
  if (state.mode === 'custom') {
    const value = Math.trunc(Number(state.customValue))
    return state.customUnit === 'hours'
      ? `0 */${value} * * *`
      : `*/${value} * * * *`
  }
  if (state.mode === 'advanced') return state.advancedCron.trim()
  return presetCron[state.mode]
}

export const validateTelegramBackupSchedule = (state: TelegramBackupScheduleState): TelegramBackupScheduleError[] => {
  if (state.mode === 'custom') {
    const value = Number(state.customValue)
    if (!Number.isInteger(value)) {
      return [state.customUnit === 'hours' ? 'customHoursRange' : 'customMinutesRange']
    }
    if (state.customUnit === 'minutes' && (value < 1 || value > 59)) {
      return ['customMinutesRange']
    }
    if (state.customUnit === 'hours' && (value < 1 || value > 23)) {
      return ['customHoursRange']
    }
  }

  if (state.mode === 'advanced') {
    const cron = state.advancedCron.trim()
    if (!cron) return []
    const parts = cron.split(/\s+/)
    if (parts.length !== 5 || parts.some(part => part.includes('/0'))) {
      return ['advancedCronInvalid']
    }
  }

  return []
}
