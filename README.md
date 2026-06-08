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

## English

**s-ui-x with sing-box-extended core support.** An advanced web panel for sing-box, running on the [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) core (the shtorm-7 fork of `SagerNet/sing-box`).

This project adds extended protocol and transport support on top of the s-ui-x panel, while inheriting the security and reliability hardening of the panel. It is based on `alireza0/s-ui` (current build: `v1.0.0-beta1`).

> **Disclaimer:** this project is intended only for personal learning and knowledge sharing. Do not use it for illegal purposes.

## Releases

The full release history and current release notes live in:

- [`CHANGELOG.md`](CHANGELOG.md)
- Current release notes: [`docs/releases/v1.0.0-beta1.md`](docs/releases/v1.0.0-beta1.md)

The README keeps installation and project overview short. For full release
history, breaking notes, upgrade guidance, and rollback notes, open the
changelog.

## Key differences vs `alireza0/s-ui`

<details>
  <summary>Show details</summary>

This fork is binary-compatible with `alireza0/s-ui` — drop the new
binary on top of an existing 1.x install, the panel migrates the DB
automatically on first start. The intent is to harden security and
reliability without changing the protocol surface.

- **Auth and session security.** bcrypt with lazy migration, randomly generated first-run password (logged once), login rate limiter, `HttpOnly` + `SameSite=Lax` + HTTPS-aware `Secure` cookies. Sensitive settings (Telegram bot token, proxy credentials, install salt) are encrypted at rest via secretbox; API tokens are stored as salted SHA-256 hashes. CSRF protection is enforced on browser `/api/*` mutating requests.
- **API token scopes.** `admin`, `read`, `write`, and `observability` scopes are documented in [`docs/scope-matrix.md`](docs/scope-matrix.md), including audit, database, Telegram, subscription-secret rotation, observability, and realtime security-event behavior.
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

## Overview

| Feature | Support |
| -------------------------------------- | :----------------: |
| Multiple protocols | :heavy_check_mark: |
| Multiple languages | :heavy_check_mark: |
| Multiple clients/inbounds | :heavy_check_mark: |
| Advanced traffic routing interface | :heavy_check_mark: |
| Client, traffic, and system status | :heavy_check_mark: |
| Subscription links (link/json/clash + info) | :heavy_check_mark: |
| Dark/light theme | :heavy_check_mark: |
| API | :heavy_check_mark: |

## Supported Platforms

| Platform | Architecture | Status |
|----------|--------------|---------|
| Linux | amd64, arm64, armv7, armv6, armv5, 386, s390x | Supported |
| Windows | amd64, 386, arm64 | Supported |
| macOS | amd64, arm64 | Experimental support |

## Default Installation Information

- Panel port: 2095
- Panel path: /app/
- Subscription port: 2096
- Subscription path: /sub/
- Subscription per-IP rate-limit changes (`subRateLimitPerIP`) take effect within 1 minute after saving.
- Username: admin
- Password (fresh install only): a random 24-character string is generated on first start and written to the application log. Look for the line `created initial admin user. username=admin password=...` in `journalctl -u s-ui` (Linux) or in the panel log on first run. After that, change it from the panel.

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

## Install v1.0.0-beta1

This is the initial public beta of s-ui-x-extended — see [`CHANGELOG.md`](CHANGELOG.md)
and [`docs/releases/v1.0.0-beta1.md`](docs/releases/v1.0.0-beta1.md). This is a beta — test first.

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0-beta1
```

Or from a local clone:

```sh
git clone https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.0-beta1
```

## Install an Older Version

Append the version tag with `v` to the installation command. For example, version `v1.0.0`:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0
```

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

## Uninstall S-UI

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

## How to read CI status

Required checks are `build`, `vet`, `test-go`, `fe-lint`, `fe-build`, and
`fe-vitest`. Additional diagnostic jobs such as race detector, gosec,
govulncheck, staticcheck/golangci-lint, chaos, perf and flaky e2e can be useful
for maintainers, but are treated as advisory unless branch protection says
otherwise.

## Manual Run for Development and Contributions

<details>
   <summary>Show details</summary>

### Build and Run the Full Project

```shell
./runSUI.sh
```

### Clone the Repository

```shell
# Clone the repository
git clone https://github.com/deposist/s-ui-x-extended
```

### Frontend

The frontend code is in the [frontend](frontend) directory.

### Backend

> Build the frontend at least once before building the backend.

Build the backend:

```shell
# Remove old frontend build files
rm -fr web/html/*
# Copy new frontend build files
cp -R frontend/dist/ web/html/
# Build
go build -o sui main.go
```

Run the backend from the repository root:

```shell
./sui
```

</details>

## Languages

- English
- Persian
- Vietnamese
- Simplified Chinese
- Traditional Chinese
- Russian

## Features

- Supported protocols:
  - General / transparent: Mixed, SOCKS, HTTP, Direct, Redirect, TProxy, Tun
  - V2Ray-based: VLESS, VMess, Trojan, Shadowsocks
  - Other proxy: ShadowTLS, AnyTLS, Naive, Hysteria, Hysteria2, TUIC, TrustTunnel, Mieru, Sudoku, SSH, Tor, MTProxy
  - Tunnel / VPN (endpoints & outbounds): WireGuard, WARP, Tailscale, AmneziaWG 2.0, MASQUE, OpenVPN, VPN server / client
  - Outbound groups & traffic control: Selector, URLTest, Bond, Failover, Fallback, Parser, and bandwidth / connection / traffic / rate limiters
  - DNS transports: TCP, UDP, DoT, DoH, DoQ, DNSCrypt (SDNS), DHCP, FakeIP, Hosts, Local, Fallback, Resolved, Tailscale
  - Most proxy protocols work as both inbound and outbound; MTProxy is inbound-only, Tor / MASQUE / OpenVPN are outbound-only.
- XTLS protocol support.
- Advanced traffic routing interface with PROXY Protocol, External, transparent proxy, SSL certificates, and port configuration support.
- Advanced inbound and outbound configuration interface.
- Client traffic limit and expiration support.
- Online clients, inbound/outbound traffic statistics, and system status monitoring.
- Subscription service supports external links and subscriptions.
- Web panel and subscription service support secure HTTPS access (you must provide your own domain and SSL certificate).
- Dark/light theme.

## Environment Variables

<details>
  <summary>Show details</summary>

### Usage

| Variable | Type | Default |
| -------------- | :--------------------------------------------: | :------------ |
| SUI_LOG_LEVEL | `"debug"` \| `"info"` \| `"warn"` \| `"error"` | `"info"` |
| SUI_DEBUG | `boolean` | `false` |
| SUI_BIN_FOLDER | `string` | `"bin"` |
| SUI_DB_FOLDER | `string` | `"db"` |
| SINGBOX_API | `string` | - |
| SUI_TRUSTED_PROXIES | comma-separated CIDRs / IPs | - (XFF ignored) |
| SUI_ALLOW_PRIVATE_SUB_URLS | `boolean` | `false` |
| SUI_SECRETBOX_KEY | `string` | - (falls back to `settings.secret`) |

For systemd installs run by `install.sh`, S-UI generates a stable
`SUI_SECRETBOX_KEY` once in `/etc/s-ui/secretbox.env`, shows the generated
value once, and loads the file through a systemd drop-in. Keep that file
private and preserve the same key across updates and restores.

</details>

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

## Acknowledgments

- **[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended)** by **shtorm-7** — the extended sing-box core this panel is built around. Thank you for the extended protocol and transport work that makes this project possible.
- **[`SagerNet/sing-box`](https://github.com/SagerNet/sing-box)** — the upstream core.
- **[`alireza0/s-ui`](https://github.com/alireza0/s-ui)** — the original panel this project is based on.

---

## Русский

**s-ui-x с поддержкой ядра sing-box-extended.** Продвинутая веб-панель для sing-box на ядре [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) (форк SagerNet/sing-box от shtorm-7).

Проект добавляет поддержку расширенных протоколов и транспортов поверх панели s-ui-x, наследуя при этом её усиление безопасности и надёжности. Основан на `alireza0/s-ui` (текущая сборка: `v1.0.0-beta1`).

> **Отказ от ответственности:** этот проект предназначен только для личного обучения и обмена опытом. Не используйте его в незаконных целях.

## Релизы

Полная история релизов и текущие release notes находятся в:

- [`CHANGELOG.md`](CHANGELOG.md)
- Release notes текущего релиза: [`docs/releases/v1.0.0-beta1.md`](docs/releases/v1.0.0-beta1.md)

README оставляет только установку и общий обзор проекта. Полная история
релизов, breaking-заметки, гайд по обновлению и инструкции по откату находятся
в changelog.

## Ключевые отличия от `alireza0/s-ui`

<details>
  <summary>Показать подробности</summary>

Этот форк бинарно совместим с `alireza0/s-ui` — новый бинарник можно
ставить поверх работающей установки 1.x, схема БД автоматически
обновится при первом старте. Цель форка — усилить безопасность и
надёжность, не меняя протокол.

- **Авторизация и сессия.** bcrypt с ленивой миграцией, случайный пароль администратора при первой установке (выводится в журнал один раз), лимит на неуспешные логины, cookie сессии — `HttpOnly` + `SameSite=Lax` + `Secure` при HTTPS. Чувствительные настройки (Telegram bot token, креденшелы прокси, install salt) шифруются at-rest через secretbox; API-токены хранятся как salted SHA-256. CSRF-защита на browser `/api/*`-mutating-запросах.
- **Scopes API-токенов.** `admin`, `read`, `write` и `observability` описаны в [`docs/scope-matrix.md`](docs/scope-matrix.md), включая audit, database, Telegram, rotation subscription-secret, observability и realtime security-event.
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

## Краткий обзор

| Возможность | Поддержка |
| -------------------------------------- | :----------------: |
| Несколько протоколов | :heavy_check_mark: |
| Несколько языков | :heavy_check_mark: |
| Несколько клиентов/входящих подключений | :heavy_check_mark: |
| Продвинутый интерфейс маршрутизации трафика | :heavy_check_mark: |
| Клиенты, трафик и состояние системы | :heavy_check_mark: |
| Ссылки подписки (link/json/clash + info) | :heavy_check_mark: |
| Темная/светлая тема | :heavy_check_mark: |
| API | :heavy_check_mark: |

## Поддерживаемые платформы

| Платформа | Архитектура | Статус |
|----------|--------------|---------|
| Linux | amd64, arm64, armv7, armv6, armv5, 386, s390x | Поддерживается |
| Windows | amd64, 386, arm64 | Поддерживается |
| macOS | amd64, arm64 | Экспериментальная поддержка |

## Информация об установке по умолчанию

- Порт панели: 2095
- Путь панели: /app/
- Порт подписки: 2096
- Путь подписки: /sub/
- Изменения лимита подписок на IP (`subRateLimitPerIP`) применяются в течение 1 минуты после сохранения.
- Имя пользователя: admin
- Пароль (только для свежей установки): при первом запуске генерируется случайная строка из 24 символов, которая выводится в журнал приложения. Найдите строку `created initial admin user. username=admin password=...` в `journalctl -u s-ui` (Linux) или в журнале панели после первого запуска. После входа смените пароль в настройках.

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

## Установка v1.0.0-beta1

Это первая публичная бета s-ui-x-extended — см. [`CHANGELOG.md`](CHANGELOG.md)
и [`docs/releases/v1.0.0-beta1.md`](docs/releases/v1.0.0-beta1.md). Это бета —
сначала протестируйте.

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0-beta1
```

Или из локального клона:

```sh
git clone https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.0-beta1
```

## Установка старой версии

Чтобы установить определённую старую версию, добавьте тег версии с `v` в конец команды установки. Например, версия `v1.0.0`:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0
```

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

## Удаление S-UI

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

## Как читать CI status

Обязательные проверки: `build`, `vet`, `test-go`, `fe-lint`, `fe-build` и
`fe-vitest`. Дополнительные diagnostic jobs вроде race detector, gosec,
govulncheck, staticcheck/golangci-lint, chaos, perf и flaky e2e полезны для
maintainer-проверки, но считаются advisory, если branch protection не требует
обратного.

## Ручной запуск для разработки и участия в проекте

<details>
   <summary>Показать подробности</summary>

### Сборка и запуск полного проекта

```shell
./runSUI.sh
```

### Клонирование репозитория

```shell
# Клонирование репозитория
git clone https://github.com/deposist/s-ui-x-extended
```

### Фронтенд

Код фронтенда находится в каталоге [frontend](frontend).

### Бэкенд

> Перед сборкой бэкенда нужно хотя бы один раз собрать фронтенд.

Сборка бэкенда:

```shell
# Удаление старых собранных файлов фронтенда
rm -fr web/html/*
# Копирование новых собранных файлов фронтенда
cp -R frontend/dist/ web/html/
# Сборка
go build -o sui main.go
```

Запуск бэкенда из корня репозитория:

```shell
./sui
```

</details>

## Языки

- Английский
- Персидский
- Вьетнамский
- Упрощенный китайский
- Традиционный китайский
- Русский

## Возможности

- Поддерживаемые протоколы:
  - Общие / прозрачные: Mixed, SOCKS, HTTP, Direct, Redirect, TProxy, Tun
  - На базе V2Ray: VLESS, VMess, Trojan, Shadowsocks
  - Прочие прокси: ShadowTLS, AnyTLS, Naive, Hysteria, Hysteria2, TUIC, TrustTunnel, Mieru, Sudoku, SSH, Tor, MTProxy
  - Туннели / VPN (endpoints и outbound): WireGuard, WARP, Tailscale, AmneziaWG 2.0, MASQUE, OpenVPN, VPN server / client
  - Outbound-группы и контроль трафика: Selector, URLTest, Bond, Failover, Fallback, Parser, а также лимитеры bandwidth / connection / traffic / rate
  - DNS-транспорты: TCP, UDP, DoT, DoH, DoQ, DNSCrypt (SDNS), DHCP, FakeIP, Hosts, Local, Fallback, Resolved, Tailscale
  - Большинство прокси-протоколов работают и как inbound, и как outbound; MTProxy — только inbound, Tor / MASQUE / OpenVPN — только outbound.
- Поддержка протокола XTLS.
- Продвинутый интерфейс маршрутизации трафика с поддержкой PROXY Protocol, External, прозрачного прокси, SSL-сертификатов и настройки портов.
- Продвинутый интерфейс настройки входящих и исходящих подключений.
- Поддержка лимита трафика и срока действия для клиентов.
- Отображение онлайн-клиентов, статистики трафика входящих и исходящих подключений, а также мониторинг состояния системы.
- Служба подписок поддерживает добавление внешних ссылок и подписок.
- Web-панель и служба подписок поддерживают безопасный доступ по HTTPS (необходимо самостоятельно предоставить домен и SSL-сертификат).
- Темная/светлая тема.

## Переменные окружения

<details>
  <summary>Показать подробности</summary>

### Использование

| Переменная | Тип | Значение по умолчанию |
| -------------- | :--------------------------------------------: | :------------ |
| SUI_LOG_LEVEL | `"debug"` \| `"info"` \| `"warn"` \| `"error"` | `"info"` |
| SUI_DEBUG | `boolean` | `false` |
| SUI_BIN_FOLDER | `string` | `"bin"` |
| SUI_DB_FOLDER | `string` | `"db"` |
| SINGBOX_API | `string` | - |
| SUI_TRUSTED_PROXIES | список CIDR/IP через запятую | - (XFF игнорируется) |
| SUI_ALLOW_PRIVATE_SUB_URLS | `boolean` | `false` |
| SUI_SECRETBOX_KEY | `string` | - (fallback на `settings.secret`) |

Для systemd-установок через `install.sh` S-UI один раз генерирует стабильный
`SUI_SECRETBOX_KEY` в `/etc/s-ui/secretbox.env`, один раз показывает
сгенерированное значение и подключает файл через systemd drop-in. Держите
этот файл в секрете и сохраняйте тот же ключ при обновлениях и
восстановлении.

</details>

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

## Благодарности

- **[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended)** от **shtorm-7** — расширенное ядро sing-box, вокруг которого построена эта панель. Спасибо за работу над расширенными протоколами и транспортами, благодаря которой возможен этот проект.
- **[`SagerNet/sing-box`](https://github.com/SagerNet/sing-box)** — исходное ядро.
- **[`alireza0/s-ui`](https://github.com/alireza0/s-ui)** — оригинальная панель, на которой основан проект.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date&theme=dark" />
  <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
  <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
</picture>
