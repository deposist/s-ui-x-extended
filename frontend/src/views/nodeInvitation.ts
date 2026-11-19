const prefix = 's-ui-node v1 '

export function parseNodeInvitation(text: string): { masterUrl: string; code: string } | null {
  const trimmed = text.trim()
  if (!trimmed.startsWith(prefix)) return null
  const [address, code, extra] = trimmed.slice(prefix.length).split(/\s+/)
  if (!address || !code || extra || !/^[A-Za-z2-7]{26,64}$/.test(code)) return null
  try {
    const url = new URL(address)
    if (url.protocol !== 'https:' || !url.hostname || url.username || url.password || /[?#\\]/.test(address)) return null
    return { masterUrl: url.href, code }
  } catch {
    return null
  }
}

export function createNodeInvitation(masterUrl: string, code: string): string {
  const text = `${prefix}${masterUrl} ${code}`
  return parseNodeInvitation(text) ? text : ''
}
