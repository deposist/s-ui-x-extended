## S-UI-X (Extended)

<p align="center">
  <img width="492" height="450" alt="s-ui-x logo" src="https://raw.githubusercontent.com/deposist/s-ui-x-extended/refs/heads/main/docs/592996937-cfc9da97-f8ea-4c68-961c-2bf164932272.png" />
</p>
<p align="center">
  <a href="https://github.com/deposist/s-ui-x-extended/releases/latest">
    <img src="https://img.shields.io/github/v/release/deposist/s-ui-x-extended?style=for-the-badge&label=release" alt="Release">
  </a>
  <a href="https://github.com/deposist/s-ui-x-extended/releases">
    <img src="https://img.shields.io/github/downloads/deposist/s-ui-x-extended/total?style=for-the-badge&label=downloads" alt="Total downloads">
  </a>
  <a href="https://github.com/deposist/s-ui-x-extended/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/deposist/s-ui-x-extended?style=for-the-badge" alt="License">
  </a>
  <a href="https://github.com/deposist/s-ui-x-extended/stargazers">
    <img src="https://img.shields.io/github/stars/deposist/s-ui-x-extended?style=for-the-badge" alt="Stars">
  </a>
</p>

<p align="center">
  <img width="1024" height="768" alt="s-ui-x logo" src="https://github.com/deposist/s-ui-x-extended/blob/main/docs/screen1.png" />
</p>

<p align="center"><b>English</b> · <a href="#русский">Русский</a></p>

> [!NOTE]
> **Public beta (`v1.0.0-beta2`).** Test it before relying on it in production.

## English

**s-ui-x with sing-box-extended core support.** An advanced web panel for sing-box, running on the [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) core (the shtorm-7 fork of `SagerNet/sing-box`).

This project adds extended protocol and transport support on top of the s-ui-x panel, while inheriting the security and reliability hardening of the panel. It is based on `alireza0/s-ui` (current build: `v1.0.0-beta2`).

> **Disclaimer:** this project is intended only for personal learning and knowledge sharing. Do not use it for illegal purposes.

## Table of Contents

- [English](#english)
- [Releases](#releases)
- [Overview](#overview)
- [Key differences vs alireza0/s-ui](#key-differences-vs-alireza0s-ui)
- [Supported Platforms](#supported-platforms)
- [Default Installation Information](#default-installation-information)
- [Install or Upgrade to the Latest Stable Version](#install-or-upgrade-to-the-latest-stable-version)
- [Installer Behavior & Upgrading](#installer-behavior--upgrading)
- [Manual Installation](#manual-installation)
- [s-ui Management Menu & CLI](#s-ui-management-menu--cli)
- [Uninstall S-UI](#uninstall-s-ui)
- [Docker Installation](#docker-installation)
- [Migrate from x-ui / 3x-ui](#migrate-from-x-ui--3x-ui)
- [Backup & Restore](#backup--restore)
- [Environment Variables](#environment-variables)
- [HTTP API](#http-api)
- [Monitoring & Observability](#monitoring--observability)
- [Security & Hardening](#security--hardening)
- [Features](#features)
- [Telegram Notifications](#telegram-notifications)
- [Paid Subscriptions (experimental)](#paid-subscriptions-experimental)
- [Subscription Service](#subscription-service)
- [SSL Certificates](#ssl-certificates)
- [Running Behind a Reverse Proxy](#running-behind-a-reverse-proxy)
- [Development & Build](#development--build)
- [FAQ / Troubleshooting](#faq--troubleshooting)
- [Languages](#languages)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)
- [Acknowledgments](#acknowledgments)

## Releases

The full release history and current release notes live in:

- [`CHANGELOG.md`](CHANGELOG.md)
- Current release notes: [`docs/releases/v1.0.0-beta2.md`](docs/releases/v1.0.0-beta2.md)

The README keeps installation and project overview short. For the full release history and breaking notes, open the changelog. Upgrade and rollback steps live in [Installer Behavior & Upgrading](#installer-behavior--upgrading).

## Overview

| Feature | Support |
| --- | :---: |
| Multiple protocols (proxy / tunnel / VPN) | ✅ |
| Inbound & outbound management | ✅ |
| Multiple clients & inbounds | ✅ |
| Advanced routing & DNS rules | ✅ |
| Subscription links (link / JSON / Clash + info) | ✅ |
| Per-client subscription secrets | ✅ |
| HTTP API with scoped tokens | ✅ |
| Migrate from x-ui / 3x-ui | ✅ |
| Backup & restore | ✅ |
| Telegram notifications | ✅ |
| Paid subscriptions (experimental) | ✅ |
| Client, traffic & system monitoring | ✅ |
| Multiple languages | ✅ |
| Dark / light / system theme | ✅ |

## Key differences vs `alireza0/s-ui`

<details>
  <summary>Show details</summary>

This fork is binary-compatible with `alireza0/s-ui` — drop the new
binary on top of an existing 1.x install, the panel migrates the DB
automatically on first start. The intent is to harden security and
reliability without changing the protocol surface.

- **Auth and session security.** bcrypt with lazy migration, randomly generated first-run password (logged once), login rate limiter, `HttpOnly` + `SameSite=Lax` + HTTPS-aware `Secure` cookies. Sensitive settings (Telegram bot token, proxy credentials, install salt) are encrypted at rest via secretbox; API tokens are stored as salted SHA-256 hashes. CSRF protection is enforced on browser `/api/*` mutating requests.
- **API token scopes.** `admin`, `read`, `write`, `database`, `telegram`, and `observability` scopes are documented in [`docs/scope-matrix.md`](docs/scope-matrix.md), including audit, database, Telegram, subscription-secret rotation, observability, and realtime security-event behavior.
- **`X-Forwarded-For` handling.** Header is ignored unless `SUI_TRUSTED_PROXIES` is configured; the chain is walked right-to-left so spoofed XFF cannot reach IP-based logic.
- **External subscription fetcher.** URL allow-list, blocks private/loopback targets by default (opt-in via `SUI_ALLOW_PRIVATE_SUB_URLS=true`), 4 MiB response cap, DNS-rebinding-safe dial-time IP re-validation. `Authorization: Bearer <token>` is the primary API token transport on `/apiv2/*`; the legacy `Token` header still works with `Deprecation` and `Sunset` headers until `Sat, 15 Aug 2026 00:00:00 GMT`.
- **Realtime WebSocket.** `/api/realtime/ws-token` + `/api/realtime/ws` enforce Origin allow-listing, per-IP handshake rate limits, single-use tokens, ping/pong heartbeat, idle close, and close-all on session rotation. Frontend has a polling fallback for degraded states.
- **Per-client subscription secrets.** `/sub/<secret>`, `/sub/json/<secret>`, `/sub/clash/<secret>`, `/json/<secret>`, `/clash/<secret>` are supported; legacy `/sub/<name>` keeps working until `subSecretRequired=true`. Subscription endpoints sanitize response headers and apply a configurable per-IP rate limit.
- **Telegram notifications (off by default).** Async bounded queue with retry/backoff and audited overflow/failure events. Egress can use validated HTTP/HTTPS/SOCKS5 proxy settings. Telegram payloads, audit details, change history, and backup captions are redacted.
- **Audit and observability.** `audit_events` table with retention GC, scoped `GET /api/security/audit` endpoint with rate limiting and cursor pagination. Bounded observability buckets (`2s`, `30s`, `1m`, `5m`) sampled by cron. Bounded logs API and fail-soft 1h-cached `GET /api/version`.
- **IP monitor (monitor-only by default).** Salted hashes, opt-in raw display, retention GC, per-client `limitIp` and `ipLimitMode`. Enforce mode rejects only new over-limit connections and never closes active connections.
- **SQL safety.** Parameterized queries throughout `service/config.go` and `service/inbounds.go`; static allow-list of inbound types in the user-fetch SQL builder.
- **Backup import / upgrade.** `ImportDB` enforces a 64 MiB cap, SQLite magic check, temporary staging, read-only `PRAGMA integrity_check`, and audit events. WAL/SHM sidecars are cleaned, schema migrations + `AdaptToCurrentVersion` run automatically (rehashes legacy plaintext passwords, refreshes indexes, bumps `settings.version`); the previous DB is restored on any failure.
- **Listen-address resilience.** When the saved `webListen` / `subListen` IP no longer exists on the host, the panel logs a warning and binds on every interface instead of failing with `EADDRNOTAVAIL`.
- **Race-free runtime.** Core lifecycle, online stats, last-update bookkeeping, v2 token store, realtime hub all pass `go test -race ./...` (requires CGO).
- **HTTP server hardening.** `Read/Write/Header/Idle` timeouts and `tls.MinVersion = 1.2` on both the panel and the subscription endpoint. Security-headers middleware (CSP, HSTS when TLS, no-store on subscription responses).
- **WARP registration.** Talks to the current Cloudflare WARP API (`v0a4005`) with proper first-party headers, falls back to `v0a2158`, retries transient TLS handshake failures.
- **Frontend hygiene.** `v-html` removed from logs, rule import errors, IP lists, and the gauge tile. Axios on an exported instance, `AbortController` instead of deprecated `CancelToken`, dedupe limited to idempotent reads, Vite code splitting on. Realtime WS store with reconnect/degraded states. Secret-aware settings fields with `••• stored •••` placeholder. IP history modal with raw-IP masking by default. Telegram settings and Audit views.
- **Localization & defaults.** Multilingual `install.sh` and `s-ui` management menu (English / Russian / Chinese), language switchable at runtime. Default `timeLocation` is `Europe/Moscow`. Default frontend locale is English (existing browsers keep their `localStorage` choice).

</details>

## Supported Platforms

| OS | Architectures |
| --- | --- |
| Linux | `amd64`, `arm64`, `armv7`, `armv6`, `armv5`, `386`, `s390x` |
| Windows | `amd64`, `386`, `arm64` |
| macOS | `amd64`, `arm64` (experimental) |

### Prebuilt vs Docker — build matrix

Not every binary ships the same feature set. What you get depends on **how** you install:

| Build path | Protocol tag set | Naive / cronet |
| --- | --- | --- |
| Prebuilt Linux tarball (default `install.sh`) | Full | Compiled in for `amd64`, `arm64`, `armv7`, `armv6`, `386` (CGO + cronet). **Not** in `armv5` / `s390x`. |
| Prebuilt Windows ZIP | Full | `amd64` only (CGO). `arm64` is built `CGO_ENABLED=0`, so cronet cannot link. |
| Docker image | Full | Yes (`amd64` / `arm64` / `arm`) |
| Self-build (`build.sh`) | Full | Yes |

**Full protocol tag set everywhere.** All four build paths — prebuilt GitHub-release Linux tarballs ([.github/workflows/release.yml](.github/workflows/release.yml)), the Windows ZIP ([.github/workflows/windows.yml](.github/workflows/windows.yml)), Docker ([Dockerfile](Dockerfile)) and self-build ([build.sh](build.sh)) — compile the **same** protocol tags: `with_dhcp`, `with_wireguard`, `with_masque`, `with_mtproxy`, `with_openvpn`, `with_sudoku`, `with_trusttunnel`, `with_ccm`, `with_ocm`, `with_oomkiller`. So WireGuard / WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel and DHCP-DNS work in every build. These protocols are pure-Go (no extra CGO), so they add no cross-compile or static-link cost. The only build-time capability that still varies by platform is **Naive** (cronet/CGO) — see below.

**Naive needs CGO.** The Naive outbound links against cronet, which requires `CGO_ENABLED=1`. On Linux, `armv5` and `s390x` are built without it (`naive: false`); on Windows, only `amd64` is CGO-enabled — `arm64` is built `CGO_ENABLED=0` and ships without cronet. In short, full Naive support is **Linux / Docker** territory.

## Default Installation Information

- Panel port: `2095`, panel path: `/app/`
- Subscription port: `2096`, subscription path: `/sub/`
- Per-IP subscription rate limit (`subRateLimitPerIP`) changes take effect within 1 minute of saving.
- Username: `admin`
- Password (fresh install only): a random 24-character string is generated on first start and written to the log. Look for `created initial admin user. username=admin password=...` in `journalctl -u s-ui` (Linux) or the first-run panel log, then change it from the panel.

Panel settings are stored in the DB and editable from the panel (Settings) or the API; defaults are seeded on first run from [`service/setting.go`](service/setting.go). The most-changed knobs:

| Setting (`key`) | Default | What it controls |
| --- | --- | --- |
| `webPort` | `2095` | Panel HTTP(S) port. |
| `webPath` | `/app/` | Panel base path. |
| `subPort` | `2096` | Subscription server port. |
| `subPath` | `/sub/` | Subscription base path. |
| `subRateLimitPerIP` | `60` | Subscription requests per IP per minute (1–10000). |
| `timeLocation` | `Europe/Moscow` | IANA timezone for scheduling and timestamps. |

Dozens more keys exist (subscription formats/encoding, session cookies, retention, Telegram, payments, certificates, branding, …) — defaulting to disabled or empty. Full source of truth: [`service/setting.go`](service/setting.go).

## Install or Upgrade to the Latest Stable Version

### Linux/macOS

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh)
```

### Windows

1. Download the latest Windows version from [GitHub Releases](https://github.com/deposist/s-ui-x-extended/releases/latest).
2. Extract the ZIP file.
3. Run `install-windows.bat` as Administrator.
4. Follow the installation wizard.

## Install v1.0.0-beta2

This is the initial public beta of s-ui-x-extended — see [`CHANGELOG.md`](CHANGELOG.md)
and [`docs/releases/v1.0.0-beta2.md`](docs/releases/v1.0.0-beta2.md). This is a beta — test first.

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0-beta2
```

Or from a local clone:

```sh
git clone https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.0-beta2
```

## Install an Older Version

Append the version tag with `v` to the installation command. For example, version `v1.0.0`:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0
```

## Installer Behavior & Upgrading

The installer ([install.sh](install.sh)) and the `s-ui` menu ([s-ui.sh](s-ui.sh)) are interactive. The installer first picks a UI language (`SUI_LANG=en|ru|zh`, or interactive — default English, saved to `/etc/s-ui/lang`), then verifies the release tarball against its `.sha256` and **aborts** if the checksum is missing or wrong. After installing it runs `sui migrate` and offers to edit settings (panel/subscription port and path, admin credentials); empty answers keep current values. On a fresh install where you skip editing, it prints a random username/password once — save them. A one-time `SUI_SECRETBOX_KEY` is stored in `/etc/s-ui/secretbox.env`; keep it the same across updates/restores. The panel DB is a single SQLite file at `/usr/local/s-ui/db/s-ui.db`.

**Upgrading from alireza0/s-ui:** binary- and DB-compatible. Stop the panel, back up `s-ui.db` (e.g. `s-ui.db.bak` — your rollback point), drop the new binary on top, and start; the schema migrates forward automatically ([cmd/migration/main.go](cmd/migration/main.go)). Migration is **forward-only** — a DB newer than the binary is left untouched, not downgraded; an incompatible version fails fast rather than corrupting data.

**Rollback:** restore the DB backup over `/usr/local/s-ui/db/s-ui.db`, then reinstall the previous version (menu **3 → Custom version**). Keep the backup until the new version is confirmed working — an upgraded DB will not migrate down, so without it rollback is impossible.

## Manual Installation

### Linux/macOS

1. Download the latest S-UI version for your system and architecture from GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. **Optional:** download the latest `s-ui.sh`: [https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh](https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh)
3. **Optional:** copy `s-ui.sh` to `/usr/bin/` and run `chmod +x /usr/bin/s-ui`.
4. Extract the s-ui tar.gz archive to your chosen directory and enter the extracted folder.
5. Copy the `*.service` files to `/etc/systemd/system/`, then run `systemctl daemon-reload`.
6. Run `systemctl enable s-ui --now` to enable autostart and start the S-UI service.
7. Run `systemctl enable sing-box --now` to start the sing-box service.

### Windows

1. Download the latest Windows version from GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. Download the appropriate Windows package, for example `s-ui-windows-amd64.zip`.
3. Extract the ZIP file to your chosen directory.
4. Run `install-windows.bat` as Administrator.
5. Follow the installation wizard.
6. Open the panel: http://localhost:2095/app

## s-ui Management Menu & CLI

Run `s-ui` (as **root**) with no arguments to open the multilingual interactive menu (English, Russian, Chinese; language persisted to `/etc/s-ui/lang` or forced via `SUI_LANG`). It covers install, update, admin credentials, panel settings, SSL, BBR, and service control.

For scripting, pass a wrapper subcommand: `s-ui start | stop | restart | status | enable | disable | log | update | uninstall` (also `install`, `help`).

The underlying binary at `/usr/local/s-ui/sui` does the real work and exposes: `admin`, `setting`, `uri`, `migrate`, `import-xui` (import from 3x-ui `x-ui.db`), and `decrypt-backup` (decrypt a Telegram backup). `sui -v` prints panel and sing-box versions.

**Locked out / lost password?** Recover from the server shell — no panel login needed:

```bash
/usr/local/s-ui/sui admin -reset
```

This resets the first admin to username `admin` with a fresh random 16-char password printed **once** — save it immediately, as it is stored only as a bcrypt hash and cannot be recovered later. To set your own: `sui admin -username myname -password 'mypass'`.

## Uninstall S-UI

> Tip: the management menu (run `s-ui`, choose **Uninstall**, or `s-ui uninstall`) is the recommended path — it also cleans `/etc/s-ui/` and the systemd drop-in. The manual steps below are a fallback.

```sh
sudo -i

systemctl disable s-ui  --now

rm -f /etc/systemd/system/sing-box.service
systemctl daemon-reload

rm -fr /usr/local/s-ui
rm /usr/bin/s-ui
```

## Docker Installation

<details>
   <summary>Show details</summary>

### Usage

**Step 1:** install Docker

```shell
curl -fsSL https://get.docker.com | sh
```

**Step 2:** install S-UI

> Docker Compose option

```shell
services:
  s-ui:
    image: ghcr.io/deposist/s-ui-x-extended
    container_name: s-ui
    hostname: "s-ui"
    network_mode: host
    volumes:
      - "./db:/app/db"
      - "./cert:/app/cert"
    tty: true
    restart: unless-stopped
    entrypoint: "./entrypoint.sh"
```

`docker compose up -d`

> Direct Docker run

```shell
mkdir s-ui && cd s-ui

docker run -itd \
    --network host \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name s-ui \
    --restart=unless-stopped \
    ghcr.io/deposist/s-ui-x-extended
```

> Build the image yourself

```shell
git clone https://github.com/deposist/s-ui-x-extended
docker build -t s-ui .
```

</details>

## Migrate from x-ui / 3x-ui

s-ui-x-extended can import an existing **x-ui / 3x-ui** deployment by reading its SQLite database (`x-ui.db`). The source file is opened **read-only** and never modified — the importer maps everything into the active s-ui DB in a single transaction that rolls back on any failure, so a failed import never leaves the panel half-migrated. It brings over inbounds, TLS/Reality, clients, panel settings, admins, aggregated traffic history, and routing (best-effort; unmapped objects are flagged for review).

**In-panel wizard** — open **Settings → 3x-ui Migration** (`/migrate-xui`): upload the DB, review a dry-run plan (per-item action `create` / `merge` / `replace` / `skip`), apply with live progress, then download a report or **roll back**. The same flow is exposed over `/api` and `/apiv2` (token scope `database`).

**CLI** — for scripted/server-side use:

```bash
sui import-xui --src /path/to/x-ui.db --strategy merge --include-routing --include-history --yes
```

(Use `--dry-run` to preview; omit `--yes` to be prompted.)

> **Safe by default.** Before any non-dry-run import the panel writes a pre-import snapshot (`s-ui-pre-xui-import-*.db`, newest 10 kept) so you can fully restore the previous database.

## Backup & Restore

Export the entire SQLite database as a single, consistent `.db` file and restore it later — on the same host or a fresh install. Both operations need an authenticated session (or an API token with the `database`/`admin` scope) and are written to the [audit log](#audit-log).

- **Export — `GET /api/getdb`** streams a clean `s-ui_<timestamp>.db` (WAL checkpointed, sidecars stripped). Optional query params let you `exclude` history tables (such as `stats`, `client_ips`, `audit_events`, `changes`) for a config-only backup, or set `encryptTelegramBackup=true` for an AES envelope reusing the [Telegram backup](#telegram-bot--notifications) passphrase. Core config is always included.
- **Restore — `POST /api/importdb`** uploads the `.db` (or `.db.aes` + passphrase) as multipart field `db`. It is defensive: 64 MiB size cap, `SQLite format 3` + read-only `integrity_check` validation, staged off the live DB, safe swap, then automatic migration/adaptation — with **rollback to the original DB on any failure**. On success the panel restarts (~3 s).

> 3x-ui / x-ui (Xray) databases are rejected — use [Migrate from 3x-ui](#migrate-from-3x-ui), not Restore.
>
> Restoring replaces the **entire** database, including admin credentials; log in with the source panel's credentials afterwards.

## Environment Variables

s-ui-x-extended reads its runtime configuration from process environment variables. Set them in the systemd unit (`/etc/systemd/system/s-ui.service`), the Docker `environment:` block, or the Windows service definition — the panel reads them at startup.

| Variable | Type | Default | Purpose |
|---|---|---|---|
| `SUI_DB_FOLDER` | path | `<app-dir>/db` | Directory holding the SQLite database (`<name>.db`). When unset, falls back to the directory of the running binary plus `db` (or `/usr/local/s-ui/db` / `C:\Program Files\s-ui\db` if the binary path cannot be resolved). |
| `SUI_SECRETBOX_KEY` | base64 | _(unset)_ | Out-of-database key used to encrypt secret settings (bot tokens, payment keys, backup passphrase). Must be base64 (standard, raw-standard, or raw-URL) decoding to **≥ 32 bytes**. If unset or invalid, the panel logs a warning and falls back to an HKDF key derived from `settings.secret` — usable, but those secrets then remain decryptable from the database alone. |
| `SUI_COOKIE_KEY` | base64 list | _(unset)_ | One or more keys for signing session cookies. Comma-, semicolon-, or newline-separated; each entry must base64-decode to **≥ 32 bytes**. If unset or invalid, the panel logs a warning and uses an HKDF-derived compatibility key from `settings.secret`. |
| `SUI_FORCE_COOKIE_SECURE` | bool | _(unset)_ | When set to a parseable boolean (`true`/`false`/`1`/`0`), forces the `Secure` flag on session cookies on/off; unset leaves the panel's automatic detection in place. An unparseable value is reported as an error. |
| `SUI_TRUSTED_PROXIES` | CIDR/IP list | _(none)_ | Comma-separated list of trusted proxy CIDRs or bare IPs (IPv4/IPv6) whose `X-Forwarded-For` is honored for client-IP resolution. Invalid entries are logged and skipped; empty means no proxy is trusted. |
| `SUI_ALLOW_PRIVATE_SUB_URLS` | bool | `false` | When exactly `true`, allows subscription-conversion fetches to target private/loopback addresses. Off by default to block SSRF against internal hosts. |
| `SUI_LOG_LEVEL` | enum | `info` | Log verbosity: `debug`, `info`, `warn`, or `error` (case-insensitive). An invalid value logs a warning and falls back to `info`. Overridden by `SUI_DEBUG`. |
| `SUI_DEBUG` | bool | `false` | When exactly `true`, enables debug mode and forces the log level to `debug`. |
| `SUI_SIGHUP_TIMEOUT_SECONDS` | int | `3` | Seconds to wait for the core to reload after a `SIGHUP` during config apply. Accepts `1`–`60`; out-of-range or non-numeric values log a warning and fall back to `3`. |
| `SUI_DB_MAX_OPEN_CONNS` | int | `8` | Maximum open SQLite connections in the pool. Must be `> 0`; otherwise the default is used. |
| `SUI_DB_MAX_IDLE_CONNS` | int | `4` | Maximum idle SQLite connections. Must be `≥ 0`; otherwise the default is used. Clamped down to `SUI_DB_MAX_OPEN_CONNS` if larger. |
| `SUI_BIN_FOLDER` | path | `bin` | **Legacy migration only.** Read solely by the one-time 1→2 schema migration to locate an old `config.json` (under `<app-dir>/<SUI_BIN_FOLDER>/config.json`) and import it into the database. Has no effect on a fresh install. |

> **systemd note:** to set `SUI_SECRETBOX_KEY` for the service, add an `Environment=SUI_SECRETBOX_KEY=...` line (or `EnvironmentFile=`) to `/etc/systemd/system/s-ui.service` and run `systemctl daemon-reload` followed by `systemctl restart s-ui`. The Linux installer can generate and persist this key for you.

### Script-only variables

These are consumed by the install/launch scripts, not by the Go binary:

| Variable | Read by | Purpose |
|---|---|---|
| `SUI_MIGRATE_ONLY` | [entrypoint.sh](entrypoint.sh) | When `1`, the Docker entrypoint runs `./sui migrate` and exits instead of starting the panel. |
| `SUI_LANG` | [install.sh](install.sh), [s-ui.sh](s-ui.sh) | Forces the installer/management-script language: `en`, `ru`, or `zh`. |
| `SUI_HOME` | [windows/s-ui-windows.bat](windows/s-ui-windows.bat) | Windows install directory; set by the installer and read by the Windows control script (defaults to `C:\Program Files\s-ui`). |

## HTTP API

s-ui-x-extended exposes two HTTP API surfaces, both under the configurable panel base path (`webPath`, default `/app/`):

- **`/api/*`** — used by the web UI. Auth is the browser session cookie plus a CSRF token (`GET /app/api/csrf`, sent in the `X-CSRF-Token` header on every mutating request).
- **`/apiv2/*`** — stateless surface for scripts and automation. No session, no CSRF — just an API token.

Create tokens under **Settings -> API tokens**; pick a scope and optional expiry. The plaintext value is shown **once** (only a hash is stored), so save it. Send it on every `/apiv2/*` request as `Authorization: Bearer <your-token>`.

Each token carries exactly one scope:

| Scope | Allows |
| --- | --- |
| `admin` | Everything (and the only scope for audit reads and Telegram test). |
| `read` | Read-only config/identity reads and metrics. |
| `write` | Everything `read` allows, plus mutations such as `save`, `restartApp`, `restartSb`, … |
| `database` | Database export (`getdb`) and import (`importdb`). |
| `telegram` | Manual Telegram backup trigger. |
| `observability` | Operational metrics and history such as `stats`, `status`, `logs`, … |

See [`docs/scope-matrix.md`](docs/scope-matrix.md) for the full per-endpoint matrix.

**Legacy `Token` header (deprecated):** older clients sent the token in a `Token` header; it still works but is on a hard sunset of `Sat, 15 Aug 2026 00:00:00 GMT`, after which it returns `401 legacy token header expired`. Migrate to `Authorization: Bearer` before then.

## Monitoring & Observability

The panel continuously samples host and core metrics and exposes them through read-only JSON endpoints under `/api/` (relative to the configured web base path). All observability data is kept **in memory** — there is no separate time-series database to run.

Key read endpoints:

- `GET /api/onlines` — active online clients (inbound tags, users, outbound tags)
- `GET /api/stats` — per-tag inbound/outbound traffic time series
- `GET /api/status` — on-demand snapshot of CPU, memory, disk, swap, network, sing-box core, and DB
- `GET /api/logs` — bounded, filterable panel + core log buffer
- `GET /api/version` — current version plus latest GitHub release (fail-soft, cached)
- `GET /api/observability/history` and `/api/observability/core-history` — bucketed host (CPU/RAM/network) and core (running, goroutines, alloc, uptime) history from in-memory ring buffers

A cron job samples every ~2s and rolls samples into fixed-size buckets (such as `2s`, `30s`, `1m`, `5m`, …), capping total memory (32 MB by default). The history endpoints require the `observability` or `admin` token scope. See [observability.go](service/observability.go).

## Security & Hardening

The panel ships with defence-in-depth turned on by default. The highlights below are user-facing; for the supply-chain rationale see [SECURITY.md](SECURITY.md).

- **Password storage.** Admin passwords are stored as bcrypt hashes ([util/common/password.go](util/common/password.go)). Legacy plaintext passwords are accepted once via a constant-time compare and lazily re-hashed to bcrypt on next use, so an upgrade transparently migrates the stored credential.
- **Random first-run password.** On a fresh install the panel generates a random 24-character admin password and writes it to the application log once (`created initial admin user. username=admin password=...`). There is no shared default password.
- **Login rate limiting.** Failed logins are throttled both **per source IP** and **per username** (`user|<name>`), so a distributed brute-force that rotates source IPs is still capped on the target account. After 5 failures within a 15-minute window the key is blocked for 15 minutes ([api/rateLimit.go](api/rateLimit.go)).
- **Timing equalization.** The unknown-username path runs a throwaway bcrypt comparison so a non-existent user costs about the same as a wrong password, defeating username enumeration via timing ([util/common/password.go](util/common/password.go)). CSRF and legacy-password checks also use constant-time comparison.
- **At-rest encryption.** Sensitive settings (Telegram bot token, proxy credentials, install salt) are encrypted at rest with AES-256-GCM via the secretbox helper, keyed by HKDF-SHA256 and tagged with associated data ([util/secretbox/secretbox.go](util/secretbox/secretbox.go)). Encrypted values carry the `sbox:v1:` prefix.
- **API tokens.** API tokens are never stored in plaintext; they are kept as SHA-256 hashes salted with the per-install salt ([service/user.go](service/user.go)).
- **CSRF protection.** Browser-driven mutating requests (`POST`/`PUT`/`PATCH`/`DELETE`) under `/api/*` require a valid `X-CSRF-Token` that matches the session token; the token is compared in constant time and expires after 2 hours. The login endpoint is exempt ([api/csrf.go](api/csrf.go)).
- **Hardened cookies.** Session and CSRF cookies are `HttpOnly`, `SameSite=Lax` by default (`Strict` when `sessionSameSiteStrict` is enabled), and `Secure` whenever the request is HTTPS or the configured web URL/domain is `https://` (force with `forceCookieSecure`).
- **Transport & headers.** Both the panel and the subscription server enforce `tls.MinVersion = TLS 1.2` and ship a security-headers middleware ([middleware/securityHeaders.go](middleware/securityHeaders.go)): the admin panel sends a strict `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, and `Strict-Transport-Security` (only over real TLS); subscription responses add `Cache-Control: no-store`.
- **Trusted-proxy XFF handling.** `X-Forwarded-For` is ignored entirely unless `SUI_TRUSTED_PROXIES` is set (comma-separated CIDRs/IPs); the chain is then walked right-to-left through trusted hops, so a spoofed XFF from an untrusted client cannot reach IP-based logic or spoof HSTS via `X-Forwarded-Proto` ([api/utils.go](api/utils.go)).
- **Audit log.** Security-relevant actions are recorded to the `audit_events` table with actor, IP, user-agent and redacted details, and exposed through a rate-limited, scoped, cursor-paginated endpoint ([service/audit.go](service/audit.go)).
- **IP monitor.** Per-client connection-source tracking stores **salted SHA-256 hashes** of client IPs (raw display is opt-in), with a `monitor`/`enforce` mode per client; enforce rejects only new over-limit connections and emits debounced realtime security events ([ipmonitor/ipmonitor.go](ipmonitor/ipmonitor.go)).

### Hardening checklist

1. **Change the default password.** Read the random first-run password from the log, then change it from the panel immediately.
2. **Set a stable `SUI_SECRETBOX_KEY`.** Provide a base64-encoded raw key of at least 32 bytes so at-rest encryption does not fall back to the HKDF-derived `settings.secret`. Keep the key stable across upgrades and back it up — losing it makes encrypted settings unrecoverable. `install.sh` can generate and persist it to `/etc/s-ui/secretbox.env`.
3. **Configure `SUI_TRUSTED_PROXIES`** if the panel sits behind a reverse proxy/load balancer; set it to the proxy's CIDRs/IPs so `X-Forwarded-For` is honoured only from trusted hops. Leave it unset for direct exposure.
4. **Serve over TLS.** Terminate HTTPS (panel TLS or a trusted reverse proxy) so cookies become `Secure` and HSTS is emitted. Optionally force it with `forceCookieSecure`.
5. **Enable `subSecretRequired`** to disable legacy `/sub/<name>` lookups and require per-client subscription secrets.
6. **Restrict the panel.** Don't expose the admin panel to the open internet unnecessarily — bind it to a private interface or front it with an allow-list, and prefer a non-default port/path.
7. **Keep `with_profiler` builds off public interfaces.** The profiler build tag registers `/debug/pprof/*` **without authentication**; it is a development-only tag and is **not** in the default builds. If you build with `-tags with_profiler`, never bind `listen` to a non-loopback address.

### Reporting a vulnerability

Please report suspected vulnerabilities privately via a GitHub **security advisory** on the repository, rather than opening a public issue. Note that [SECURITY.md](SECURITY.md) currently documents supply-chain decisions (dependency forks, build tags, base-image pinning) rather than a contact policy.

## Features

- Supported protocols:
  - General / transparent: Mixed, SOCKS, HTTP, Direct, Redirect, TProxy, Tun
  - V2Ray-based: VLESS, VMess, Trojan, Shadowsocks
  - Other proxy: ShadowTLS, AnyTLS, Naive, Hysteria, Hysteria2, TUIC, TrustTunnel, Mieru, Sudoku, SSH, Tor, MTProxy
  - Tunnel / VPN (endpoints & outbounds): WireGuard, WARP, Tailscale, AmneziaWG 2.0, MASQUE, OpenVPN, VPN server / client
  - Outbound groups & traffic control: Selector, URLTest, Bond, Failover, Fallback, Parser, and bandwidth / connection / traffic / rate limiters
  - DNS transports: TCP, UDP, DoT, DoH, DoQ, DNSCrypt (SDNS), DHCP, FakeIP, Hosts, Local, Fallback, Resolved, Tailscale
  - Most proxy protocols work as both inbound and outbound; MTProxy is inbound-only, Tor / MASQUE / OpenVPN are outbound-only.

> [!IMPORTANT]
> **Build-tag caveat (Naive only).** Current prebuilt GitHub-release Linux tarballs and the Windows ZIP are compiled with the **full protocol tag set**, so **WireGuard / WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel and DHCP DNS** work out of the box — same as Docker and [`build.sh`](build.sh). The one feature that still depends on the build is **Naive**, which links cronet (`CGO_ENABLED=1`); see the [build matrix](#prebuilt-vs-docker--build-matrix). If a protocol still errors with *"… is not included in this build"*, your binary predates this change — upgrade to a current release.

### Transports & TLS

- **Stream transports:** HTTP/2, WebSocket, gRPC, HTTPUpgrade, XHTTP, and mKCP. See [transport.ts](frontend/src/types/transport.ts) and [Transport.vue](frontend/src/components/Transport.vue).
- **REALITY** with one-click key-pair generation (private/public key and Short IDs) and an optional max time difference. Configured in [Tls.vue](frontend/src/layouts/modals/Tls.vue).
- **uTLS fingerprints** for client TLS — `chrome`, `firefox`, `edge`, `safari`, `360`, `qq`, `ios`, `android`, `random`, `randomized` ([OutTLS.vue](frontend/src/components/tls/OutTLS.vue)).
- **ECH** (Encrypted Client Hello) with config text or path and query server name.
- **Multiplex** (`smux`, `yamux`, `h2mux`) with padding and **TCP Brutal** congestion control (up/down Mbps). See [Multiplex.vue](frontend/src/components/Multiplex.vue).
- **XTLS flow** `xtls-rprx-vision` for compatible VLESS clients.
- TLS knobs: self-signed certificate generation, ACME, min/max version, ALPN, cipher suites, curve preferences (incl. `X25519MLKEM768`), client authentication, certificate pinning (`certificate_public_key_sha256`), kernel TLS (kTLS), and TLS fragmentation.

### Routing & DNS rules

- **Routing rules** with a logical/simple rule builder, rich matchers (inbound, network, protocol, sniffed client, domain / domain-suffix / keyword / regex, source & destination IP CIDR, ports & port ranges, process, package, user, clash mode, network type/SSID/BSSID, rule-sets, and more) and the `invert` flag. See [rules.ts](frontend/src/types/rules.ts) and [Rule.vue](frontend/src/layouts/modals/Rule.vue).
- **Rule actions:** `route`, `route-options`, `bypass`, `reject`, `hijack-dns`, `sniff`, and `resolve` — including override address/port, UDP options, TLS fragmentation, sniffers (HTTP, TLS, QUIC, STUN, DNS, BitTorrent, DTLS, SSH, RDP, NTP), and resolve strategies.
- **Rule-sets:** `inline`, `local`, and `remote` (source/binary format, update interval, download detour).
- **DNS rule engine** with its own servers, rules, and actions (`route`, `route-options`, `reject`, `predefined`), DNS server types including Local, Hosts, TCP, UDP, DoT, DoQ, DoH, HTTP/3, DHCP, FakeIP, Tailscale, Resolved, DNSCrypt (SDNS), and Fallback. See [dns.ts](frontend/src/types/dns.ts).
- **Shared SSRF validator** ([util/ssrf/validator.go](util/ssrf/validator.go)) backs every panel-initiated outbound fetch — subscription import, custom outbound check targets, and Telegram proxy egress — rejecting private, loopback, link-local and other reserved ranges with DNS-rebinding-safe dial-time re-validation.

### Outbound providers

- Define outbound providers as **inline**, **local** (file path), or **remote** (URL) sets. See [providers.go](database/model/providers.go), [service/providers.go](service/providers.go), and [providers.ts](frontend/src/types/providers.ts).
- Optional per-provider **health checks** (URL, interval, timeout), plus emoji stripping and include/exclude filters for remote providers.
- Remote providers support a custom user agent, update interval, and download detour. Panel-side URL fetching shares the same SSRF blocklist described above, so remote URLs pointing at private/loopback ranges are blocked by default.

### Dashboard & themes

- **Dual dashboard:** the new **Nexus** UI and the **Classic** dashboard.
- **Themes:** Light, Dark, and **System** (default). See [vuetify.ts](frontend/src/plugins/vuetify.ts).

### More

- Advanced traffic routing interface with PROXY Protocol, External, transparent proxy, SSL certificates, and port configuration support.
- Advanced inbound and outbound configuration interface.
- Client traffic limit and expiration support.
- Online clients, inbound/outbound traffic statistics, and system status monitoring.
- Subscription service supports external links and subscriptions.
- Web panel and subscription service support secure HTTPS access (you must provide your own domain and SSL certificate).

## Telegram Notifications

Off by default. When enabled, the panel sends short event messages to a Telegram chat through your own bot. Configure everything under **Settings -> Telegram**: set the bot token (from [@BotFather](https://t.me/BotFather)) and the numeric chat id, then use **Test** to send a probe. Every message and audit record is redacted first, so bot tokens, proxy credentials, cookies and OTP/TOTP secrets never reach Telegram.

Once on, it notifies on events such as successful/failed logins, all-admins logout, sing-box core restarts, high/normal CPU (hysteresis over `telegramCpuThreshold`, opt-in), and scheduled cron heartbeat reports. Login/restart events carry only privacy-preserving metadata (client IP, a SHA-256 hash of the user agent, timestamp). Messages are queued and delivered asynchronously with retries.

By default the panel reaches `api.telegram.org` directly; set **Transport** to route egress through an HTTP/HTTPS/SOCKS5 proxy or through a running sing-box outbound (by tag).

It can also deliver an encrypted database backup to the same chat as a `*.db.aes` document (Argon2id + AES-GCM, client-side). Enable it, set a passphrase (min 12 chars) and a schedule. Decrypt with the bundled CLI:

```bash
sui decrypt-backup --in s-ui-backup-YYYYMMDD-HHMMSSZ.db.aes --out restored.db
```

> Heads-up: message text itself is plaintext in Telegram. Treat the chat as any external channel.

## Paid Subscriptions (experimental)

> **Experimental and OFF by default.** The module is isolated from the core (own DB tables, started only when enabled) and ships behind an `experimental` chip. Review the flows before pointing it at real money.

This module turns the panel into a small subscription business via a **separate, client-facing Telegram bot** (a token different from the admin notifier). End users fetch their own link / QR / usage stats and **buy or renew access** through a payment provider, without touching the admin panel. Configure it under **Paid Subscriptions** ([PaidSubscriptions.vue](frontend/src/views/paidsub/PaidSubscriptions.vue)); the `paidSubEnabled` setting toggles the bot at runtime.

Each Telegram user maps one-to-one to a single panel client and can only act on that client. Define purchasable **Tariffs** (price, currency, Stars amount, +days, +traffic), and optionally enable **auto-registration** to provision a trial client on first `/start`. Renewals apply exactly once. Supported providers (each off by default) include **Telegram Stars, YooKassa, Stripe, PayMaster, CryptoBot, and an external link** — each appears only when enabled and its token/template is set. **Refunds**: only Telegram Stars are refunded automatically via the Bot API; for every other provider the panel just marks the order refunded — **you must return the money in that provider's own dashboard.**

> **Security:** payment tokens are stored encrypted at rest. For production set a stable **`SUI_SECRETBOX_KEY`** env var (a key kept outside the DB); otherwise a key derived from the database is used. Keep it stable across restarts — rotating or losing it makes stored tokens unrecoverable.

## Subscription Service

A separate HTTP(S) listener (default port `2096`, path prefix `/sub/`) with its own TLS, timeouts, and `no-store` security headers. Each enabled client is served via a per-client secret URL (`sub_secret`, a UUIDv4 auto-generated on first use) — never the client name.

The same client resolves through several routes: `/sub/<secret>` (link list), `/sub/json/<secret>` and `/sub/clash/<secret>`, plus top-level aliases `/json/<secret>` and `/clash/<secret>`. All accept `GET` and `HEAD`. The link route also re-dispatches by query: `?format=json` / `?format=clash` (any other value returns `400`). For backward compatibility `/sub/<name>` still resolves by name unless `subSecretRequired=true`.

Output formats: link list (base64 when `subEncode=true`, else raw), a full sing-box JSON config, and Clash/Mihomo YAML — each gated by `subLinkEnable` / `subJsonEnable` / `subClashEnable`. Responses set subscription info headers such as `Subscription-Userinfo`, `Profile-Update-Interval`, `Profile-Title`, …, all sanitized (control chars stripped, max 512 bytes) so client data can't inject headers. JSON/Clash output is further shaped by `subJson*` / `subClashExt` settings (see [sub/jsonService.go](sub/jsonService.go)).

Hardening: a per-IP rate limit (default `60` req/min, `subRateLimitPerIP`; over the limit returns `429` + `Retry-After`), `Cache-Control: no-store`, and `Host` rejection when `subDomain` is set.

## SSL Certificates

<details>
  <summary>Show details</summary>

### Certbot

```bash
snap install core; snap refresh core
snap install --classic certbot
ln -s /snap/bin/certbot /usr/bin/certbot

certbot certonly --standalone --register-unsafely-without-email --non-interactive --agree-tos -d <your domain>
```

</details>

## Running Behind a Reverse Proxy

Putting nginx (or any reverse proxy) in front of the panel is supported. Terminate TLS at the proxy, pass the panel's loopback port through, and forward the WebSocket upgrade so the realtime feed at `/app/api/realtime/ws` works.

```nginx
location /app/ {
    proxy_pass http://127.0.0.1:2095;
    proxy_http_version 1.1;
    # WebSocket upgrade — required for /app/api/realtime/ws
    proxy_set_header Upgrade    $http_upgrade;
    proxy_set_header Connection $connection_upgrade;  # map default upgrade; '' close;
    proxy_set_header Host              $host;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 3600s;  # realtime stream is long-lived
}
```

**Set `SUI_TRUSTED_PROXIES`** (comma-separated IPs/CIDRs of your proxy, e.g. `127.0.0.1,::1`) or the panel ignores `X-Forwarded-For`/`X-Forwarded-Proto` entirely — every request looks like it came from the proxy, breaking rate limiting, login lockout, audit logs, HSTS, and the `Secure` cookie. Set it as tightly as possible.

**Do not strip the panel's own security headers** (CSP, `X-Frame-Options`, HSTS, etc. — see [middleware/securityHeaders.go](middleware/securityHeaders.go)). Don't add `proxy_hide_header` or layer a second, conflicting CSP/`X-Frame-Options` in nginx. Keep the panel bound to loopback so it's only reachable through the proxy.

## Development & Build

s-ui-x-extended builds in two stages: compile the Vue frontend to static assets, then build the Go backend with `with_*` tags. The reference is [build.sh](build.sh).

```bash
# 1) Frontend -> static assets
cd frontend && npm i && npm run build && cd ..
# 2) Copy UI into the Go embed tree
mkdir -p web/html && rm -fr web/html/* && cp -R frontend/dist/* web/html/
# 3) Backend (see build.sh for the full -tags string)
go build -ldflags '-w -s -checklinkname=0 ...' -tags "with_quic,with_grpc,...,with_wireguard,..." -o sui main.go
```

> **Build profiles differ.** Prebuilt GitHub release tarballs (the default `install.sh` path) ship a **reduced** tag set and do **not** include WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, or DHCP-DNS — configuring those returns a "rebuild with `-tags with_…`" error. The **full** set (`with_wireguard`, `with_masque`, `with_openvpn`, `with_mtproxy`, `with_sudoku`, `with_trusttunnel`, `with_dhcp`, …) is only compiled by Docker (`docker build .`) and [build.sh](build.sh). Naive also needs a pinned cronet native lib, easiest via Docker.

Required CI checks ([ci.yml](.github/workflows/ci.yml)): `build`, `vet`, `test-go`, `fe-lint`, `fe-build`, `fe-vitest`. Run them locally via the [Makefile](Makefile) `make audit` targets.

<details><summary>Full tag → feature matrix</summary>

`with_quic` (Hysteria/Hysteria2/TUIC, DoQ), `with_grpc`, `with_utls`, `with_acme`, `with_gvisor`, `with_tailscale` and `with_naive_outbound` ship in all profiles. Full-build-only: `with_wireguard` (incl. WARP), `with_masque`, `with_openvpn`, `with_mtproxy`, `with_sudoku`, `with_trusttunnel`, `with_dhcp`, plus `with_ccm`/`with_ocm`/`with_oomkiller`. `with_profiler` is dev-only (unauthenticated `/debug/pprof/*`, never bind to non-loopback). See [build.sh](build.sh) and [core/](core) for exact tags and stubs.

</details>

## FAQ / Troubleshooting

**I lost the admin password.** Reset it from the host with `s-ui admin -reset` (or the management menu). See [s-ui Management Menu & CLI](#s-ui-management-menu--cli).

**IP features (IP monitor, real client IP in logs) show the proxy IP.** The panel ignores `X-Forwarded-For` unless `SUI_TRUSTED_PROXIES` lists your proxy. See [Running Behind a Reverse Proxy](#running-behind-a-reverse-proxy).

**An external subscription URL fails to import.** Private, loopback and link-local targets are blocked by default (SSRF protection). Opt in with `SUI_ALLOW_PRIVATE_SUB_URLS=true` only if you trust the target.

**A protocol (WireGuard, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS) errors with `… is not included in this build, rebuild with -tags with_…`.** Current builds (prebuilt tarballs, Windows ZIP, Docker, `build.sh`) all ship the full tag set, so this means your binary predates that change. Upgrade to a current release, or rebuild from source / use Docker. See [Supported Platforms](#supported-platforms).

**The panel will not start with `EADDRNOTAVAIL`.** If the saved `webListen` / `subListen` IP no longer exists on the host, the panel logs a warning and binds on all interfaces instead — check the log and fix the listen address.

**Secrets became unreadable after a restore or migration.** Encrypted-at-rest settings need the same `SUI_SECRETBOX_KEY`. Preserve `/etc/s-ui/secretbox.env` across reinstalls. See [Environment Variables](#environment-variables).

## Languages

- English
- Persian
- Vietnamese
- Simplified Chinese
- Traditional Chinese
- Russian

## Contributing

Contributions are welcome. See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the workflow, and run the local checks before opening a PR. Required CI checks and build details are in [Development & Build](#development--build).

## License

Released under the GNU General Public License v3.0 — see [`LICENSE`](LICENSE).

## Support

- Bugs and feature requests: [GitHub Issues](https://github.com/deposist/s-ui-x-extended/issues)
- Security reports: see [Security & Hardening](#security--hardening)
- Acknowledgments and upstream projects: see below

## Acknowledgments

- **[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended)** by **shtorm-7** — the extended sing-box core this panel is built around. Thank you for the extended protocol and transport work that makes this project possible.
- **[`SagerNet/sing-box`](https://github.com/SagerNet/sing-box)** — the upstream core.
- **[`alireza0/s-ui`](https://github.com/alireza0/s-ui)** — the original panel this project is based on.

---

<p align="center"><a href="#s-ui-x-extended">English</a> · <b>Русский</b></p>

## Русский

**s-ui-x с поддержкой ядра sing-box-extended.** Продвинутая веб-панель для sing-box на ядре [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) (форк SagerNet/sing-box от shtorm-7).

Проект добавляет поддержку расширенных протоколов и транспортов поверх панели s-ui-x, наследуя при этом её усиление безопасности и надёжности. Основан на `alireza0/s-ui` (текущая сборка: `v1.0.0-beta2`).

> **Отказ от ответственности:** этот проект предназначен только для личного обучения и обмена опытом. Не используйте его в незаконных целях.

## Оглавление

- [Русский](#русский)
- [Релизы](#релизы)
- [Краткий обзор](#краткий-обзор)
- [Ключевые отличия от alireza0/s-ui](#ключевые-отличия-от-alireza0s-ui)
- [Поддерживаемые платформы](#поддерживаемые-платформы)
- [Информация об установке по умолчанию](#информация-об-установке-по-умолчанию)
- [Установка или обновление до последней стабильной версии](#установка-или-обновление-до-последней-стабильной-версии)
- [Поведение установщика и обновление](#поведение-установщика-и-обновление)
- [Ручная установка](#ручная-установка)
- [Меню управления s-ui и CLI](#меню-управления-s-ui-и-cli)
- [Удаление S-UI](#удаление-s-ui)
- [Установка с помощью Docker](#установка-с-помощью-docker)
- [Миграция из x-ui / 3x-ui](#миграция-из-x-ui--3x-ui)
- [Резервное копирование и восстановление](#резервное-копирование-и-восстановление)
- [Переменные окружения](#переменные-окружения)
- [HTTP API](#http-api-1)
- [Мониторинг и observability](#мониторинг-и-observability)
- [Безопасность и харднинг](#безопасность-и-харднинг)
- [Возможности](#возможности)
- [Telegram-уведомления](#telegram-уведомления)
- [Платные подписки (экспериментально)](#платные-подписки-экспериментально)
- [Служба подписок](#служба-подписок)
- [SSL-сертификаты](#ssl-сертификаты)
- [Запуск за обратным прокси](#запуск-за-обратным-прокси)
- [Разработка и сборка](#разработка-и-сборка)
- [FAQ / Решение проблем](#faq--решение-проблем)
- [Языки](#языки)
- [Участие в проекте](#участие-в-проекте)
- [Лицензия](#лицензия)
- [Поддержка](#поддержка)
- [Благодарности](#благодарности)

## Релизы

Полная история релизов и текущие release notes находятся в:

- [`CHANGELOG.md`](CHANGELOG.md)
- Release notes текущего релиза: [`docs/releases/v1.0.0-beta2.md`](docs/releases/v1.0.0-beta2.md)

README оставляет только установку и общий обзор проекта. Полная история релизов и breaking-заметки — в changelog. Шаги обновления и отката описаны в разделе [Поведение установщика и обновление](#поведение-установщика-и-обновление).

## Краткий обзор

| Возможность | Поддержка |
| --- | :---: |
| Множество протоколов (proxy / tunnel / VPN) | ✅ |
| Управление входящими и исходящими | ✅ |
| Несколько клиентов и входящих | ✅ |
| Продвинутая маршрутизация и DNS-правила | ✅ |
| Ссылки подписки (link / JSON / Clash + info) | ✅ |
| Per-client секреты подписок | ✅ |
| HTTP API со scoped-токенами | ✅ |
| Миграция из x-ui / 3x-ui | ✅ |
| Бэкап и восстановление | ✅ |
| Telegram-уведомления | ✅ |
| Платные подписки (экспериментально) | ✅ |
| Мониторинг клиентов, трафика и системы | ✅ |
| Несколько языков | ✅ |
| Тёмная / светлая / системная тема | ✅ |

## Ключевые отличия от `alireza0/s-ui`

<details>
  <summary>Показать подробности</summary>

Этот форк бинарно совместим с `alireza0/s-ui` — новый бинарник можно
ставить поверх работающей установки 1.x, схема БД автоматически
обновится при первом старте. Цель форка — усилить безопасность и
надёжность, не меняя протокол.

- **Авторизация и сессия.** bcrypt с ленивой миграцией, случайный пароль администратора при первой установке (выводится в журнал один раз), лимит на неуспешные логины, cookie сессии — `HttpOnly` + `SameSite=Lax` + `Secure` при HTTPS. Чувствительные настройки (Telegram bot token, креденшелы прокси, install salt) шифруются at-rest через secretbox; API-токены хранятся как salted SHA-256. CSRF-защита на browser `/api/*`-mutating-запросах.
- **Scopes API-токенов.** `admin`, `read`, `write`, `database`, `telegram` и `observability` описаны в [`docs/scope-matrix.md`](docs/scope-matrix.md), включая audit, database, Telegram, rotation subscription-secret, observability и realtime security-event.
- **`X-Forwarded-For`.** Заголовок игнорируется без `SUI_TRUSTED_PROXIES`; цепочка обходится справа налево, поддельный XFF не может обойти IP-логику.
- **Загрузчик внешних подписок.** Allow-list URL, блок приватных/loopback по умолчанию (opt-in `SUI_ALLOW_PRIVATE_SUB_URLS=true`), лимит ответа 4 МиБ, защита от DNS rebinding на dial. `Authorization: Bearer <token>` — основной способ передачи API-токена в `/apiv2/*`; legacy `Token`-header работает с `Deprecation` и `Sunset` до `Sat, 15 Aug 2026 00:00:00 GMT`.
- **Realtime WebSocket.** `/api/realtime/ws-token` + `/api/realtime/ws` с Origin allow-list, per-IP rate-limit handshake, одноразовыми токенами, ping/pong heartbeat, idle close и close-all при ротации сессии. На фронте есть polling-фолбэк для degraded-состояний.
- **Per-client subscription secrets.** Поддерживаются `/sub/<secret>`, `/sub/json/<secret>`, `/sub/clash/<secret>`, `/json/<secret>`, `/clash/<secret>`; legacy `/sub/<name>` работает пока `subSecretRequired=false`. Subscription-эндпоинты санитизируют response-заголовки и применяют конфигурируемый per-IP rate-limit.
- **Telegram-уведомления (off by default).** Асинхронная bounded-очередь с retry/backoff и audit-событиями overflow/failure. Egress может идти через валидированные HTTP/HTTPS/SOCKS5-прокси. Payload, audit-детали, changes и captions проходят redaction.
- **Audit и observability.** Таблица `audit_events` с retention GC, scoped эндпоинт `GET /api/security/audit` с rate-limit и cursor pagination. Bounded observability buckets (`2s`, `30s`, `1m`, `5m`), сэмплятся cron'ом. Bounded logs API и fail-soft 1h-cached `GET /api/version`.
- **IP monitor (monitor-only по умолчанию).** Соль+SHA-256 hashing, opt-in raw-display, retention GC, per-client `limitIp` и `ipLimitMode`. Enforce отбрасывает только новые сверхлимитные подключения и не разрывает активные.
- **Безопасность SQL.** Параметризованные запросы в `service/config.go` и `service/inbounds.go`; в выборке пользователей по inbound — статический whitelist допустимых типов.
- **Импорт бэкапа / обновление.** `ImportDB` имеет cap 64 МиБ, проверку SQLite magic, временную staging-копию, read-only `PRAGMA integrity_check` и audit-события. WAL/SHM сайдкары очищаются, schema-миграции и `AdaptToCurrentVersion` запускаются автоматически (перешивка plaintext-паролей в bcrypt, обновление индексов, поднятие `settings.version`); при ошибке БД восстанавливается из staging.
- **Листен-адрес, устойчивый к переезду.** Если в `webListen` / `subListen` сохранён IP, которого нет на текущем хосте, панель пишет warning и слушает на всех интерфейсах вместо краша `EADDRNOTAVAIL`.
- **Race-free runtime.** core lifecycle, online-stats, last-update, v2 token store и realtime hub проходят `go test -race ./...` (требует CGO).
- **HTTP server hardening.** Таймауты `Read/Write/Header/Idle` и `tls.MinVersion = 1.2` для панели и для эндпоинта подписки. Middleware security-headers (CSP, HSTS при TLS, no-store на subscription-ответах).
- **WARP-регистрация.** Поддержка актуального API Cloudflare (`v0a4005`) с заголовками первого клиента, фоллбэк на `v0a2158`, ретраи переходящих TLS-ошибок.
- **Чистота фронтенда.** `v-html` удалён из логов, ошибок импорта правил, IP-листов и gauge-плитки. Axios через экспортируемый instance, `AbortController` вместо устаревшего `CancelToken`, дедупликация только для идемпотентных запросов, code splitting Vite. Realtime WS-store с reconnect/degraded состояниями. Secret-aware-поля настроек с placeholder'ом `••• stored •••`. IP-history modal с маской raw-IP по умолчанию. Views Telegram-настроек и Audit.
- **Локализация и значения по умолчанию.** Многоязычные `install.sh` и меню `s-ui` (английский / русский / китайский), язык переключается на лету. Часовой пояс по умолчанию — `Europe/Moscow`. Локаль фронтенда по умолчанию — английский (существующие браузеры сохраняют выбор из `localStorage`).

</details>

## Поддерживаемые платформы

| ОС | Архитектуры |
| --- | --- |
| Linux | `amd64`, `arm64`, `armv7`, `armv6`, `armv5`, `386`, `s390x` |
| Windows | `amd64`, `386`, `arm64` |
| macOS | `amd64`, `arm64` (экспериментально) |

### Готовые сборки vs Docker — матрица сборки

Не все бинарники несут одинаковый набор возможностей. Что вы получите, зависит от **способа** установки:

| Путь сборки | Набор тегов протоколов | Naive / cronet |
| --- | --- | --- |
| Готовый Linux-tarball (по умолчанию `install.sh`) | Урезанный — см. примечание ниже | Включён для `amd64`, `arm64`, `armv7`, `armv6`, `386` (CGO + cronet). **Нет** в `armv5` / `s390x`. |
| Готовый Windows-ZIP | Урезанный | Только `amd64` (CGO). `arm64` собирается с `CGO_ENABLED=0`, поэтому cronet не линкуется. |
| Docker-образ | Полный | Да (`amd64` / `arm64` / `arm`) |
| Самостоятельная сборка (`build.sh`) | Полный | Да |

**Урезанный набор протоколов.** Готовые tarball'ы из GitHub-релизов собираются с урезанным списком тегов ([.github/workflows/release.yml](.github/workflows/release.yml)) и **не** включают `with_wireguard`, `with_masque`, `with_mtproxy`, `with_openvpn`, `with_sudoku`, `with_trusttunnel` и `with_dhcp`. Поэтому WireGuard, WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel и DHCP-DNS доступны **только в Docker / при самостоятельной сборке** — полный набор тегов задаётся в [Dockerfile](Dockerfile) и [build.sh](build.sh).

**Naive требует CGO.** Outbound Naive линкуется с cronet, для чего нужен `CGO_ENABLED=1`. На Linux `armv5` и `s390x` собираются без него (`naive: false`); на Windows CGO включён только для `amd64` — `arm64` собирается с `CGO_ENABLED=0` и поставляется без cronet. Иными словами, полноценная поддержка Naive — это территория **Linux / Docker**.

## Информация об установке по умолчанию

- Порт панели: `2095`, путь панели: `/app/`
- Порт подписки: `2096`, путь подписки: `/sub/`
- Изменения лимита подписок на IP (`subRateLimitPerIP`) применяются в течение 1 минуты после сохранения.
- Имя пользователя: `admin`
- Пароль (только при чистой установке): случайная строка из 24 символов генерируется при первом запуске и записывается в журнал. Ищите строку `created initial admin user. username=admin password=...` в `journalctl -u s-ui` (Linux) или в логе панели при первом запуске, затем смените его в панели.

Настройки панели хранятся в БД и редактируются из панели (Настройки) или через API; значения по умолчанию инициализируются при первом запуске из [`service/setting.go`](service/setting.go). Наиболее часто изменяемые параметры:

| Параметр (`key`) | По умолчанию | Что управляет |
| --- | --- | --- |
| `webPort` | `2095` | Порт HTTP(S) панели. |
| `webPath` | `/app/` | Базовый путь панели. |
| `subPort` | `2096` | Порт сервера подписок. |
| `subPath` | `/sub/` | Базовый путь подписок. |
| `subRateLimitPerIP` | `60` | Запросов подписки с одного IP в минуту (1–10000). |
| `timeLocation` | `Europe/Moscow` | Часовой пояс IANA для планирования и меток времени. |

Существует много других ключей (форматы и кодировка подписки, cookie сессий, сроки хранения, Telegram, платежи, сертификаты, брендирование, …) — по умолчанию отключены или пусты. Полный источник истины: [`service/setting.go`](service/setting.go).

## Установка или обновление до последней стабильной версии

### Linux/macOS

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh)
```

### Windows

1. Скачайте последнюю версию для Windows из [GitHub Releases](https://github.com/deposist/s-ui-x-extended/releases/latest).
2. Распакуйте ZIP-файл.
3. Запустите `install-windows.bat` от имени администратора.
4. Следуйте инструкциям мастера установки.

## Установка v1.0.0-beta2

Это первая публичная бета s-ui-x-extended — см. [`CHANGELOG.md`](CHANGELOG.md)
и [`docs/releases/v1.0.0-beta2.md`](docs/releases/v1.0.0-beta2.md). Это бета —
сначала протестируйте.

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0-beta2
```

Или из локального клона:

```sh
git clone https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.0-beta2
```

## Установка старой версии

Чтобы установить определённую старую версию, добавьте тег версии с `v` в конец команды установки. Например, версия `v1.0.0`:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0
```

## Поведение установщика и обновление

Установщик ([install.sh](install.sh)) и меню `s-ui` ([s-ui.sh](s-ui.sh)) интерактивны. Сначала установщик выбирает язык интерфейса (`SUI_LANG=en|ru|zh` или диалог — по умолчанию английский, сохраняется в `/etc/s-ui/lang`), затем проверяет архив релиза по его `.sha256` и **прерывается**, если сумма отсутствует или не совпадает. После установки он запускает `sui migrate` и предлагает изменить настройки (порт и путь панели/подписки, учётные данные администратора); пустой ответ сохраняет текущее значение. При новой установке, если пропустить редактирование, случайные логин и пароль выводятся один раз — сохраните их. Одноразовый `SUI_SECRETBOX_KEY` хранится в `/etc/s-ui/secretbox.env`; держите его неизменным при обновлениях и восстановлении. База панели — единственный файл SQLite по пути `/usr/local/s-ui/db/s-ui.db`.

**Обновление с alireza0/s-ui:** совместимо по бинарнику и базе. Остановите панель, сделайте резервную копию `s-ui.db` (например `s-ui.db.bak` — точка отката), поставьте новый бинарник поверх и запустите; схема мигрирует вперёд автоматически ([cmd/migration/main.go](cmd/migration/main.go)). Миграция **только вперёд** — база новее бинарника остаётся нетронутой, понижения нет; несовместимая версия падает сразу, не повреждая данные.

**Откат:** восстановите резервную копию поверх `/usr/local/s-ui/db/s-ui.db`, затем переустановите предыдущую версию (меню **3 → Пользовательская версия**). Храните копию, пока новая версия не подтвердится: обновлённая база не мигрирует вниз, поэтому без неё откат невозможен.

## Ручная установка

### Linux/macOS

1. Скачайте последнюю версию S-UI для вашей системы и архитектуры из GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. **Необязательно:** скачайте последнюю версию `s-ui.sh`: [https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh](https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh)
3. **Необязательно:** скопируйте `s-ui.sh` в `/usr/bin/` и выполните `chmod +x /usr/bin/s-ui`.
4. Распакуйте tar.gz-архив s-ui в выбранный каталог и перейдите в распакованную папку.
5. Скопируйте файлы `*.service` в `/etc/systemd/system/`, затем выполните `systemctl daemon-reload`.
6. Выполните `systemctl enable s-ui --now`, чтобы включить автозапуск и запустить службу S-UI.
7. Выполните `systemctl enable sing-box --now`, чтобы запустить службу sing-box.

### Windows

1. Скачайте последнюю версию для Windows из GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. Скачайте подходящий пакет для Windows, например `s-ui-windows-amd64.zip`.
3. Распакуйте ZIP-файл в выбранный каталог.
4. Запустите `install-windows.bat` от имени администратора.
5. Следуйте инструкциям мастера установки.
6. Откройте панель: http://localhost:2095/app

## Меню управления s-ui и CLI

Запустите `s-ui` (от **root**) без аргументов, чтобы открыть многоязычное интерактивное меню (английский, русский, китайский; язык сохраняется в `/etc/s-ui/lang` или задаётся через `SUI_LANG`). Оно охватывает установку, обновление, учётные данные администратора, настройки панели, SSL, BBR и управление службой.

Для скриптов передавайте подкоманду-обёртку: `s-ui start | stop | restart | status | enable | disable | log | update | uninstall` (а также `install`, `help`).

Базовый бинарник `/usr/local/s-ui/sui` выполняет фактическую работу и предоставляет: `admin`, `setting`, `uri`, `migrate`, `import-xui` (импорт из 3x-ui `x-ui.db`) и `decrypt-backup` (расшифровка резервной копии Telegram). `sui -v` выводит версии панели и sing-box.

**Потеряли доступ / забыли пароль?** Восстановите из консоли сервера — вход в панель не нужен:

```bash
/usr/local/s-ui/sui admin -reset
```

Это сбрасывает первого администратора на имя `admin` со свежим случайным паролем из 16 символов, выводимым **один раз** — сохраните его сразу, так как он хранится только как bcrypt-хеш и не может быть восстановлен позже. Чтобы задать свой: `sui admin -username myname -password 'mypass'`.

## Удаление S-UI

> Совет: рекомендуемый путь — меню управления (запустите `s-ui`, пункт **Удалить**, или `s-ui uninstall`): оно также чистит `/etc/s-ui/` и systemd drop-in. Ручные шаги ниже — запасной вариант.

```sh
sudo -i

systemctl disable s-ui  --now

rm -f /etc/systemd/system/sing-box.service
systemctl daemon-reload

rm -fr /usr/local/s-ui
rm /usr/bin/s-ui
```

## Установка с помощью Docker

<details>
   <summary>Показать подробности</summary>

### Использование

**Шаг 1:** установите Docker

```shell
curl -fsSL https://get.docker.com | sh
```

**Шаг 2:** установите S-UI

> Вариант с Docker Compose

```shell
services:
  s-ui:
    image: ghcr.io/deposist/s-ui-x-extended
    container_name: s-ui
    hostname: "s-ui"
    network_mode: host
    volumes:
      - "./db:/app/db"
      - "./cert:/app/cert"
    tty: true
    restart: unless-stopped
    entrypoint: "./entrypoint.sh"
```

`docker compose up -d`

> Прямой запуск через Docker

```shell
mkdir s-ui && cd s-ui

docker run -itd \
    --network host \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name s-ui \
    --restart=unless-stopped \
    ghcr.io/deposist/s-ui-x-extended
```

> Самостоятельная сборка образа

```shell
git clone https://github.com/deposist/s-ui-x-extended
docker build -t s-ui .
```

</details>

## Миграция из x-ui / 3x-ui

s-ui-x-extended умеет импортировать существующую установку **x-ui / 3x-ui**, читая её базу SQLite (`x-ui.db`). Исходный файл открывается **только на чтение** и никогда не изменяется — импортёр переносит всё в активную базу s-ui в рамках одной транзакции, которая откатывается при любой ошибке, поэтому неудачный импорт не оставляет панель «наполовину перенесённой». Переносятся inbound'ы, TLS/Reality, клиенты, настройки панели, администраторы, агрегированная история трафика и маршрутизация (best-effort; несопоставленные объекты помечаются на проверку).

**Мастер в панели** — откройте **Настройки → 3x-ui Migration** (`/migrate-xui`): загрузите БД, проверьте план в режиме dry-run (действие для объекта `create` / `merge` / `replace` / `skip`), примените с прогрессом в реальном времени, затем скачайте отчёт или выполните **откат**. Тот же поток доступен через `/api` и `/apiv2` (область токена `database`).

**CLI** — для скриптовой/серверной миграции:

```bash
sui import-xui --src /path/to/x-ui.db --strategy merge --include-routing --include-history --yes
```

(Используйте `--dry-run` для предпросмотра; без `--yes` запросит подтверждение.)

> **Безопасно по умолчанию.** Перед любым импортом (не dry-run) панель пишет снимок базы (`s-ui-pre-xui-import-*.db`, хранятся 10 новейших), чтобы полностью восстановить предыдущую базу.

## Резервное копирование и восстановление

Панель выгружает всю базу SQLite одним согласованным файлом `.db` и позже восстанавливает её — на том же хосте или на чистой установке. Обе операции требуют авторизованной сессии (или API-токена со scope `database`/`admin`) и пишутся в [журнал аудита](#журнал-аудита).

- **Экспорт — `GET /api/getdb`** отдаёт чистый `s-ui_<метка_времени>.db` (с checkpoint WAL, без файлов-спутников). Параметры запроса позволяют через `exclude` исключить таблицы истории (такие как `stats`, `client_ips`, `audit_events`, `changes`) для бэкапа только с конфигурацией, либо задать `encryptTelegramBackup=true` для AES-конверта на парольной фразе [Telegram-бэкапа](#telegram-бот--уведомления). Базовая конфигурация включается всегда.
- **Восстановление — `POST /api/importdb`** принимает `.db` (или `.db.aes` + парольную фразу) как multipart-поле `db`. Оно защищённое: лимит 64 МиБ, проверка сигнатуры `SQLite format 3` и `integrity_check` в режиме только-чтение, стейджинг вне рабочей базы, безопасная замена, затем авто-миграция/адаптация — с **откатом к исходной базе при любом сбое**. При успехе панель перезапускается (~3 с).

> Базы 3x-ui / x-ui (Xray) отклоняются — используйте [Миграцию с 3x-ui](#миграция-с-3x-ui), а не Восстановление.
>
> Восстановление заменяет **всю** базу, включая учётные данные администратора; затем входите с учётными данными исходной панели.

## Переменные окружения

s-ui-x-extended читает свою конфигурацию во время выполнения из переменных окружения процесса. Задавайте их в юните systemd (`/etc/systemd/system/s-ui.service`), в блоке `environment:` Docker или в определении службы Windows — панель читает их при запуске.

| Переменная | Тип | По умолчанию | Назначение |
|---|---|---|---|
| `SUI_DB_FOLDER` | путь | `<каталог-приложения>/db` | Каталог с базой данных SQLite (`<name>.db`). Если не задана, используется каталог запущенного бинарника плюс `db` (либо `/usr/local/s-ui/db` / `C:\Program Files\s-ui\db`, если путь к бинарнику определить не удалось). |
| `SUI_SECRETBOX_KEY` | base64 | _(не задана)_ | Ключ вне базы данных для шифрования секретных настроек (токены ботов, ключи платёжных систем, парольная фраза бэкапа). Должен быть base64 (standard, raw-standard или raw-URL), декодирующимся в **≥ 32 байта**. Если не задана или некорректна, панель пишет предупреждение и переходит на HKDF-ключ, выведенный из `settings.secret` — это работает, но такие секреты тогда остаются расшифровываемыми только из базы данных. |
| `SUI_COOKIE_KEY` | список base64 | _(не задана)_ | Один или несколько ключей для подписи сессионных cookie. Разделители — запятая, точка с запятой или перевод строки; каждый элемент должен декодироваться из base64 в **≥ 32 байта**. Если не задана или некорректна, панель пишет предупреждение и использует совместимый HKDF-ключ из `settings.secret`. |
| `SUI_FORCE_COOKIE_SECURE` | bool | _(не задана)_ | При установке в разбираемое булево значение (`true`/`false`/`1`/`0`) принудительно включает/выключает флаг `Secure` у сессионных cookie; без значения сохраняется автоопределение панели. Неразбираемое значение возвращается как ошибка. |
| `SUI_TRUSTED_PROXIES` | список CIDR/IP | _(нет)_ | Список доверенных прокси через запятую — CIDR или одиночные IP (IPv4/IPv6), чей `X-Forwarded-For` учитывается при определении IP клиента. Некорректные элементы логируются и пропускаются; пусто — ни один прокси не доверенный. |
| `SUI_ALLOW_PRIVATE_SUB_URLS` | bool | `false` | Ровно при `true` разрешает запросам конвертации подписок обращаться к приватным/loopback-адресам. По умолчанию выключено для защиты от SSRF к внутренним хостам. |
| `SUI_LOG_LEVEL` | enum | `info` | Подробность логов: `debug`, `info`, `warn` или `error` (без учёта регистра). Некорректное значение логирует предупреждение и откатывается к `info`. Переопределяется `SUI_DEBUG`. |
| `SUI_DEBUG` | bool | `false` | Ровно при `true` включает режим отладки и принудительно ставит уровень логирования `debug`. |
| `SUI_SIGHUP_TIMEOUT_SECONDS` | int | `3` | Секунды ожидания перезагрузки ядра после `SIGHUP` при применении конфигурации. Принимает `1`–`60`; значения вне диапазона или нечисловые логируют предупреждение и откатываются к `3`. |
| `SUI_DB_MAX_OPEN_CONNS` | int | `8` | Максимум открытых соединений SQLite в пуле. Должно быть `> 0`; иначе используется значение по умолчанию. |
| `SUI_DB_MAX_IDLE_CONNS` | int | `4` | Максимум простаивающих соединений SQLite. Должно быть `≥ 0`; иначе используется значение по умолчанию. Если больше `SUI_DB_MAX_OPEN_CONNS`, обрезается до него. |
| `SUI_BIN_FOLDER` | путь | `bin` | **Только устаревшая миграция.** Читается лишь однократной миграцией схемы 1→2 для поиска старого `config.json` (по пути `<каталог-приложения>/<SUI_BIN_FOLDER>/config.json`) и его импорта в базу данных. На чистой установке не влияет. |

> **Примечание по systemd:** чтобы задать `SUI_SECRETBOX_KEY` для службы, добавьте строку `Environment=SUI_SECRETBOX_KEY=...` (или `EnvironmentFile=`) в `/etc/systemd/system/s-ui.service`, затем выполните `systemctl daemon-reload` и `systemctl restart s-ui`. Установщик для Linux может сгенерировать и сохранить этот ключ за вас.

### Переменные только для скриптов

Эти переменные используются скриптами установки/запуска, а не бинарником на Go:

| Переменная | Кем читается | Назначение |
|---|---|---|
| `SUI_MIGRATE_ONLY` | [entrypoint.sh](entrypoint.sh) | При `1` точка входа Docker выполняет `./sui migrate` и завершается вместо запуска панели. |
| `SUI_LANG` | [install.sh](install.sh), [s-ui.sh](s-ui.sh) | Принудительно задаёт язык установщика/скрипта управления: `en`, `ru` или `zh`. |
| `SUI_HOME` | [windows/s-ui-windows.bat](windows/s-ui-windows.bat) | Каталог установки в Windows; задаётся установщиком и читается управляющим скриптом Windows (по умолчанию `C:\Program Files\s-ui`). |

## HTTP API

s-ui-x-extended предоставляет две поверхности HTTP API, обе под настраиваемым базовым путём панели (`webPath`, по умолчанию `/app/`):

- **`/api/*`** — используется веб-интерфейсом. Аутентификация — сессионная cookie браузера плюс CSRF-токен (`GET /app/api/csrf`, отправляется в заголовке `X-CSRF-Token` при каждом изменяющем запросе).
- **`/apiv2/*`** — поверхность без состояния для скриптов и автоматизации. Нет сессии, нет CSRF — только API-токен.

Создавайте токены в разделе **Settings -> API tokens**; выберите область (scope) и необязательный срок действия. Открытое значение показывается **один раз** (хранится только хеш), поэтому сохраните его. Отправляйте его в каждом запросе `/apiv2/*` как `Authorization: Bearer <your-token>`.

Каждый токен несёт ровно одну область:

| Область | Что разрешает |
| --- | --- |
| `admin` | Всё (и единственная область для чтения аудита и теста Telegram). |
| `read` | Чтение конфигурации/идентичности только для чтения и метрики. |
| `write` | Всё, что разрешает `read`, плюс изменения, такие как `save`, `restartApp`, `restartSb`, … |
| `database` | Экспорт базы данных (`getdb`) и импорт (`importdb`). |
| `telegram` | Ручной запуск резервного копирования в Telegram. |
| `observability` | Операционные метрики и история, такие как `stats`, `status`, `logs`, … |

Полную матрицу по эндпоинтам см. в [`docs/scope-matrix.md`](docs/scope-matrix.md).

**Устаревший заголовок `Token` (deprecated):** старые клиенты отправляли токен в заголовке `Token`; он ещё работает, но имеет жёсткую дату прекращения `Sat, 15 Aug 2026 00:00:00 GMT`, после которой возвращается `401 legacy token header expired`. Переведите автоматизацию на `Authorization: Bearer` до этой даты.

## Мониторинг и observability

Панель непрерывно снимает метрики хоста и ядра и отдаёт их через JSON-эндпоинты только на чтение под `/api/` (относительно настроенного web base path). Все данные observability хранятся **в памяти** — отдельную базу данных временных рядов запускать не нужно.

Основные эндпоинты на чтение:

- `GET /api/onlines` — активные онлайн-клиенты (теги inbound, пользователи, теги outbound)
- `GET /api/stats` — временные ряды трафика inbound/outbound по тегам
- `GET /api/status` — снимок по запросу: CPU, память, диск, swap, сеть, ядро sing-box и БД
- `GET /api/logs` — ограниченный фильтруемый буфер логов панели и ядра
- `GET /api/version` — текущая версия плюс последний релиз GitHub (fail-soft, кэшируется)
- `GET /api/observability/history` и `/api/observability/core-history` — история хоста (CPU/RAM/сеть) и ядра (running, goroutines, alloc, uptime) по бакетам из кольцевых буферов в памяти

Cron-задача снимает метрики каждые ~2 с и агрегирует их в бакеты фиксированного размера (такие как `2s`, `30s`, `1m`, `5m`, …), ограничивая суммарную память (32 МБ по умолчанию). Эндпоинты истории требуют scope токена `observability` или `admin`. См. [observability.go](service/observability.go).

## Безопасность и харднинг

Панель поставляется с включённой по умолчанию эшелонированной защитой. Ниже — пользовательская сводка; обоснование решений по цепочке поставок см. в [SECURITY.md](SECURITY.md).

- **Хранение паролей.** Пароли администратора хранятся как bcrypt-хеши ([util/common/password.go](util/common/password.go)). Унаследованные plaintext-пароли принимаются один раз через сравнение с постоянным временем и лениво перехешируются в bcrypt при следующем использовании, поэтому обновление прозрачно мигрирует сохранённые учётные данные.
- **Случайный пароль при первом запуске.** На свежей установке панель генерирует случайный 24-символьный пароль администратора и один раз записывает его в журнал приложения (`created initial admin user. username=admin password=...`). Общего пароля по умолчанию нет.
- **Ограничение частоты входов.** Неудачные входы троттлятся как **по IP-источнику**, так и **по имени пользователя** (`user|<name>`), поэтому распределённый перебор со сменой IP всё равно ограничивается для целевой учётной записи. После 5 неудач в окне 15 минут ключ блокируется на 15 минут ([api/rateLimit.go](api/rateLimit.go)).
- **Выравнивание времени.** Путь для несуществующего имени пользователя выполняет холостое bcrypt-сравнение, поэтому несуществующий пользователь стоит примерно столько же, сколько неверный пароль, что не даёт перечислять имена по таймингу ([util/common/password.go](util/common/password.go)). Проверки CSRF и унаследованных паролей также используют сравнение с постоянным временем.
- **Шифрование при хранении.** Чувствительные настройки (токен Telegram-бота, учётные данные прокси, install salt) шифруются при хранении через AES-256-GCM с помощью хелпера secretbox, с ключом из HKDF-SHA256 и тегированием associated data ([util/secretbox/secretbox.go](util/secretbox/secretbox.go)). Зашифрованные значения несут префикс `sbox:v1:`.
- **API-токены.** API-токены никогда не хранятся в открытом виде; они хранятся как SHA-256-хеши, посоленные per-install salt ([service/user.go](service/user.go)).
- **Защита от CSRF.** Браузерные мутирующие запросы (`POST`/`PUT`/`PATCH`/`DELETE`) под `/api/*` требуют валидный `X-CSRF-Token`, совпадающий с токеном сессии; токен сравнивается с постоянным временем и истекает через 2 часа. Эндпоинт входа исключён ([api/csrf.go](api/csrf.go)).
- **Усиленные cookie.** Cookie сессии и CSRF имеют флаги `HttpOnly`, по умолчанию `SameSite=Lax` (`Strict` при включённой настройке `sessionSameSiteStrict`) и `Secure`, если запрос пришёл по HTTPS или сконфигурированный web URL/домен начинается с `https://` (можно форсировать через `forceCookieSecure`).
- **Транспорт и заголовки.** И панель, и сервер подписок требуют `tls.MinVersion = TLS 1.2` и подключают middleware security-заголовков ([middleware/securityHeaders.go](middleware/securityHeaders.go)): админ-панель отдаёт строгий `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy` и `Strict-Transport-Security` (только по реальному TLS); ответы подписок добавляют `Cache-Control: no-store`.
- **Обработка XFF за доверенным прокси.** `X-Forwarded-For` полностью игнорируется, пока не задан `SUI_TRUSTED_PROXIES` (CIDR/IP через запятую); затем цепочка обходится справа налево через доверенные узлы, поэтому поддельный XFF от недоверенного клиента не может дотянуться до IP-логики или подделать HSTS через `X-Forwarded-Proto` ([api/utils.go](api/utils.go)).
- **Журнал аудита.** Связанные с безопасностью действия записываются в таблицу `audit_events` с актором, IP, user-agent и отредактированными деталями и доступны через эндпоинт с rate-limit, скоупами и курсорной пагинацией ([service/audit.go](service/audit.go)).
- **IP-монитор.** Отслеживание источников подключений по клиентам хранит **посоленные SHA-256-хеши** IP клиентов (показ сырого IP — opt-in), с режимом `monitor`/`enforce` для каждого клиента; enforce отклоняет только новые подключения сверх лимита и публикует дебаунсированные realtime-события безопасности ([ipmonitor/ipmonitor.go](ipmonitor/ipmonitor.go)).

### Чек-лист харднинга

1. **Смените пароль по умолчанию.** Считайте случайный пароль первого запуска из журнала и немедленно смените его из панели.
2. **Задайте стабильный `SUI_SECRETBOX_KEY`.** Предоставьте сырой ключ в base64 длиной не менее 32 байт, чтобы шифрование при хранении не откатывалось на производный от HKDF `settings.secret`. Держите ключ неизменным между обновлениями и сделайте его резервную копию — потеря ключа делает зашифрованные настройки невосстановимыми. `install.sh` может сгенерировать его и сохранить в `/etc/s-ui/secretbox.env`.
3. **Сконфигурируйте `SUI_TRUSTED_PROXIES`**, если панель стоит за обратным прокси/балансировщиком; задайте CIDR/IP прокси, чтобы `X-Forwarded-For` учитывался только от доверенных узлов. Оставьте пустым при прямом доступе.
4. **Используйте TLS.** Терминируйте HTTPS (TLS панели или доверенный обратный прокси), чтобы cookie стали `Secure` и отдавался HSTS. При необходимости форсируйте через `forceCookieSecure`.
5. **Включите `subSecretRequired`**, чтобы отключить унаследованные обращения `/sub/<name>` и требовать per-client subscription-секреты.
6. **Ограничьте доступ к панели.** Не выставляйте админ-панель в открытый интернет без необходимости — привяжите её к приватному интерфейсу или закройте allow-list-ом и предпочтите нестандартные порт/путь.
7. **Держите сборки с `with_profiler` подальше от публичных интерфейсов.** Build-тег profiler регистрирует `/debug/pprof/*` **без аутентификации**; это тег только для разработки, и его **нет** в сборках по умолчанию. Если собираете с `-tags with_profiler`, никогда не привязывайте `listen` к не-loopback адресу.

### Сообщение об уязвимости

Пожалуйста, сообщайте о подозреваемых уязвимостях приватно через GitHub **security advisory** в репозитории, а не открывая публичный issue. Учтите, что [SECURITY.md](SECURITY.md) сейчас документирует решения по цепочке поставок (форки зависимостей, build-теги, пиннинг базовых образов), а не политику контактов.

## Возможности

- Поддерживаемые протоколы:
  - General / transparent: Mixed, SOCKS, HTTP, Direct, Redirect, TProxy, Tun
  - На базе V2Ray: VLESS, VMess, Trojan, Shadowsocks
  - Прочие прокси: ShadowTLS, AnyTLS, Naive, Hysteria, Hysteria2, TUIC, TrustTunnel, Mieru, Sudoku, SSH, Tor, MTProxy
  - Туннели / VPN (endpoints и outbounds): WireGuard, WARP, Tailscale, AmneziaWG 2.0, MASQUE, OpenVPN, VPN server / client
  - Outbound-группы и контроль трафика: Selector, URLTest, Bond, Failover, Fallback, Parser, а также лимитеры bandwidth / connection / traffic / rate
  - DNS-транспорты: TCP, UDP, DoT, DoH, DoQ, DNSCrypt (SDNS), DHCP, FakeIP, Hosts, Local, Fallback, Resolved, Tailscale
  - Большинство прокси-протоколов работают и как inbound, и как outbound; MTProxy — только inbound, Tor / MASQUE / OpenVPN — только outbound.

> [!IMPORTANT]
> **Важно про build-теги (только Naive).** Текущие готовые Linux-тарболлы из GitHub-релизов и Windows ZIP собираются с **полным набором тегов протоколов**, поэтому **WireGuard / WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel и DHCP DNS** работают «из коробки» — как в Docker и [`build.sh`](build.sh). Единственное, что всё ещё зависит от сборки, — **Naive**, который линкует cronet (`CGO_ENABLED=1`). Если протокол всё равно выдаёт *«… is not included in this build»*, значит ваш бинарник собран до этого изменения — обновитесь до актуального релиза.

### Транспорты и TLS

- **Stream-транспорты:** HTTP/2, WebSocket, gRPC, HTTPUpgrade, XHTTP и mKCP. См. [transport.ts](frontend/src/types/transport.ts) и [Transport.vue](frontend/src/components/Transport.vue).
- **REALITY** с генерацией ключевой пары в один клик (private/public ключ и Short IDs) и опциональной максимальной разницей во времени. Настраивается в [Tls.vue](frontend/src/layouts/modals/Tls.vue).
- **uTLS-отпечатки** для клиентского TLS — `chrome`, `firefox`, `edge`, `safari`, `360`, `qq`, `ios`, `android`, `random`, `randomized` ([OutTLS.vue](frontend/src/components/tls/OutTLS.vue)).
- **ECH** (Encrypted Client Hello) с конфигом в виде текста или пути и query server name.
- **Multiplex** (`smux`, `yamux`, `h2mux`) с паддингом и управлением перегрузкой **TCP Brutal** (up/down Mbps). См. [Multiplex.vue](frontend/src/components/Multiplex.vue).
- **XTLS flow** `xtls-rprx-vision` для совместимых VLESS-клиентов.
- Настройки TLS: генерация самоподписанного сертификата, ACME, min/max версия, ALPN, cipher suites, curve preferences (вкл. `X25519MLKEM768`), client authentication, certificate pinning (`certificate_public_key_sha256`), kernel TLS (kTLS) и TLS-фрагментация.

### Маршрутизация и DNS-правила

- **Правила маршрутизации** с конструктором logical/simple, богатым набором матчеров (inbound, network, protocol, sniffed client, domain / domain-suffix / keyword / regex, source и destination IP CIDR, порты и диапазоны портов, process, package, user, clash mode, тип сети/SSID/BSSID, rule-sets и др.) и флагом `invert`. См. [rules.ts](frontend/src/types/rules.ts) и [Rule.vue](frontend/src/layouts/modals/Rule.vue).
- **Действия правил:** `route`, `route-options`, `bypass`, `reject`, `hijack-dns`, `sniff` и `resolve` — включая override address/port, UDP-опции, TLS-фрагментацию, снифферы (HTTP, TLS, QUIC, STUN, DNS, BitTorrent, DTLS, SSH, RDP, NTP) и resolve-стратегии.
- **Rule-sets:** `inline`, `local` и `remote` (формат source/binary, интервал обновления, download detour).
- **Движок DNS-правил** с собственными серверами, правилами и действиями (`route`, `route-options`, `reject`, `predefined`), типы DNS-серверов: Local, Hosts, TCP, UDP, DoT, DoQ, DoH, HTTP/3, DHCP, FakeIP, Tailscale, Resolved, DNSCrypt (SDNS) и Fallback. См. [dns.ts](frontend/src/types/dns.ts).
- **Общий SSRF-валидатор** ([util/ssrf/validator.go](util/ssrf/validator.go)) лежит в основе каждого исходящего запроса, инициированного панелью — импорт подписки, кастомные check-таргеты outbound и egress Telegram-прокси, — отклоняя приватные, loopback, link-local и прочие зарезервированные диапазоны с повторной валидацией на этапе dial (защита от DNS rebinding).

### Outbound-провайдеры

- Описывайте outbound-провайдеры как наборы **inline**, **local** (путь к файлу) или **remote** (URL). См. [providers.go](database/model/providers.go), [service/providers.go](service/providers.go) и [providers.ts](frontend/src/types/providers.ts).
- Опциональные **health checks** для каждого провайдера (URL, interval, timeout), а также удаление эмодзи и фильтры include/exclude для remote-провайдеров.
- Remote-провайдеры поддерживают кастомный user agent, интервал обновления и download detour. Загрузка URL на стороне панели использует тот же SSRF-блоклист, описанный выше, поэтому remote-URL, указывающие на приватные/loopback-диапазоны, блокируются по умолчанию.

### Дашборд и темы

- **Двойной дашборд:** новый UI **Nexus** и **Classic**-дашборд.
- **Темы:** Light, Dark и **System** (по умолчанию). См. [vuetify.ts](frontend/src/plugins/vuetify.ts).

### Дополнительно

- Продвинутый интерфейс маршрутизации трафика с поддержкой PROXY Protocol, External, прозрачного прокси, SSL-сертификатов и настройки портов.
- Продвинутый интерфейс настройки входящих и исходящих подключений.
- Поддержка лимита трафика и срока действия для клиентов.
- Онлайн-клиенты, статистика входящего/исходящего трафика и мониторинг состояния системы.
- Сервис подписок поддерживает внешние ссылки и подписки.
- Веб-панель и сервис подписок поддерживают защищённый доступ по HTTPS (нужно предоставить свой домен и SSL-сертификат).

## Telegram-уведомления

Выключено по умолчанию. После включения панель отправляет короткие сообщения о событиях в чат Telegram через вашего собственного бота. Всё настраивается в разделе **Settings -> Telegram**: укажите токен бота (от [@BotFather](https://t.me/BotFather)) и числовой chat id, затем кнопкой **Test** отправьте пробу. Каждое сообщение и запись аудита сначала редактируются, поэтому токены бота, учётные данные прокси, cookie и секреты OTP/TOTP никогда не попадают в Telegram.

После включения уведомления приходят о событиях, таких как успешный/неудачный вход, отзыв всех сессий администраторов, перезапуск ядра sing-box, высокая/нормальная нагрузка CPU (гистерезис вокруг `telegramCpuThreshold`, по выбору) и плановые heartbeat-отчёты по cron. События входа/перезапуска несут только метаданные, не нарушающие приватность (IP клиента, SHA-256-хеш user-agent, метка времени). Сообщения ставятся в очередь и доставляются асинхронно с повторами.

По умолчанию панель обращается к `api.telegram.org` напрямую; в поле **Transport** можно направить egress через HTTP/HTTPS/SOCKS5-прокси или через работающий outbound sing-box (по тегу).

Также можно доставлять зашифрованную резервную копию базы в тот же чат документом `*.db.aes` (Argon2id + AES-GCM, на стороне панели). Включите, задайте парольную фразу (минимум 12 символов) и расписание. Расшифровка встроенным CLI:

```bash
sui decrypt-backup --in s-ui-backup-YYYYMMDD-HHMMSSZ.db.aes --out restored.db
```

> Важно: сам текст сообщения в Telegram открытый. Относитесь к чату как к любому внешнему каналу.

## Платные подписки (экспериментально)

> **Экспериментальная функция, по умолчанию ВЫКЛЮЧЕНА.** Модуль изолирован от ядра (собственные таблицы БД, запускается только при включении) и помечен плашкой `experimental`. Изучите сценарии, прежде чем подключать реальные платежи.

Модуль превращает панель в небольшой подписочный сервис через **отдельного Telegram-бота для клиентов** (токен отличается от админского уведомителя). Конечные пользователи получают свою ссылку / QR / статистику трафика и могут **купить или продлить доступ** через платёжного провайдера, не касаясь админ-панели. Настройка — в разделе **Paid Subscriptions** ([PaidSubscriptions.vue](frontend/src/views/paidsub/PaidSubscriptions.vue)); настройка `paidSubEnabled` включает бота на лету.

Каждый Telegram-пользователь сопоставляется один к одному с единственным клиентом и может управлять только им. Задайте **тарифы** для покупки (цена, валюта, сумма в Stars, +дни, +трафик) и при желании включите **авторегистрацию** — бот создаёт пробного клиента при первом `/start`. Продление применяется ровно один раз. Поддерживаемые провайдеры (все по умолчанию выключены): **Telegram Stars, YooKassa, Stripe, PayMaster, CryptoBot и внешняя ссылка** — каждый появляется, только если включён и задан его токен/шаблон. **Возвраты**: автоматически через Bot API возвращаются только Telegram Stars; для остальных провайдеров панель лишь помечает заказ возвращённым — **деньги нужно вернуть в личном кабинете самого провайдера.**

> **Безопасность:** платёжные токены хранятся в зашифрованном виде. Для продакшена задайте стабильную переменную **`SUI_SECRETBOX_KEY`** (ключ вне БД); иначе используется ключ, выведенный из самой базы. Держите его неизменным между перезапусками — ротация или потеря делает сохранённые токены невосстановимыми.

## Служба подписок

Отдельный HTTP(S)-листенер (порт по умолчанию `2096`, префикс `/sub/`) с собственным TLS, таймаутами и security-заголовками `no-store`. Каждый включённый клиент доступен по персональной секретной ссылке (`sub_secret`, UUIDv4, генерируется при первом обращении) — а не по имени.

На одного клиента указывают несколько маршрутов: `/sub/<secret>` (список ссылок), `/sub/json/<secret>` и `/sub/clash/<secret>`, плюс алиасы верхнего уровня `/json/<secret>` и `/clash/<secret>`. Все принимают `GET` и `HEAD`. Маршрут ссылок также переключает формат по query: `?format=json` / `?format=clash` (другое значение даёт `400`). Для совместимости `/sub/<name>` ищет по имени, пока `subSecretRequired` не равен `true`.

Форматы вывода: список ссылок (base64 при `subEncode=true`, иначе сырой), полный конфиг sing-box JSON и Clash/Mihomo YAML — каждый с переключателем `subLinkEnable` / `subJsonEnable` / `subClashEnable`. Ответы несут заголовки подписки, такие как `Subscription-Userinfo`, `Profile-Update-Interval`, `Profile-Title`, …, все санитизируются (управляющие символы вырезаются, макс. 512 байт), чтобы данные клиента не внедрили заголовки. Вывод JSON/Clash настраивается параметрами `subJson*` / `subClashExt` (см. [sub/jsonService.go](sub/jsonService.go)).

Защита: лимит запросов на IP (по умолчанию `60` запр/мин, `subRateLimitPerIP`; при превышении `429` + `Retry-After`), `Cache-Control: no-store` и отклонение запросов с несовпадающим `Host`, если задан `subDomain`.

## SSL-сертификаты

<details>
  <summary>Показать подробности</summary>

### Certbot

```bash
snap install core; snap refresh core
snap install --classic certbot
ln -s /snap/bin/certbot /usr/bin/certbot

certbot certonly --standalone --register-unsafely-without-email --non-interactive --agree-tos -d <ваш домен>
```

</details>

## Запуск за обратным прокси

Размещение nginx (или любого обратного прокси) перед панелью поддерживается. Терминируйте TLS на прокси, пробросьте loopback-порт панели и пробросьте WebSocket-апгрейд, чтобы работал realtime-канал по адресу `/app/api/realtime/ws`.

```nginx
location /app/ {
    proxy_pass http://127.0.0.1:2095;
    proxy_http_version 1.1;
    # WebSocket-апгрейд — требуется для /app/api/realtime/ws
    proxy_set_header Upgrade    $http_upgrade;
    proxy_set_header Connection $connection_upgrade;  # map default upgrade; '' close;
    proxy_set_header Host              $host;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 3600s;  # realtime-поток долгоживущий
}
```

**Задайте `SUI_TRUSTED_PROXIES`** (IP/CIDR прокси через запятую, напр. `127.0.0.1,::1`), иначе панель полностью игнорирует `X-Forwarded-For`/`X-Forwarded-Proto` — каждый запрос выглядит как пришедший с прокси, что ломает rate-limit, блокировку входа, журналы аудита, HSTS и флаг `Secure` у cookie. Указывайте значение как можно уже.

**Не вырезайте собственные заголовки безопасности панели** (CSP, `X-Frame-Options`, HSTS и др. — см. [middleware/securityHeaders.go](middleware/securityHeaders.go)). Не добавляйте `proxy_hide_header` и не навешивайте в nginx второй, конфликтующий CSP/`X-Frame-Options`. Держите панель привязанной к loopback, чтобы до неё нельзя было достучаться иначе как через прокси.

## Разработка и сборка

s-ui-x-extended собирается в два этапа: фронтенд на Vue компилируется в статические ассеты, затем бэкенд на Go собирается с тегами `with_*`. Эталон — [build.sh](build.sh).

```bash
# 1) Фронтенд -> статические ассеты
cd frontend && npm i && npm run build && cd ..
# 2) Скопировать UI в дерево embed для Go
mkdir -p web/html && rm -fr web/html/* && cp -R frontend/dist/* web/html/
# 3) Бэкенд (полную строку -tags см. в build.sh)
go build -ldflags '-w -s -checklinkname=0 ...' -tags "with_quic,with_grpc,...,with_wireguard,..." -o sui main.go
```

> **Профили сборки различаются.** Готовые release-тарболы GitHub (путь по умолчанию для `install.sh`) — это **сокращённый** набор тегов, который **не** включает WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel и DHCP-DNS — их настройка вернёт ошибку «rebuild with `-tags with_…`». **Полный** набор (`with_wireguard`, `with_masque`, `with_openvpn`, `with_mtproxy`, `with_sudoku`, `with_trusttunnel`, `with_dhcp`, …) собирают только Docker (`docker build .`) и [build.sh](build.sh). Naive дополнительно требует зафиксированную нативную библиотеку cronet — проще всего через Docker.

Обязательные проверки CI ([ci.yml](.github/workflows/ci.yml)): `build`, `vet`, `test-go`, `fe-lint`, `fe-build`, `fe-vitest`. Локально — через цели `make audit` в [Makefile](Makefile).

<details><summary>Полная матрица тег → возможность</summary>

`with_quic` (Hysteria/Hysteria2/TUIC, DoQ), `with_grpc`, `with_utls`, `with_acme`, `with_gvisor`, `with_tailscale` и `with_naive_outbound` есть во всех профилях. Только в полных сборках: `with_wireguard` (вкл. WARP), `with_masque`, `with_openvpn`, `with_mtproxy`, `with_sudoku`, `with_trusttunnel`, `with_dhcp`, а также `with_ccm`/`with_ocm`/`with_oomkiller`. `with_profiler` — только для разработки (`/debug/pprof/*` без аутентификации, никогда не привязывайте к не-loopback). Точные теги и stub-ы см. в [build.sh](build.sh) и [core/](core).

</details>

## FAQ / Решение проблем

**Потерян пароль администратора.** Сбросьте его на хосте командой `s-ui admin -reset` (или через меню управления). См. [Меню управления s-ui и CLI](#меню-управления-s-ui-и-cli).

**IP-функции (IP-монитор, реальный IP клиента в логах) показывают IP прокси.** Панель игнорирует `X-Forwarded-For`, пока ваш прокси не указан в `SUI_TRUSTED_PROXIES`. См. [Запуск за обратным прокси](#запуск-за-обратным-прокси).

**Внешний URL подписки не импортируется.** Приватные, loopback и link-local адреса блокируются по умолчанию (защита от SSRF). Включайте `SUI_ALLOW_PRIVATE_SUB_URLS=true` только если доверяете источнику.

**Протокол (WireGuard, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS) выдаёт ошибку `… is not included in this build, rebuild with -tags with_…`.** Текущие сборки (тарболлы, Windows ZIP, Docker, `build.sh`) идут с полным набором тегов, так что это значит, что ваш бинарник собран до этого изменения. Обновитесь до актуального релиза или пересоберите из исходников / используйте Docker. См. [Поддерживаемые платформы](#поддерживаемые-платформы).

**Панель не стартует с `EADDRNOTAVAIL`.** Если сохранённый IP `webListen` / `subListen` больше не существует на хосте, панель пишет warning и слушает на всех интерфейсах — проверьте лог и поправьте адрес.

**Секреты стали нечитаемыми после восстановления или миграции.** Зашифрованным at-rest настройкам нужен тот же `SUI_SECRETBOX_KEY`. Сохраняйте `/etc/s-ui/secretbox.env` между переустановками. См. [Переменные окружения](#переменные-окружения).

## Языки

- Английский
- Персидский
- Вьетнамский
- Упрощенный китайский
- Традиционный китайский
- Русский

## Участие в проекте

Вклад приветствуется. См. [`CONTRIBUTING.md`](CONTRIBUTING.md) для рабочего процесса и запустите локальные проверки перед открытием PR. Обязательные CI-проверки и детали сборки — в разделе [Разработка и сборка](#разработка-и-сборка).

## Лицензия

Распространяется под GNU General Public License v3.0 — см. [`LICENSE`](LICENSE).

## Поддержка

- Баги и запросы функций: [GitHub Issues](https://github.com/deposist/s-ui-x-extended/issues)
- Сообщения об уязвимостях: см. [Безопасность и харднинг](#безопасность-и-харднинг)
- Благодарности и upstream-проекты: см. ниже

## Благодарности

- **[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended)** от **shtorm-7** — расширенное ядро sing-box, вокруг которого построена эта панель. Спасибо за работу над расширенными протоколами и транспортами, благодаря которой возможен этот проект.
- **[`SagerNet/sing-box`](https://github.com/SagerNet/sing-box)** — исходное ядро.
- **[`alireza0/s-ui`](https://github.com/alireza0/s-ui)** — оригинальная панель, на которой основан проект.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date&theme=dark" />
  <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
  <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
</picture>
