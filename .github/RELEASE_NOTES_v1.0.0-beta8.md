# s-ui-x-extended v1.0.0-beta8

Pre-release. This release follows v1.0.0-beta7 and focuses on panel support for sing-box-extended control-plane features and option coverage.

## Changes since beta7

- Added panel-owned outbound groups and provider-backed group membership for selector, URLTest, fallback, and failover flows.
- Added failover state, outbound health, provider health, bounded realtime payloads, and a Nexus overview widget for operational health.
- Added explicit all-down failover policies. The default keeps the current member. Direct fallback is available only when selected.
- Added panel support for outbound `block`, native core failover outbound, inbound `bond`, and native core failover inbound.
- Added shared inbound advanced fields, outbound `domain_strategy`, endpoint advanced fields, and protocol-specific option coverage in TypeScript and UI where safe to edit.
- Added provider entries to `/api/capabilities` and frontend generated capability types.
- Added a generated option coverage matrix and a drift test against the sing-box-extended option structs. The current matrix has zero unexplained missing fields.
- Kept the core dependency unchanged. All work is in the panel layer.

## Upgrade notes

No manual database migration is needed. The binary runs the normal migration chain on startup. If upgrading from a previous s-ui-x-extended beta, the settings.version row is updated to `1.0.0-beta8` automatically.

Panel-managed failover does not provide generic session recovery. It switches new connections; existing sessions may break when the active member changes.

See `CHANGELOG-EN.md`, `CHANGELOG-RU.md`, and `CHANGELOG-ZH.md` for the full history. This is a pre-release. Review `SECURITY.md` before exposing the panel.
