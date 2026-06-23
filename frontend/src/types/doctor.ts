export type DoctorSeverity = 'ok' | 'warn' | 'error'

export interface DoctorItem {
  id: string
  title: string
  severity: DoctorSeverity
  message: string
  action?: string
  details?: unknown
}

export interface DoctorReport {
  status: DoctorSeverity
  summary: string
  items: DoctorItem[]
  ranAt: number
  durationMs: number
}
