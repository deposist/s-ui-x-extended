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
    const parts = ip.split('::')
    if (parts.length > 2) return 'masked'
    let groups = parts[0].split(':').filter(Boolean)
    if (parts.length === 2) {
      const suffix = parts[1].split(':').filter(Boolean)
      const omitted = 8 - groups.length - suffix.length - (suffix.at(-1)?.includes('.') ? 1 : 0)
      if (omitted < 1) return 'masked'
      groups = [...groups, ...new Array(omitted).fill('0'), ...suffix]
    }
    if (groups.length < 4) return 'masked'
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
