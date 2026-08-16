// Suggestion logic for the managed AWG "Public endpoint (host:port)" field.
// Kept pure (no component state) so it stays unit-testable; the panel address
// is a best-effort default the admin confirms, never a silent override.

export const AWG_DEFAULT_LISTEN_PORT = 51820

// Hostnames that only make sense for the admin's own machine; suggesting them
// would hand devices an unreachable endpoint.
export function isLocalHostname(hostname: string): boolean {
  const host = (hostname ?? '').trim().toLowerCase()
  return host === '' || host === 'localhost' || host === '127.0.0.1' || host === '::1' || host === '[::1]'
}

export function panelHostname(): string {
  if (typeof window === 'undefined' || !window.location) return ''
  return window.location.hostname || ''
}

export function suggestAWGPublicEndpoint(hostname: string, listenPort: number | null | undefined): string {
  if (isLocalHostname(hostname)) return ''
  const port = listenPort && listenPort > 0 ? listenPort : AWG_DEFAULT_LISTEN_PORT
  return `${hostname.trim()}:${port}`
}
