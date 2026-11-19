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
  nodeAssignments?: DoctorNodeAssignment[]
}

export interface DoctorNodeAssignment {
  nodeUuid: string
  nodeName: string
  inboundId?: number
  inboundTag?: string
  linkReason: string
  quotaState: 'unknown' | 'stale' | 'measured' | 'exhausted' | 'unlimited'
  used?: number
  remaining?: number
  sampledAt?: number
  probe?: { inboundId: number; checkedAt: number; result: string; stage: string }
}
