export interface ClientIPHistoryRow {
  id?: number
  ip: string
  firstSeen: number
  lastSeen: number
}

const ipv4Pattern = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/

export const isBackendMaskedIP = (ip: string): boolean => {
  return ip === 'masked' || ip.startsWith('masked:')
}

export const maskRawIP = (ip: string): string => {
  if (!ip || isBackendMaskedIP(ip)) return ip

  const ipv4 = ip.match(ipv4Pattern)
  if (ipv4) {
    return `${ipv4[1]}.${ipv4[2]}.${ipv4[3]}.x`
  }

  if (ip.includes(':')) {
    const groups = ip.split(':').filter(Boolean)
    if (groups.length === 0) return 'masked'
    return `${groups.slice(0, 4).join(':')}::/64`
  }

  return 'masked'
}

export const displayIP = (ip: string, showRaw: boolean): string => {
  if (showRaw || isBackendMaskedIP(ip)) return ip
  return maskRawIP(ip)
}

export const hasRawIPRows = (rows: ClientIPHistoryRow[]): boolean => {
  return rows.some(row => row.ip.length > 0 && !isBackendMaskedIP(row.ip))
}
