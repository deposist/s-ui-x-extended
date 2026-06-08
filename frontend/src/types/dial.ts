export type DomainResolveOptions = string | {
  server: string
  strategy?: '' | 'prefer_ipv4' | 'prefer_ipv6' | 'ipv4_only' | 'ipv6_only'
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string
}

export interface Dial {
  detour?: string
  bind_interface?: string
  inet4_bind_address?: string
  inet6_bind_address?: string
  bind_address_no_port?: boolean
  protect_path?: string
  routing_mark?: number
  reuse_addr?: boolean
  netns?: string
  connect_timeout?: string
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  fallback_delay?: string
  domain_resolver?: DomainResolveOptions
  network_strategy?: 'default' | 'fallback' | 'hybrid'
  network_type?: ('wifi' | 'cellular' | 'ethernet' | 'other')[]
  fallback_network_type?: ('wifi' | 'cellular' | 'ethernet' | 'other')[]
  disable_tcp_keep_alive?: boolean
  tcp_keep_alive?: string
  tcp_keep_alive_interval?: string
}
