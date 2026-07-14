# Changelog (English)

All notable changes to this project are documented in this file.

This is the English-language changelog. See `CHANGELOG-RU.md` for Russian and
`CHANGELOG-ZH.md` for Simplified Chinese.

## [Unreleased]

- No unreleased changes.

## [1.0.2] - 2026-07-14 - stable 1.0.2 release

- Added managed personal AmneziaWG 2.0 devices with encrypted keys, per-tariff limits, Telegram delivery, QR codes, traffic accounting, key rotation, revocation, and crash-safe reconciliation.
- Managed endpoint peers are protected from manual edits. The core receives preshared keys only in memory.
- Fixed `s-ui` waiting on standard input before showing the menu when a saved language exists.
- Hardened release downloads with HTTPS-only curl retries, timeouts, and mandatory SHA-256 verification.

Full release notes: [`docs/releases/v1.0.2.md`](docs/releases/v1.0.2.md).

## [1.0.1] - 2026-07-07 - stable 1.0.1 release

Stable v1.0.1 promotes the v1.0.1 beta line and adds the Telegram Chat ID setup fix. No manual database or configuration migration is required.

- Added a Detect Chat ID action to Telegram settings. After the bot receives `/start` or any message, the panel can read Telegram updates and fill the Chat ID field.
- Detection works with a newly typed bot token or with the encrypted token already saved in the panel. Saved tokens are not returned to the browser.
- Telegram Test now saves changed Telegram settings before sending the test message, so a freshly entered or detected Chat ID is used immediately.
- Includes the v1.0.1 beta-line hardening: API token ownership checks, safer `subSecretRequired` default for fresh installs, forced password reset sessions, WARP/WireGuard reserved-field cleanup, and inbound sniff migration.

Full release notes: [`docs/releases/v1.0.1.md`](docs/releases/v1.0.1.md).

## [1.0.1-beta5] - 2026-07-02 - admin security hardening

Security hardening release. No manual database migration is required.

- API token delete and enable/disable now require token ownership.
- Fresh installs now default `subSecretRequired` to `true`; existing installs keep their saved setting.
- `force_password_reset=true` admins now receive a restricted session and must change the password before accessing the panel.
- API tokens for admins that must reset a password are rejected until the password changes.
- The login page now supports the forced password reset flow.

Full release notes: [`.github/RELEASE_NOTES_v1.0.1-beta5.md`](.github/RELEASE_NOTES_v1.0.1-beta5.md).

## [1.0.1-beta4] - 2026-06-30 - WARP endpoint reserved field fix

This beta keeps WARP and WireGuard endpoint output compatible with the sing-box-extended core used by this build. No manual database migration is required.

- WARP registration no longer writes `reserved` inside `peers[]`. The current endpoint schema rejects that field and returned `endpoints[0].peers[0].reserved: json: unknown field "reserved"` after save.
- Core config generation now strips unsupported `peers[].reserved` fields from WARP and WireGuard endpoints, so stale stored rows no longer break sing-box startup.
- The WARP form now reads reserved bytes safely from old rows without requiring `reserved` inside the peer object.
- Added backend regression tests for stored WARP and WireGuard endpoint rows with legacy `reserved` fields.

Full release notes: [`.github/RELEASE_NOTES_v1.0.1-beta4.md`](.github/RELEASE_NOTES_v1.0.1-beta4.md).

## [1.0.1-beta3] - 2026-06-30 - sing-box inbound sniff migration

Compatibility release for sing-box 1.11+. The database migration runs automatically.

- Inbound advanced settings no longer expose `sniff`, `sniff_override_destination`, or `sniff_timeout`, and Apply inbound recommendations no longer writes those deprecated inbound fields.
- Backend inbound JSON filters legacy rule-action fields before handing config to sing-box: `sniff`, `sniff_override_destination`, `sniff_timeout`, `domain_strategy`, and `udp_disable_domain_unmapping`.
- Existing inbound `sniff` settings migrate to scoped `route.rules` with `action: "sniff"`; `sniff_timeout` becomes `timeout`, `domain_strategy` becomes `action: "resolve"`, and `udp_disable_domain_unmapping` becomes `action: "route-options"`.
- `sniff_override_destination` is removed because sing-box has no equivalent route-action field.
- Migrated rules are inserted before existing routing rules and duplicate equivalent rules are skipped. The old `config.json` import path now preserves migrated inbound sniff-related settings during import.
- Option coverage marks the deprecated inbound sniff fields as intentionally hidden from panel TS types.

Full release notes: [`.github/RELEASE_NOTES_v1.0.1-beta3.md`](.github/RELEASE_NOTES_v1.0.1-beta3.md).

## [1.0.1-beta2] - 2026-06-29 - frontend guidance and documentation

Frontend and documentation release. No database migration is required.

- Dashboard: Traffic statistics now has a timezone selector in the KPI controls. It defaults to the browser timezone, persists the selected timezone in `localStorage`, formats chart labels as `YYYY-MM-DD HH:MM`, and shows the active timezone next to the range selector. The menu opens with a short default list and searches the full IANA list only while text is entered.
- Configuration forms: Outbound, Service, Endpoint, TLS, DNS server, DNS rule, and route rule forms now use field-level help and create-mode preset buttons where safe. Settings, Subscription JSON, and Subscription Clash show field help without apply buttons. Edit forms are not auto-filled.
- Inbound setup: add inbound forms use protocol-aware guidance. VLESS has a dedicated preset, other supported protocols get presets only where defaults are safe, and deployment-specific protocols show guidance without a preset button.
- Inbound safety: Shadowsocks and Sudoku credentials are generated when new inbounds are created, ShadowTLS passwords are generated only for version 2, and TLS/Reality templates are filtered so Reality is shown only for VLESS and Trojan.
- Locales and tests: new recommendation hints were added for all supported UI locales. Tests cover locale coverage, recommendation helpers, inbound credential generation, TLS template compatibility, and Dashboard timezone selection.
- Documentation: README was shortened into an English entry point, `README-RU.md` was added, supported protocols were refreshed from the capability matrix and configuration docs, and README beta links now point to `v1.0.1-beta2`.

Full release notes: [`.github/RELEASE_NOTES_v1.0.1-beta2.md`](.github/RELEASE_NOTES_v1.0.1-beta2.md).

## [1.0.1-beta1] - 2026-06-28 - hardening

Hardening release. No database migration is required.

- Shell: removed `eval` from custom version installer; `s-ui uri` no longer contacts external IP services; all positional arguments quoted; `read -r` throughout; tilde paths corrected.
- Panel: `trafficAge=0` is now rejected (prevented silent data loss); SubSecret generation is atomic under concurrency; import preserves client link mappings for skipped inbounds; broadcast delivery and UI count use the same active-client filter.
- API: Telegram bot parses `ok:false` in responses and redacts tokens from error messages; error responses no longer leak internal details.
- Validation: `paidSubCurrency` allowlist matches the frontend picker; `paidSubAutoInbounds` rejects duplicates and missing inbounds; `paidSubExternalUrlTemplate` blocks local IPs and control characters; `saveBinding` validates `tgUserId`.
- Frontend: Nexus is now the only selectable/default UI mode under the Nexus gate; the visible mode switch to classic was removed; QUIC transport editor restored as a deprecated-visible option for legacy `v2rayquic` compatibility; bulk outbound save correctly awaits all operations; Settings restart button uses `window.location.reload()`; polling loop bounded; tariff deletion requires confirmation; TLS PEM parsing uses substring match; rule import validates URL protocol; inbound dirty tracking includes user list changes.
- Cron jobs propagate context and cancel on shutdown; WARP retries respect cancellation; IP monitor salt bootstrap hardened; protocol coverage additions from the prior implementation wave remain included in this release.
- Auto-registered client names are verified against the database before use.

Full release notes: [`.github/RELEASE_NOTES_v1.0.1-beta1.md`](.github/RELEASE_NOTES_v1.0.1-beta1.md).




## [1.0.0-hotfix4] - 2026-06-26 - stable hotfix

Hotfix for the first stable release. No manual database migration is required.

- Fixed the client config editor not showing the MTProxy `secret` field. The generic config loop only rendered `password`, `uuid`, and `auth_str`, so MTProxy credentials were invisible and uneditable.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-hotfix4.md`](.github/RELEASE_NOTES_v1.0.0-hotfix4.md).

## [1.0.0-hotfix3] - 2026-06-26 - stable hotfix



Hotfix for the first stable release. No manual database migration is required.



- Fixed extended protocol inbounds (Mieru, TrustTunnel, SSH, MTProxy, and others) failing with `users is empty` when existing clients lacked per-protocol credentials. The backend now backfills missing config blocks when clients are linked to an inbound.

- Hardened credential backfill error handling: UUID, shadowsocks password, and mtproxy secret generation errors are now propagated instead of silently producing empty values.



Full release notes: [`.github/RELEASE_NOTES_v1.0.0-hotfix3.md`](.github/RELEASE_NOTES_v1.0.0-hotfix3.md).



## [1.0.0-hotfix2] - 2026-06-26 - stable hotfix

Hotfix for the first stable release. No manual database migration is required.

- Fixed stale frontend cache recovery after reinstalling the panel.
- Fixed Mieru inbound generation when enabled clients are missing `config.mieru`.
- Added missing English and Russian labels for extended protocol editors, including Sudoku, Mieru, MTProxy, TrustTunnel, MASQUE, OpenVPN, provider, VPN, xHTTP, and mKCP fields.
- Restored the frontend asset filename prefixes checked by release tests.
- Cleared service linter findings in Doctor, IP certificate storage, and panel self-update code.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-hotfix2.md`](.github/RELEASE_NOTES_v1.0.0-hotfix2.md).

## [1.0.0-hotfix1] - 2026-06-26 - stable hotfix

Hotfix for the first stable release. No manual database migration is required.

- Fixed a login loop that could appear after reinstalling the panel in a browser with old cached assets.
- Fixed user injection for Mieru, TrustTunnel, SSH, and MTProxy inbounds, so sing-box receives `users` for these protocols.
- New Mieru, TrustTunnel, SSH, and MTProxy inbounds now select all existing clients by default and block saving when no client is selected.
- Restored the xHTTP and mKCP transport editors in the inbound UI.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-hotfix1.md`](.github/RELEASE_NOTES_v1.0.0-hotfix1.md).

## [1.0.0] - 2026-06-26 - first stable release

First stable release of s-ui-x-extended. No manual database migration is required.

- Shipped the extended protocol panel on the `shtorm-7/sing-box-extended` core with upstream s-ui-x `v1.5.10-beta7` panel parity.
- Included panel support for Sudoku, TrustTunnel, MASQUE, OpenVPN, MTProxy, WireGuard/WARP endpoints, DHCP DNS transport, outbound `block`, native inbound `bond`, and native core failover.
- Included panel-managed outbound groups, provider-backed group membership, failover runtime state, provider and outbound health snapshots, and Nexus operational health display.
- Included shared inbound advanced options, outbound `domain_strategy`, endpoint advanced fields, protocol-specific option coverage, and the generated option coverage matrix with zero unexplained missing fields.
- Built official release artifacts with the supported protocol tag set and verified Linux and Windows artifacts with runtime smoke checks for `/api/capabilities`.
- Added English and Russian configuration guides under `docs/`.
- Kept `shtorm-7/sing-box-extended` unchanged; release work stayed in the panel repository, build workflows, and documentation.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0.md`](.github/RELEASE_NOTES_v1.0.0.md).

## [1.0.0-beta9] - 2026-06-26 - full protocol tags in release builds

Pre-release. Follow-up to v1.0.0-beta8. No manual database migration is required.

- Updated Linux release builds to include the supported protocol tags declared by the panel capability manifest, including Sudoku, TrustTunnel, MASQUE, OpenVPN, MTProxy, WireGuard/WARP endpoints, and DHCP DNS transport.
- Updated Windows release builds and local Windows build scripts to use the same supported protocol tag set.
- Added a regression test that checks release workflows, local build scripts, `build.sh`, and the Dockerfile against `core/capabilities/protocols.json`.
- Kept `with_naive_outbound` conditional on Linux targets that prepare the cronet/naive toolchain.
- Kept `shtorm-7/sing-box-extended` unchanged; this release only fixes build and release packaging for the panel repository.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-beta9.md`](.github/RELEASE_NOTES_v1.0.0-beta9.md).

## [1.0.0-beta8] - 2026-06-26 - panel control plane and option coverage

Pre-release. Follow-up to v1.0.0-beta7. No manual database migration is required.

- Added panel-managed outbound groups with provider-backed membership, group preview, tag reference validation, and capability metadata.
- Added failover runtime state, outbound health, provider health, bounded realtime payloads, and a Nexus overview widget for operational health.
- Added explicit all-down failover policy handling. The default keeps the current member; direct fallback must be selected by an admin.
- Added panel support for outbound `block`, native core failover outbound, inbound `bond`, and native core failover inbound.
- Added shared inbound advanced options, outbound `domain_strategy`, endpoint advanced fields, and protocol-specific option coverage in panel TypeScript and UI where safe to edit.
- Added provider entries to `/api/capabilities` and generated frontend capability types.
- Added a generated option coverage matrix and drift test against sing-box-extended option structs. The matrix now has zero unexplained `missing` entries.
- Kept `shtorm-7/sing-box-extended` unchanged; all new behavior is implemented in the panel layer.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-beta8.md`](.github/RELEASE_NOTES_v1.0.0-beta8.md).

## [1.0.0-beta7] - 2026-06-23 - session handling and extended UI coverage

Pre-release. Follow-up to v1.0.0-beta6. No manual database migration is required.

- Fixed a frontend loop that could keep showing `Invalid login` and `Error: CSRF token was not returned` after a lost or expired session. Invalid sessions now trigger a local logout, clear the cached CSRF token, and return the user to the login page without first calling the CSRF-protected logout endpoint.
- CSRF token loading now preserves backend errors such as `Invalid login` instead of replacing them with a missing-token message.
- Repeated invalid-session responses from polling or parallel requests are collapsed into one notification until the next successful login.
- The panel self-update checker now queries `deposist/s-ui-x-extended`, not the old `deposist/s-ui-x` repository.
- Extended protocol editors are available in both Classic and Nexus interfaces.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-beta7.md`](.github/RELEASE_NOTES_v1.0.0-beta7.md).

## [1.5.10-beta7] - 2026-06-25 - exact dashboard traffic totals

Seventh beta of the 1.5.10 line. This update fixes the Nexus Dashboard Traffic statistics KPI. No manual migration is required.

- Added `GET /api/stats/traffic` for dashboard traffic summaries with exact bucket sums.
- The Traffic statistics KPI now reads the new summary endpoint instead of combining per-inbound `/api/stats` responses in the browser.
- Download and upload totals now come from `SUM(traffic)` over the selected period, so long ranges no longer use average-downsampled `/api/stats` rows.
- The dashboard summary covers all historical inbound traffic for the selected period, including traffic from inbounds that were later disabled or removed.
- Added an index on `stats(resource, date_time)` for the all-inbound time-range query.

Full release notes: [`docs/releases/v1.5.10-beta7.md`](docs/releases/v1.5.10-beta7.md).

## [1.5.10-beta6] - 2026-06-24 - dashboard card heights and traffic history

Sixth beta of the 1.5.10 line. This is a Nexus dashboard update. No database, API, or sing-box configuration migration is required.

- Top clients now shows up to 10 clients and scrolls inside a fixed-height card aligned with System status.
- Recent events now uses the same fixed card height and scrolls inside the card when the event list is longer.
- System status is shorter and sized around its 8 current values.
- The first dashboard KPI now shows Traffic statistics instead of Live traffic. It reads existing `/api/stats` data for enabled inbound tags and plots download and upload separately.
- The traffic timeline selector now supports 1 hour, 6 hours, 12 hours, 24 hours, 7 days, and 30 days.
- Traffic history still depends on Traffic Maximum Age. Set it above `0` to save stats; `0` disables historical stats.
Full release notes: [`docs/releases/v1.5.10-beta6.md`](docs/releases/v1.5.10-beta6.md).

## [1.5.10-beta5] - 2026-06-24 - Nexus dashboard and Settings layout refresh

Fifth beta of the 1.5.10 line. This is a frontend release. No database, API, or sing-box configuration migration is required.

- Reworked the Nexus dashboard spacing and card alignment across KPI cards, overview panels, protocol summaries, recent events, system status, and dense tables.
- Updated Top clients with real summary totals, online count, total client count, compact traffic columns, and a link to the Clients page.
- Added a local time-window selector to the live traffic KPI for 1 minute through 24 hours, using existing realtime samples.
- Updated the Nexus sidebar branding to S-UI-X and added CPU/RAM percentages to the lower-left server status block when the status API returns those metrics.
- Added Emerald and Dracula Nexus palettes in dark and light variants, with better light-theme contrast.
- Redesigned Nexus Settings into card-based sections and moved sing-box Basics into Settings as the Basics (Singbox) tab. `/basics` now opens that tab, and the old Basics menu item was removed from Nexus and Classic navigation.
- Fixed the migrated Basics Save button state and ensured the loading state clears after failed config saves.

Full release notes: [`docs/releases/v1.5.10-beta5.md`](docs/releases/v1.5.10-beta5.md).

## [1.0.0-beta6] - 2026-06-23 - rebase onto s-ui-x v1.5.10-beta3 with extended core

Pre-release. Rebases the panel onto s-ui-x v1.5.10-beta3 while keeping the sing-box-extended core. Every upstream feature is now available on the extended core, alongside the existing extended-protocol set.

- Upstream parity: Nexus and Classic UI, routing/DNS preset drawers, Config Doctor, in-panel self-update, IP-address TLS certificates, hot-reload without core restart, paid subscriptions, Telegram notifications, subscription caching, failover live status.
- Extended core preserved: all sing-box-extended replace directives stay in go.mod. Mieru, Sudoku, TrustTunnel, SSH, MTProxy, MASQUE, OpenVPN, Bond, Failover, Fallback, limiters, VPN endpoints, providers, profiler, VLESS encryption, mKCP/XHTTP, unified_delay remain functional.
- Provider manager integrated into the upstream codebase: database model, backup/restore, service, config builder, core box.go, API, frontend.
- Branding updated to "S-UI-X Extended" throughout.
- Version stamping and migration logic handles the fork's own versioning line alongside upstream schema versions.

Full release notes: [`.github/RELEASE_NOTES_v1.0.0-beta6.md`](.github/RELEASE_NOTES_v1.0.0-beta6.md).

## [1.5.10-beta3] - 2026-06-23 - fix regional preset drawer rendering and button state

Third beta of the 1.5.10 line. No database or configuration migration is required. Frontend tests, lint, and build pass.

- Fixed a bug where Vuetify components inside the Regional Presets drawer failed to resolve because of dynamically resolved components in JS render functions (bypassing the `vite-plugin-vuetify` compiler).
- Fixed a bug where the "Preview Changes" button was permanently locked (disabled) because the switches were not functional and the "disable presets" state was not considered a previewable state.
- Fixed layout and spacing issues in the Regional Presets drawer by using standard Vuetify cards, dividers, and typography for better visual separation.

Full release notes: [`docs/releases/v1.5.10-beta3.md`](docs/releases/v1.5.10-beta3.md).

## [1.5.10-beta2] - 2026-06-23 - RU/ZH preset drawer with preview and security warnings

Second beta of the 1.5.10 line. No manual database, API, or configuration migration is required. Frontend tests, lint, and build pass.

- The old flat preset gallery on the Rules and DNS pages is replaced by a side drawer. RU and ZH are independent cards with a Direct / Through proxy direction switch.
- A preview step shows what will be added, changed, and kept before applying. Security warnings appear when direction is Through proxy (DNS leak risk, route exposure risk).
- Per-region domain exceptions in Advanced options route excepted domains to the opposite direction.
- Preset-managed items are labeled in both nexus and classic interfaces (badge column in nexus, chip in classic).
- The preset catalog changed from three overlapping presets to four bidirectional ones (ru-direct, ru-proxy, zh-direct, zh-proxy). Preset-managed items use deterministic tag names; no unknown metadata fields are written to sing-box config.
- RU private ranges always route direct regardless of direction (safety constraint).
- Old RoutingDnsPresetGallery component removed.

Full release notes: [`docs/releases/v1.5.10-beta2.md`](docs/releases/v1.5.10-beta2.md).

## [1.5.10-beta1] - 2026-06-23 - performance fixes and safer large-database paths

First beta of the 1.5.10 line. No manual database, API, or configuration migration is required. The full Go suite, Go vet, frontend lint, frontend build, and frontend unit tests pass.

- Stats charts now prepare large result sets with one pass of bucket assignment after sorting, instead of scanning all rows for every bucket and direction. Stats writes now use explicit SQLite-safe batches.
- `/api/load` now reads the settings needed for a full reload through one snapshot and reads independent entities in parallel.
- Unencrypted local DB export now streams the prepared SQLite backup file to the browser instead of buffering the whole backup in memory. Encrypted backup downloads keep the existing whole-payload envelope path.
- Base, JSON, and Clash subscription outputs now use a short TTL cache that is cleared on config save. Only successful responses are cached.
- The frontend build now splits stable vendor chunks for Vue, Vuetify, and HTTP dependencies. The DateTime picker is lazy-loaded from client modals, and extra Moment locale imports were removed.

Full release notes: [`docs/releases/v1.5.10-beta1.md`](docs/releases/v1.5.10-beta1.md).

## [1.5.9-beta6] - 2026-06-18 - WireGuard endpoint editor opens reliably

Small follow-up to v1.5.9-beta5. No database, API, or configuration changes
are required.

- **Fix (UI): the WireGuard endpoint editor could open empty.** When adding a
  WireGuard endpoint, the editor sometimes showed only the type and tag because
  it rendered before its key data was ready. Key, address, port, and peer fields
  now load on the first open.

Full release notes: [`docs/releases/v1.5.9-beta6.md`](docs/releases/v1.5.9-beta6.md).

## [1.5.9-beta5] - 2026-06-17 - cleaner forms in the default interface

Small follow-up to v1.5.9-beta4. No database, API, or configuration changes
are required.

- **Fix (UI): cramped, overlapping fields in the default interface.** The add and
  edit forms were packed too tightly, so a field's label could touch or overlap
  the field, heading, or tab above it, especially when adding a client or a
  failover group. The forms now have enough vertical spacing.

Full release notes: [`docs/releases/v1.5.9-beta5.md`](docs/releases/v1.5.9-beta5.md).

## [1.5.9-beta4] - 2026-06-17 - failover group editor fix in the Nexus UI

Small follow-up to v1.5.9-beta3. No database, API, or configuration changes
are required. The affected frontend build, frontend tests, and Nexus editor
Playwright check pass.

- **Fix (UI): the Failover group editor was missing in the Nexus (default) UI.**
  The `failover` outbound type added in v1.5.9-beta3 was wired into the classic
  outbound modal but not into the Nexus drawer (`OutboundDrawer.vue`), so
  selecting *Failover* in the default UI opened an empty editor and still rendered
  the server/port fields. The Nexus drawer now mounts the Failover editor and
  hides server/port for the group. A Playwright e2e covers the Nexus UI path.

Full release notes: [`docs/releases/v1.5.9-beta4.md`](docs/releases/v1.5.9-beta4.md).

## [1.5.9-beta3] - 2026-06-17 - in-panel web update + automatic outbound failover

Two new capabilities since v1.5.9-beta2. No manual database, API, or
configuration changes are required. The affected Go packages and the frontend
test suite pass.

- **Feat (maintenance): update the panel from the web UI.** **Settings ->
  Maintenance** now has a *Panel updates* card. Pick a channel (*Main* tracks
  stable releases, *Beta* tracks the newest release including pre-releases), check
  for updates, and update in one click. The panel downloads the selected version, verifies it
  against the release's published SHA-256 over verified HTTPS, replaces itself,
  and restarts on the new version, with the previous binary backed up and an
  automatic rollback if a freshly-installed version repeatedly fails to start.
  Only admins can run it, and the action requires password re-entry. Every check
  and update attempt, including rejected ones, is audited. The password is never logged. Only newer
  versions are offered (no downgrades).
- **Feat (routing): automatic strict-priority outbound failover.** A new
  `failover` outbound type holds an ordered member list (the first is the
  primary, the rest are backups). An in-process manager probes each member over
  HTTPS (operator-chosen target host/IP and interval, default 30s) and switches
  the active member via the sing-box selector: immediate failover when the active
  member stops responding, hysteresis-gated failback to the highest-priority
  member once it recovers, and a direct/hold-senior fallback when every member is
  down. Use the group as **Routing -> Default Outbound** or in any rule. It
  behaves like any outbound, and the Outbounds list shows the currently-active
  member. No schema migration (the group assembles to a sing-box selector; a
  non-authoritative `failover_state` observability table is created idempotently
  at startup).

Full release notes: [`docs/releases/v1.5.9-beta3.md`](docs/releases/v1.5.9-beta3.md).

## [1.5.9-beta2] - 2026-06-16 - frontend CSS fixes + install/update download timeout

Small follow-up to v1.5.9-beta1. No API, database, or configuration changes; the
frontend suite is green.

- **Fix (UI): unreadable hint tooltips in the dark theme.** The (i) SettingInfo
  tooltips added in v1.5.9-beta1 were too muted to read in the Nexus dark theme
  (it sets `surface-variant` but no `on-surface-variant`). All tooltips now use a
  solid dark-background/light-text scheme that reads well in both themes.
- **Fix (UI): truncated floating field labels.** `persistent-placeholder` plus an
  append-inner (i) icon constrained the floating label, clipping short labels to
  "Add...", "P...", "Do...". Floating labels now size to their content.
- **Fix (install): installer/self-update could hang on a stalled mirror.**
  `install.sh` (the tarball and its `.sha256`) and the `s-ui.sh` self-update ran
  `wget` with no timeout, so one stuck `release-assets.githubusercontent.com` node
  could block the install for ~15 minutes. Downloads now use
  `--timeout=20 --tries=5 --retry-connrefused`.

## [1.5.9-beta1] - 2026-06-16 - security & reliability hardening + settings UX from a full codebase audit

Remediates the findings of a full codebase audit: 28 fixes across the Go
backend, the Vue frontend, and the build/install scripts, with the
deterministic toolchain (build, vet, staticcheck, gosec, govulncheck) clean and
the affected Go packages and the full frontend suite green. The two most serious
findings are integrity-class. The new stats index is created automatically on
startup; no manual database migration and no configuration changes are required.

- **Fix (integrity): half-applied transaction on a save panic.** The
  subscription/out-JSON TLS assembler `addTls` panicked on operator-supplied
  Reality/ECH blobs whose client and server halves were not in lockstep (nil-map
  write, bare type assertions). Because `ConfigService.Save` branched
  commit-versus-rollback only on the returned error, the panic committed a
  half-applied transaction and left the running core out of sync with the
  database. `addTls` is now total (nil-safe client map, comma-ok on every
  assertion); `Save` recovers any panic into a transaction rollback. Regression
  tests cover all panic vectors.
- **Fix (security): root RCE via TLS-disabled installer downloads.** `install.sh`
  and the `s-ui.sh` self-update fetched their artifacts - and, for install.sh,
  the `.sha256` validating the tarball - with `wget --no-check-certificate`, so
  an active man-in-the-middle could swap both the artifact and its checksum and
  the verification would still pass. Certificate validation is restored on every
  download; the self-update writes to a temp file and swaps it in atomically.
- **Supply chain.** The prebuilt `libcronet` native library (`dlopen`ed in-process
  by the root core) is pinned to an immutable cronet-go release tag and verified
  by a per-architecture SHA-256 in the Dockerfile and `windows.yml`, instead of
  a mutable `releases/latest` URL with no checksum. Docker base images are
  digest-pinned; every GitHub Actions step across all workflows is pinned to a
  full commit SHA. The `s-ui.sh` SSL menu hardens the acme.sh install
  (`curl -fsSL --proto '=https'`) and no longer enables `--auto-upgrade`.
- **Fix: migration abort on a config-less database.** The 1.2→1.3 migration read
  the `config` settings row with `First()` and aborted with `record not found`
  when no row existed (a panel managed only via the entity UIs), blocking
  upgrades and backup restores; it now treats the missing row as a no-op.
- **Fix: listen fallback widened a restricted bind to 0.0.0.0.** An unbindable
  literal-IP listen address (e.g. after restoring a backup on a new host) fell
  back to the all-interfaces wildcard, silently exposing a deliberately-narrowed
  admin panel and subscription server. It now falls back to loopback
  (`127.0.0.1`) only.
- **Auth & correctness.** Logout is now a CSRF-protected POST (was a forgeable
  GET); `ChangePass` rejects empty and duplicate usernames; a traffic-tariff
  refund of an older order no longer overwrites the current billing window's
  usage; a `resetDays=0` periodic-reset misconfiguration is clamped so counters
  are not wiped every minute; the external-subscription SSRF filter now reuses
  the central validator, closing the CGNAT and other reserved ranges it
  previously let through.
- **Concurrency & lifecycle.** A deplete-cron hot reload could race a full core
  restart and panic inside sing-box - manager mutations now hold the core read
  lock for their duration. An IP-certificate auto-renewal could report success
  while silently skipping the panel restart that loads the new certificate; it
  now uses a non-droppable restart.
- **Performance.** Added a composite `stats(resource, tag, date_time)` index
  matching the dashboard query; WebSocket events are marshalled once per
  broadcast instead of once per connection; the subscription server now
  gzip-compresses its payloads.
- **Frontend / settings UX.** Every settings field across the panel (web and
  subscription settings, the sing-box core Basics page, Telegram) now shows its
  default/recommended value as a placeholder plus an (i) tooltip describing the
  field. Strictly-enumerated fields became pickers: the timezone is an
  autocomplete over the full IANA list, the Clash API default mode is a
  dropdown, and payment currencies are a combobox. The misleading routing label
  "Invalid IP Ranges" / "Invalid Source IPs" - which match private IP ranges,
  not invalid ones - was corrected in every locale; the login form shows an
  inline error instead of only a toast; and several hard-coded English strings
  (Telegram transport labels, routing action cards) and KPI captions were moved
  into i18n. Frontend-only; no API/database/configuration change.

Full release notes: [`docs/releases/v1.5.9-beta1.md`](docs/releases/v1.5.9-beta1.md).

## [1.5.8] - 2026-06-15 - stable 1.5.8: full no-restart apply, Personal Ops Pack, IP certificates, sing-box v1.13.13

First stable release of the 1.5.8 line, consolidating 1.5.8-beta1..beta4 plus
the final Nexus polish pass. No manual database migration.

- **No-restart apply for every managed object.** Inbounds, outbounds, endpoints,
  and services now hot-replace the changed object in the running sing-box core
  instead of restarting it. Captured adapter references still escalate to a full
  restart, while route-rule and `route.final` references stay hot. Managed
  shadowsocks edits also recreate the bound `ssm-api` service.
- **Referenced-tag guard.** Deleting or renaming a referenced outbound,
  endpoint, or managed inbound is rejected with an error listing every
  referencing site, preventing the next core start from failing on a dangling
  reference. Unchanged sing-box settings saves no longer restart the core, and
  apply/restart paths are serialized.
- **Personal Ops Pack.** Config Doctor, client diagnosis, RU/ZH routing and DNS
  preset galleries, and a platform-aware subscription Delivery dialog were
  added. Diagnostics are read-only and dry-check the generated sing-box config
  without starting or restarting the running core.
- **Settings maintenance cleanup.** Config Doctor moved into Settings ->
  Maintenance and is shared by both layouts; the Nexus Home overview was
  streamlined with IPs in the live-traffic KPI and 10 recent events.
- **IP-address TLS certificates.** The panel can issue and auto-renew real
  Let's Encrypt shortlived certificates for bare IP addresses via in-process
  `go-acme/lego`, with target-IP validation, audit logging, and support for
  applying to the panel HTTPS listener or an inbound TLS profile.
- **Terminal issuance and badCSR fix.** IP-certificate issuance moved out of the
  web UI into `s-ui.sh` and the new `sui ip-cert <issue|renew|status|disable>`
  CLI. CSR construction now leaves Common Name empty and puts the IP only in
  SubjectAltName, fixing Let's Encrypt `badCSR` rejections and making auto-renew
  work correctly.
- **Embedded sing-box v1.13.13.** The embedded core moved from v1.13.12 to
  v1.13.13. The module pins the fixed release commit because upstream moved the
  tag after publishing.
- **Performance and code health.** `GET /load` is about 3.7x faster by skipping
  redundant default-setting seed writes; dead code was removed; security helpers
  are pinned by contract tests; generation paths gained determinism, round-trip,
  golden, and benchmark coverage.
- **Stable Nexus polish.** Mobile topbar actions collapse into a compact menu,
  search no longer overlaps controls, dense tables scroll horizontally on mobile,
  overview panels share common primitives, and Settings/Rules/DNS/Audit/Login/
  Admins received spacing, action-row, date/time, and accessibility fixes.
- **Warnings are no longer surfaced as core errors.** ERROR/FATAL logs show error
  toasts, WARN/WARNING logs show warning toasts, and INFO logs no longer produce
  false "Sing-Box Error" notifications. The `warning` key is present in every
  supported locale.

Breaking changes vs v1.5.7:

- Deleting or renaming a referenced outbound, endpoint, or managed inbound is
  now blocked until the reference is removed or pointed elsewhere.
- IP-certificate issuance is no longer available from the web UI; use the
  terminal menu or `sui ip-cert`.

Full release notes: [`docs/releases/v1.5.8.md`](docs/releases/v1.5.8.md).

## [1.5.8-beta4] - 2026-06-14 - Embedded sing-box v1.13.13; IP-certificate badCSR fix (terminal issuance)

Updates the embedded sing-box core to v1.13.13 and fixes a critical CSR
construction bug that caused Let's Encrypt to reject every IP-address
certificate issuance. Moves the issuance workflow out of the web UI and into
the terminal management menu. The in-panel auto-renewal cron is preserved; with
the CSR fix it now works correctly. No database migration.

- **Embedded sing-box updated to v1.13.13** (from v1.13.12) - upstream bug
  fixes (TUN loopback in the direct outbound, ping-timeout fix, build fixes).
  Upstream moved the `v1.13.13` tag after publishing, so `go.mod` keeps
  `require v1.13.13` but `replace`s it with the fixed release commit; the
  original proxy-cached commit fails to compile on Windows. The connection/
  stats tracker contract was revalidated against v1.13.13 (unchanged).
- **Fix: badCSR.** `Obtain(ObtainRequest{Domains:[ip]})` copied the IP string
  into the CSR Subject.CommonName; RFC 8738 requires an empty CN with the IP
  only in `SubjectAltName.iPAddress`. Fixed: a new `buildIpCSR(ip)` helper
  builds a CSR with `certcrypto.CreateCSR{Domain:"", SAN:[ip]}` and issues
  via `ObtainForCSR`, producing the correct `Type:"ip"` ACME identifier.
- **Terminal issuance.** The Settings → Maintenance "IP certificate" card is
  removed. New: s-ui.sh item 20 → option 5 - stops the panel (freeing port 80
  and exclusive DB access), runs `sui ip-cert issue`, restarts the panel.
  Sources `/etc/s-ui/secretbox.env` so the CLI shares `SUI_SECRETBOX_KEY`
  with the running panel.
- **New CLI subcommand:** `sui ip-cert <issue|renew|status|disable>` with
  flags `-ip`, `-email`, `-port`, `-no-renew`.
- **Web feature removed.** `POST /api/ip-cert/issue`, `GET /api/ip-cert/status`,
  `IpCertificateCard.vue`, `types/ipcert.ts`, the `ipCert` i18n block (en/ru),
  and the `shield-check` icon mapping are all deleted.
- **Auto-renewal preserved.** The `@every 12h` in-panel cron (`certRenewJob`)
  is unchanged and now correctly re-issues shortlived certs before the 72-hour
  threshold.
- **Tests.** New `ip_certificate_acme_test.go` checks empty CN, correct IP SAN,
  and valid ECDSA signature. New `TestIssueForCLIAppliesToPanelWithoutRuntime`
  confirms the nil-runtime CLI path is safe.
- **Performance: `GET /load` ~3.7× faster.** `SettingService.GetAllSetting`
  opened a ~90-statement default-seeding write transaction on *every* call,
  serializing the panel's busiest read path on SQLite's single writer and
  dominating `/load` CPU under load (profiled at ~85% of the endpoint). It now
  skips seeding once all default keys exist; the returned map is byte-identical
  and concurrent first-init stays exactly-once (issue #19). `BenchmarkAPI_Load`:
  ~4.86 ms → ~1.30 ms/op, allocations -37%.
- **Internal cleanup.** Removed proven-dead code (10 unreachable functions plus
  an orphaned variable) flagged by `deadcode`/`staticcheck`. The three
  security helpers that looked dead (`config.GetSecret`, `ValidateIssuableIP`,
  `ssrf.IsInfrastructureAddr`) are kept and pinned with new contract tests.
- **Generation test coverage.** Added determinism and link↔json round-trip
  tests, a byte-exact subscription golden test for the database-driven
  `GetSubs` path, and benchmark anchors for link and Clash generation. No
  behavior change.

Full release notes: [`docs/releases/v1.5.8-beta4.md`](docs/releases/v1.5.8-beta4.md).

## [1.5.8-beta3] - 2026-06-14 - IP-address TLS certificates (issue + auto-renew); Config Doctor moves to Settings

Beta: issue and auto-renew a Let's Encrypt TLS certificate for a bare IP address
(no domain needed), done entirely in-process. Additive - no database migration,
no configuration changes.

- IP-address TLS certificates: a new "IP certificate" card in Settings →
  Maintenance issues a Let's Encrypt cert for a bare IP (RFC 8738 `shortlived`
  profile) in-process via `go-acme/lego` - no `acme.sh`, identical on Linux,
  Windows and Docker. HTTP-01 standalone on a configurable port (default 80).
  Apply it to the panel HTTPS listener (panel restarts to load it) or to an
  inbound TLS profile (hot-reload, no core restart). A 12-hourly job re-issues
  when under 72h of validity remain (shortlived certs live ~6.7 days) and when
  the target IP changes. Cert/key live under `<db>/certs` (key `0600`); the ACME
  account key is encrypted at rest and never exposed. The target IP is validated
  against private/loopback/link-local/CGNAT/metadata/reserved ranges (no DNS) on
  both manual issue and auto-renew; issuing is privileged and audited.
- Config Doctor moves to Settings → Maintenance, shared by both layouts; the two
  previous copies (the Nexus Home overview card and the classic Home inline card)
  are removed. Same read-only dry-check behaviour.
- Nexus Home overview streamlined: the standalone System Status panel and the
  Config Doctor card are gone; IPv4/IPv6 addresses move into the live-traffic KPI
  tile and Recent events shows 10 entries (was 6).
- Hardening: auto-renew re-issues on a target-IP change; a direct issue now
  validates the ACME email (via the Go stdlib mail parser) and the challenge port
  like the settings page; a missing `shield-check` icon mapping that would have
  rendered blank is fixed.

Full release notes: [`docs/releases/v1.5.8-beta3.md`](docs/releases/v1.5.8-beta3.md).

## [1.5.8-beta2] - 2026-06-12 - Personal Ops Pack: Config Doctor, RU/ZH routing/DNS presets, subscription delivery UX, client diagnosis

Beta: an operations and diagnostics layer for RU/ZH single-panel admins.
Additive - new read-only diagnostic endpoints and frontend surfaces, no database
migration and no configuration changes.

- Config Doctor on the Home page (Nexus and classic layouts): one click
  assembles the full sing-box config and dry-checks it without starting or
  restarting the running core (parse + `NewBox` construction with no listener
  binds, outbound dials, or rule-set downloads), reporting ok/warn/error items
  for config build, DNS/route references, remote rule-set URL shape, subscription
  health, recent core warnings, and outbound reachability. Report-only; it never
  mutates the config.
- Client "why doesn't it work" diagnosis from each client row: enabled / expired
  / traffic-limit, inbound membership, delivery links, subscription secret and
  formats, core state, online signal, and outbound reachability.
- RU/ZH routing & DNS preset gallery on the Rules and DNS pages: named `.srs`
  sources (SagerNet sing-geosite/sing-geoip, runetfreedom russia-blocked-geoip),
  preview diff, applied only to the local unsaved config; you choose your own
  proxy/direct outbound tags.
- Subscription "Delivery" dialog (reworked client QR): per-platform tabs
  (sing-box, Clash/Mihomo, Hiddify, raw links), copy/QR/test-URL. Settings now
  exposes the previously hidden subscription controls (title/support/profile/
  announce, format toggles, secret-required, per-IP rate limit, name-in-remark).
- Hardening: the client-diagnose connectivity target is now validated with the
  same SSRF guard the outbound-check endpoints use (HTTPS only, no userinfo, no
  internal/metadata addresses) before any probe; subscription-format checks warn
  on a settings read error instead of falsely reporting "all formats disabled";
  outbound probes cancelled by the doctor's own time budget are reported as "not
  tested" rather than failed.

Full release notes: [`docs/releases/v1.5.8-beta2.md`](docs/releases/v1.5.8-beta2.md).

## [1.5.8-beta1] - 2026-06-11 - hot reload for all objects: saves without core restarts; referenced-tag delete guard

Beta: no-restart apply now covers every object the panel manages. Backend-only,
no manual migration. One behavior change: deleting/renaming a referenced
outbound, endpoint, or managed inbound is now rejected with an explanatory
error instead of breaking the next core start.

- Saving inbounds, outbounds, endpoints, and services no longer restarts the
  sing-box core: the affected object is hot-replaced in the running core (the
  same mechanism clients and TLS edits already use). Existing connections on
  unrelated objects survive every save. A failed hot apply still falls back to
  a full core restart, so the core never serves a stale configuration.
  - An outbound or endpoint whose tag is captured at adapter construction -
    another outbound's `detour`, a `selector`/`urltest` member list or
    `default`, a service dial detour, `dns`/`ntp` detours, a rule-set
    `download_detour`, or the Clash API UI download detour - is conservatively
    applied via a full restart. References from route rules and `route.final`
    are resolved per connection, so those edits stay hot.
  - Editing a managed-shadowsocks inbound also recreates the `ssm-api` service
    bound to it, so the service keeps tracking the fresh inbound.
- Deleting (or renaming) an outbound, endpoint, or managed inbound that is
  still referenced anywhere - including route rules and `route.final` - is now
  blocked with an error that lists every referencing site. Previously the save
  went through and the next core start failed on the dangling reference,
  taking the whole proxy down.
- Re-saving the sing-box settings blob without any changes no longer restarts
  the core (and no longer drops every active connection).
- Post-save core synchronization, user-initiated restarts, and the cron core
  starter are now serialized: a save can no longer interleave with a restart
  and silently leave the running core out of sync with the database.
- Fixed a latent data race in the panel-restart scheduler (timer callback read
  an unsynchronized variable); it predates this release.

Full release notes: [`docs/releases/v1.5.8-beta1.md`](docs/releases/v1.5.8-beta1.md).

## [1.5.7-hotfix1] - 2026-06-11 - Fix "record not found" after deleting a client; SUI_COOKIE_KEY generator

Hotfix for v1.5.7 plus a session cookie key generator in the `s-ui` menu and
the installer. No breaking, manual-migration, or configuration changes.

### Fixes

- Deleting a client whose database row was already gone (a stale list row, a
  concurrent delete from another session or tab, a resubmitted request) no
  longer fails with "record not found": client delete and bulk delete are now
  idempotent - deleting an absent client is a no-op success, and a bulk delete
  skips already-gone ids while deleting the rest. The delete paths of inbounds,
  outbounds, endpoints, services, and TLS were verified already idempotent and
  are now covered by regression tests.

### Management script and installer

- New `s-ui` menu item 23 generates the session cookie key (`SUI_COOKIE_KEY`)
  into `/etc/s-ui/secretbox.env`; an existing key is detected and offered a
  rotation with rollover (previous keys stay accepted, so nobody is signed
  out).
- The installer generates `SUI_COOKIE_KEY` automatically when absent - it
  never overwrites an existing key and never prompts. On the first upgrade
  that introduces the key, sessions signed with the previous fallback key are
  signed out once.

Full release notes: [`docs/releases/v1.5.7-hotfix1.md`](docs/releases/v1.5.7-hotfix1.md).

## [1.5.7] - 2026-06-11 - first stable 1.5.7: Paid Subscriptions, Nexus redesign, hardening, no-restart apply

First stable release of the 1.5.7 line, consolidating 1.5.7-beta1..beta10 - the
experimental **Paid Subscriptions** Telegram bot (six payment providers, trials,
refunds, broadcasts), the refreshed dark "technical" Nexus interface, and three
rounds of independent security hardening. The entries below are new since
beta10.

- Adding, editing, enabling/disabling, and deleting clients no longer restarts
  the sing-box core: only the affected inbounds are hot-reloaded in the running
  core. Existing connections on those inbounds are closed, so revoked or
  disabled credentials stop working immediately; connections on unrelated
  inbounds survive the change. If a hot reload fails, the panel falls back to a
  full core restart so the core never keeps serving a stale configuration.
- IP-limit changes (`limitIp`, IP-limit mode) take effect immediately: saving a
  client invalidates the IP-limit enforcement cache instead of waiting out its
  30-second TTL.
- Editing a TLS certificate now hot-reloads only the inbounds and services that
  reference it (previously: a full core restart), and the reload runs after the
  database transaction commits. Creating or deleting a TLS entry no longer
  touches the core at all.
- Fixed: editing a TLS entry referenced by at least one service failed with a
  database scan error and rolled back the whole edit.

Full release notes: [`docs/releases/v1.5.7.md`](docs/releases/v1.5.7.md).

## [1.5.7-beta10] - 2026-06-10 - Remediation from a full quality, security, and supply-chain review

Applies the fixes from a full code-quality, optimization, security, and
supply-chain review. One additive database column is created automatically (no
manual migration). Two behavior changes: changing the admin password signs out
all other sessions, and the per-username login lockout is now a delay (tarpit)
rather than a hard block.

### Paid Subscriptions (financial correctness)

- A CryptoBot payment confirmed after the local order timeout is no longer lost
  (the poll runs before expiry; polled orders use a long grace window).
- Refunds restore the usage counters symmetrically (via an internal snapshot),
  not just the volume.
- Stars refunds return the money before finalizing, so a transient failure leaves
  the order retryable.
- CryptoBot polling re-validates the paid amount and currency against the order.
- Tariffs reject negative values server-side.

### Security

- Changing the admin password rotates the session generation: other web sessions
  and all WebSocket tokens are invalidated, and only the current session stays
  signed in.
- The per-username login throttle is now an escalating, capped delay (tarpit)
  instead of a hard block; the per-IP block is unchanged.
- x-ui import validates geoip/geosite codes before using them in a rule-set URL.
- The subscription link service no longer panics on a malformed vmess `ps` field.
- Fixed a database-import race on the shared connection handle.

### Detection and audit

- New audit signals: `/sub` enumeration, login from a new IP, cross-user order
  access on the client bot, IP-limit enforcement, and an audit-pipeline drop
  marker.
- Real-time alerts on account lockout and database export.

### Interface (Nexus)

- Unsaved-changes confirmation now covers every entity form (drawers and the
  classic dialogs) in both interface modes.
- Case-insensitive client search; TLS option defaults no longer leak between
  forms; the inbound form Save is disabled until valid; a correct per-row checkbox
  label; and removed dead UI code.
- The dashboard status poll pauses while the browser tab is hidden.

### Supply chain (CI)

- Third-party and Docker GitHub Actions are pinned to commit SHAs, and a
  Dependabot config keeps dependencies current. The Docker frontend build uses
  `npm ci`.

Full release notes: [`docs/releases/v1.5.7-beta10.md`](docs/releases/v1.5.7-beta10.md).

## [1.5.7-beta9-hotfix1] - 2026-06-10 - Fix blank panel from dropped underscore-named chunks

Hotfix for v1.5.7-beta9. No backend logic, breaking, manual-migration, or
configuration changes.

### Fixes

- Fixed a blank panel ("Failed to fetch dynamically imported module", 404 on a
  JS chunk) that occurred when the build emitted an asset name starting with
  "_": Go's `//go:embed` drops such files unless the `all:` prefix is used. The
  embed now uses `all:`, and the frontend prefixes asset names
  (`app-`/`chunk-`/`style-[hash]`) so they never start with "_".

Full release notes: [`docs/releases/v1.5.7-beta9-hotfix1.md`](docs/releases/v1.5.7-beta9-hotfix1.md).

## [1.5.7-beta9] - 2026-06-10 - Nexus interface aligned to the reference design

Frontend-only release. The default Nexus interface now matches the dark
"technical" reference design. There are no backend, breaking, manual-migration,
or configuration changes.

### New look

- **Refreshed Nexus interface.** The default interface now matches the dark
  "technical" reference design end to end: the exact colour palette (surfaces,
  `#2a2a2a` borders, the cyan accent, status and text colours, and a cyan primary
  button), a single system font, and monospace IP / port / UUID cells.

### Improvements

- **Clearer status at a glance.** Online / Offline / Disabled now render as
  coloured status badges and TLS shows On / Off pills; the status column is
  labelled "Status".
- **Page header in the top bar.** Each page shows its title, a live count summary
  (for example "8 inbounds • 3 online") and the search box in the top bar, with
  the global controls on the right.
- **Compact sidebar and refreshed icons.** A tighter, flat sidebar with a cyan
  "S" brand mark and refreshed icons throughout. The first menu entry is now
  "Dashboard".

### Fixes

- Fixed the bulk client **Add** and **Edit** dialogs where dropdown menus could
  stack open and not close; they now open as proper drawers.
- Fixed the bulk client **Edit** "Save" button staying disabled when "All
  clients" was selected.
- Fixed the untranslated "Refresh" and "Cancel" buttons on the Paid
  Subscriptions screen.

### Localization

- Localized the Paid Subscriptions screen (English and Russian; other locales
  fall back to English) and page subtitles with natural per-language phrasing.

This frontend-only release was validated with lint, 128 unit tests (including new
Lucide icon-set and English/Russian key-parity tests), the production build, the
Nexus end-to-end specs, and a multi-agent regression review of the diff (no
regressions).

Full release notes: [`docs/releases/v1.5.7-beta9.md`](docs/releases/v1.5.7-beta9.md).

## [1.5.7-beta8] - 2026-06-07 - Internal reliability and audit-tooling hardening

Internal reliability and audit hardening. There are no user-facing feature
changes, breaking changes, manual migrations, or configuration changes.

### Reliability

- Synchronized publication and access to the global GORM database handle,
  eliminating a data race when the handle is initialized or replaced.
- Stabilized SQLite test teardown on Windows across database, import-xui,
  cron-job, and IP-monitor tests by checkpointing WAL files, closing database
  handles, retrying temporary-directory cleanup, and isolating asynchronous
  audit-writer activity during database resets.

### Security and quality assurance

- Limited the `gosec` audit target to project Go sources by excluding local Go
  scratch caches and `frontend/node_modules`.
- Revalidated the full backend and frontend quality gates: Go tests (including
  race detection), build, vet, static analysis, linters, `gosec`,
  `govulncheck`, frontend lint, 88 frontend tests, production build, and
  dependency audit all pass with no reported security findings.

Full release notes: [`docs/releases/v1.5.7-beta8.md`](docs/releases/v1.5.7-beta8.md).

## v1.5.7 Beta Summary

Summary of changes added since stable **v1.5.6**. Full per-release notes:
[`docs/releases/whats-new-1.5.7.md`](docs/releases/whats-new-1.5.7.md).

The 1.5.7 line adds **Paid Subscriptions**, an experimental self-service
Telegram bot. It is disabled by default. Existing setups are unchanged until an
administrator enables it.

**Main changes**
- **Paid Subscriptions client bot:** subscription & per-protocol share links,
  **QR codes**, and live usage (used vs. limit, days left, online status, traffic).
- **Self-service sign-up** with a configurable free trial (capped + rate-limited).
- **Built-in payments across 6 providers** - Telegram Stars, YooKassa, Stripe,
  CryptoBot, PayMaster, and external links - with safe, no-double-charge renewals.
- **In-bot Payment menu** (*Buy / Renew*, *My purchases*, *Request a refund*) with
  automatic Telegram Stars refunds; other providers route a request to the admin.
- **Admin refund tool** with an optional per-refund claw-back of granted days/traffic.
- **Broadcasts** to every bound client, plus an **editable /start greeting**.
- **Flexible Telegram routing:** proxy (HTTP/HTTPS/SOCKS5) or a sing-box Outbound,
  set independently for the client bot and admin notifications.

**Fixes**
- **No more accidental duplicates:** the Save button locks while saving and the
  server rejects duplicate submissions. One action creates one record.

**Security**
- Bot & payment tokens are **encrypted at rest** and masked in the UI; set
  `SUI_SECRETBOX_KEY` in production. Sensitive payment identifiers never reach the
  browser or logs.

---

## [1.5.7-beta7] - 2026-06-06 - Independent-review hardening; 3x-ui scheduled sync & remote import removed

This beta ships the **removal of the 3x-ui scheduled sync / remote import** plus a
round of **low-risk fixes and hardening** from a fresh independent code-quality,
optimization, and security review of the panel. No new features and **no manual
migration**. The 3x-ui removal changes existing behavior (Breaking changes below);
none of the review fixes are breaking.

**3x-ui import** is trimmed to the **one-shot local `.db` upload** importer. The
recurring **scheduled sync** and the **on-demand remote import** (pull from a running
3x-ui over SSH/HTTP) are removed, along with all of their machinery: the SSH/HTTP/file
import sources, the SSRF-guarded remote client, the SSH host-key (TOFU) store, and
at-rest encryption of saved sync profiles.

**Breaking changes**
- **Scheduled sync is gone.** The "3x-ui Sync" schedule page, sync profiles, and
  the background cron job no longer exist; any previously configured schedule
  stops running after upgrade. The `xui_sync_profiles` and `xui_known_hosts`
  tables are dropped automatically on first start - no manual step.
- **Remote import is gone.** The `POST /api/import-xui/remote/*` and
  `/api/import-xui/sync/*` endpoints are removed. Import is now upload-only:
  `POST /api/import-xui[/plan|/apply|/rollback]` and `GET /api/import-xui/reports`.
- **CLI:** the `s-ui sync-xui` command and the `import-xui --remote` / `--schedule`
  flags are removed; `s-ui import-xui --src <x-ui.db>` (local file) remains.
- **API tokens:** the `xui_remote` token scope is removed and is no longer valid.
  Re-issue any such token with an appropriate scope.

### Security & privacy (review fixes)

- **Login no longer reveals whether a username exists** - the not-found path performs
  the same bcrypt work as a wrong-password attempt, closing a timing oracle that
  enabled admin-username enumeration.
- **URL credentials are masked in logs** - a `user:pass@host` in any logged URL is
  redacted in free text and under secret-named setting keys.
- **Session cookies are Secure by default** in the session store (production login/CSRF
  flows already set this explicitly).
- **Refunds reject corrupted orders** - a paid order with a non-positive amount is never
  processed (defense in depth).
- **IP-limit failures are observable** - a database error during the per-client IP-limit
  check still fails open but now logs the event (throttled) instead of silently
  disabling enforcement.

### Reliability & fixes (review fixes)

- **Correct traffic chart** - the per-client statistics graph summed each time bucket
  with a no-op reducer and displayed only the first sample instead of the total; it now
  sums correctly.
- **Safer 1.3 migration** - the anytls / domain-strategy migration runs inside a
  transaction and checks every write (it previously ignored save errors and carried a
  dead filter clause that loaded every row).
- **IDN panel domains work** - a Unicode panel domain (e.g. `панель.рф`) now matches the
  punycode `Host` header browsers send instead of being rejected with `403`.
- **Bounded public-IP probe** - the `s-ui uri` public-IP lookup caps the response body
  (1 MiB), matching every other outbound reader.
- **No drawer thrash** - the default layout's `isMobile` is a pure computed again; the
  drawer's default open state follows the breakpoint through a watcher.
- **Clearer core-start log** - a sing-box core that fails to start is logged explicitly;
  the panel intentionally stays up so the config can be fixed from the UI.

### Performance & cleanup (review fixes)

- **Indexed order history** - `payment_orders.telegram_user_id` is now indexed, so a
  user's order / refund history no longer scans the whole table.
- **Lighter frontend install** - removed three unused dependencies (`core-js`,
  `roboto-fontface`, `material-design-icons-iconfont`).

**Kept**
- One-shot local **`.db` upload** import - the UI wizard, the API, and
  `import-xui --src` - including dry-run, conflict strategy, plan/apply, and rollback.

No manual migration is required; the deprecated 3x-ui tables are dropped on startup.

## [1.5.7-beta6-hotfix1] - 2026-06-05 - Fix beta6 panel black screen (frontend build)

Emergency **build** hotfix for v1.5.7-beta6. No code, config, or data changes -
it only fixes the broken frontend build that shipped in the beta6 artifacts.

**Fixed**
- **Web panel failed to load on v1.5.7-beta6** - black screen with a `404` for a
  JS chunk (e.g. `assets/_WJiVkoC.js`). `frontend/package-lock.json` had drifted
  out of sync with `package.json` (icons moved from `@mdi/font` to `@mdi/js`
  without regenerating the lock); the release built the frontend with the lenient
  `npm install` and embedded an inconsistent, unvalidated bundle. The lockfile is
  regenerated in sync and the build is verified consistent - no dangling chunk.

**Release-pipeline hardening**
- The release workflow now builds the frontend fail-closed - `npm ci` plus the
  same lint and unit-test gates CI runs - so a desynced lockfile or any
  CI-rejected frontend can no longer ship.

## [1.5.7-beta6] - 2026-06-05 - Security & reliability hardening, performance, and accessibility

A hardening release driven by a full code-quality, optimization, and security
audit of the panel. No new features and **no manual migration** - it closes
several security gaps, removes silent-failure and panic risks, trims the frontend
bundle by ~60%, and fixes a few data-integrity bugs. Two items change existing
behavior - see **Breaking changes**.

### Security

- **Updated to Go 1.26.4**, closing two reachable Go standard-library
  vulnerabilities (`GO-2026-5037` in `crypto/x509`, `GO-2026-5039` in
  `net/textproto`).
- **API-token scopes are now enforced.** The `apiv2` action endpoints used to run
  every action regardless of token scope, so a `read`/`observability`/`telegram`/
  `database` token could write config, restart the panel, or read settings. Each
  action is now gated: writes and restarts require `write`/`admin`, config and
  identity reads require `read`/`write`/`admin`, and metrics also allow
  `observability`. Browser (admin-session) access is unchanged. *(Breaking.)*
- **Remote x-ui import/sync hardened against SSRF.** Remote imports validate the
  target URL and **re-check the resolved IP at connection time** (defeating
  DNS-rebinding), with redirects bounded; cloud-metadata, loopback, and private
  ranges are blocked for untrusted (scoped-token) callers. **`file` and `ssh`
  import sources are admin-only**, and scheduled sync runs a `file`/`ssh` source
  only from an admin-saved profile. *(Breaking.)*
- **Secrets at rest.** With `SUI_SECRETBOX_KEY` set, stored secrets are now
  **re-sealed once at startup** under that out-of-database key, so a value written
  before you adopted the key is no longer recoverable from the database alone.
  Remote-panel credential encryption derives its key from a random per-install
  secret instead of a predictable default.
- **Login brute-force.** Added a **per-username** login throttle on top of the
  existing per-IP limit, so a distributed attack on a single account is also
  slowed.
- **Session fixation.** The session ID is now **rotated on login**, so a planted
  pre-auth session cookie cannot survive authentication.
- **Transport & headers.** HSTS honors `X-Forwarded-Proto` only from a trusted
  proxy; the CSRF cookie honors a strict `SameSite` policy; `s-ui admin -reset`
  generates a random password instead of a fixed default.
- **Telegram payments** verify the payer's Telegram id; proxy URLs carrying
  embedded credentials are masked in logs.

### Reliability and fixes

- **No more false "running" core.** If the generated sing-box config fails to
  parse, the core now surfaces the error instead of silently starting an empty
  instance and reporting healthy while nothing is listening.
- **Background jobs can't crash the panel.** Cron jobs are panic-isolated and
  skip-if-still-running; the WAL-checkpoint job is guarded against a startup
  nil-dereference.
- **Sturdier subscription generation.** Malformed inbound/client configuration no
  longer panics the link, Clash, or JSON subscription builders - they skip the bad
  field gracefully.
- **Change feed stays available.** A client name containing a quote or other JSON
  metacharacter no longer corrupts the stored change log, which previously made the
  admin **Changes** view return an empty response for everyone.
- **Correct bulk-edit links.** Editing a group of clients with *different* inbound
  sets now regenerates each client's subscription links from its own inbounds
  instead of copying the first client's.
- **Consistent API errors** with a documented success envelope and internal
  details redacted.

### Performance

- **Backend.** The IP monitor writes pending records in a single batched upsert
  (instead of one statement per IP); the subscription hot path caches its display
  settings (~8 fewer queries per request) and `settings` reads now use an index.
- **Frontend bundle 6.2 MB → 2.5 MB (-60%).** `moment` and the date-picker load
  lazily with the pages that use them, and icons moved from the full Material
  Design webfont (~2.9 MB) to inline SVG paths.

### Accessibility

- The icon-only admin action buttons (edit / changes / delete) now have accessible
  names for screen readers.

### Breaking changes

- **Scoped API tokens lose access they should not have had.** An integration using
  a `read`, `observability`, `telegram`, or `database` token to write config,
  restart the panel, or read settings (which only worked because of the enforcement
  gap above) is now rejected - use an `admin` or appropriately scoped `write` token.
- **`file`/`ssh` x-ui sync profiles must be admin-saved.** After upgrading, a
  scheduled sync profile whose source is a local `file` or `ssh` target will not run
  until an **admin re-saves it** (the panel can no longer prove a pre-upgrade
  profile was admin-created). Re-saving it from an admin session restores it.

### Upgrade

No manual migration or config change. The `settings` table gains a unique index
automatically on first start; if you use `SUI_SECRETBOX_KEY`, the one-time secret
re-seal runs at startup. Review the two **Breaking changes** if you use scoped API
tokens or `file`/`ssh` scheduled sync. Release notes:
[`docs/releases/v1.5.7-beta6.md`](docs/releases/v1.5.7-beta6.md).

## [1.5.7-beta5] - 2026-06-04 - Paid Subscriptions admin UI: Bindings/Orders columns, unbind confirm, tab order

- **Paid Subscriptions tabs reordered.** **Bindings** is now the first (default)
  tab and **Bot** is last (after *Orders*).
- **Bindings table:** new **Client ID**, **Description**, and **Expiry** columns.
  Expiry shows the date plus a remaining-days chip (green = unlimited, red =
  expired), reusing the Clients-page formatter.
- **Unbind confirmation:** removing a client's Telegram binding now opens a
  confirmation dialog instead of unbinding on the first click; the binding is
  cleared only after you confirm (the client itself stays in the panel).
- **Orders table:** new **Client name** (replacing the numeric client id),
  **Telegram ID**, and **Description** columns, joined server-side from the
  clients table via a LEFT JOIN. The Orders API still never exposes the provider
  charge id, the invoice idempotency key, or the provider payload.
- Release notes: [`docs/releases/v1.5.7-beta5.md`](docs/releases/v1.5.7-beta5.md).

## [1.5.7-beta4] - 2026-06-04 - Paid Subscriptions Payment menu & refunds; duplicate-create fix

- **Paid Subscriptions bot: new "Payment" section.** The flat "Buy / Renew"
  button is replaced by a **Payment** menu that opens a submenu: **Buy / Renew**,
  **My purchases**, **Request a refund**. The **Stats** button is renamed to
  **My subscription** (person icon); the view itself is unchanged.
- **My purchases:** a read-only list of the client's own orders (tariff, amount,
  status, date), scoped strictly to the requesting Telegram user.
- **Refunds.** Telegram Stars are refunded automatically via the Bot API
  (`refundStarPayment`); other providers (YooKassa/Stripe/PayMaster/CryptoBot/
  external) send the admin a refund request, since the Bot API has no fiat/crypto
  refund. A new **Refund** action on the admin *Orders* tab refunds Stars or marks
  the order refunded, with a per-refund "revoke granted days/traffic" toggle.
- **Refund rollback policy** `paidSubRefundRevoke` (default on) governs the bot's
  user-initiated Stars refund: a successful refund also rolls back the days and
  traffic that order granted (anti-abuse), idempotently and without disabling the
  client. The user never chooses this - the admin does (globally, plus per-refund
  in the panel).
- **Hardening:** the admin Orders API no longer exposes the Telegram charge id or
  the invoice idempotency key; a concurrent bot/panel refund that returns "already
  refunded" is treated as success (Stars refunds are charge-idempotent).
- **Fix: duplicate creation from double-submitted saves.** Saving an entity
  (client/inbound/outbound/...) synchronously restarts the sing-box core before
  responding, so a second submission during that slow window created a duplicate
  row. The Save button is now disabled while a save is in flight (all create/edit
  modals), and the server skips an identical create that arrives while the first
  is still in flight or within a short window after it - so one action creates one
  row even under a slow core restart.
- Release notes: [`docs/releases/v1.5.7-beta4.md`](docs/releases/v1.5.7-beta4.md).

## [1.5.7-beta3] - 2026-06-04 - fix Paid Subscriptions admin writes + hardening

- **Fix:** the Paid Subscriptions admin page now works end-to-end. Writes
  (`/api/paidsub/*`: bindings, tariffs, broadcast) were sent as form-urlencoded
  while the backend parsed JSON, AND every paidsub response omitted empty
  `msg`/`obj` keys (rejected by the frontend as "unknown data", so reads were
  empty too). Requests now send JSON and responses always include the
  `success`/`msg`/`obj` envelope.
- **Fix:** `/start` only auto-registers on a genuine "not found"; a transient DB
  error no longer risks creating-and-rebinding a new client over an existing one.
- **Fix:** connection leak in the bot poll loop (idle connections of discarded
  proxy/outbound transports are now closed).
- **Hardening:** rate-limiter refuses new keys when saturated; CryptoBot invoice
  ids URL-escaped; long link lists hard-split under Telegram's limit; custom
  greeting defensively truncated.
- **Payments: PayMaster provider** added (Telegram-native invoicing with a
  BotFather `provider_token`), alongside YooKassa/Stripe/Stars/CryptoBot/external.
- **Fix:** Orders table shows Telegram Stars (XTR) amounts as whole units (a
  1-Star order had shown as "0.01 XTR").
- Release notes: [`docs/releases/v1.5.7-beta3.md`](docs/releases/v1.5.7-beta3.md).

## [1.5.7-beta2] - 2026-06-04 - Telegram transport selector, broadcast & greeting

- **Telegram transport selector, per module.** The Paid Subscriptions bot and
  the admin Telegram module (notifications/backups) can each egress either via a
  **proxy** (http/https/socks5, own credentials) or via a configured **sing-box
  outbound** (requires the core running), configured independently.
- **Broadcast to all clients.** New *Messages* tab sends a one-off announcement
  to every bound Telegram user (throttled, sent/failed report, confirmation).
- **Editable /start greeting** on the *Messages* tab (empty = built-in default).
- **Fixes (beta1 UI):** *Auto-registration* inbound dropdown now lists inbounds
  (API response was read incorrectly); *Bindings* tab gained an explicit **Add
  binding** action with a clear empty state.
- Release notes: [`docs/releases/v1.5.7-beta2.md`](docs/releases/v1.5.7-beta2.md).

## [1.5.7-beta1] - 2026-06-04 - experimental Paid Subscriptions Telegram bot

- New **experimental "Paid Subscriptions" module** (off by default, isolated
  from the core). A client-facing Telegram bot on a separate, encrypted token
  lets a bound client get their subscription link, per-inbound share links and
  server-rendered QR codes, and view current usage (used/limit + progress bar,
  days left, online status, lifetime traffic).
- **Telegram ID ↔ client binding** is managed on a new **Paid Subscriptions**
  admin page (separate left-menu item); the existing client card and the core
  `clients` table are not touched (bindings live in their own table).
- **Self-registration with a trial:** an unknown user opening the bot can be
  auto-registered with admin-selected inbounds and a configurable trial period,
  guarded by a global cap and a per-user `/start` rate limit.
- **Tariff-based payments, multi-provider:** admins define tariffs (name, price,
  +days, +traffic); clients pay/renew in the bot and the subscription extends
  automatically. Selectable providers (several at once): Telegram Stars (XTR),
  YooKassa, Stripe, CryptoBot, and an external payment link. Renewals are
  idempotent, amounts are verified server-side against the order snapshot, and
  zero-price tariffs cannot grant a renewal.
- The module lives in its own `paidsub` package behind a single `paidSubEnabled`
  flag with its own HTTP endpoints and DB tables (created idempotently at
  startup); the UI is a lazy-loaded page marked *experimental*. For production,
  set `SUI_SECRETBOX_KEY` so payment tokens are encrypted with a key kept
  outside the database (the UI warns when unset).
- Release notes: [`docs/releases/v1.5.7-beta1.md`](docs/releases/v1.5.7-beta1.md).

## [1.5.6] - 2026-06-04 - first stable 1.5.6: 3x-ui import-correctness fixes

- First stable release of the 1.5.6 line, consolidating 1.5.6-beta1..beta9 - the
  3x-ui → s-ui-x migration and the panel-recovery terminal menu. The entries below
  are the import-correctness fixes added since beta9.
- An Xray `blackhole` outbound now migrates to a `reject` rule action instead of a
  dangling `outbound: "block"` reference. sing-box 1.11+ has no `block` outbound,
  so the reference made the imported config fail at route time with "outbound not
  found: block"; this supersedes the `blackhole`→`block` mapping shipped in
  1.5.6-beta7/beta8.
- A DNS-only source config (no routing rules, no proxy outbounds, no endpoints) is
  no longer skipped during import - its DNS was being dropped silently.
- A built-in `direct` outbound is ensured whenever migrated routing routes to it
  (a rule, or a remote rule-set download detour), and the check now consults the
  database so the InitDB-seeded `direct` outbound is not re-reported as a skipped
  duplicate.
- The reject / hijack-dns routing targets use non-colliding sentinels, so a user
  proxy legitimately tagged `block`/`blocked`/`dns` keeps routing to itself instead
  of being turned into an action.
- Full release notes: [`docs/releases/v1.5.6.md`](docs/releases/v1.5.6.md).

## [1.5.6-beta9] - 2026-06-03 - panel domain/address reset menu & SSL force-reissue

- New terminal-menu item *Clear panel domain and address* (also a
  `s-ui setting -clearDomain` CLI flag and a `ClearWebDomainAndAddress()` service
  method) resets the panel domain (`webDomain`), listen address (`webListen`) and
  web URI (`webURI`) to their defaults, recovering panel access when a wrong
  domain or an unbindable listen IP locks you out. Takes effect after a manual
  panel restart; adding the item renumbered the later menu entries (`11..21` →
  `12..22`) and widened the choice-range prompts to `0-22`.
- The `Get SSL` menu can now re-issue a certificate acme.sh already holds instead
  of dead-ending with "Certificate already exists; cannot reissue": it shows the
  existing certificate, warns about the Let's Encrypt duplicate-certificate limit
  (5 per week), and on confirmation runs `acme.sh --issue --force` plus
  `--installcert` so `/root/cert/<domain>/` is rewritten too. The existence check
  no longer inspects only the last `acme.sh --list` row, so it matches the right
  domain when several certificates are present.
- Full release notes: [`docs/releases/v1.5.6-beta9.md`](docs/releases/v1.5.6-beta9.md).

## [1.5.6-beta8] - 2026-06-03 - 3x-ui migration: outbounds, routing matchers, TLS certs & DNS

- Proxy outbounds now migrate as s-ui outbounds: previously only WARP
  (WireGuard), `freedom` and `blackhole` outbounds were handled, so an Xray
  `outbounds` entry of type `vmess`/`vless`/`trojan`/`shadowsocks`/`socks`/`http`
  was dropped silently and a chained/proxy outbound vanished from the migrated
  panel. Each is now converted to a first-class sing-box outbound (server/port,
  `uuid`/`password`/`method`/user-pass, VLESS `flow`, the TLS/Reality block, and
  the `ws`/`grpc`/`http`/`httpupgrade` transport) and registered as a routing
  target, so a rule that referenced it resolves to the migrated outbound instead
  of "requires manual review".
- System outbounds map to their sing-box home: `freedom`→`direct`,
  `blackhole`→`block`, and a `dns` outbound becomes a `hijack-dns` route action
  (sing-box has no `dns` outbound). `loopback` and any protocol Xray does not
  emit (e.g. `hysteria`) are surfaced as a warning to recreate manually rather
  than dropped silently.
- No silent loss when routing import is off: proxy/WARP outbounds live in the
  source `xrayConfig`, read only during routing import. When routing import is
  disabled but the source contains outbounds, the plan now warns that they were
  not migrated and how to migrate them.
- Outbounds are created only when new (a re-import or scheduled sync does not
  clobber an operator-edited outbound of the same tag), and the import report
  gains an `outbounds` (imported/skipped) counter.
- Migrated routing and DNS now apply to the live config: they are merged into
  the active sing-box `config` setting (route rules/rule sets, DNS servers/rules),
  preserving existing rules and de-duplicating rule sets/servers by tag.
  Previously the import wrote them to a separate setting the panel never loaded,
  so imported routing had no effect - it does now.
- Routing rules cover many more matchers instead of "manual review": `port`/
  `sourcePort` (including ranges), `network`, `protocol`, `source`, `inboundTag`
  (→`inbound`), `user` (→`auth_user`), and non-`geosite` domains
  (`domain:`/`full:`/`keyword:`/`regexp:`/bare → `domain_suffix`/`domain`/
  `domain_keyword`/`domain_regex`). `geosite:`/`geoip:` matches become remote
  `rule_set`s (MetaCubeX `meta-rules-dat`), since sing-box 1.12 removed the inline
  `geoip`/`geosite` fields; a source `geoip` uses `rule_set_ip_cidr_match_source`.
  `attrs` and `balancerTag` still need manual review (no sing-box equivalent).
- The Xray `dns` block is translated to sing-box's format (typed servers
  `udp`/`tls`/`https`/`h3`/`quic`/`tcp`/`local`, domain-scoped servers become DNS
  rules, plus `final`, query strategy and `client_subnet`) instead of being
  copied verbatim, which produced an invalid block. `hosts`/`fakedns` are flagged
  for manual setup.
- Non-reality TLS certificates now migrate: an inbound whose `tlsSettings` carry
  an inline certificate/key gets a real s-ui TLS record (server cert + client
  block); a certificate referenced only by file path is flagged for manual
  upload, since the importer reads only the database, not the source host's disk.
- WebSocket transport carries every request header, not just `Host`.
- Outbound extras: `packet_encoding` (`xudp`) is set when the source used it; a
  multi-server outbound becomes per-server members plus a `urltest` group; Xray
  `mux` is reported rather than enabled, because sing-box multiplex is not
  wire-compatible with Xray mux and enabling it would break the outbound.
- Web admin keeps unsaved edits during the background refresh: the Basics, DNS
  and Routing pages bound their forms to the live store config, so the 10-second
  config poll (and WS reload events) silently reverted in-progress edits - they
  now edit a local copy that persists until you save.
- Routing import no longer produces a config sing-box rejects: an Xray external
  geoip reference (`ext:<file>:<code>`, e.g. `ext:geoip_RU.dat:ru`) and bare IPs
  were written into `ip_cidr` verbatim, which sing-box refuses to parse
  (`ipcidr: parse: no '/'`) so the core would not start. `ext:` now maps to a
  geoip rule set, a bare IP gets a host mask, and an unparseable value is dropped
  with a warning instead of breaking the whole config.
- Migrated DNS no longer stops the core from starting: a DNS server reached over
  a domain (`https://dns.google/...`, `tls://...`) was emitted without a
  `domain_resolver`, which sing-box 1.13 rejects (`missing domain resolver for
  domain server address`). Each domain-addressed server now gets a
  `domain_resolver` - an IP-addressed server from the migration, or an appended
  local bootstrap - the same way s-ui's own DNS editor sets it; TLS/HTTP servers
  also get the `tls`/`headers` blocks so a migrated server matches a
  natively-created one.
- A Trojan inbound no longer crashes the core: the inbound editor wrote a
  top-level `password`, which sing-box's Trojan inbound rejects (`unknown field
  "password"` - it authenticates per user via `users`). The password field is now
  outbound-only, and any leftover top-level password is dropped when the config
  is built (so existing inbounds recover without an edit).
- Full release notes: [`docs/releases/v1.5.6-beta8.md`](docs/releases/v1.5.6-beta8.md).

## [1.5.6-beta7] - 2026-06-02 - 3x-ui migration: subscription links, WARP & import timeout

- Migrated clients now get subscription links: the importer left each client's
  `Links` empty and the subscription reads only stored links, so an imported
  client's inbounds never appeared in the subscription or QR/Links view. Links
  are generated during import with the same generator the panel uses on a normal
  client save; the hostname is the panel request host (web), the new `--host`
  CLI flag, or the configured sub/web domain (scheduled sync). A re-import merge
  preserves a client's external/sub links, and a NULL `Links` column is now
  tolerated so a later inbound edit regenerates links instead of skipping the
  client.
- Cloudflare WARP migrates as a WireGuard endpoint with its routing rule: 3x-ui
  stores WARP as a WireGuard outbound referenced by rules, while s-ui models WARP
  as an endpoint routed to via Rules. The WARP outbound becomes a WARP endpoint
  (Cloudflare peer, MTU, addresses, reserved) and its rule is rewired to target
  it instead of "requires manual review"; source blackhole/freedom outbounds
  resolve to block/direct by protocol, so `blocked`/`direct` rules map too. The
  endpoint is created only when new (a re-import or scheduled sync no longer
  clobbers an edited WARP endpoint), and a `reserved` value that is not exactly
  three bytes is dropped with a warning so the config still loads.
- Import no longer fails with "Network Error": a large import could run past the
  web server's 30s write timeout and sever the HTTP response mid-import even
  though it completed server-side. The deadline is now lifted on the raw
  connection - the gzip middleware wraps the response writer so
  `http.NewResponseController` could not reach it - only after the request is
  authenticated, scope-checked and rate-limited, and the work stays bounded by
  the request context.
- Full release notes: [`docs/releases/v1.5.6-beta7.md`](docs/releases/v1.5.6-beta7.md).

## [1.5.6-beta6] - 2026-06-02 - 3x-ui migration hardening

- Fixes a runaway re-import loop: importing a large 3x-ui database takes longer
  than the web server's 30s write timeout, so the response was severed mid-import
  - the client never saw success and resubmitted, and each retry ran a full
  import and wrote another pre-import backup. The import endpoints now lift that
  deadline (only after authentication; the work stays bounded by the request
  context), so the client receives the result and stops resubmitting.
- Caps pre-import backups: only the newest 10 `s-ui-pre-xui-import-*.db` files
  are kept, so a slow or retried import can no longer fill the database
  directory.
- Restore now rejects a 3x-ui / x-ui database up front with a clear message
  ("use Migrate from 3x-ui") instead of failing later with the cryptic
  `no such table: changes`; schema migration also tolerates a missing `changes`
  table so a genuinely old s-ui backup still restores.
- Backup & Restore dialog: clarifies that Restore is for s-ui backups only and
  distinguishes the quick 3x-ui import from the full review wizard.
- Full release notes: [`docs/releases/v1.5.6-beta6.md`](docs/releases/v1.5.6-beta6.md).

## [1.5.6-beta5] - 2026-06-02 - 3x-ui migration fixes

- Fixes a total-failure bug in the built-in 3x-ui import (`migrate-xui`): the
  dialect hard-coded an `all_time` column (and `last_online`) that neither
  vanilla mhsanaei 3x-ui nor current normalized forks actually have, so every
  real-world import aborted with `no such column: all_time` before reading a
  single row. Source reads are now column-aware (`tableColumns`/`selectColumns`,
  case-insensitive) and default any column a fork omits. Verified end to end
  against a real export: all inbounds, clients, WireGuard endpoints, reality TLS
  (deduplicated), routing and history migrate.
- Fixes setting migration that wrote 3x-ui keys s-ui ignores (`webBasePath`,
  `tgBotEnable`, `tgBotToken`, `tgBotChatId`, `tgRunTime`, `subEnable`). Keys now
  map to their canonical s-ui names (`webPath`, `telegram*`, ...) and the mapping
  is expanded from 9 to 34 settings (web/sub endpoints, display toggles, and the
  Telegram bot incl. CPU threshold, backup and proxy). Source settings with no
  s-ui equivalent are surfaced as visible, skipped plan items instead of being
  dropped silently.
- Makes cross-host / cross-domain migration safe: host- and domain-specific
  settings (listen address, panel/sub domain, on-disk TLS certificate paths, and
  the host-embedding subscription URLs) now default to skip in the plan so they
  do not overwrite the destination server's working configuration and break it;
  ports and paths still migrate, and the operator can re-enable any item in the
  review step. An inbound bound to a specific source listen address now emits a
  warning that it may not exist on the destination host.
- The migrate-xui wizard now includes settings, routing and history by default;
  admin import stays opt-in.
- Full release notes: [`docs/releases/v1.5.6-beta5.md`](docs/releases/v1.5.6-beta5.md).

## [1.5.6-beta4] - 2026-06-02 - security & static-analysis hardening

- Drives backend static analysis to zero findings (`staticcheck`,
  `golangci-lint`, `gosec`), with `go vet` (nilness), the full `go test ./...`
  suite including `-race`, `govulncheck`, and the frontend lint/typecheck/unit
  suites all green. Migrates the deprecated `sing/common/atomic.Int64` to
  `sync/atomic` and the deprecated `net.Error.Temporary()` check to a
  timeout-based one, removes dead code, and makes previously unchecked error
  returns explicit.
- APIv2: an invalid or expired bearer token now returns HTTP `401 Unauthorized`
  instead of HTTP `200` with a `success:false` body. The browser UI is
  unaffected because it uses cookie sessions on `/api`, not bearer tokens on
  `/apiv2`; external API consumers must now check the HTTP status.
- Adds the opt-in `sessionSameSiteStrict` setting (default off) that issues
  session cookies with `SameSite=Strict`; rejects embedded credentials in
  Telegram proxy URLs and private/loopback/link-local/multicast IP literals in
  optional HTTP URL settings; and hardens the constant-time API-token-scope
  comparison against length-multiple-of-256 truncation.
- Emits a `settings_save_succeeded` audit event on successful settings saves and
  enforces backup → restore table-count consistency, including the `tls` no-TLS
  sentinel row.
- Full release notes: [`docs/releases/v1.5.6-beta4.md`](docs/releases/v1.5.6-beta4.md).

## [1.5.6-beta3] - 2026-05-29 - administrator management beta

- Adds administrator creation and deletion to the shared Classic/Nexus
  `/admins` page. Both actions require the current administrator password.
- Blocks self-delete in the backend and UI. Deleting another administrator also
  removes their API tokens, reloads the APIV2 token cache, and makes their
  existing browser sessions invalid because session validation now checks that
  the user still exists.
- Adds `admin_created` and `admin_deleted` audit events, exposes `isCurrent` in
  `/api/users`, and maps the new admin audit events in the Nexus overview.
- Full release notes: [`docs/releases/v1.5.6-beta3.md`](docs/releases/v1.5.6-beta3.md).

## [1.5.6-beta2] - 2026-05-28 - sing-box 1.13.12 settings coverage beta

- Extends sing-box 1.13.12 settings coverage across Classic and Nexus UI by
  sharing the advanced editor surfaces for basics, rules, DNS, TLS, inbounds,
  outbounds, endpoints and services.
- Adds `DomainResolveOptions` editing, route network presets, Dial/Listen/TUN
  advanced fields, top-level certificate trust presets, rule route-options TLS
  fragmentation controls, rule `client` matcher support, HTTP/Mixed system
  proxy controls and protocol-specific advanced options.
- Preserves top-level `certificate` and unknown top-level sing-box config fields
  through backend round-trips, so runtime config generation no longer drops
  certificate trust settings.
- Keeps default/no-op JSON clean: `Off` removes fields, default delays and
  zero marks are not written, empty app/package selections are rejected, and
  `tls_record_fragment` stays mutually exclusive with `tls_fragment`.
- Validation: `npm run build`, `npm run test`, `npm run lint`,
  `go test ./...`, and `go test -tags
  "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale"
  ./core` passed locally.

## [1.5.6-beta1] - 2026-05-27 - sing-box 1.13 UI parity beta

- Adds first-class UI coverage for sing-box 1.13 TLS advanced options,
  including curve preferences, client authentication/certificates,
  certificate public key pins and outbound kTLS controls.
- Fixes route/DNS rule interface-address wire shapes and adds network/Wi-Fi
  state matchers across route rules, DNS rules and inline/source headless
  rule-set rules.
- Adds inline rule-set editing, route `bypass` option serialization, route
  reject `reply`, Naive receive windows/UoT version selection, TUN reset
  mark/NFQUEUE, Tailscale advertise tags, OCM/CCM headers and the
  `oom-killer` service UI/backend registration.
- Adds representative sing-box 1.13 option-unmarshal coverage plus an OOM
  service registry regression test.
- Validation: `npm --prefix frontend run build`, `npm --prefix frontend run
  test`, `npm --prefix frontend run lint`, and `go test -tags
  "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale"
  ./...` passed locally.

## [1.5.5] - 2026-05-26 - stable 1.5.5 release

- Promotes `v1.5.5-beta1` through `v1.5.5-beta4-hotfix2` to stable `v1.5.5`.
- Fixes subscription correctness for shared VLESS UUIDs and Clash WebSocket
  Host headers: `xtls-rprx-vision` is stripped from non-TCP transports, and
  Clash/Mihomo exports keep a usable `ws-opts.headers.Host`.
- Hardens backup export, restore and import rollback. The no-TLS `tls.id=0`
  sentinel is preserved safely, failed imports reopen the live DB,
  `settings.config` carries DNS/routing restore coverage, and backup export no
  longer lets the sentinel collide with real TLS rows.
- Carries the beta4 security/reliability work: forced password reset for
  imported administrators, safer token handling, audit prioritization, streamed
  large X-UI import plans, rollback realtime invalidation, configurable SQLite
  pools, fail-closed IP-monitor reads, bounded rate-limit state, realtime
  self-healing, retry/backoff improvements and data-race fixes.
- Includes frontend hotfixes for the npm lockfile, Playwright/Vite e2e
  stability, reconnect chaos tests and accessibility baseline timeout.
- Updates Go to `1.26.3`, `github.com/sagernet/sing-box` to `v1.13.12`, and
  synchronizes the cronet-go source pin used by release/Docker builds.
- Validation: `go vet ./...`, `go test -race -timeout=10m ./...`, release-tag
  `go build`, and `git diff --check` passed locally. Docker was not available
  in the local workspace; GitHub release/Docker workflows run on tag push.

## [1.5.5-beta4-hotfix2] - 2026-05-26 - backup export TLS sentinel hotfix

- **Backup export with real TLS rows.**
  Problem: the no-TLS sentinel row `tls.id=0` was copied through GORM's normal
  auto-increment create path. When a database also had a real TLS row, SQLite
  could assign the sentinel a generated id and the next real row copy failed
  with `UNIQUE constraint failed: tls.id`.
  Impact: backup export now skips `tls.id=0` during the generic table copy and
  restores that sentinel explicitly with `INSERT OR IGNORE`, so no-TLS inbounds
  keep a valid parent row without colliding with real TLS configs.
- Added regression coverage for a database that contains both `tls.id=0` and a
  regular TLS row.
- Updated release metadata, README install examples and manual workflow default
  tags to `v1.5.5-beta4-hotfix2`.

## [1.5.5-beta4] - 2026-05-26 - problem-fix and technical-debt report

### 1. Security, Authentication And Audit

- **Forced password reset during import.**
  Problem: the UI offered `reset_required` when migrating administrators from
  x-ui, but the backend did not have a durable state for mandatory password
  reset and fell back to the generated-password scenario.
  Impact: users now have `force_password_reset` state, the API contract matches
  the UI, and this import mode no longer generates or exposes a temporary
  password in the import report.
- **Token attack resistance and legacy header sunset.**
  Problem: WebSocket token checks had measurable timing differences, the legacy
  `Token` authorization header had no enforced cutoff date, and legacy API token
  migration could re-enable previously disabled tokens.
  Impact: WebSocket token consumption uses a safer match-and-delete path, the
  legacy `Token` header is rejected after Sunset, and token migration preserves
  the original enabled/disabled state.
- **System-data leak reduction.**
  Problem: system information could expose private and link-local server
  addresses, Telegram backup secrets needed clearer memory ownership, and
  generated administrator passwords in MigrateXui were too easy to reveal on
  screen.
  Impact: internal addresses are filtered from system info, Telegram backup
  payloads and passphrases are zeroed after use, and generated administrator
  passwords stay hidden until explicit reveal and are cleared automatically.
- **Audit priority and signal quality.**
  Problem: under audit queue pressure, warn/security events could be evicted by
  ordinary `info` events; successful legacy secret decrypts created noisy audit
  records; stats commit failures lacked a durable trail; and optional URL
  settings accepted control characters.
  Impact: the audit writer preserves warning/security priority, redundant
  secretbox fallback noise is removed, stats commit failures are recorded, and
  optional URL settings reject control characters and unsafe input shapes.

### 2. X-UI Import, Sync And Admin UI

- **Respecting saved import policy.**
  Problem: background X-UI sync used hard-coded behavior and could ignore saved
  profile fields such as `OnlyNew`, settings/history/routing import and
  administrator handling mode.
  Impact: the scheduler now passes the saved import policy into planning and
  apply, so cron sync follows the administrator's profile settings.
- **Large import handling.**
  Problem: migration plans were read as ordinary multipart fields with an 8 MiB
  limit, which blocked larger panels from using the same apply contract, and
  interrupted uploads could leave temporary directories behind.
  Impact: the multipart `plan` field streams through temporary storage under
  the 200 MiB request cap, and stale `xui-import-*` temporary directories are
  cleaned up by a safe age-based rule.
- **Import isolation and report accuracy.**
  Problem: TLS delete errors in replace mode could be ignored before creating
  replacement records, and skipped WireGuard endpoints were counted as skipped
  inbounds.
  Impact: TLS delete errors abort the transaction and roll back safely, while
  import reports now count skipped endpoints separately.
- **Rollback and recovery UX.**
  Problem: apply errors could send the user back a step without useful context,
  rollback waited a fixed one-second delay before reload, and other active
  sessions were not notified about restored configuration state.
  Impact: MigrateXui shows apply errors inline, rollback waits for a health
  check before reload, and the backend publishes `config_invalidated` after a
  successful rollback.

### 3. Database, Backup And Resilience

- **Backup and migration safety.**
  Problem: SIGHUP timeout was fixed at three seconds, WAL checkpoint failures
  could abort backup on a locked SQLite database, missing `settings.config`
  blocked versioned restore, and post-migration adapt failures were only logged
  as warnings.
  Impact: timeout is environment-configurable, WAL checkpoint falls back from
  `TRUNCATE` to `FULL`, backups without `settings.config` restore with a
  warning, and broken post-migration adapt now stops startup.
- **Database scaling and first-start races.**
  Problem: SQLite pool limits were fixed, and concurrent first-start paths could
  create duplicate default settings.
  Impact: SQLite pool limits can be tuned through environment variables, and
  default settings are created through a database-level idempotent insert path.
- **IP monitor fail-closed behavior.**
  Problem: a transient database read error in the IP-monitor path could allow an
  unknown address in enforce mode.
  Impact: failed `client_ips` reads mark the cache entry as unreliable and move
  enforcement to fail-closed behavior.

### 4. Network, Data Races And Core Stability

- **OOM protection and realtime self-healing.**
  Problem: import-xui rate-limit state could grow without a hard bound under a
  stream of unique IPs, and the frontend could remain in degraded polling mode
  after a short network outage.
  Impact: the rate-limit cache now uses bounded eviction and expired-bucket
  cleanup, while the WebSocket runtime attempts healing reconnects from
  fallback mode.
- **Data race fixes.**
  Problem: concurrent access to core restart timers, the Telegram HTTP client
  and token-use flushing could trigger race detector failures, panics or writes
  through an outdated database handle.
  Impact: critical paths are protected with mutex, single-flight and barrier
  mechanisms, and token-use flush lifecycle is synchronized with database reset
  and API test lifecycle.
- **Smarter retries and storm protection.**
  Problem: cron sync retried too aggressively, token-use write failures lacked a
  backoff circuit, update checks called GitHub without ETag caching, sync error
  reasons were flattened, and WARP authorization headers were spread across
  fragile code paths.
  Impact: retry policies use exponential backoff, token-use flushing has a
  circuit breaker, release checks use `If-None-Match`, sync-failure summaries
  include sanitized error class/detail, and WARP authorized headers are
  centralized.
- **IPv6-safe system info and shared API route registry.**
  Problem: system info could panic on short interface flag/address data,
  including unusual IPv6-only environments, and import-xui routes could drift
  between v1 and v2 API registration.
  Impact: network interfaces are checked by content and length, and import-xui
  endpoints are registered from one route spec for `/api` and `/apiv2`.

## [1.5.5-beta3] - 2026-05-22 - backup config restore safety for DNS and routing

- Config saves now recreate a missing `settings.config` row, and restore rejects
  versioned S-UI database backups that already lost that sing-box config instead
  of accepting an import without DNS and routing rules.
- Added restore coverage that exports and reimports `settings.config` with DNS
  servers and routing rules intact.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.5-beta3`.

## [1.5.5-beta2] - 2026-05-22 - backup restore safety for no-TLS inbounds

- Kept backup exports with no-TLS inbounds foreign-key valid by explicitly
  preserving the `tls(id=0)` sentinel that their `tls_id=0` rows reference.
- Restore now recreates that no-TLS foreign-key parent before migration checks,
  so backups produced before this prerelease can restore instead of failing
  with `Foreign key check failed: inbounds=1`.
- Failed database imports reopen the rolled-back live database instead of
  leaving the running panel with a closed DB handle; SQLite sessions follow
  the live DB after swap and settings reads fail cleanly if the DB is
  temporarily unavailable.
- Added regression coverage for no-TLS backup FK validity, no-TLS migration
  repair and rollback/reopen after a rejected restore.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.5-beta2`.

## [1.5.5-beta1] - 2026-05-22 - subscription correctness for shared VLESS UUID and Clash WS Host

- Stripped `xtls-rprx-vision` flow from non-TCP transports when the same
  client UUID is shared across multiple VLESS inbounds. Affects panel
  sing-box config (`fetchUsersByCondition`), JSON subscription
  (`sub/jsonService.go`) and shareable links (`vlessLink`). Aligns with
  Xray-core's TCP-only flow contract so a TCP+REALITY inbound and a
  gRPC+TLS (or WS) inbound can serve the same UUID without breaking the
  non-TCP one (alireza0/s-ui#1127).
- Fixed Clash `ws-opts.headers` so the WebSocket `Host` header is
  emitted again. The previous `[]interface{}` cast against a
  map-shaped header silently dropped it, causing Mihomo handshake
  failures through strict CDN / Nginx upstreams. The exporter also
  falls back to TLS `server_name` when no explicit Host is set so the
  upstream always sees a Host that matches the SNI
  (alireza0/s-ui#1126).
- Added regression coverage in `service/inbounds_vless_flow_test.go`,
  `util/genLink_vless_flow_test.go` and `sub/clashService_ws_host_test.go`.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.5-beta1`.

## [1.5.4] - 2026-05-22 - stable Nexus UI line + localization cleanup

- Promoted `1.5.4-beta1` through `1.5.4-beta5` to stable `1.5.4`.
- Includes the opt-in Nexus UI mode, canceled read toast hotfix, denser Nexus
  Overview, systemd installer secretbox key bootstrap, and reserved `/ws` path
  boundary fix from the beta line.
- Finished the release localization pass: Persian Telegram, Audit, maintenance,
  backup and IP-limit strings; Vietnamese machine-translation cleanup across
  Telegram, Audit, settings, networking, DNS, TLS, rules and stats; remaining
  Simplified/Traditional Chinese maintenance path strings; and final Russian
  terminology polish.
- Release, Windows and Docker workflow dispatch defaults now target `v1.5.4`.

## [1.5.4-beta5] - 2026-05-22 - reserved path prefix hotfix

- Reserved path validation now matches slashless framework paths on a
  path-segment boundary instead of rejecting every string prefix match.
- Custom paths such as `/wsub/` no longer collide with the reserved
  `/ws` route, while `/ws`, `/ws/` and descendants under `/ws/` remain
  blocked.
- Added regression coverage for the `/ws` boundary behavior used by
  saved panel and subscription path settings.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.4-beta5`.

## [1.5.4-beta4] - 2026-05-22 - installer secretbox key bootstrap

- Systemd installs through `install.sh` now generate a stable
  `SUI_SECRETBOX_KEY` for encrypted settings when no installer-managed
  key exists yet.
- The generated secretbox key is shown once during installation, stored
  in root-only `/etc/s-ui/secretbox.env`, and loaded through an
  installer-owned systemd drop-in.
- Upgrade runs preserve the existing installer-managed key instead of
  rotating it; uninstall removes the drop-in with the rest of the
  systemd install state.
- Documented the installer-managed secretbox key path and retention
  requirement for systemd users.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.4-beta4`.

## [1.5.4-beta3] - 2026-05-22 - Nexus Overview density refinement

- Re-graded Nexus dark surfaces to a deeper navy palette with teal and
  violet accents while keeping classic themes unchanged.
- Removed the standalone Traffic overview panel and the duplicate Health
  KPI, keeping the Live traffic KPI spark on a compact live status sample
  window.
- Reflowed the Overview into a denser three-column primary row and
  compacted Top clients, Recent events and Protocol summaries so the
  dark LTR `en` dashboard fits one `1440x900` viewport.
- Kept the refinement frontend-only: no backend/API/CSRF/CSP drift and no
  runtime or development dependency changes.
- Verified frontend test/lint/build gates, Nexus source/build artifact
  external-origin gates, `TestAdminSecurityHeaders`, and LTR `en` plus
  RTL `fa` viewport coverage across desktop, narrow desktop, tablet and
  mobile widths.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.4-beta3`.

## [1.5.4-beta2] - 2026-05-21 - Nexus overview cancel toast hotfix

- Suppressed the failed notification for canceled duplicate frontend
  reads. Nexus Overview can trigger overlapping dashboard reads during
  startup, and the shared axios dedupe path intentionally cancels the
  older request; this is now kept silent instead of surfacing
  `CanceledError: canceled` as a user-visible toast.
- Added frontend regression coverage for silent cancellations while
  keeping failed notifications for real request errors.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.4-beta2`.

## [1.5.4-beta1] - 2026-05-21 - Nexus UI mode opt-in beta

- Added the opt-in `nexus` UI mode alongside the existing `classic`
  interface. Classic remains the default and Nexus is a per-browser
  localStorage preference.
- Added the UI mode contract, `VITE_ENABLE_NEXUS` kill switch,
  CSP-safe pre-mount anti-FOUC bootstrap, authenticated layout host,
  mode controls and localized Nexus strings.
- Added the Nexus shell, responsive sidebar/topbar behavior, RTL `fa`
  support, Nexus design tokens/themes and the fixed Nexus Overview
  dashboard built from existing data sources.
- Preserved the backend/API/CSRF/CSP surface: no new endpoints, no new
  WebSocket flow, no inline scripts, no external-origin Nexus source
  literals and no runtime/dev dependency changes.
- Verified the final beta with `npm run test`, `npm run lint`,
  `npm run build`, external-origin gates, supply-chain invariance and
  Nexus viewport checks for LTR `en` and RTL `fa` at desktop, narrow
  desktop, tablet and mobile widths.
- Release, Windows and Docker workflow dispatch defaults now target
  `v1.5.4-beta1`.

## [1.5.3] - 2026-05-21 - stable release + Telegram backup schedule UX

- Promoted the release line from `1.5.3-beta` to stable `1.5.3`.
- Telegram database backup scheduling is now configured through friendly
  presets and custom minute/hour intervals while continuing to store the
  existing `telegramBackupCron` setting.
- Existing custom cron expressions remain supported through Advanced cron mode.
- Release, Windows, and Docker workflow dispatch defaults now target
  `v1.5.3`.

## [1.5.3-beta] - 2026-05-20 - aggregated remediation + upstream parity (#1114)

### Multi-chat delivery ledger (P0-P5)

#### Security

- [P0] Hardened SSRF filtering and dial-time validation; tightened backup
  restore path/symlink checks.
- [P1] Hardened CSRF/session lifecycle behavior, including token renewal after
  logout/logout-all and tighter WS token handling.
- [P2] Expanded secret/settings safety checks and migration guardrails.
- [P3] Added listen fallback auditing and restart-path consistency hardening.

#### Reliability / data integrity

- [P0] Closed race-condition paths in tracker/session options/audit writer code.
- [P1] Stabilized realtime fallback behavior and frontend unit-test harness.
- [P2] Added reset hooks, tracker wait guards, and foreign-key migration checks.
- [P3] Unified restart scheduling and reduced global side effects with an
  initial DI slice.
- [P4] Moved the remaining service runtime globals behind a DI-compatible
  runtime while preserving zero-value service compatibility.
- [P5] Completed the logging backend cleanup without changing API endpoint
  behavior.

#### API and runtime behavior

- [P0] Improved trusted-proxy handling and safer import error classification.
- [P1] Tightened realtime/session/CSRF flows and Telegram error taxonomy.
- [P2] Normalized batching and timeout behavior in heavy data paths.
- [P3] Added an initial slog adapter path for gradual migration from
  go-logging.
- [P4] Promoted `slog` to the logger facade, leaving `op/go-logging` isolated
  behind deprecated compatibility APIs.
- [P5] Removed deprecated `logger.InitLogger`/`logger.GetLogger`, moved facade
  output fully onto standard `log/slog`, and kept panel/core log buffering.
- [P5] Removed the legacy `github.com/op/go-logging` module from `go.mod` and
  `go.sum`.
- [P4] Added a checked sing-box tracker revalidation policy for
  `github.com/sagernet/sing-box v1.13.11`.
- [P4] Added a checked SemVer release/version policy and prevented migration
  code from downgrading future `settings.version` values.

#### Frontend

- [P1] Fixed Vitest harness configuration in `frontend/vitest.config.ts`.
- [P1/P2] Aligned CSRF cache clearing, request dedupe boundaries, and realtime
  degraded-mode behavior.

#### Tests and verification

- Baseline and phase reports:
  - `plans/lint-baseline.txt`
  - `plans/lint-baseline-normalized.txt`
  - `plans/fix-validation.txt` (P0)
  - `plans/p1-validation.txt` (P1)
  - `plans/p2-validation.txt` (P2)
  - `plans/p3-architecture-validation.txt` (P3)
  - `plans/p4-architecture-debt-validation.txt` (P4)
  - `plans/p5-logging-cleanup-validation.txt` (P5)
- Each phase includes targeted checks and a final command pass set in its
  validation artifact.

### Traceability (multi-chat policy)

- Prefix each completed item with a phase tag: `[P0]`, `[P1]`, `[P2]`, `[P3]`, `[P4]`, `[P5]`.
- Add references in this format: `(ref: <commit|PR|chat-id>)`.
- Use combined tags for cross-phase items, for example `[P1/P2]`.
- Keep deferred architecture items in a separate section and do not mix them
  into completed bullets.

### Upgrade notes (aggregate window)

- Treat P0->P5 as one release window; create a full SQLite backup before
  upgrade.
- Validate behavior changes around session/CSRF/realtime and listen fallback in
  staging before production rollout.
- Use the phase validation files above as upgrade verification evidence.
- External Go integrations that imported `logger.InitLogger` or
  `logger.GetLogger` must migrate to `logger.Init(logger.Level*)`,
  `logger.Slog(source)`, or `slog.Default()`.

### Rollback (aggregate window)

- Restore the pre-window SQLite snapshot and previous binary/image.
- If rollback crosses session/token behavior changes, invalidate active sessions
  after downgrade and rotate admin credentials.

### Deferred architecture debt

- [P5] No P5 scope item is deferred. The legacy `op/go-logging` dependency and
  deprecated logger compatibility APIs have been removed.

### Reusable template for next multi-chat release

- Use domain sections: Security, Reliability/Data integrity, API/Runtime,
  Frontend, Tests.
- Tag each bullet with phase marker(s) and append traceability refs.
- Add explicit `Upgrade notes` and `Rollback` for the full aggregated window.

### Fixed

- TUIC subscription/share links and Clash export now include `udp_relay_mode`,
  preserving the configured value and defaulting generated links to `quic`
  when it is absent.

### Added

- Scheduled and manual encrypted SQLite database backup to Telegram. The backup
  passphrase is configured only in the Telegram tab, and the feature is off by
  default. New settings and defaults: `telegramBackupEnabled="false"`,
  `telegramBackupPassphrase=""`, `telegramBackupCron=""`,
  `telegramBackupExcludeTables="stats,client_ips,audit_events,changes"`, and
  `telegramBackupMaxSizeMB="45"`. New manual trigger routes:
  `POST /api/telegram/backup/run` and `POST /apiv2/telegram/backup/run`.
- Restore now auto-detects uploaded `SUI-TGBKP\x00` backup envelopes and shows
  a Backup passphrase field in Backup & Restore. Plaintext `.db` uploads are
  still accepted without that field.
- The existing Backup button can optionally download the same encrypted
  envelope via checkbox "Encrypt with Telegram backup passphrase". The checkbox
  is unchecked by default, plaintext download behavior is preserved, and the
  existing `getdb` endpoint uses the new non-breaking query parameter
  `encryptTelegramBackup=true`.
- The main release binary now includes `s-ui decrypt-backup` for offline
  envelope decryption. No separate artifact is required.
- `docs/scope-matrix.md` now documents the `tg_backup_run` operation.

### Changed

- BREAKING: legacy `POST /api/telegram/backup` and
  `POST /apiv2/telegram/backup` now delegate to the new Telegram backup
  service. `backupKey` is removed from every response,
  `telegramBackupEnabled=true` is required, and successful responses include
  `trigger="manual"`. There is no compatibility window. Strict migration step:
  after upgrading, enable `telegramBackupEnabled` in the Telegram tab; otherwise
  the legacy call returns HTTP 503 with `errorClass=disabled`.
- `util/secretbox` now has `EncryptBytes` and `DecryptBytes` helpers for
  byte-oriented secret handling.
- `api/rateLimit.go` has a shared manual Telegram backup bucket for all four
  manual trigger routes: 3 attempts per 60 seconds with `Retry-After`.
- New audit event types: `tg_backup_sent`, `tg_backup_failed`,
  `tg_backup_passphrase_changed`, `tg_backup_manual_encrypted`, and
  `tg_backup_restore_failed`.

### Upgrade notes

- Back up the SQLite database before upgrading. If using the system service,
  stop `s-ui`, copy `s-ui.db` plus any `-wal`/`-shm` sidecars, then start the
  service again.
- Telegram database backup remains disabled until `telegramBackupEnabled` is
  turned on in the Telegram tab and a Backup passphrase is configured.
- Existing integrations that call the legacy Telegram backup endpoints must
  handle the removed `backupKey` field and the new HTTP 503 `disabled` response
  until the setting is enabled.

### Rollback

- Restore the pre-upgrade SQLite backup and previous binary/image if rollback
  is required.
- Encrypted `.db.aes` files remain decryptable with the passphrase that created
  them via any binary containing `s-ui decrypt-backup`.

## [1.5.2-beta-hotfix2] - 2026-05-18 - drop legacy client_ips unique index

### Fixed

- `UNIQUE constraint failed: client_ips.client_name, client_ips.ip` during
  the 3x-ui pre-import auto-backup. `client_ips.ip` is a legacy column
  kept only for backfill since 1.5.x and is empty for new rows; the
  canonical unique key is `(client_name, ip_hash)`. The model still
  carried an obsolete `gorm:"index:idx_client_ips_client_ip,unique"`
  on `(client_name, ip)`, so `database/backup.go` re-created the bad
  index in the temporary backup DB via `AutoMigrate` and the chunked
  copy of `client_ips` failed as soon as one client owned more than
  one row with empty `ip`. After this hotfix the only unique index on
  the model is `(client_name, ip_hash)`.

### Changed

- `database/model/model.go` - removed the legacy
  `idx_client_ips_client_ip,unique` tag from `ClientIP.ClientName` and
  `ClientIP.IP`.
- `cmd/migration/1_5.go` - the `1.5` schema migration drops the legacy
  `idx_client_ips_client_ip` and creates a partial non-unique
  `idx_client_ips_client_legacy_ip ON client_ips(client_name, ip)
  WHERE ip IS NOT NULL AND ip != ''` for fast legacy lookups. The
  migration is fully idempotent (`DROP INDEX IF EXISTS` /
  `CREATE INDEX IF NOT EXISTS`), so installs already on
  `1.5.2-beta` re-run it cleanly when the runner re-enters the `1.5`
  branch on the next start.
- `database/db.go: ensureIndexes` - drops the obsolete unique index at
  every `InitDB`. This is a runtime safety net for installs that
  bypass `MigrateDb` (for example, restoring an older backup outside
  the panel) and ensures the temporary backup DB built by `GetDb("")`
  no longer carries the bad index either.

### Notes

- No new columns, tables, settings, endpoints, scopes or environment
  variables. Combine with the previous hotfix's chunked-backup helpers.
- Regression coverage:
  - `cmd/migration/migration_1_5_test.go` proves the obsolete index is
    no longer created and accepts multiple empty-`ip` rows for one
    client.
  - `database/db_test.go: TestInitDBDropsObsoleteClientIPUniqueIndex`
    boots an old-shape DB with the legacy unique index already in
    place and verifies `InitDB` removes it.
  - `database/backup_test.go: TestGetDbHandlesHashedClientIPsWithEmptyLegacyIP`
    rounds-trips multiple `ip_hash` rows with empty `ip` for the same
    client through `GetDb("")`.

## [1.5.2-beta-hotfix] - 2026-05-18 - backup chunking and SPA upgrade safety

### Fixed

- `too many SQL variables` during database backup and 3x-ui migration on
  installs with large `stats`, `client_ips`, `audit_events`, `changes` or
  `clients` tables. The backup routine in `database/backup.go` no longer
  emits a single multi-row `INSERT VALUES (...)` that exceeded SQLite's
  compile-time variable limit (`SQLITE_MAX_VARIABLE_NUMBER = 999` in
  `mattn/go-sqlite3`). This unblocks `WritePreImportBackup` and the
  3x-ui migration on production-sized databases (≈40k+ rows in `stats`).
- Stale `index.html` after upgrade no longer breaks the Clients tab.
  `/<base>/assets/*` now returns a real 404 for missing files instead of
  falling through to the SPA fallback, so browsers stop receiving
  `text/html` for JS module requests
  (`Failed to load module script` / `Failed to fetch dynamically imported
  module`). `index.html` is served with `Cache-Control: no-cache, no-store,
  must-revalidate`; hashed assets keep `public, max-age=31536000, immutable`.
- The Vue Router now listens for `vite:preloadError` and triggers one
  guarded `window.location.reload()` (a `sessionStorage` flag prevents
  reload loops), so tabs left over from the previous build pick up the
  new bundle automatically.
- `service/client.go` (`addbulk`, `editbulk`, `ResetClients`,
  `DepleteClients`) and `database/importxui/history_routing.go` (historical
  traffic import) now chunk their bulk `Save`/`Create` calls through new
  `database/bulk.go` helpers (`SafeSQLiteBatchSize`, `CreateInBatchesSafe`,
  `SaveInBatchesSafe`). Reset/deplete jobs and historical-stats imports
  no longer fail on installs with thousands of clients.

### Notes

- No schema migrations, new endpoints, scopes or environment variables.
- A regression test (`database/backup_test.go`) now creates ≈43k `stats`
  rows plus 5k `client_ips` and verifies `GetDb("")` round-trips them.

## [1.5.2-beta] - 2026-05-18 - 3x-ui migration suite

### Added

- 3x-ui configuration import: `s-ui import-xui` CLI, `POST /api/import-xui`
  HTTP endpoint, and a "Migrate from 3x-ui" section in the Backup & Restore
  modal. Import runs in one transaction with auto-backup, supports
  `merge`/`replace`/`skip` strategies, and writes `xui_import` audit events.
- Full migration wizard at `/migrate-xui`: per-object plan/apply with
  `Source.Hash` validation, WebSocket `xui_import_progress` events, JSON
  preview, rollback to the auto-backup, and downloadable JSON/Markdown
  reports. Reports live in `audit_events.details`.
- Remote 3x-ui sources via `--remote ssh://...` and `--remote http://...`
  (xuihttp), plus `s-ui sync-xui` for scheduled incremental syncs. SSH uses
  host-key TOFU with a `xui_known_hosts` table; HTTP supports the 3x-ui
  login flow.
- Encrypted `xui_sync_profiles` (AES-GCM with HKDF-SHA256 from
  `config.GetSecret()`, override via `XUI_PROFILE_KEY_FILE`),
  `cmd/migration/1_7.go` schema migration, `xuiSyncJob` cron job, and the
  `/migrate-xui/schedule` UI for managing profiles.
- Best-effort historical traffic import (`client_traffics`/`outbound_traffics`
  → `stats` aggregates) and Xray routing rules import (`geosite:*`/`geoip:*`,
  block, direct) into sing-box `route.rules`/`dns.servers`. Balancers are
  reported as warnings.
- New `xui_remote` token scope required for all remote/sync endpoints;
  local `/api/import-xui*` endpoints stay under `database`/`admin`.
  `XUI_DISABLE_REMOTE=1` disables remote sources and the cron mode.

### Notes

- `test-db/` holds local 3x-ui import fixtures with real production data
  and is no longer tracked in the repository (see `.gitignore`). Tests that
  need those fixtures are skipped automatically on CI; run them locally
  with the fixtures present in `test-db/`.

## [1.5.1-beta] - 2026-05-17 - remediation hardening and UI completion

### Security

- Telegram notifications now use an async bounded queue with retry/backoff and
  audited overflow/failure events, so login and other handlers are not blocked
  by Telegram network failures.
- Telegram event payloads, audit details, change history payloads, and backup
  captions are redacted so bot tokens, proxy credentials, API tokens, and
  backup keys are not written to logs, audit, changes, or captions.
- Realtime WebSocket handshakes now enforce Origin allow-listing, per-IP
  handshake rate limits, one-time token replay rejection, ping/pong heartbeat,
  idle close, and session-rotation close-all semantics.
- `GET /api/security/audit` now has admin scope gating for API-token requests,
  endpoint rate limiting, cursor pagination, and validated `event`/`severity`
  filters.
- `POST /api/telegram/test` is admin-scoped for API-token requests and writes
  an audit event containing only success/errorClass metadata.
- Security headers middleware was added for the panel and subscription server,
  with no-store cache handling on subscription responses.
- Fresh-install admin passwords are no longer written to application logs; the
  generated password is saved once to `<dataDir>/initial-admin.txt` with
  owner-only permissions, and startup only prints the file path.
- `s-ui admin -show` no longer prints the stored password hash; it shows the
  username and reset guidance instead.
- The frontend clears cached CSRF tokens after logout, logout-all, and
  realtime session-rotation closes so the next mutating request fetches a new
  token.
- `install.sh` now downloads the release `*.sha256` file and verifies the
  Linux tarball with `sha256sum -c` before extraction.
- Added a pull-request CI workflow for Go vet/race tests and frontend
  lint/unit/build checks.
- Admin web sessions now use a SQLite-backed server-side store; the browser
  cookie contains only a signed session ID and session data lives in the local
  `sessions` table.

### Privacy and subscriptions

- Client IP history is stored with salted hashes by default, raw display is
  disabled unless explicitly opted in, and retention is handled by cron GC.
- IP limiting still starts in monitor mode; enforce mode rejects only new
  over-limit connections and does not close active sessions.
- Subscription settings from the design are now persisted and used by link,
  JSON, and Clash subscription responses. Subscription paths are validated
  against reserved prefixes, headers are sanitized centrally, and the per-IP
  subscription rate limit is configurable.
- `POST /api/rotateSubSecret` rotates per-client subscription secrets with an
  audit event. When `subSecretRequired=true`, legacy name URLs return 404.

### Telegram and observability

- Telegram egress can use validated HTTP/HTTPS/SOCKS5 proxy settings stored as
  secret-aware settings. Error classes are normalized to
  `unauthorized`, `chat_not_found`, `rate_limited`, `network`, or `unknown`.
- CPU hysteresis alerts, scheduled Telegram reports, and encrypted Telegram
  database backup export are implemented and remain opt-in.
- Observability history now uses bounded buckets (`2s`, `30s`, `1m`, `5m`),
  sampled by cron, with validated metric/bucket/since API parameters.
- `GET /api/logs` accepts bounded `count`, `level`, `source`, and substring
  `filter` parameters; `GET /api/version` performs a fail-soft 1h-cached
  GitHub release check.
- Database import/export now enforces a 64 MiB cap, SQLite magic validation,
  temporary staging, read-only `PRAGMA integrity_check`, and audit events.

### Frontend

- Added the realtime frontend store with websocket reconnect/degraded states
  and polling fallback.
- Added secret-aware settings fields that show `••• stored •••` and never
  submit the placeholder as a secret value.
- Added IP history modal with raw-IP masking by default and confirmation before
  showing raw IPs to admins.
- Added Telegram settings and Audit views. The Audit view uses cursor
  pagination and server-side `event`/`severity` filters.

### Packaging and CI

- Docker builds now include a `CRONET_GO_VERSION` argument synchronized with
  `release.yml` and document the dated fallback to upstream's latest prebuilt
  `libcronet` asset until commit-addressable assets are available.
- The Docker image default `TZ` now matches the panel default
  `Europe/Moscow`.
- The manual release workflow now defaults to tag `v1.5.1-beta`.
- The container entrypoint no longer runs a duplicate automatic migration
  before startup; use `SUI_MIGRATE_ONLY=1` for a manual migration-only run.
- The migration runner now performs the SQLite WAL checkpoint only after a
  successful transaction commit, fixing `database table is locked` failures
  seen during `1.4.x` to `1.5.1-beta` upgrades.
- The admin frontend no longer depends on an inline base-path script, so the
  strict Content Security Policy is honored and custom web paths route API,
  CSRF and realtime fallback requests correctly.

### Tests

- Added or extended regression coverage for secret settings migration,
  redaction, IP monitor cache/enforce behavior, audit filtering/rate limits,
  subscription header injection and 404 legacy URL behavior, realtime Origin,
  replay token and heartbeat behavior, migrations, and frontend websocket/IP
  helper behavior.
- Verification in this workspace: `go vet ./...`, `go test ./...`,
  `npm run test:unit`, `npm run build`, and `npm run lint` pass. Race tests
  require CGO and a C compiler; this Windows workspace currently lacks `gcc`.

### Upgrade notes

- Back up the SQLite database before upgrading. If using the system service,
  stop `s-ui`, copy `s-ui.db` plus any `-wal`/`-shm` sidecars, then start the
  service again.
- Legacy `/apiv2/*` `Token` header support remains temporary. Move clients to
  `Authorization: Bearer <token>` before the Sunset date:
  `Sat, 15 Aug 2026 00:00:00 GMT`.
- All new features remain off by default except realtime websocket support
  with frontend polling fallback and monitor-only IP tracking.

## [1.5.0] - 2026-05-15 - security foundation and realtime platform

### Security

- Added an Admins panel action to invalidate all admin web sessions at once.
  The action rotates the session generation and clears the initiator's current
  cookie; API tokens are not revoked.
- Added an AES-GCM/HKDF secretbox helper for sensitive settings. New
  secret-aware settings are encrypted with `SUI_SECRETBOX_KEY` when set, or
  with the legacy `settings.secret` compatibility key with a startup warning.
- Secret-aware settings are masked from `api/settings` as `<key>HasSecret`;
  saving an empty value keeps the previously stored secret.
- Added the `audit_events` table, redaction helper, retention setting, and
  `/api/security/audit` endpoint. Login, logout, logout-all-admins, credential
  changes, and API token create/delete actions now write redacted audit events.
- Added CSRF protection for browser `/api/*` mutating requests. `GET /api/csrf`
  issues a session-bound token, frontend requests send it as `X-CSRF-Token`,
  and invalid or expired tokens return HTTP 403. Bearer-token `/apiv2/*`
  requests are not affected.
- API tokens are now migrated from plaintext to salted SHA-256 hashes using
  the per-install `installSalt`; new tokens are shown only once, stored as
  hash/prefix metadata, and can be enabled or disabled from the Admins UI.
- `/apiv2/*` now accepts `Authorization: Bearer <token>` as the primary API
  token transport. The legacy `Token` header still works, emits audit events,
  and returns `Deprecation` plus `Sunset: Sat, 15 Aug 2026 00:00:00 GMT`.
- Added per-client subscription secrets. New `/sub/<secret>`,
  `/sub/json/<secret>`, `/sub/clash/<secret>`, `/json/<secret>`, and
  `/clash/<secret>` routes are supported; legacy `/sub/<name>` remains enabled
  until `subSecretRequired=true`.
- Subscription endpoints now sanitize response headers, validate configured
  subscription paths, and apply a per-IP rate limit.

### API

- Added grouped API route placeholders for the `1.5.0` security,
  notification, observability, and bulk outbound-check work while preserving
  the existing one-level `/api/<action>` endpoints.
- Added `GET /api/observability/history`,
  `GET /api/observability/core-history`, and `GET /api/version`.
- Added `POST /api/checkOutbounds` for bounded bulk outbound checks with
  concurrency `8`, per-outbound timeout `5s`, total timeout `60s`, and an
  HTTPS/public-IP target validator.
- Added disabled-by-default Telegram notification service and
  `POST /api/telegram/test`. Bot token and proxy-related settings are
  secret-aware; login, logout-all-admins, and core restart events notify only
  when Telegram is explicitly enabled.
- Added authenticated realtime WebSocket foundation under
  `/api/realtime/ws-token` and `/api/realtime/ws` with one-time tokens,
  bounded client queues, per-user/per-IP connection limits, and frontend
  polling fallback. `logoutAllAdmins` closes active realtime sockets with
  close code `4401`.
- Added batched client IP monitoring with `client_ips`, per-client `limitIp`
  and `ipLimitMode`, last-online/IP-count metadata, Admins-audited clear
  action, and Clients UI controls. `monitor` is the default mode; `enforce`
  rejects only new over-limit connections and never closes active connections.

### Localization

- `install.sh` and the `s-ui` management menu now also offer Chinese as
  option **3. 中文**; `SUI_LANG=zh` is supported for non-interactive installs.

## [1.4.3] - 2026-05-15 - sing-box runtime update

This release updates the embedded sing-box runtime from `v1.13.4` to
`v1.13.11` and keeps the panel, REST API, frontend forms, and database
schema unchanged.

### Runtime

- Updated `github.com/sagernet/sing-box` to `v1.13.11`.
- Accepted the matching upstream dependency set, including `sing v0.8.9`,
  `sing-tun v0.8.9`, `sing-quic v0.6.1`, and the April 2026 `cronet-go`
  modules required by NaiveProxy.
- Pinned the Linux release workflow to the full `cronet-go` commit
  `e4926ba205fae5351e3d3eeafff7e7029654424a` so release builds do not use a
  short commit prefix for the source checkout.

### Compatibility and Security

- No database migration is required; stored inbound/outbound/endpoint/service
  JSON remains compatible with `sing-box v1.13.11`.
- No web UI fields were added because `sing-box 1.13.5` through `1.13.11`
  only contain fixes and runtime updates, including the fake-ip DNS fix,
  NaiveProxy update, and process searcher regression fix.
- Production upgrades should deploy the full release archive or rebuilt image
  so the updated `libcronet.so`/`libcronet.dll` stays in sync with the new
  binary.

### Verification

- `go mod verify`
- `go test ./...`
- `go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" ./...`

## [1.4.2-beta] - 2026-05-14 - security and reliability hardening

This release rewrites large parts of the auth, transaction, and runtime
control flow, hardens the external-subscription fetcher against SSRF,
and renames the Go module to `github.com/deposist/s-ui-x`.

The full backend test suite (`go test`, `go test -race`,
`go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale"`)
and the full frontend pipeline (`npm ci`, `npm run build`, `npm run lint`,
`npm audit --audit-level=high`) pass clean.

### Changes
- Plaintext passwords replaced with bcrypt; existing accounts migrate
  transparently on first successful login.
- First-run admin password is randomly generated and printed once to the
  application log (no more shipped `admin/admin`).
- Login rate limiter (5 failures / 15 minutes / 15 minutes block) with
  bounded memory.
- Bilingual (English/Russian) `install.sh` and `s-ui` management menu;
  language pickable on first run, switchable from menu item **21.
  Language**, persisted in `/etc/s-ui/lang`. Default language is English.
- Default panel timezone changed from `Asia/Shanghai` to `Europe/Moscow`.
- Default frontend locale changed from Simplified Chinese to English
  (existing installations keep their saved `localStorage.locale`).
- External subscription URL fetcher rejects private/loopback/link-local
  targets and re-validates the resolved IP at dial time, blocking
  DNS-rebinding attacks.
- Configuration saves no longer leave the panel and sing-box out of sync
  on commit/start failures.
- Race-free core lifecycle, online-stats tracking, last-update
  bookkeeping, and v2 token store.
- Frontend code splitting re-enabled; `v-html` removed from the
  remaining surfaces; `AbortController` replaces deprecated
  `axios.CancelToken`.

### Breaking / behaviour changes

- **Module path**: `github.com/alireza0/s-ui` → `github.com/deposist/s-ui-x`.
  Source consumers must update imports. Pre-built binaries are unaffected.
- **Default admin password**: on a fresh database, a random 24-character
  password is generated. Look for the line
  `created initial admin user. username=admin password=...` in the
  application log on first start. **Existing databases keep their
  configured admin user**; nothing is reset.
- **`X-Forwarded-For`**: ignored unless `SUI_TRUSTED_PROXIES` lists the
  immediate client. When set, the chain is walked **right-to-left** and
  the first non-trusted hop wins. Previously the leftmost (easily
  spoofed) value was returned.
- **Login lockout**: 5 failed logins from the same client IP within 15
  minutes block that IP for 15 minutes.
- **Subscription fetcher TLS**: `InsecureSkipVerify` was removed.
  Self-signed origins must now use a certificate trusted by the system
  store.
- **Subscription fetcher private targets**: blocked by default. Set
  `SUI_ALLOW_PRIVATE_SUB_URLS=true` to opt back in (e.g. for `127.0.0.1`
  origins on the same host).
- **Sub fetcher size cap**: responses larger than 4 MiB are rejected.
- **Cookie store**: cookies are now `HttpOnly`, `SameSite=Lax`, and
  `Secure` when the request is HTTPS (directly or via a trusted proxy
  that sent `X-Forwarded-Proto: https`).
- **Frontend dedupe**: only `GET`/`HEAD`/`OPTIONS` requests are deduped;
  concurrent mutating requests no longer cancel each other.

### Security

| Severity | Change |
| --- | --- |
| High | Replaced plaintext password storage with bcrypt hashes (`util/common/password.go`). Existing entries are detected via the `bcrypt:` prefix or the `$2[aby]$` cost markers. |
| High | Lazy migration: a successful login with an unhashed password updates the DB record to a bcrypt hash. |
| High | Fixed `admin/admin` default removed; first-run admin password is randomly generated by `common.Random(24)` and logged once (`database/db.go.initUser`). |
| High | Login rate limiter introduced (`api/rateLimit.go`), with periodic state pruning and a hard cap of 4096 tracked keys to prevent unbounded memory growth. |
| High | Hardened session cookies with `HttpOnly`, `SameSite=Lax`, and HTTPS-aware `Secure` (`api/session.go`). |
| High | `X-Forwarded-For` is only consulted when `SUI_TRUSTED_PROXIES` is set; the parser now walks the chain right-to-left and returns the first non-trusted hop instead of the easily spoofed leftmost value (`api/utils.go`). |
| High | Replaced unsafe SQL string concatenation with parameterized queries in `service/config.go.GetChanges` and `service/config.go.CheckChanges`. |
| High | Static identifier allow-list inside the inbound user-fetch SQL builder (`service/inbounds.go.fetchUsersByCondition`) so future inbound types cannot become a SQL-injection vector. |
| High | Removed default TLS verification bypass for external subscription fetches (`util/subToJson.go`). |
| High | External subscription URL validation: HTTP/HTTPS only, blocks `localhost`/private/link-local/multicast/unspecified by default, opt-in via `SUI_ALLOW_PRIVATE_SUB_URLS=true`, response capped at 4 MiB. |
| High | DNS-rebinding-resistant dialer: a custom `http.Transport.DialContext` re-validates each resolved IP and dials the validated address directly, so an attacker DNS that swaps records between validation and dial cannot escape the filter. |
| Medium | Replaced `error` swallowing in `WarpService.getWarpInfo`/`RegisterWarp`/`SetWarpLicense` with explicit status-code and JSON-parse checks; replaced manual JSON formatting with `encoding/json` to avoid escaping bugs. |
| Medium | Domain validator middleware now compares case-insensitively and handles bare IPv6 hosts. |

### Reliability / data integrity

- Backup export now includes the `services` and API `tokens` tables (`database/backup.go`).
- Backup import (UI: **Backup → Restore**) now also runs the schema migrations and the post-migration adapter (`database.AdaptToCurrentVersion`) automatically. Old backups (S-UI 1.0/1.1/1.2/1.3 layouts, plaintext passwords, missing `services`/`tokens` tables, missing `version` row) are upgraded to the current shape on the fly. If migration fails, the previous live database is restored and an error is returned to the panel - no half-migrated state on disk.
- Schema migrations (`cmd/migration`) now return errors instead of calling `log.Fatal`, so a bad import no longer kills the panel process; the version pin is upserted instead of expecting an existing row.
- The same migration + adaptation pipeline runs at panel start (`app.Init`), so a fresh binary on top of an existing 1.x database upgrades automatically.
- Added `database.AdaptToCurrentVersion`, an idempotent post-migration step that:
  - rehashes any plaintext passwords with bcrypt (legacy backups before this fork shipped them in clear);
  - re-applies the new `idx_stats_lookup`/`idx_changes_lookup`/`idx_clients_name` indexes;
  - bumps the `settings.version` row to the build version so the migration runner short-circuits next time.
- Database path construction uses `filepath.Join` instead of string concatenation.
- Database init creates `idx_stats_lookup`, `idx_changes_lookup`, and `idx_clients_name` indexes for the hottest queries (`database/db.go.ensureIndexes`).
- SQLite connection pool tuned: `SetMaxOpenConns(8)`, `SetMaxIdleConns(4)`, `SetConnMaxLifetime(time.Hour)`, with `_busy_timeout=10000` and `_journal_mode=WAL` already in the DSN. Avoids `SQLITE_BUSY` storms during stats inserts.
- Transaction commits in `service.config.Save`, `service.stats.SaveStats`, and `service.client.DepleteClients` are checked; a failed commit is now reported up the call chain instead of being silently dropped.
- Configuration saves only mutate sing-box runtime state **after** a successful DB commit. The previous behaviour could end with a runtime change applied but a rolled-back DB.
- User-driven core restarts (`RestartCore`) bypass the cron cooldown so the API reflects the real start status. The cron `CheckCoreJob` continues to respect the cooldown.
- Inbound restart and `GetSingboxInfo` are now nil-safe against a concurrent core stop/start (previously could panic with `nil pointer dereference` on `corePtr.GetInstance().ConnTracker()`).
- Race-detector-clean synchronization around:
  - API tokens (`api/apiV2Handler.go`, now a `map[string]TokenInMemory` with O(1) lookup).
  - Online stats (`service/stats.go.onlineResources`) - readers receive a deep copy under `RWMutex`.
  - Core running state and instance pointer (`core/main.go.Core`).
  - Last-update bookkeeping (`service/config.go.LastUpdate`).
- HTTP server now sets `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `tls.Config.MinVersion = tls.VersionTLS12` for both the panel and the subscription server.

### Frontend / tooling

- Fixed `npm ci` by syncing `package-lock.json`.
- Migrated ESLint to flat config (`frontend/eslint.config.mjs`).
- Lint script now reports without auto-fixing (`"lint": "eslint ."`).
- `npm audit --audit-level=high` reports 0 vulnerabilities.
- Axios setup moved onto the exported instance; deprecated `CancelToken` replaced with `AbortController`. Dedupe limited to idempotent reads.
- Removed unsafe `v-html` from `Logs.vue`, `RuleImport.vue`, the IP lists in `Main.vue`, and the gauge tile (`components/tiles/Gauge.vue`).
- Fixed `enableTraffic=false` not propagating to the store, `loadClients` crashing on empty results, and the unused filtered status request list in `Main.vue.reloadData`.
- Re-enabled Vite code splitting; bundle output uses `[hash].js`/`[hash].css` filenames.

### Localization & defaults

- `install.sh` and the `s-ui` management menu are now bilingual
  (English / Russian). On first run the user is asked to pick a
  language; the choice is stored in `/etc/s-ui/lang` and reused on
  subsequent runs. `SUI_LANG=en|ru` overrides interactively or in CI.
- Added menu item **21. Language** so the user can switch UI language
  without editing files.
- Default `timeLocation` for the panel changed from `Asia/Shanghai`
  to `Europe/Moscow`.
- Default frontend locale (and Vuetify locale) changed from
  `zhHans` (Simplified Chinese) to `en`. The user-selected locale
  saved in `localStorage` is still honoured, so existing browsers
  keep their language.

### Repository / packaging

- Go module renamed to `github.com/deposist/s-ui-x`; all internal imports updated.
- `frontend/go.mod` keeps root-level `go` commands away from `frontend/node_modules`.
- README, `install.sh`, `s-ui.sh`, `docker-compose.yml` updated to point at `https://github.com/deposist/s-ui-x` and `ghcr.io/deposist/s-ui-x`.

### Tests

New regression tests:

- `util/common/password_test.go` - hashing, plaintext detection, migration flag.
- `util/subToJson_test.go` - URL validation rejects `file://`, `localhost`, RFC1918, IPv6 loopback; opt-in restores private targets.
- `util/subToJson_dial_test.go` - dialer hook rejects loopback addresses post-validation; opt-in allows them.
- `service/setting_test.go` - default port omission for `subURI`.
- `database/backup_test.go` - backup includes `services` and `tokens`.
- `database/adapt_test.go` - legacy plaintext password rehashing during import is correct, idempotent, and bumps `settings.version`.
- `api/rateLimit_test.go` - block on max failures, reset clears state, concurrent access.
- `api/utils_test.go` - XFF parsing matrix (untrusted client, rightmost untrusted hop, all-trusted fallback, spoofed XFF from untrusted client).

### Verification

| Command | Result |
| --- | --- |
| `go build ./...` | OK |
| `go vet ./...` | OK |
| `go test -count=1 ./...` | OK |
| `go test -count=1 -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale" ./...` | OK |
| `go test -race -count=1 ./...` | OK (requires CGO and a C compiler, e.g. `C:\msys64\ucrt64\bin\gcc.exe`) |
| `npm ci` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `npm audit --audit-level=high` | OK (0 vulnerabilities) |

## Upgrade guide (English, TL;DR)

You can upgrade in place without losing data or reconfiguring the server.
The DB schema is migrated automatically on every panel start
(`app.Init` → `cmd/migration` → `database.AdaptToCurrentVersion`),
existing settings/inbounds/outbounds/clients/tokens stay intact, and
plaintext admin passwords migrate to bcrypt automatically on the next
login. Backups taken from older S-UI builds (1.0/1.1/1.2/1.3) can be
restored straight from the panel and will be brought up to the current
schema in the same flow.

1. Make a backup, just in case:
   - via panel: **Backup → Backup**, save the resulting `s-ui_*.db`;
   - or copy the file: `cp /usr/local/s-ui/db/s-ui.db /root/s-ui.db.bak`.
2. Stop the service: `systemctl stop s-ui`.
3. Replace the binary or the docker image with the new build:
   - manual: extract the new tarball into `/usr/local/s-ui/`;
   - docker: bump the image tag to `ghcr.io/deposist/s-ui-x` and `docker compose pull && docker compose up -d`.
4. Start the service: `systemctl start s-ui`.
5. Log in as usual. Your password is stored in plaintext today; the
   panel hashes it transparently on first successful login.

What you should review after the upgrade:

- If the panel sits behind a reverse proxy and you relied on
  `X-Forwarded-For` (e.g. for IP audit logs), set
  `SUI_TRUSTED_PROXIES=10.0.0.0/8,192.168.0.0/16,...` to the CIDRs your
  proxy lives in. Without this variable, XFF is ignored and audit logs
  show the proxy IP instead of the real client.
- If you fetch external subscriptions from a private endpoint
  (`http://127.0.0.1:.../sub` etc.), set `SUI_ALLOW_PRIVATE_SUB_URLS=true`.
- If you used the old install / update script (`deposist/s-ui`), grab
  the new one once: `wget -O /usr/bin/s-ui https://raw.githubusercontent.com/deposist/s-ui-x/main/s-ui.sh && chmod +x /usr/bin/s-ui`.

## Rollback

If something goes wrong, restoring your backup is enough:

1. `systemctl stop s-ui`.
2. `cp /root/s-ui.db.bak /usr/local/s-ui/db/s-ui.db`.
3. Either restore the previous binary or `docker compose` to the
   previous image tag.
4. `systemctl start s-ui`.

The bcrypt prefix in the `users.password` column is forward- and
backward-compatible with the old binary in the sense that the old binary
will simply not match a hashed password, in which case `s-ui admin -reset`
restores a known credential. So data is safe; only the admin password
might need a CLI reset on rollback.
