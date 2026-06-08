export interface KpiSummary {
  liveTrafficBps: number
  onlineClients: number
  activeInbounds: number
  totalInbounds: number
  health: 'healthy' | 'degraded' | 'down'
}

export interface KpiInboundInput {
  enable?: boolean
  tag?: string
}

export interface KpiSelectorInput {
  inbounds?: readonly KpiInboundInput[] | null
  onlines?: {
    inbound?: readonly string[] | null
    user?: readonly string[] | null
  } | null
  liveTraffic?: {
    downloadBps?: number | null
    uploadBps?: number | null
  } | null
  health?: {
    online?: boolean | null
    singboxRunning?: boolean | null
  } | null
}

const toTrafficBps = (value: number | null | undefined): number => {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
}

const selectHealth = (input?: KpiSelectorInput['health']): KpiSummary['health'] => {
  if (input?.online === false || input?.singboxRunning === false) return 'down'
  if (input?.online === true && input.singboxRunning === true) return 'healthy'
  return 'degraded'
}

const uniqueStrings = (values: readonly string[] | null | undefined): Set<string> => {
  return new Set((values ?? []).filter((value) => value.length > 0))
}

export const selectKpiSummary = (input?: KpiSelectorInput | null): KpiSummary => {
  const onlineInboundTags = uniqueStrings(input?.onlines?.inbound)
  const enabledInbounds = (input?.inbounds ?? []).filter((inbound) => inbound.enable !== false)

  return {
    liveTrafficBps:
      toTrafficBps(input?.liveTraffic?.downloadBps) +
      toTrafficBps(input?.liveTraffic?.uploadBps),
    onlineClients: uniqueStrings(input?.onlines?.user).size,
    activeInbounds: enabledInbounds.filter((inbound) => {
      return typeof inbound.tag === 'string' && onlineInboundTags.has(inbound.tag)
    }).length,
    totalInbounds: enabledInbounds.length,
    health: selectHealth(input?.health),
  }
}
