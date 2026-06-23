# s-ui-x-extended v1.0.0-beta6

Pre-release. Rebases the panel onto s-ui-x v1.5.10-beta3 while keeping the
sing-box-extended core. Every protocol, service, routing feature, DNS option,
subscription format, and management tool from upstream 1.5.10-beta3 is now
available on the extended core, alongside the extended-protocol set that was
already present (Mieru, Sudoku, TrustTunnel, SSH, MTProxy, MASQUE, OpenVPN,
Bond, Failover, Fallback, limiters, VPN endpoints, providers, profiler).

## Highlights

- **Upstream parity.** The panel now matches s-ui-x v1.5.10-beta3 feature-for-
  feature: Nexus and Classic UI layouts, routing/DNS preset drawers with preview,
  Config Doctor, in-panel self-update, IP-address TLS certificates, hot-reload
  without core restart, paid-subscription management, Telegram notifications,
  subscription caching, failover live status, and more.
- **Extended core preserved.** All sing-box-extended replace directives stay in
  go.mod. Extended protocols (Mieru, Sudoku, TrustTunnel, SSH inbound, MTProxy,
  MASQUE, OpenVPN, Bond, Failover, Fallback, bandwidth/connection/traffic/rate
  limiters, VPN server/client endpoints, parser outbound, SDNS, DNS Fallback,
  DNS Resolved, VLESS encryption, mKCP/XHTTP transports, unified_delay,
  profiler service) remain selectable and functional.
- **Provider manager.** The provider subsystem (inline/local/remote providers)
  is integrated into the upstream codebase: database model, backup/restore,
  service layer, config builder, core box.go manager, API endpoints, and
  frontend types/store.
- **Branding.** Logo and product name read "S-UI-X Extended" throughout the
  panel, scripts, and documentation.
- **Version logic.** The version stamping and migration logic now handles the
  extended fork's own versioning line (1.0.0-betaN) alongside upstream schema
  versions (1.4.x, 1.5.x) without false downgrade guards.

## Upgrade notes

No manual database migration is needed. The binary runs the normal migration
chain on startup. If upgrading from a previous s-ui-x-extended beta, the
settings.version row is updated to 1.0.0-beta6 automatically.

See [`CHANGELOG.md`](CHANGELOG.md) for the full history and
[`SECURITY.md`](SECURITY.md) for supply-chain and hardening notes. This is a
pre-release. Review `SECURITY.md` before exposing the panel.
