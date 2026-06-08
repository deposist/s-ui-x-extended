# s-ui-x-extended v1.0.0-beta4

Pre-release. Makes the extended protocols actually usable end-to-end: the panel
could already *configure* ssh / mieru / sudoku / trusttunnel / mtproxy inbounds,
but had nothing to hand the client — the JSON subscription was empty for them and
mtproxy couldn't even start. This release fixes delivery and consolidates protocol
knowledge into a single embedded source of truth.

## Highlights

- **Client delivery for ssh / mieru / sudoku / trusttunnel.** These inbounds now
  emit a working client outbound in the JSON subscription (previously the server
  config was generated but the subscription dropped them). Per-user credentials map
  correctly (`name → username` for mieru/trusttunnel, `name → user` for ssh).
- **MTProxy via Telegram — and it finally starts.** mtproxy inbounds now produce a
  `tg://proxy` deep link and are excluded from JSON/Clash (there is no sing-box
  mtproxy outbound). Crucially, the per-user `secret` is now a valid faketls (`ee`)
  secret — `0xee || 16-byte key || faketls SNI host` — instead of the bare hex the
  panel used to generate, which the core (`mtglib`) rejected at inbound start.
- **Single source of truth for protocol capabilities.** A new embedded manifest
  (`core/capabilities/protocols.json`) drives the backend maps, the frontend lists,
  the `docs/protocol-matrix.md` table, and a new **admin-only `/api/capabilities`**
  endpoint. The inbound-type picker now greys out protocols not compiled into the
  running binary (detected via `//go:build` flags, not by parsing build scripts).

## Security

- **No server-secret leakage.** Every out_json builder is a strict allow-list (ssh
  copies *nothing* — private host keys never leave the server). A forbidden-keys
  invariant test recursively scans every out_json and subscription body and fails on
  TLS keys, reality private keys, ssh `host_key*`, server `fallback` /
  `handshake_timeout`, etc.; a static test forbids a builder from range-copying the
  whole inbound.
- **`/api/capabilities` is admin-authenticated** and returns only boolean build-tag
  flags and UI capability metadata — no paths, versions, builder names or secrets.

## Diagnostics

- **ShadowTLS is marked `broken`** (not delivered as working): the panel never
  creates the required backing-shadowsocks detour, and the core inbound is not
  fail-closed without one. Auto-pair is deferred.

See [`CHANGELOG.md`](CHANGELOG.md) for the full bilingual list and
[`SECURITY.md`](SECURITY.md) for supply-chain and hardening notes. This is a
pre-release — review `SECURITY.md` before exposing the panel.
