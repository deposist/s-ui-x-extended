import { describe, expect, it } from 'vitest'
import {
  parseTelegramBackupSchedule,
  serializeTelegramBackupSchedule,
  validateTelegramBackupSchedule,
  type TelegramBackupScheduleState,
} from './telegramBackupSchedule'

const state = (overrides: Partial<TelegramBackupScheduleState>): TelegramBackupScheduleState => ({
  mode: 'manual',
  customValue: 15,
  customUnit: 'minutes',
  advancedCron: '',
  ...overrides,
})

describe('telegram backup schedule helpers', () => {
  it('parses blank cron as manual only', () => {
    expect(parseTelegramBackupSchedule('')).toMatchObject({ mode: 'manual' })
    expect(serializeTelegramBackupSchedule(state({ mode: 'manual' }))).toBe('')
  })

  it.each([
    ['every15m', '*/15 * * * *'],
    ['every30m', '*/30 * * * *'],
    ['hourly', '0 * * * *'],
    ['every6h', '0 */6 * * *'],
    ['every12h', '0 */12 * * *'],
    ['daily3', '0 3 * * *'],
  ] as const)('round-trips the %s preset', (mode, cron) => {
    expect(parseTelegramBackupSchedule(cron)).toMatchObject({ mode })
    expect(serializeTelegramBackupSchedule(state({ mode }))).toBe(cron)
  })

  it('converts custom minute and hour intervals to cron', () => {
    expect(serializeTelegramBackupSchedule(state({
      mode: 'custom',
      customValue: 5,
      customUnit: 'minutes',
    }))).toBe('*/5 * * * *')
    expect(parseTelegramBackupSchedule('*/5 * * * *')).toMatchObject({
      mode: 'custom',
      customValue: 5,
      customUnit: 'minutes',
    })

    expect(serializeTelegramBackupSchedule(state({
      mode: 'custom',
      customValue: 2,
      customUnit: 'hours',
    }))).toBe('0 */2 * * *')
    expect(parseTelegramBackupSchedule('0 */2 * * *')).toMatchObject({
      mode: 'custom',
      customValue: 2,
      customUnit: 'hours',
    })
  })

  it('keeps unknown cron expressions in advanced mode', () => {
    const parsed = parseTelegramBackupSchedule('10 4 * * 1')
    expect(parsed).toMatchObject({
      mode: 'advanced',
      advancedCron: '10 4 * * 1',
    })
    expect(serializeTelegramBackupSchedule(parsed)).toBe('10 4 * * 1')
  })

  it.each([
    state({ mode: 'custom', customValue: 0, customUnit: 'minutes' }),
    state({ mode: 'custom', customValue: -1, customUnit: 'minutes' }),
    state({ mode: 'custom', customValue: 60, customUnit: 'minutes' }),
    state({ mode: 'custom', customValue: 0, customUnit: 'hours' }),
    state({ mode: 'custom', customValue: -1, customUnit: 'hours' }),
    state({ mode: 'custom', customValue: 24, customUnit: 'hours' }),
  ])('rejects invalid custom interval %#', (schedule) => {
    expect(validateTelegramBackupSchedule(schedule).length).toBeGreaterThan(0)
  })
})
