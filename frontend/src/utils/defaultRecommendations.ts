import { DnsTypes } from '@/types/dns'
import { EpTypes } from '@/types/endpoints'
import { InTypes } from '@/types/inbounds'
import { OutTypes } from '@/types/outbounds'
import { SrvTypes } from '@/types/services'
import { RECOMMENDED, dnsResolvers, dohPaths, sniFrontHosts, tlsAlpn } from '@/types/recommended'
import type { RecommendationContext, RecommendationSpec } from '@/utils/recommendations'

function typeAvailable(context: RecommendationContext<any>): boolean {
  return !context.type || !context.unavailableTypes?.includes(context.type)
}

export const inboundRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'inbound-listen-any', label: 'Listen on all interfaces', description: 'Use :: for dual-stack listen when the protocol supports Listen.', path: 'listen', value: '::', when: (ctx) => typeAvailable(ctx) && ctx.type !== InTypes.Tun },
  { id: 'inbound-domain-resolver', label: 'Use local domain resolver', description: 'For local proxy inbounds, resolve domains through the local DNS server.', path: 'domain_resolver.server', value: 'local', when: (ctx) => [InTypes.SOCKS, InTypes.HTTP, InTypes.Mixed].includes(ctx.type as any) },
]

export const outboundRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'outbound-server-port-443', label: 'Use HTTPS port', description: 'Common TLS/QUIC protocols usually listen on port 443.', path: 'server_port', value: 443, when: (ctx) => typeAvailable(ctx) && [OutTypes.VLESS, OutTypes.VMess, OutTypes.Trojan, OutTypes.Hysteria2, OutTypes.TUIC, OutTypes.AnyTls, OutTypes.Naive].includes(ctx.type as any) },
  { id: 'outbound-tls-min-13', label: 'Require TLS 1.3', description: 'Prefer TLS 1.3 where outbound TLS settings are available.', path: 'tls.min_version', value: RECOMMENDED.tlsMinVersion, when: (ctx) => typeAvailable(ctx) && ctx.model != null && Object.hasOwn(ctx.model as object, 'tls') },
  { id: 'outbound-utls-chrome', label: 'Use Chrome uTLS fingerprint', description: 'Recommended client fingerprint for outbound TLS.', path: 'tls.utls', value: { enabled: true, fingerprint: 'chrome' }, when: (ctx) => typeAvailable(ctx) && ctx.model != null && Object.hasOwn(ctx.model as object, 'tls') },
]

export const serviceRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'service-listen-any', label: 'Listen on all interfaces', description: 'Use :: for dual-stack listen when this service has Listen settings.', path: 'listen', value: '::', when: (ctx) => ![SrvTypes.OOMKiller, SrvTypes.Profiler].includes(ctx.type as any) && (ctx.model as any)?.listen_port != null },
]

export const endpointRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'endpoint-wireguard-mtu', label: 'WireGuard MTU 1420', description: 'A safe MTU for common WireGuard deployments.', path: 'mtu', value: 1420, when: (ctx) => ctx.type === EpTypes.Wireguard },
  { id: 'endpoint-vpn-server-route', label: 'VPN server first client address', description: 'Use the default first client address for a new VPN server user.', path: 'users.0.address', value: '10.0.0.2', when: (ctx) => ctx.type === EpTypes.VpnServer },
]

export const tlsRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'tls-min-13', label: 'Minimum TLS 1.3', description: 'Use TLS 1.3 as the minimum version when enabled.', path: 'server.min_version', value: RECOMMENDED.tlsMinVersion, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-max-13', label: 'Maximum TLS 1.3', description: 'Pin maximum TLS version to TLS 1.3 when enabled.', path: 'server.max_version', value: RECOMMENDED.tlsMaxVersion, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-alpn-modern', label: 'Modern ALPN list', description: 'Advertise HTTP/3, HTTP/2 and HTTP/1.1.', path: 'server.alpn', value: tlsAlpn.map((item) => item.value), when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-utls-chrome', label: 'Chrome client fingerprint', description: 'Use Chrome as the outbound uTLS fingerprint.', path: 'client.utls', value: { enabled: true, fingerprint: 'chrome' }, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'reality-handshake-server', label: 'Reality handshake server', description: 'Use a well-known TLS 1.3 host as the Reality handshake destination.', path: 'server.reality.handshake.server', value: sniFrontHosts[0], when: (ctx) => Boolean((ctx.model as any)?.server?.reality) },
]

export const dnsRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'dns-server-cloudflare', label: 'Cloudflare resolver', description: 'Use 1.1.1.1 as an editable upstream DNS server.', path: 'server', value: dnsResolvers[0], when: (ctx) => [DnsTypes.TCP, DnsTypes.UDP, DnsTypes.TLS, DnsTypes.QUIC].includes(ctx.type as any) },
  { id: 'dns-doh-server', label: 'Cloudflare DoH host', description: 'Use cloudflare-dns.com with the standard DoH path.', path: 'server', value: 'cloudflare-dns.com', when: (ctx) => [DnsTypes.HTTPS, DnsTypes.HTTP3].includes(ctx.type as any) },
  { id: 'dns-doh-path', label: 'Standard DoH path', description: 'Use /dns-query for DoH and HTTP/3 DNS.', path: 'path', value: dohPaths[0], when: (ctx) => [DnsTypes.HTTPS, DnsTypes.HTTP3].includes(ctx.type as any) },
]

export const dnsRuleRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'dns-rule-mode-or', label: 'Logical OR mode', description: 'OR mode is usually expected when adding multiple match rules.', path: 'mode', value: 'or', when: (ctx) => (ctx.model as any).type === 'logical' },
  { id: 'dns-rule-strategy-ipv4', label: 'Prefer IPv4 strategy', description: 'Prefer IPv4 for DNS route actions when a strategy is needed.', path: 'strategy', value: 'prefer_ipv4', when: (ctx) => (ctx.model as any).action === 'route' },
]

export const ruleRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'rule-mode-or', label: 'Logical OR mode', description: 'OR mode is usually expected when adding multiple match rules.', path: 'mode', value: 'or', when: (ctx) => (ctx.model as any).type === 'logical' },
  { id: 'rule-udp-timeout', label: 'UDP timeout 5m', description: 'Use an explicit UDP timeout for route options.', path: 'udp_timeout', value: '5m', when: (ctx) => ['route', 'route-options', 'bypass'].includes((ctx.model as any).action) },
]
