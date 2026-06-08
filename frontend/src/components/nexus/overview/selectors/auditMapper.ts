import { isSelectorRecord, nonNegativeNumber, plainText } from './selectorUtils'

export const auditDisplayIcons = [
  'mdi-shield-outline',
  'mdi-shield-alert-outline',
  'mdi-login',
  'mdi-account-alert-outline',
  'mdi-account-lock-outline',
  'mdi-logout',
  'mdi-logout-variant',
  'mdi-account-plus-outline',
  'mdi-account-remove-outline',
  'mdi-account-key-outline',
  'mdi-key-plus',
  'mdi-key-minus',
  'mdi-key',
  'mdi-database-import-outline',
  'mdi-database-export-outline',
  'mdi-lock-reset',
] as const

export type AuditDisplayIcon = typeof auditDisplayIcons[number]
export type AuditDisplayTone = 'info' | 'success' | 'warning' | 'error'

export interface AuditDisplayItem {
  id: number
  timestamp: number
  icon: AuditDisplayIcon
  tone: AuditDisplayTone
  text: string
  detail?: string
}

type AuditPresentation = {
  icon: AuditDisplayIcon
  tone: AuditDisplayTone
  text: string
}

const knownAuditEvents = {
  login_success: {
    icon: 'mdi-login',
    tone: 'success',
    text: 'Login succeeded',
  },
  login_failed: {
    icon: 'mdi-account-alert-outline',
    tone: 'warning',
    text: 'Login failed',
  },
  login_blocked: {
    icon: 'mdi-account-lock-outline',
    tone: 'warning',
    text: 'Login blocked',
  },
  logout: {
    icon: 'mdi-logout',
    tone: 'info',
    text: 'Admin logged out',
  },
  logout_all_admins: {
    icon: 'mdi-logout-variant',
    tone: 'warning',
    text: 'All admin sessions logged out',
  },
  admin_credentials_changed: {
    icon: 'mdi-account-key-outline',
    tone: 'warning',
    text: 'Admin credentials changed',
  },
  admin_created: {
    icon: 'mdi-account-plus-outline',
    tone: 'warning',
    text: 'Admin created',
  },
  admin_deleted: {
    icon: 'mdi-account-remove-outline',
    tone: 'warning',
    text: 'Admin deleted',
  },
  api_token_created: {
    icon: 'mdi-key-plus',
    tone: 'warning',
    text: 'API token created',
  },
  api_token_deleted: {
    icon: 'mdi-key-minus',
    tone: 'warning',
    text: 'API token deleted',
  },
  api_token_enabled_changed: {
    icon: 'mdi-key',
    tone: 'warning',
    text: 'API token access changed',
  },
  db_imported: {
    icon: 'mdi-database-import-outline',
    tone: 'warning',
    text: 'Database imported',
  },
  db_exported: {
    icon: 'mdi-database-export-outline',
    tone: 'warning',
    text: 'Database exported',
  },
  sub_secret_rotated: {
    icon: 'mdi-lock-reset',
    tone: 'warning',
    text: 'Client subscription secret rotated',
  },
  xui_import: {
    icon: 'mdi-database-import-outline',
    tone: 'success',
    text: '3x-ui import applied',
  },
} satisfies Record<string, AuditPresentation>

type KnownAuditEvent = keyof typeof knownAuditEvents

const unknownPresentation: AuditPresentation = {
  icon: 'mdi-shield-alert-outline',
  tone: 'info',
  text: 'Audit event',
}

const isKnownAuditEvent = (event: string): event is KnownAuditEvent => {
  return Object.hasOwn(knownAuditEvents, event)
}

const presentationForUnknownEvent = (severity?: string): AuditPresentation => {
  if (severity === 'error') return { ...unknownPresentation, tone: 'error' }
  if (severity === 'warn' || severity === 'warning') {
    return { ...unknownPresentation, tone: 'warning' }
  }
  return unknownPresentation
}

const wholeNumber = (value: unknown): number => {
  const number = nonNegativeNumber(value)
  return number === undefined ? 0 : Math.floor(number)
}

const displayDetail = (
  actor: string | undefined,
  resource: string | undefined,
  unknownEvent: string | undefined,
): string | undefined => {
  const fields: string[] = []
  if (actor) fields.push(`actor: ${actor}`)
  if (resource) fields.push(`resource: ${resource}`)
  if (unknownEvent) fields.push(`event: ${unknownEvent}`)
  return fields.length > 0 ? fields.join('; ') : undefined
}

export const mapAuditDisplayItem = (payload?: unknown): AuditDisplayItem => {
  const event = isSelectorRecord(payload) ? payload : {}
  const eventName = plainText(event.event)
  const severity = plainText(event.severity)
  const knownEventName = eventName && isKnownAuditEvent(eventName) ? eventName : undefined
  const presentation = knownEventName
    ? knownAuditEvents[knownEventName]
    : presentationForUnknownEvent(severity)
  const detail = displayDetail(
    plainText(event.actor),
    plainText(event.resource),
    knownEventName ? undefined : eventName,
  )

  const item: AuditDisplayItem = {
    id: wholeNumber(event.id),
    timestamp: wholeNumber(event.dateTime ?? event.timestamp),
    icon: presentation.icon,
    tone: presentation.tone,
    text: presentation.text,
  }

  if (detail) item.detail = detail
  return item
}

export const mapAuditDisplayItems = (payloads?: readonly unknown[] | null): AuditDisplayItem[] => {
  return (payloads ?? []).map(mapAuditDisplayItem)
}
