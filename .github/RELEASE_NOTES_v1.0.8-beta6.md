# S-UI-X Extended v1.0.8-beta6

This beta adds panel-managed regional rule-sets, nested route and DNS conditions, and a broad set of UI fixes. It also updates the bundled core to `sing-box-extended v1.13.14-extended-2.5.4`.

The panel downloads regional `.srs` files before saving a preset, checks their size and binary format, then stores local paths in the sing-box configuration. Direct downloads accept HTTPS only and block private network targets, redirects to unsafe addresses, and DNS rebinding. File and manifest updates are transactional, so a failed download or daily refresh leaves the last working files in place.

Route and DNS editors now support nested `and` and `or` groups while preserving action fields. The backend pinpoints invalid descendants and enforces request, depth, and node limits.

List pages now distinguish an empty collection from filtered results and provide relevant setup actions. Drawers explain why Save is blocked. Login, Settings, the Nexus shell, and Overview behave better on narrow screens; the shell also adds accessible logout controls, remembers the collapsed sidebar, and announces server status through a dedicated live region.

The new core includes the WireGuard endpoint IPC used by AmneziaWG provisioning. It closes rule-set files and zlib readers after use, rejects truncated files, bad checksums, and trailing data, and keeps the REALITY client version at `26.7.11` for current Xray defaults.

The default Compose image and frontend package version now use `v1.0.8-beta6`; both still pointed at beta4. nftables permissions remain opt-in, and this release needs no database migration.

This is a beta. Test AmneziaWG provisioning, one rule-set preset, and the main panel workflows on a non-critical server before upgrading production.

Full release notes: [`docs/releases/v1.0.8-beta6.md`](../docs/releases/v1.0.8-beta6.md).
