# s-ui-x-extended v1.0.0-beta7

Pre-release. This release builds on v1.0.0-beta6. It keeps the s-ui-x v1.5.10-beta3 rebase and the sing-box-extended core, then fixes follow-up panel issues found after the beta6 branch push.

## Changes since beta6

- Fixed a frontend loop that could keep showing `Invalid login` together with `Error: CSRF token was not returned` after the session was lost or expired. The panel now handles an invalid session with a local logout, clears the cached CSRF token, and redirects to the login page without trying to call the CSRF-protected logout endpoint first.
- CSRF token loading now preserves backend errors such as `Invalid login` instead of replacing them with the misleading `CSRF token was not returned` message.
- Repeated `Invalid login` responses from polling or parallel API requests are collapsed into one notification until the next successful login.
- The panel self-update checker now reads releases from `deposist/s-ui-x-extended`. The download URL already used the extended repository; the GitHub API URL now matches it.
- Extended protocol editors are available in both UI modes. Classic modals, Nexus drawers, and shared endpoint, DNS, and client dialogs now expose the extended options restored from the extended fork.

## Included from beta6

- s-ui-x v1.5.10-beta3 UI and feature parity on the extended core.
- sing-box-extended replace directives are preserved in go.mod.
- Extended protocols remain available: Mieru, Sudoku, TrustTunnel, SSH inbound, MTProxy, MASQUE, OpenVPN, Bond, Failover, Fallback, bandwidth, connection, traffic, and rate limiters, VPN server and client endpoints, parser outbound, SDNS, DNS Fallback, DNS Resolved, VLESS encryption, mKCP/XHTTP transports, unified_delay, providers, and profiler.
- Provider support is integrated across database migration, backup and restore, service layer, config builder, core runtime, API, and frontend types/store.
- Branding remains `S-UI-X Extended`.
- Version and migration logic supports the fork's `1.0.0-betaN` line alongside upstream schema versions.

## Upgrade notes

No manual database migration is needed. The binary runs the normal migration chain on startup. If upgrading from a previous s-ui-x-extended beta, the settings.version row is updated to `1.0.0-beta7` automatically.

See `CHANGELOG-EN.md`, `CHANGELOG-RU.md`, and `CHANGELOG-ZH.md` for the full history. This is a pre-release. Review `SECURITY.md` before exposing the panel.
