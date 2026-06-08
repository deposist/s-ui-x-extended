// Centralized, backend-verified option lists and "recommended" presets used to
// pre-fill admin create-forms with security-first defaults, to populate fixed-value
// dropdowns (v-select) and to suggest editable values in comboboxes (v-combobox).
//
// Tokens here were verified against the Go backend
// (github.com/shtorm-7/sing-box-extended v1.13.12-extended-2.4.0). Do NOT add tokens
// that are not accepted by the core — an invalid enum breaks config save/start.

/* ───────────── Protocol enums (fixed value sets → v-select) ───────────── */

// TrustTunnel congestion controller. Verified: transport/trusttunnel/quic.go switch.
// NOTE: "new_reno" is NOT valid for trusttunnel (it is "reno"); bbr_profile is a no-op
// in this core version, so it is intentionally NOT pre-filled anywhere.
export const trustTunnelCongestion = ['bbr', 'bbr_standard', 'bbr2', 'bbr2_variant', 'cubic', 'reno']

// TUIC congestion control. Type union: "cubic" | "new_reno" | "bbr".
export const tuicCongestion = ['bbr', 'cubic', 'new_reno']
export const tuicUdpRelayMode = ['native', 'quic']

// Naive QUIC congestion control. Verified: protocol/naive/outbound.go switch.
export const naiveCongestion = ['bbr', 'bbr2', 'cubic', 'reno']

// Mieru. Verified: transport must be TCP/UDP; multiplexing must be a MultiplexingLevel enum.
export const mieruTransport = ['TCP', 'UDP']
export const mieruMultiplexing = [
  'MULTIPLEXING_DEFAULT',
  'MULTIPLEXING_OFF',
  'MULTIPLEXING_LOW',
  'MULTIPLEXING_MIDDLE',
  'MULTIPLEXING_HIGH',
]

// Sudoku. Verified: transport/sudoku/config.go + crypto/aead.go.
export const sudokuAeadMethods = ['chacha20-poly1305', 'aes-128-gcm', 'none']
export const sudokuHttpMaskMode = ['legacy', 'stream', 'poll', 'auto', 'ws']
export const sudokuHttpMaskMultiplex = ['off', 'auto', 'on']

// Tun stack. Fixed set per sing-box.
export const tunStacks = ['system', 'gvisor', 'mixed']

// MTProxy prefer_ip (editable combobox — keep free in case core extends it).
export const mtproxyPreferIp = ['prefer-ipv4', 'prefer-ipv6', 'ipv4-only', 'ipv6-only']

// SOCKS outbound version.
export const socksVersions = ['4', '4a', '5']

// VMess security (outbound). "auto" lets the core pick the strongest AEAD.
export const vmessSecurity = ['auto', 'aes-128-gcm', 'chacha20-poly1305', 'none', 'zero']

// VLESS UDP packet encoding. Verified: protocol/vless/outbound.go ("packetaddr"/"xudp").
export const vlessPacketEncoding = ['none', 'packetaddr', 'xudp']
export const vmessPacketEncoding = ['', 'packetaddr', 'xudp']

// VLESS flow (per-client). Only "xtls-rprx-vision" or empty.
export const vlessFlows = ['xtls-rprx-vision', '']

// Shadowsocks methods (full valid set, incl. SIP022 2022-blake3-*).
export const shadowsocksMethods = [
  '2022-blake3-aes-256-gcm',
  '2022-blake3-aes-128-gcm',
  '2022-blake3-chacha20-poly1305',
  'aes-256-gcm',
  'aes-192-gcm',
  'aes-128-gcm',
  'chacha20-ietf-poly1305',
  'xchacha20-ietf-poly1305',
  'none',
]

/* ───────────── TLS (shared between InTLS/OutTLS/Tls modal) ───────────── */

export const tlsVersions = ['1.0', '1.1', '1.2', '1.3']

export const tlsAlpn = [
  { title: 'H3', value: 'h3' },
  { title: 'H2', value: 'h2' },
  { title: 'Http/1.1', value: 'http/1.1' },
]

export const tlsCurvePreferences = ['X25519MLKEM768', 'X25519', 'P256', 'P384', 'P521']

export const utlsFingerprints = [
  { title: 'Chrome', value: 'chrome' },
  { title: 'Firefox', value: 'firefox' },
  { title: 'Microsoft Edge', value: 'edge' },
  { title: 'Apple Safari', value: 'safari' },
  { title: '360', value: '360' },
  { title: 'QQ', value: 'qq' },
  { title: 'Apple IOS', value: 'ios' },
  { title: 'Android', value: 'android' },
  { title: 'Random', value: 'random' },
  { title: 'Randomized', value: 'randomized' },
]

export const tlsCipherSuites = [
  { title: 'AES128-GCM-SHA256', value: 'TLS_AES_128_GCM_SHA256' },
  { title: 'AES256-GCM-SHA384', value: 'TLS_AES_256_GCM_SHA384' },
  { title: 'CHACHA20-POLY1305-SHA256', value: 'TLS_CHACHA20_POLY1305_SHA256' },
  { title: 'ECDHE-ECDSA-AES128-GCM-SHA256', value: 'TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256' },
  { title: 'ECDHE-ECDSA-AES256-GCM-SHA384', value: 'TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384' },
  { title: 'ECDHE-RSA-AES128-GCM-SHA256', value: 'TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256' },
  { title: 'ECDHE-RSA-AES256-GCM-SHA384', value: 'TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384' },
  { title: 'ECDHE-ECDSA-CHACHA20-POLY1305-SHA256', value: 'TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256' },
  { title: 'ECDHE-RSA-CHACHA20-POLY1305-SHA256', value: 'TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256' },
]

// Well-known, reachable TLS 1.3 / CDN-fronted hosts suitable as Reality dest /
// ShadowTLS handshake / SNI camouflage. SUGGESTIONS ONLY — never force a default
// (a forced, well-known dest is fingerprintable and may go stale).
export const sniFrontHosts = [
  'www.microsoft.com',
  'www.apple.com',
  'www.cloudflare.com',
  'www.amazon.com',
  'aws.amazon.com',
  'dl.google.com',
  'www.icloud.com',
  'www.bing.com',
  'www.tesla.com',
]

/* ───────────── Editable comboboxes (free text + suggestions) ───────────── */

// Go duration strings.
export const durationPresets = ['1s', '5s', '10s', '30s', '1m', '5m', '10m', '30m', '1h', '6h', '12h', '1d', '7d']

// Byte-size quotas (sing-box size strings).
export const sizePresets = ['100MB', '500MB', '1GB', '5GB', '10GB', '50GB', '100GB', '500GB', '1TB']

// Bandwidth speeds.
export const speedPresets = ['512KB', '1MB', '2MB', '5MB', '10MB', '20MB', '50MB', '100MB']

// Listen addresses.
export const listenAddresses = ['::', '0.0.0.0', '127.0.0.1']

// Privacy/anti-censorship DNS resolvers (DoH/DoT/plain) — editable suggestions.
export const dnsResolvers = ['1.1.1.1', '1.0.0.1', '8.8.8.8', '8.8.4.4', '9.9.9.9', 'dns.google', 'cloudflare-dns.com', 'dns.quad9.net']

// DoH/DoH3 query path.
export const dohPaths = ['/dns-query']

// Curated IANA time zones.
export const timeZones = [
  'UTC', 'Europe/Moscow', 'Europe/Kyiv', 'Europe/Berlin', 'Europe/London', 'Europe/Paris',
  'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'America/Sao_Paulo',
  'Asia/Shanghai', 'Asia/Hong_Kong', 'Asia/Singapore', 'Asia/Tokyo', 'Asia/Seoul',
  'Asia/Dubai', 'Asia/Tehran', 'Asia/Kolkata', 'Australia/Sydney',
]

// NTP servers reachable in most regions.
export const ntpServers = ['time.apple.com', 'time.google.com', 'time.cloudflare.com', 'pool.ntp.org', 'time.windows.com']

// HTTP transport methods.
export const httpMethods = ['GET', 'PUT', 'POST']

// gRPC service name suggestions (must match peer).
export const grpcServiceNames = ['GunService', 'TunService', 'grpc']

// Provider subscription User-Agent suggestions (keep "sing-box" as the safe default).
export const providerUserAgents = ['sing-box', 'clash', 'clash-verge', 'v2rayN', 'SFA', 'SFI', 'mihomo']

// Provider / outbound health-check probe URLs.
export const healthCheckUrls = [
  'https://www.gstatic.com/generate_204',
  'https://cp.cloudflare.com/generate_204',
  'http://www.google.com/generate_204',
  'http://www.gstatic.com/generate_204',
]

// SSH client/server identification strings (cosmetic fingerprinting).
export const sshVersions = ['SSH-2.0-OpenSSH_9.7', 'SSH-2.0-OpenSSH_9.6', 'SSH-2.0-OpenSSH_9.0', 'SSH-2.0-OpenSSH_8.9']

// SSH host key algorithms (modern first).
export const sshHostKeyAlgorithms = [
  'ssh-ed25519',
  'rsa-sha2-512',
  'rsa-sha2-256',
  'ecdsa-sha2-nistp256',
  'ecdsa-sha2-nistp384',
  'ecdsa-sha2-nistp521',
]

// OpenVPN data ciphers / auth digests (editable; must match server).
export const openvpnCiphers = ['AES-256-GCM', 'AES-128-GCM', 'CHACHA20-POLY1305', 'AES-256-CBC', 'AES-128-CBC']
export const openvpnAuthDigests = ['SHA256', 'SHA384', 'SHA512', 'SHA1']

// Common torrc option keys (free-solo; torrc allows arbitrary keys).
export const torrcKeys = ['ClientOnly', 'UseBridges', 'Bridge', 'EntryNodes', 'ExitNodes', 'ExcludeNodes', 'StrictNodes', 'SocksPort']

// WireGuard / endpoint DNS resolver suggestions.
export const wireguardDns = ['1.1.1.1', '8.8.8.8', '9.9.9.9']

/* ───────────── Recommended scalar defaults (security-first) ───────────── */

export const RECOMMENDED = {
  tlsMinVersion: '1.3',
  tlsMaxVersion: '1.3',
  tuicCongestion: 'bbr',
  tuicUdpRelayMode: 'native',
  naiveCongestion: 'bbr',
  trustTunnelCongestion: 'bbr',
  vlessPacketEncoding: 'xudp',
  vmessPacketEncoding: 'xudp',
  vmessSecurity: 'auto',
  sudokuAead: 'chacha20-poly1305',
  sudokuPaddingMin: 10,
  sudokuPaddingMax: 30,
  sudokuHandshakeTimeout: 5,
  openvpnCipher: 'AES-256-GCM',
  openvpnAuth: 'SHA256',
  healthCheckUrl: 'https://www.gstatic.com/generate_204',
  healthCheckInterval: '1m',
  healthCheckTimeout: '5s',
  wsPath: '/',
  httpMethod: 'GET',
}
