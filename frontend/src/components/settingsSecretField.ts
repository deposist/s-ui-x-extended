export const STORED_SECRET_PLACEHOLDER = '••• stored •••'
export const SECRET_MARKER_SUFFIX = 'HasSecret'

type SettingValue = string | boolean | number | null | undefined
type SettingsMap = Record<string, SettingValue>

export const hasStoredSecret = (value: SettingValue): boolean => {
  return value === true || value === 'true'
}

export const secretKeyFromMarker = (key: string): string | null => {
  if (!key.endsWith(SECRET_MARKER_SUFFIX)) return null
  return key.slice(0, -SECRET_MARKER_SUFFIX.length)
}

export const normalizeSecretFields = <T extends SettingsMap>(settings: T): T & SettingsMap => {
  const normalized = { ...settings } as SettingsMap
  for (const key of Object.keys(normalized)) {
    const secretKey = secretKeyFromMarker(key)
    if (!secretKey) continue
    if (normalized[secretKey] === undefined || normalized[secretKey] === null) {
      normalized[secretKey] = ''
    }
  }
  return normalized as T & SettingsMap
}

export const stripSecretPlaceholders = <T extends SettingsMap>(settings: T): T & SettingsMap => {
  const stripped = { ...settings } as SettingsMap
  for (const key of Object.keys(stripped)) {
    const secretKey = secretKeyFromMarker(key)
    if (!secretKey) continue
    if (stripped[secretKey] === STORED_SECRET_PLACEHOLDER && secretKey !== 'telegramBackupPassphrase') {
      stripped[secretKey] = ''
    }
  }
  return stripped as T & SettingsMap
}
