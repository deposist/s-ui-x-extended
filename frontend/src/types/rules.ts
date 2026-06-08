interface generalRule {
  invert: boolean
  action: 'route' | 'route-options' | 'reject' | 'hijack-dns' | 'sniff' | 'resolve' | 'bypass'
  outbound?: string
  override_address?: string
  override_port?: number
  network_strategy?: string
  fallback_delay?: number
  udp_disable_domain_unmapping?: boolean
  udp_connect?: boolean
  udp_timeout?: string
  tls_fragment?: boolean
  tls_fragment_fallback_delay?: string
  tls_record_fragment?: boolean
  method?: string
  no_drop?: boolean
  sniffer: string[]
  timeout: string
  strategy: string
  server: string
}

export const actionKeys = [
  'invert',
  'action',
  'outbound',
  'override_address',
  'override_port',
  'network_strategy',
  'fallback_delay',
  'udp_disable_domain_unmapping',
  'udp_connect',
  'udp_timeout',
  'tls_fragment',
  'tls_fragment_fallback_delay',
  'tls_record_fragment',
  'method',
  'no_drop',
  'sniffer',
  'timeout',
  'strategy',
  'server'
]
export interface logicalRule extends generalRule {
  type: 'logical' | 'simple'
  mode: 'and' | 'or'
  rules: rule[]
}

export interface rule extends generalRule {
  inbound?: string[]
  ip_version?: 4 | 6
  network?: string[]
  auth_user?: string[]
  protocol?: string[]
  client?: ('chromium' | 'safari' | 'firefox' | 'quic-go' | 'unknown')[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  source_ip_cidr?: string[]
  source_ip_is_private?: boolean
  ip_cidr?: string[]
  ip_is_private?: boolean
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
  preferred_by?: string[]
  network_type?: ('wifi' | 'cellular' | 'ethernet' | 'other')[]
  network_is_expensive?: boolean
  network_is_constrained?: boolean
  wifi_ssid?: string[]
  wifi_bssid?: string[]
  interface_address?: { [interfaceName: string]: string[] }
  network_interface_address?: { wifi?: string[]; cellular?: string[]; ethernet?: string[]; other?: string[] }
  default_interface_address?: string[]
}

export interface ruleset {
  type: 'inline' | 'local' | 'remote'
  tag: string
  format?: 'source' | 'binary'
  rules?: headlessRule[]
  path?: string
  url?: string
  download_detour?: string
  update_interval?: string
}

export interface headlessRule {
  type?: 'logical' | 'simple'
  mode?: 'and' | 'or'
  rules?: headlessRule[]
  invert?: boolean
  query_type?: string[]
  network?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  source_ip_cidr?: string[]
  ip_cidr?: string[]
  source_port?: number[]
  source_port_range?: string[]
  port?: number[]
  port_range?: string[]
  process_name?: string[]
  process_path?: string[]
  process_path_regex?: string[]
  package_name?: string[]
  network_type?: ('wifi' | 'cellular' | 'ethernet' | 'other')[]
  network_is_expensive?: boolean
  network_is_constrained?: boolean
  wifi_ssid?: string[]
  wifi_bssid?: string[]
  network_interface_address?: { wifi?: string[]; cellular?: string[]; ethernet?: string[]; other?: string[] }
  default_interface_address?: string[]
}
