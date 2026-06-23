# s-ui-x-extended v1.0.0

First stable release — the s-ui-x web panel on the `sing-box-extended` core
(`shtorm-7/sing-box-extended`, a fork of `SagerNet/sing-box`). This GA
consolidates the `1.0.0-beta1`…`1.0.0-beta3` line into one stable build.

## Highlights

- **Recommended defaults pre-filled across the admin panel.** Creating inbounds,
  outbounds, endpoints, DNS servers, services, TLS templates, transports and
  routing rules now opens with security-first, ready-to-use values instead of
  blank fields — TUIC / Naive / TrustTunnel default to `bbr` congestion control,
  VLESS / VMess to `xudp` packet encoding, VMess to `auto` security, OpenVPN to
  `AES-256-GCM` / `SHA256`, Sudoku to the core's recommended AEAD / padding, and
  new TLS templates to `min_version: 1.3`.
- **Typed inputs that prevent mistakes.** Fixed-value fields are now dropdowns
  (congestion controls, Mieru transport / multiplexing, Sudoku AEAD / mask modes,
  SOCKS version, Tun stack, TLS cipher suites, …), so an invalid token can no
  longer be typed by hand; free-text fields offer editable suggestion comboboxes
  with a sensible default (Go durations, byte-size quotas, bandwidth speeds,
  listen addresses, time zones, NTP servers, DNS resolvers, SSH versions /
  algorithms, health-check URLs, …) while still accepting custom input.
- **Extended protocol & transport set.** OpenVPN, MASQUE, MTProxy, TrustTunnel,
  WireGuard/AmneziaWG, CCM/OCM, DHCP, QUIC, mKCP/XHTTP, providers, and
  rate/traffic/bandwidth/connection limiters — plus the inherited security and
  reliability hardening from the core.
- **Full protocol tag set in every build path.** Prebuilt Linux tarballs and the
  Windows packages now ship the same protocols as the Docker image / `build.sh`
  (WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS,
  CCM/OCM/OOMKiller) — no more `not included in this build, rebuild with -tags ...`
  stub at runtime. Only Naive (cronet/CGO) still varies by platform — present on
  Linux amd64/arm64/armv7/armv6/386, Docker and Windows amd64; absent on
  armv5/s390x and Windows arm64.
- **Fix — `database` token scope.** API tokens scoped to `database` now actually
  grant database export/import (`getdb`/`importdb`) and x-ui / 3x-ui migration
  (`import-xui`), instead of being unintentionally admin-only.
- **Docs.** `docs/scope-matrix.md` documents all six token scopes (`admin`,
  `read`, `write`, `database`, `telegram`, `observability`) with per-endpoint
  gates; the README was restructured and fact-checked (HTTP API, migration,
  backup, Telegram, paid subscriptions, security & hardening, monitoring,
  transports/TLS, build matrix).

All option tokens were verified against the `sing-box-extended` core. Secrets are
never hard-coded (UUIDs / passwords / keys are still generated), and camouflage
targets (Reality dest / ShadowTLS handshake / SNI) are intentionally left empty.

See [`CHANGELOG.md`](CHANGELOG.md) for the full list and [`SECURITY.md`](SECURITY.md)
for supply-chain and hardening notes.

This is the first stable release — still review `SECURITY.md` before exposing the
panel.
