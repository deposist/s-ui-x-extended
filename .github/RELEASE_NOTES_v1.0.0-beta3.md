# s-ui-x-extended v1.0.0-beta3

Third public beta — the s-ui-x web panel on the `sing-box-extended` core
(`shtorm-7/sing-box-extended`, a fork of `SagerNet/sing-box`).

## Highlights

- **Recommended defaults pre-filled across the admin panel.** Creating inbounds,
  outbounds, endpoints, DNS servers, services, TLS templates, transports and
  routing rules now starts from security-first, ready-to-use values instead of
  blank fields — TUIC / Naive / TrustTunnel default to `bbr` congestion control,
  VLESS / VMess to `xudp` packet encoding, VMess to `auto` security, OpenVPN to
  `AES-256-GCM` / `SHA256`, Sudoku to the core's recommended AEAD / padding, and
  new TLS templates to `min_version: 1.3`.
- **Dropdowns for fixed-value fields.** Parameters that accept only a fixed set
  of values are now proper dropdowns (congestion controls, Mieru transport /
  multiplexing, Sudoku AEAD / mask modes, SOCKS version, Tun stack, TLS cipher
  suites, …), so an invalid token can no longer be typed by hand.
- **Editable suggestion fields for free text.** Free-text parameters with common
  values now offer an editable combobox with suggestions plus a sensible default
  (Go durations, byte-size quotas, bandwidth speeds, listen addresses, time
  zones, NTP servers, DNS resolvers, SSH versions / algorithms, health-check
  URLs, …) while still accepting custom input.

All option tokens were verified against the `sing-box-extended` core. Secrets are
never hard-coded (UUIDs / passwords / keys are still generated), and camouflage
targets (Reality dest / ShadowTLS handshake / SNI) are intentionally left empty.

See [`CHANGELOG.md`](CHANGELOG.md) for the full list and [`SECURITY.md`](SECURITY.md)
for supply-chain and hardening notes.

This is a beta build — review `SECURITY.md` before exposing the panel.
