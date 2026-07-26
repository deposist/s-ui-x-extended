interface generalRule {
  invert: boolean
  action: 'route' | 'route-options' | 'direct' | 'reject' | 'hijack-dns' | 'sniff' | 'resolve' | 'bypass'
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
  'server',
  // Present on the resolve action alongside `server`/`strategy`, and on
  // `route-options`. Their absence used to push them into the match half of the
  // modal, which silently rewrote them as conditions on save.
  'override_gateway',
  'disable_cache',
  'rewrite_ttl',
  'client_subnet',
]

/**
 * `DialerOptions`, which the fork decodes into a rule only when its action is
 * `direct` (`DirectActionOptions` is a type alias of `DialerOptions`, selected by
 * `C.RuleActionTypeDirect`).
 *
 * Kept separate from `actionKeys` because of one genuine schema collision:
 * `network_type` is a `DialerOptions` field *and* a `RawDefaultRule` match field.
 * A flat key lookup has to guess wrong in one direction or the other, so the
 * partition is resolved by action instead — see `isRouteActionKey`.
 *
 * `network_strategy` and `fallback_delay` are intentionally absent: they are
 * already in `actionKeys` via `RawRouteOptionsActionOptions`, and listing a key
 * in both would make the partition order-dependent.
 */
export const routeDialerActionKeys = [
  'detour',
  'bind_interface',
  'inet4_bind_address',
  'inet6_bind_address',
  'bind_address_no_port',
  'protect_path',
  'routing_mark',
  'reuse_addr',
  'netns',
  'connect_timeout',
  'tcp_fast_open',
  'tcp_multi_path',
  'disable_tcp_keep_alive',
  'tcp_keep_alive',
  'tcp_keep_alive_interval',
  'udp_fragment',
  'domain_resolver',
  'network_type',
  'fallback_network_type',
  'domain_strategy',
] as const

/**
 * Whether a key belongs to the action half of a route rule.
 *
 * Context-sensitive on purpose. `network_type` is the only key the fork treats as
 * both an action and a match field, and which of the two it is depends entirely
 * on the rule's action, so this is decidable rather than a coin flip.
 */
export const isRouteActionKey = (key: string, action: unknown): boolean =>
  actionKeys.includes(key) ||
  (action === 'direct' && (routeDialerActionKeys as readonly string[]).includes(key))
/**
 * Every JSON field of the pinned fork's `option.RawDefaultRule` except `invert`,
 * which is owned by the node shape control rather than by the match editor.
 *
 * Transcribed from the struct tags of
 * `github.com/deposist/sing-box-extended@v1.13.14-extended-2.5.4`, not from the
 * `rule` interface below, because the decoder also accepts deprecated aliases
 * that the interface never modelled: `geosite`, `geoip`, `source_geoip`, and
 * `rule_set_ipcidr_match_source`. A default -> logical conversion deletes exactly
 * these keys, so anything missing here would survive the conversion and be
 * silently misread as an action or passthrough field on the resulting node.
 */
export const routeDefaultMatchKeys = [
  'inbound',
  'ip_version',
  'network',
  'auth_user',
  'protocol',
  'client',
  'domain',
  'domain_suffix',
  'domain_keyword',
  'domain_regex',
  'geosite',
  'source_geoip',
  'geoip',
  'source_ip_cidr',
  'source_ip_is_private',
  'ip_cidr',
  'ip_is_private',
  'source_port',
  'source_port_range',
  'port',
  'port_range',
  'process_name',
  'process_path',
  'process_path_regex',
  'package_name',
  'user',
  'user_id',
  'clash_mode',
  'network_type',
  'network_is_expensive',
  'network_is_constrained',
  'wifi_ssid',
  'wifi_bssid',
  'interface_address',
  'network_interface_address',
  'default_interface_address',
  'preferred_by',
  'rule_set',
  'rule_set_ip_cidr_match_source',
  'rule_set_ipcidr_match_source',
] as const

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
