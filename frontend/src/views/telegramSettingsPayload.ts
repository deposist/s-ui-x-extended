import { STORED_SECRET_PLACEHOLDER } from '@/components/settingsSecretField'

export type TelegramSettingsMap = Record<string, string>

export const telegramSettingKeys = [
  'telegramEnabled',
  'telegramBotToken',
  'telegramChatID',
  'telegramProxyURL',
  'telegramProxyUsername',
  'telegramProxyPassword',
  'telegramTransportMode',
  'telegramOutboundTag',
  'telegramCpuThreshold',
  'telegramNotifyCpu',
  'telegramReport',
  'telegramReportCron',
  'telegramBackupEnabled',
  'telegramBackupPassphrase',
  'telegramBackupCron',
  'telegramBackupExcludeTables',
  'telegramBackupMaxSizeMB',
]

const telegramSecretSettingKeys = [
  'telegramBotToken',
  'telegramProxyURL',
  'telegramProxyUsername',
  'telegramProxyPassword',
  'telegramBackupPassphrase',
]

export const minTelegramBackupPassphraseLength = 12

export const hasWeakTelegramBackupPassphrase = (value: string): boolean => {
  return value !== ''
    && value !== STORED_SECRET_PLACEHOLDER
    && Array.from(value).length < minTelegramBackupPassphraseLength
}

export const pickTelegramSettings = (source: TelegramSettingsMap): TelegramSettingsMap => {
  const picked: TelegramSettingsMap = {}
  for (const key of telegramSettingKeys) {
    picked[key] = String(source[key] ?? '')
  }
  for (const key of telegramSecretSettingKeys) {
    picked[key + 'HasSecret'] = String(source[key + 'HasSecret'] ?? 'false')
  }
  return picked
}
