export interface Dns {
  servers: DnsServer[]
  rules: dnsRule[]
  final?: string
  strategy?: string
  disable_cache?: boolean,
  disable_expire?: boolean,
  // 1.14 deprecated this (schema:"omit"); the migration strips it from stored
  // blobs and the editor no longer offers it. Kept only to read pre-migration.
  independent_cache?: boolean,
  cache_capacity?: number,
  reverse_mapping?: boolean,
  client_subnet?: string,
}

export const DnsTypes = {
  Local: 'local',
  Hosts: 'hosts',
  TCP: 'tcp',
  UDP: 'udp',
  TLS: 'tls',
  QUIC: 'quic',
  HTTPS: 'https',
  HTTP3: 'h3',
  DHCP: 'dhcp',
  FakeIP: 'fakeip',
  MDNS: 'mdns',
  Tailscale: 'tailscale',
  Resolved: 'resolved',
  SDNS: 'sdns',
  Fallback: 'fallback',
  OpenVPN: 'openvpn',
  OpenConnect: 'openconnect',
}

export type DnsType = typeof DnsTypes[keyof typeof DnsTypes]

type InterfaceMap = {
  [Key in keyof typeof DnsTypes]: {
    type: string
    [otherProperties: string]: any
  }
}

export type DnsServer = InterfaceMap[keyof InterfaceMap]

const defaultValues: Record<DnsType, DnsServer> = {
  local: { type: 'local' },
  hosts: { type: 'hosts', path: ['/etc/hosts'] },
  tcp: { type: 'tcp', server_port: 53 },
  udp: { type: 'udp', server_port: 53 },
  tls: { type: 'tls', server_port: 853, tls: { enabled: true } },
  quic: { type: 'quic', server_port: 853, tls: { enabled: true } },
  https: { type: 'https', server_port: 443, tls: { enabled: true }, headers: {} },
  h3: { type: 'h3', server_port: 443, tls: { enabled: true }, headers: {} },
  predefined: { type: 'predefined', rcode: 'NOERROR' },
  dhcp: { type: 'dhcp' },
  mdns: { type: 'mdns' },
  fakeip: { type: 'fakeip', inet4_range: '198.18.0.0/15', inet6_range: 'fc00::/18' },
  tailscale: { type: 'tailscale' },
  resolved: { type: 'resolved' },
  sdns: { type: 'sdns', stamp: '' },
  fallback: { type: 'fallback', servers: [], strategy: 'sequential' },
  openvpn: { type: 'openvpn' },
  openconnect: { type: 'openconnect' },
}
export function createDnsServer<T extends DnsServer>(type: string, json?: Partial<T>): DnsServer {
  const defaultObject: DnsServer = { ...defaultValues[type], ...(json || {}) }
  return defaultObject
}

interface generalDnsRule {
  invert: boolean
  action: 'route' | 'route-options' | 'reject' | 'predefined' | 'evaluate' | 'respond'
  server?: string
  strategy?: string
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string
  method?: string
  no_drop?: boolean
  rcode?: string
  answer?: string[]
  // evaluate action options
  tag?: string
  speculative?: boolean
  ns?: string[]
  extra?: string[]
  // shared by every DNS route/evaluate/route-options action
  race?: boolean
  timeout?: string
  disable_optimistic_cache?: boolean
  remove_client_subnet?: boolean
}

export const actionDnsRuleKeys = [
  'invert',
  'action',
  'server',
  'strategy',
  'disable_cache',
  'rewrite_ttl',
  'client_subnet',
  'method',
  'no_drop',
  'rcode',
  'answer',
  'tag',
  'speculative',
  'ns',
  'extra',
  'race',
  'timeout',
  'disable_optimistic_cache',
  'remove_client_subnet',
]
/**
 * Every JSON field of the pinned fork's `option.RawDefaultDNSRule` except
 * `invert`, transcribed from the struct tags of
 * `github.com/deposist/sing-box-extended@v1.14.0-extended-2.7.1`.
 *
 * Two entries are easy to get wrong and are the reason this is transcribed from
 * the fork rather than from the `dnsRule` interface below:
 *   - `outbound` is a *match* field on a DNS rule (the legacy outbound matcher),
 *     even though it is an *action* field on a route rule. Classifying it as an
 *     action here would leave it behind on a converted node.
 *   - the deprecated aliases `geosite`, `geoip`, `source_geoip`, and
 *     `rule_set_ipcidr_match_source` are still decoded by the fork.
 */
export const dnsDefaultMatchKeys = [
  'inbound',
  'ip_version',
  'query_type',
  'query_client_subnet',
  'query_dnssec',
  'network',
  'auth_user',
  'protocol',
  'domain',
  'domain_suffix',
  'domain_keyword',
  'domain_regex',
  'geosite',
  'source_geoip',
  'geoip',
  'ip_cidr',
  'ip_is_private',
  'ip_accept_any',
  'source_ip_cidr',
  'source_ip_is_private',
  'source_port',
  'source_port_range',
  'port',
  'port_range',
  'process_name',
  'process_path',
  'process_path_regex',
  'package_name',
  'package_name_regex',
  'user',
  'user_id',
  'outbound',
  'clash_mode',
  'network_type',
  'network_is_expensive',
  'network_is_constrained',
  'wifi_ssid',
  'wifi_bssid',
  'interface_address',
  'network_interface_address',
  'default_interface_address',
  'source_mac_address',
  'source_hostname',
  'preferred_by',
  'match_response',
  'response_rcode',
  'response_answer',
  'response_ns',
  'response_extra',
  'rule_set',
  'rule_set_ip_cidr_match_source',
  'rule_set_ip_cidr_accept_empty',
  'rule_set_ipcidr_match_source',
] as const

export interface logicalDnsRule extends generalDnsRule {
  type: 'logical' | 'simple'
  mode: 'and' | 'or'
  rules: dnsRule[]
}

export interface dnsRule extends generalDnsRule {
  inbound?: string[]
  ip_version?: 4 | 6
  query_type?: string[]
  network?: string[]
  auth_user?: string[]
  protocol?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  source_ip_cidr?: string[]
  source_ip_is_private?: boolean
  ip_cidr?: string[]
  ip_is_private?: boolean
  ip_accept_any?: boolean
  source_port?: number[]
  source_port_range?: string[]
  port?: number[]
  port_range?: string[]
  process_name?: string[]
  process_path?: string[]
  process_path_regex?: string[]
  package_name?: string[]
  user?: string[]
  user_id?: number[]
  clash_mode?: string
  rule_set?: string[]
  rule_set_ip_cidr_match_source?: boolean
  rule_set_ip_cidr_accept_empty?: boolean
  network_type?: ('wifi' | 'cellular' | 'ethernet' | 'other')[]
  network_is_expensive?: boolean
  network_is_constrained?: boolean
  wifi_ssid?: string[]
  wifi_bssid?: string[]
  interface_address?: { [interfaceName: string]: string[] }
  network_interface_address?: { wifi?: string[]; cellular?: string[]; ethernet?: string[]; other?: string[] }
  default_interface_address?: string[]
  query_client_subnet?: string[]
  query_dnssec?: boolean
  package_name_regex?: string[]
  source_mac_address?: string[]
  source_hostname?: string[]
  preferred_by?: string[]
  match_response?: boolean | string
  response_rcode?: string
  response_answer?: string[]
  response_ns?: string[]
  response_extra?: string[]
}
