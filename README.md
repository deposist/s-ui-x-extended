## S-UI-X Extended

<p align="center">
  <img width="492" height="450" alt="S-UI-X Extended logo" src="https://raw.githubusercontent.com/deposist/s-ui-x-extended/refs/heads/main/docs/592996937-cfc9da97-f8ea-4c68-961c-2bf164932272.png" />
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
  <img width="1024" alt="S-UI-X Extended panel screenshots" src="https://github.com/deposist/s-ui-x-extended/blob/main/docs/screen1.png" />
</p>

<p align="center">
  <img width="1024" alt="Support S-UI-X Extended development" src="docs/support-s-ui-x.png" />
</p>

## Support S-UI-X Extended

S-UI-X Extended is maintained as an open-source project. Donations help fund continued
development, security hardening, testing, and release work.

- WEB: [https://web.tribute.tg/d/LRJ](https://web.tribute.tg/d/LRJ)
- Telegram: [https://t.me/tribute/app?startapp=dLRJ](https://t.me/tribute/app?startapp=dLRJ)

| Network | Address |
| ------- | ------- |
| TON | `UQB5-DZ3q5vjXGf3_tVUeOHPNuXMLh8lfY0MPW3uGzjdOzke` |
| ETH | `0x0e67e1b4363a163c36943Ef4F9227c3126bB952B` |
| SOL | `BtFm5E1BrUjpoaDNwv3emc2qbyvqkn6ECnDzwhgRn7Df` |
| TRX | `TFqEbp1Z82ZQebzDdsW1MbytMvVsHJGpPd` |
| BTC | `bc1qn86mfmsnackfwvjd4czjaalv75sh830fws7xc9` |

## English

Web panel built on the [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) core (the shtorm-7 fork of `SagerNet/sing-box`).

**Note:** this repository is based on `alireza0/s-ui` starting from `v1.4.1`, with security and reliability hardening applied on top. This extended fork achieves functional parity with upstream s-ui-x `v1.5.10-beta7` while running on the `sing-box-extended` core.

**This fork keeps the original project structure and maintains the documentation and install links for this repository. You can use these scripts directly, or fork the repository and build it yourself.**

> **Disclaimer:** this project is intended only for personal learning and knowledge sharing. Do not use it for illegal purposes.

## Releases

The full per-release notes live in the language-specific changelog files:

- English: [`CHANGELOG-EN.md`](CHANGELOG-EN.md)
- Русский: [`CHANGELOG-RU.md`](CHANGELOG-RU.md)
- 简体中文: [`CHANGELOG-ZH.md`](CHANGELOG-ZH.md)
- Latest pre-release notes: [`docs/releases/v1.0.0-beta9.md`](docs/releases/v1.0.0-beta9.md)
- Upstream parity reference: [`docs/releases/v1.5.10-beta7.md`](docs/releases/v1.5.10-beta7.md)

The README keeps installation and project overview short. For full release
history, breaking notes, upgrade guidance, and rollback notes, open the
changelog in your preferred language.

## How this differs from `alireza0/s-ui`

<details>
  <summary>Show details</summary>

This fork stays compatible with existing 1.x installations. You can replace the binary on top of an existing install, and the panel applies database migrations on first start. The protocol behavior is kept close to upstream; most changes are around security, safer operations, observability, and the admin UI.

- Authentication is stricter. Fresh installs get a random first-run admin password, passwords are stored with bcrypt, browser sessions use hardened cookies, and mutating browser API calls require CSRF protection. API tokens are hashed and scoped as `admin`, `read`, `write`, or `observability`.
- Secrets are treated as secrets. Telegram credentials, proxy credentials, install salt, and other sensitive settings are encrypted at rest through secretbox. Telegram messages, audit details, backup captions, and change history are redacted before they leave the panel.
- Network-facing inputs have tighter guardrails. `X-Forwarded-For` is ignored unless trusted proxies are configured. External subscription fetches use URL and IP checks, block private and loopback targets by default, cap responses at 4 MiB, and re-check resolved IPs at dial time.
- Subscriptions are safer to operate. Per-client subscription secrets are supported for link, JSON, and Clash formats. Legacy name-based subscription URLs still work while `subSecretRequired=false`. Subscription responses sanitize headers, apply per-IP rate limits, support gzip, and use a short output cache for successful responses.
- The core is managed in-process. sing-box runs as an embedded Go library, not as a subprocess. Saves for clients, TLS, inbounds, outbounds, endpoints, and services hot-apply the affected object where possible. A full restart is used only when the change needs it or when hot apply cannot keep the running config safe.
- Routing has panel-managed outbound groups and provider-backed membership. Selector, URLTest, fallback, and failover groups have tag validation, previews, and capability metadata. The panel-managed `failover` type probes members over HTTPS, switches new connections away from a failed active member, and supports explicit all-down policies.
- The panel can update itself from the web UI. Admins can check stable or beta releases from Settings, verify the downloaded binary against the release SHA-256, apply the update, and roll back automatically if the new binary repeatedly fails to start. The action is audited and requires password re-entry.
- Backups and imports are more defensive. Imports enforce a 64 MiB cap, SQLite magic checks, staging, read-only integrity checks, schema migrations, and rollback to the previous DB on failure. Local unencrypted backup export streams the prepared SQLite file instead of buffering the whole backup in memory.
- Audit and observability are built in. The panel stores audit events with retention cleanup, exposes a scoped and paginated audit API, keeps bounded logs, samples bounded observability buckets, and publishes realtime events over a hardened WebSocket path with single-use tokens and Origin checks.
- IP monitoring is privacy-aware by default. Client IP history uses salted hashes, raw IP display is opt-in, retention is configurable, and enforce mode rejects only new over-limit connections instead of closing active ones.
- Server defaults are safer. Panel and subscription HTTP servers have read, write, header, and idle timeouts. TLS uses `MinVersion = 1.2`. Security headers are enabled, and subscription responses are marked no-store. If a saved listen IP no longer exists on the host, fallback stays restricted instead of silently widening exposure.
- The frontend is maintained as a full panel UI. Nexus is the default interface, classic mode remains available, settings show defaults and help text, update release notes render as safe Markdown, and route-based code splitting keeps heavy views out of the initial path.
- Performance work is included. Stats queries and chart downsampling are optimized, stats writes use safe batching, `/api/load` avoids duplicate settings reads and parallelizes independent reads, WebSocket broadcasts marshal once per event, and frontend vendor chunks are split for better browser caching.
- Localization is part of the fork. The panel, install script, and terminal menu support English, Russian, and Chinese. The default timezone is `Europe/Moscow`; existing browsers keep their saved locale choice.

</details>

## Overview

| Feature | Support |
| -------------------------------------- | :----------------: |
| Multiple protocols | :heavy_check_mark: |
| Multiple languages | :heavy_check_mark: |
| Multiple clients/inbounds | :heavy_check_mark: |
| Traffic routing interface | :heavy_check_mark: |
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

## Install or upgrade

Use the stable build for normal installations. Use the beta only if you want to test the newest changes before they become stable.

| Channel | Version | Notes |
|---|---|---|
| Stable | not yet released | The first stable release will follow after the beta cycle completes. |
| Beta | `v1.0.0-beta9` | Pre-release build. Rebuilds official artifacts with the full supported protocol tag set, including Sudoku, TrustTunnel, MASQUE, and OpenVPN. Release notes: [`docs/releases/v1.0.0-beta9.md`](docs/releases/v1.0.0-beta9.md). |

### Linux/macOS, beta

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0-beta9
```

The command above installs the latest beta release. Once a stable release is published, the command without a version argument will install stable by default.

### Local clone

```sh
git clone https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.0-beta9
```

### Windows

- Beta: download `v1.0.0-beta9` from [its release page](https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta9), extract the ZIP, and run `install-windows.bat` as Administrator.

Existing installations keep their settings, users, inbounds, outbounds, clients, TLS, services, and tokens. Database migrations run automatically on first start. Upgrade and rollback notes are in the changelog files: [EN](CHANGELOG-EN.md), [RU](CHANGELOG-RU.md), [中文](CHANGELOG-ZH.md).

## Manual Installation

### Linux/macOS

1. Download the latest S-UI-X Extended version for your system and architecture from GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. **Optional:** download the latest `s-ui.sh`: [https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh](https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh)
3. **Optional:** copy `s-ui.sh` to `/usr/bin/` and run `chmod +x /usr/bin/s-ui`.
4. Extract the S-UI-X Extended tar.gz archive to your chosen directory and enter the extracted folder.
5. Copy the `*.service` files to `/etc/systemd/system/`, then run `systemctl daemon-reload`.
6. Run `systemctl enable s-ui --now` to enable autostart and start the `s-ui` service used by S-UI-X Extended.
7. Run `systemctl enable sing-box --now` to start the sing-box service.

### Windows

1. Download the latest Windows version from GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. Download the appropriate Windows package, for example `s-ui-windows-amd64.zip`.
3. Extract the ZIP file to your chosen directory.
4. Run `install-windows.bat` as Administrator.
5. Follow the installation wizard.
6. Open the panel: http://localhost:2095/app

## Uninstall S-UI-X Extended

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

**Step 2:** install S-UI-X Extended

> Docker Compose option

```shell
services:
  s-ui:
    image: ghcr.io/deposist/s-ui-x-extended
    container_name: s-ui-x-extended
    hostname: "s-ui-x-extended"
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
mkdir s-ui-x-extended && cd s-ui-x-extended

docker run -itd \
    --network host \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name s-ui-x-extended \
    --restart=unless-stopped \
    ghcr.io/deposist/s-ui-x-extended
```

> Build the image yourself

```shell
git clone https://github.com/deposist/s-ui-x-extended
docker build -t s-ui-x-extended .
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
  - General protocols: Mixed, SOCKS, HTTP, HTTPS, Direct, Redirect, TProxy
  - V2Ray-based protocols: VLESS, VMess, Trojan, Shadowsocks
  - Other protocols: ShadowTLS, Hysteria, Hysteria2, Naive, TUIC
- XTLS protocol support.
- Traffic routing interface with PROXY Protocol, External, transparent proxy, SSL certificates, and port configuration support.
- Inbound and outbound configuration interface.
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

For systemd installs run by `install.sh`, S-UI-X Extended generates a stable
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

#### Credits

- Original panel project: [alireza0/s-ui](https://github.com/alireza0/s-ui)
- Extended core: [shtorm-7/sing-box-extended](https://github.com/shtorm-7/sing-box-extended)

---

## Русский

Web-панель на базе [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) (форк shtorm-7 от `SagerNet/sing-box`).

**Примечание:** этот репозиторий основан на `alireza0/s-ui`, начиная с `v1.4.1`, с применённым набором исправлений по безопасности и надёжности. Этот extended-форк достигает функционального паритета с upstream s-ui-x `v1.5.10-beta7` при работе на ядре `sing-box-extended`.

**Этот fork сохраняет структуру оригинального проекта и поддерживает документацию и ссылки установки для этого репозитория. Можно использовать эти скрипты напрямую или сделать fork и собрать проект самостоятельно.**

> **Отказ от ответственности:** этот проект предназначен только для личного обучения и обмена опытом. Не используйте его в незаконных целях.

## Релизы

Полные release notes лежат в отдельных файлах changelog по языкам:

- English: [`CHANGELOG-EN.md`](CHANGELOG-EN.md)
- Русский: [`CHANGELOG-RU.md`](CHANGELOG-RU.md)
- 简体中文: [`CHANGELOG-ZH.md`](CHANGELOG-ZH.md)
- Последние pre-release notes: [`docs/releases/v1.0.0-beta9.md`](docs/releases/v1.0.0-beta9.md)
- Реферс паритета с upstream: [`docs/releases/v1.5.10-beta7.md`](docs/releases/v1.5.10-beta7.md)

README оставляет только установку и общий обзор проекта. Полная история
релизов, breaking-заметки, гайд по обновлению и инструкции по откату находятся
в changelog на выбранном языке.

## Чем отличается от `alireza0/s-ui`

<details>
  <summary>Показать подробности</summary>

Форк остаётся совместимым с существующими установками 1.x. Новый бинарник можно поставить поверх старого, а миграции базы применятся при первом старте. Поведение протоколов держится близко к upstream; основные изменения касаются безопасности, безопасных операций, наблюдаемости и интерфейса администратора.

- Авторизация строже. На свежей установке создаётся случайный первый пароль администратора, пароли хранятся через bcrypt, browser sessions используют усиленные cookie, а mutating browser API calls требуют CSRF. API tokens хранятся как hashes и имеют scopes `admin`, `read`, `write` или `observability`.
- Секреты не хранятся открытым текстом. Telegram credentials, proxy credentials, install salt и другие чувствительные settings шифруются at rest через secretbox. Telegram messages, audit details, backup captions и change history проходят redaction перед отправкой наружу.
- Входные сетевые данные проверяются жёстче. `X-Forwarded-For` игнорируется без настроенных trusted proxies. Загрузка внешних подписок проверяет URL и IP, по умолчанию блокирует private и loopback targets, ограничивает ответ 4 MiB и повторно проверяет resolved IP во время dial.
- Подписки безопаснее в эксплуатации. Поддерживаются per-client subscription secrets для link, JSON и Clash formats. Legacy URLs по имени клиента работают, пока `subSecretRequired=false`. Subscription responses очищают headers, применяют per-IP rate limits, поддерживают gzip и используют короткий output cache для успешных ответов.
- Core управляется внутри процесса. sing-box запускается как встроенная Go library, не как subprocess. Изменения clients, TLS, inbounds, outbounds, endpoints и services применяются hot apply к затронутому объекту, когда это безопасно. Полный restart используется только когда изменение требует его или hot apply не может безопасно сохранить running config.
- Есть panel-managed outbound-группы и provider-backed участники. Selector, URLTest, fallback и failover используют проверку tag-ссылок, preview и capability metadata. Тип `failover` проверяет участников по HTTPS, переключает новые подключения с отказавшего активного участника и поддерживает явные all-down политики.
- Панель умеет обновляться из веб-интерфейса. Администратор может проверить stable или beta releases в Settings, сверить скачанный бинарь с release SHA-256, применить обновление и автоматически откатиться, если новый бинарь несколько раз не стартует. Действие попадает в audit и требует повторного ввода пароля.
- Backup и import стали осторожнее. Import ограничен 64 MiB, проверяет SQLite magic, использует staging, read-only integrity check, schema migrations и rollback к предыдущей базе при ошибке. Локальный незашифрованный export базы стримит подготовленный SQLite file, а не буферизует весь backup в памяти.
- Audit и observability встроены в панель. Есть audit events с retention cleanup, scoped и paginated audit API, bounded logs, bounded observability buckets и realtime events через защищённый WebSocket path с одноразовыми tokens и Origin checks.
- IP monitoring по умолчанию бережёт приватность. История IP клиентов хранится как salted hashes, raw IP display включается отдельно, retention настраивается, а enforce mode отклоняет только новые подключения сверх лимита и не закрывает активные.
- Server defaults безопаснее. У panel и subscription HTTP servers есть read, write, header и idle timeouts. TLS использует `MinVersion = 1.2`. Security headers включены, subscription responses помечены no-store. Если сохранённый listen IP больше не существует на хосте, fallback остаётся ограниченным и не расширяет доступ молча.
- Frontend поддерживается как полноценный интерфейс панели. Nexus является интерфейсом по умолчанию, classic mode остаётся доступен, settings показывают defaults и help text, release notes в Panel updates рендерятся как безопасный Markdown, а route-based code splitting не тянет тяжёлые views в initial path.
- Производительность тоже входит в форк. Оптимизированы stats queries и chart downsampling, запись stats использует safe batching, `/api/load` не делает повторные settings reads и параллелит независимые reads, WebSocket broadcasts сериализуют payload один раз, а frontend vendor chunks разделены для лучшего browser cache.
- Локализация поддерживается в панели и скриптах. Panel, install script и terminal menu доступны на English, Russian и Chinese. Default timezone: `Europe/Moscow`; существующие браузеры сохраняют выбранную locale из localStorage.

</details>

## Краткий обзор

| Возможность | Поддержка |
| -------------------------------------- | :----------------: |
| Несколько протоколов | :heavy_check_mark: |
| Несколько языков | :heavy_check_mark: |
| Несколько клиентов/Inbounds | :heavy_check_mark: |
| Интерфейс маршрутизации трафика | :heavy_check_mark: |
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

## Установка или обновление

Для обычных установок используйте stable. Beta нужна только если вы хотите проверить свежие изменения до стабильного релиза.

| Канал | Версия | Заметки |
|---|---|---|
| Stable | ещё не выпущена | Первый стабильный релиз выйдет после завершения beta-цикла. |
| Beta | `v1.0.0-beta9` | Pre-release сборка. Собирает официальные artifacts с полным набором protocol build tags, включая Sudoku, TrustTunnel, MASQUE и OpenVPN. Release notes: [`docs/releases/v1.0.0-beta9.md`](docs/releases/v1.0.0-beta9.md). |

### Linux/macOS, beta

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh) v1.0.0-beta9
```

Эта команда ставит последнюю beta-версию. Когда выйдет stable-релиз, команда без указания версии будет ставить stable по умолчанию.

### Локальный clone

```sh
git clone https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.0-beta9
```

### Windows

- Beta: скачайте `v1.0.0-beta9` на [странице релиза](https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta9), распакуйте ZIP и запустите `install-windows.bat` от имени администратора.

Существующие установки сохраняют settings, users, inbounds, outbounds, clients, TLS, services и tokens. Миграции базы запускаются автоматически при первом старте. Заметки по обновлению и откату находятся в changelog: [EN](CHANGELOG-EN.md), [RU](CHANGELOG-RU.md), [中文](CHANGELOG-ZH.md).

## Ручная установка

### Linux/macOS

1. Скачайте последнюю версию S-UI-X Extended для вашей системы и архитектуры из GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. **Необязательно:** скачайте последнюю версию `s-ui.sh`: [https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh](https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/s-ui.sh)
3. **Необязательно:** скопируйте `s-ui.sh` в `/usr/bin/` и выполните `chmod +x /usr/bin/s-ui`.
4. Распакуйте tar.gz-архив S-UI-X Extended в выбранный каталог и перейдите в распакованную папку.
5. Скопируйте файлы `*.service` в `/etc/systemd/system/`, затем выполните `systemctl daemon-reload`.
6. Выполните `systemctl enable s-ui --now`, чтобы включить автозапуск и запустить службу `s-ui`, используемую S-UI-X Extended.
7. Выполните `systemctl enable sing-box --now`, чтобы запустить службу sing-box.

### Windows

1. Скачайте последнюю версию для Windows из GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. Скачайте подходящий пакет для Windows, например `s-ui-windows-amd64.zip`.
3. Распакуйте ZIP-файл в выбранный каталог.
4. Запустите `install-windows.bat` от имени администратора.
5. Следуйте инструкциям мастера установки.
6. Откройте панель: http://localhost:2095/app

## Удаление S-UI-X Extended

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

**Шаг 2:** установите S-UI-X Extended

> Вариант с Docker Compose

```shell
services:
  s-ui:
    image: ghcr.io/deposist/s-ui-x-extended
    container_name: s-ui-x-extended
    hostname: "s-ui-x-extended"
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
mkdir s-ui-x-extended && cd s-ui-x-extended

docker run -itd \
    --network host \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name s-ui-x-extended \
    --restart=unless-stopped \
    ghcr.io/deposist/s-ui-x-extended
```

> Самостоятельная сборка образа

```shell
git clone https://github.com/deposist/s-ui-x-extended
docker build -t s-ui-x-extended .
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
  - Общие протоколы: Mixed, SOCKS, HTTP, HTTPS, Direct, Redirect, TProxy
  - Протоколы на базе V2Ray: VLESS, VMess, Trojan, Shadowsocks
  - Другие протоколы: ShadowTLS, Hysteria, Hysteria2, Naive, TUIC
- Поддержка протокола XTLS.
- Интерфейс маршрутизации трафика с поддержкой PROXY Protocol, External, прозрачного прокси, SSL-сертификатов и настройки портов.
- Интерфейс настройки Inbounds и Outbounds.
- Поддержка лимита трафика и срока действия для клиентов.
- Отображение онлайн-клиентов, статистики трафика Inbounds/Outbounds и мониторинг состояния системы.
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

Для systemd-установок через `install.sh` S-UI-X Extended один раз генерирует стабильный
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

#### Благодарности

- Автор оригинальной панели: [alireza0/s-ui](https://github.com/alireza0/s-ui)
- Extended-ядро: [shtorm-7/sing-box-extended](https://github.com/shtorm-7/sing-box-extended)

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date&theme=dark" />
  <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
  <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
</picture>
