import { Dial } from "./dial"

export interface tls {
  id: number
  name: string
  server: iTls
  client: oTls
}

export interface iTls {
  enabled?: boolean
  server_name?: string
  alpn?: string[]
  min_version?: string
  max_version?: string
  cipher_suites?: string[]
  curve_preferences?: string[]
  certificate?: string[]
  certificate_path?: string
  key?: string[]
  key_path?: string
  client_authentication?: string
  client_certificate?: string[]
  client_certificate_path?: string[]
  client_certificate_public_key_sha256?: string[]
  // 1.14 replaced the inline acme block with certificate_provider; the core
  // rejects acme and certificate_provider together, and the startup migration
  // rewrites stored blobs. The editor only writes certificate_provider.
  certificate_provider?: certificateProvider
  ech?: ech
  reality?: reality
  store?: 'mozilla' | 'chrome'
  kernel_tx?: boolean
  kernel_rx?: boolean
  handshake_timeout?: string
}

export interface certificateProvider {
  type: 'acme'
  tag?: string
  domain: string[]
  data_directory?: string
  default_server_name?: string
  email?: string
  provider?: string
  account_key?: string
  key_type?: 'ed25519' | 'p256' | 'p384' | 'rsa2048' | 'rsa4096'
  profile?: string
  disable_http_challenge?: boolean
  disable_tls_alpn_challenge?: boolean
  alternative_http_port?: number
  alternative_tls_port?: number
  external_account?: {
    key_id: string
    mac_key: string
  }
  dns01_challenge?: {
    provider: string
    [key: string]: string
  }
}

export interface ech {
  enabled: boolean
  key?: string[]
  key_path?: string
}

interface realityHanshake extends Dial {
  server: string
  server_port: number
}

export interface reality {
  enabled: boolean
  handshake: realityHanshake
  private_key: string
  short_id: string[]
  max_time_difference?: string
}

export const defaultInTls: iTls = {
  alpn: ['h3', 'h2', 'http/1.1'],
  min_version: "1.2",
  max_version: "1.3",
  cipher_suites: [],
}

export interface oTls {
  enabled?: boolean
  engine?: 'go' | 'apple' | 'windows'
  disable_sni?: boolean
  server_name?: string
  insecure?: boolean
  alpn?: string[]
  min_version?: string
  max_version?: string
  cipher_suites?: string[]
  curve_preferences?: string[]
  certificate?: string
  certificate_path?: string
  certificate_public_key_sha256?: string[]
  client_certificate?: string[]
  client_certificate_path?: string
  client_key?: string[]
  client_key_path?: string
  fragment?: boolean
  fragment_fallback_delay?: string
  record_fragment?: boolean
  spoof?: string
  spoof_method?: string
  kernel_tx?: boolean
  kernel_rx?: boolean
  handshake_timeout?: string
  ech?: {
    enabled: boolean
    pq_signature_schemes_enabled?: boolean
    dynamic_record_sizing_disabled?: boolean
    config?: string[]
    config_path?: string
    query_server_name?: string
  },
  utls?: {
    enabled: boolean
    fingerprint: string
  },
  reality?: {
    enabled: boolean
    public_key: string
    short_id: string
  }
}

export const defaultOutTls: oTls = {
  alpn: ['h3', 'h2', 'http/1.1'],
  min_version: "1.2",
  max_version: "1.3",
  cipher_suites: [],
  utls: {
    enabled: true,
    fingerprint: "chrome",
  },
  reality: {
    enabled: true,
    public_key: "",
    short_id: "",
  },
  ech: {
    enabled: true,
    pq_signature_schemes_enabled: false,
    dynamic_record_sizing_disabled: false,
    config_path: "",
  }
}
