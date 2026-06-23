# S-UI v1.5.2-beta - 3x-ui migration suite

> Pre-release. Adds a complete 3x-ui → s-ui migration suite (CLI, HTTP API,
> wizard UI, scheduled remote sync) on top of `v1.5.1-beta`.
> Full changelog and upgrade guide:
> [CHANGELOG-EN.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-EN.md)
> · [CHANGELOG-RU.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-RU.md)
> · [CHANGELOG-ZH.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-ZH.md).

The embedded `sing-box` runtime behaviour is unchanged from `v1.5.1-beta`.

---

## English

This release introduces a complete suite for migrating 3x-ui (Xray) into
s-ui (sing-box): one-shot local import, an interactive wizard with
plan/apply and rollback, remote sources over SSH and the 3x-ui HTTP API,
and scheduled incremental sync.

You can drop the new binary on top of an existing 1.x install. Schema
migrations and the post-migration adapter run automatically. **No data
loss, no manual steps.**

### Changes
- **Local import.** `s-ui import-xui --src x-ui.db` and a new section in
  the Backup & Restore modal. One transaction, auto-backup before write,
  `merge`/`replace`/`skip` strategies, dry-run mode, and `xui_import` audit
  events.
- **Migration wizard.** New page `/migrate-xui` with a four-step flow
  (upload → review → progress → result). Per-object plan/apply with
  `Source.Hash` validation, JSON preview for every inbound/endpoint/tls/
  client, WebSocket `xui_import_progress` events, rollback to the
  auto-backup, and JSON/Markdown report download. Reports live in
  `audit_events.details` only.
- **Remote sources.** `--remote ssh://...` (SFTP, host-key TOFU,
  `xui_known_hosts`) and `--remote http://...` (xuihttp, 3x-ui login
  flow). Acquired files are validated as SQLite and against
  `PRAGMA integrity_check` before any mapping happens.
- **Scheduled sync.** `s-ui sync-xui` plus a `/migrate-xui/schedule` page
  for managing profiles. Profiles live in `xui_sync_profiles` with
  `source_json` encrypted via AES-GCM and a key derived through
  HKDF-SHA256 from `config.GetSecret()` (override with
  `XUI_PROFILE_KEY_FILE`). The `xuiSyncJob` cron job runs incremental
  imports with `OnlyNew=true` and 10-minute minimum interval.
- **Historical and routing best-effort.** Optional import of
  `client_traffics`/`outbound_traffics` as `stats` aggregates, plus a
  best-effort mapping of Xray `routing.rules` and `dns.servers`
  (`geosite:*`, `geoip:*`, block, direct) into sing-box
  `route.rules`/`dns.servers`. Balancers and unsupported domain matching
  are reported as warnings.
- **`xui_remote` token scope.** A new scope, required for all remote and
  sync endpoints. Local `/api/import-xui*` endpoints stay under
  `database`/`admin`. `XUI_DISABLE_REMOTE=1` disables remote sources and
  the cron mode entirely.

### Compatibility

- All 1.5.1-beta endpoints, sessions, and CLI commands continue to work
  unchanged. New endpoints and scopes are additive.
- The `Dialect` interface ships with a single implementation
  (`dialect_3xui_mhsanaei`). Other 3x-ui forks return `xui_dialect_unknown`
  on import; future adapters will plug in without changes to the core.
- Tailscale-aware remote import is intentionally not a separate mode:
  s-ui already ships with `with_tailscale` and `endpoints.type=tailscale`,
  so the SSH source automatically works over a tailscale address when one
  is configured.

### Install / upgrade

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.2-beta
```

Or from a local clone:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.2-beta
```

Do not publish database backups, subscription URLs, private keys,
certificates, admin credentials, API tokens, Telegram bot tokens, proxy
credentials, Telegram backup keys, 3x-ui database files (`x-ui.db`),
3x-ui SSH credentials or 3x-ui session tokens in issues, pull requests,
CI logs or support chats.

### Verification

| Command | Result |
| --- | --- |
| `go vet ./...` | OK |
| `go build ./...` | OK |
| `go test ./...` | OK (importxui tests skip when `test-db/` fixtures are absent) |
| `npm run test:unit` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `go test -race ./...` | requires CGO and a C compiler |

`test-db/` is intentionally excluded from the repository (see `.gitignore`).
The importxui, ssh, xuihttp and CLI/API import tests skip gracefully when
those fixtures are not present.

### Rollback

1. `systemctl stop s-ui`.
2. Restore the backed-up `s-ui.db` and any matching `-wal`/`-shm` sidecars.
3. Reinstall the previous binary or pin the previous Docker image tag.
4. `systemctl start s-ui`.

Keep the `SUI_SECRETBOX_KEY` and (if used) `XUI_PROFILE_KEY_FILE` values
stable across upgrade and rollback. Losing those keys can make encrypted
settings or sync profiles unreadable.

---

## Русский

Этот релиз вносит полный пакет миграции из 3x-ui (Xray) в s-ui (sing-box):
разовый локальный импорт, интерактивный мастер с plan/apply и откатом,
удалённые источники по SSH и через HTTP-API 3x-ui, а также периодическая
инкрементальная синхронизация.

Можно положить новый бинарник поверх существующей установки 1.x. Миграции
схемы и post-migration-адаптер выполнятся автоматически. **Данные не
теряются, ручных действий не требуется.**

### Главное

- **Локальный импорт.** `s-ui import-xui --src x-ui.db` и новая секция в
  модале Backup & Restore. Одна транзакция, автобэкап до записи,
  стратегии `merge`/`replace`/`skip`, dry-run и audit-события `xui_import`.
- **Мастер миграции.** Новая страница `/migrate-xui` с четырёхшаговым
  flow (upload → review → progress → result). Per-object plan/apply с
  валидацией `Source.Hash`, JSON-предпросмотр каждого
  inbound/endpoint/tls/client, WebSocket-события `xui_import_progress`,
  откат к автобэкапу и выгрузка отчёта в JSON/Markdown. Отчёты лежат
  только в `audit_events.details`.
- **Удалённые источники.** `--remote ssh://...` (SFTP, host-key TOFU,
  таблица `xui_known_hosts`) и `--remote http://...` (xuihttp, login-flow
  3x-ui). Полученный файл проверяется как SQLite и через
  `PRAGMA integrity_check` до начала маппинга.
- **Периодическая синхронизация.** `s-ui sync-xui` и страница
  `/migrate-xui/schedule` для управления профилями. Профили хранятся в
  `xui_sync_profiles` с `source_json`, зашифрованным AES-GCM на ключе,
  выведенном через HKDF-SHA256 от `config.GetSecret()` (override через
  `XUI_PROFILE_KEY_FILE`). Cron-job `xuiSyncJob` выполняет инкрементальный
  импорт с `OnlyNew=true` и минимальным интервалом 10 минут.
- **Historical и routing best-effort.** Опциональный импорт
  `client_traffics`/`outbound_traffics` агрегатами в `stats`, плюс
  best-effort маппинг `routing.rules` и `dns.servers` Xray
  (`geosite:*`, `geoip:*`, block, direct) в sing-box
  `route.rules`/`dns.servers`. Balancers и нестандартное domain matching
  попадают в warning'и.
- **Scope токена `xui_remote`.** Обязателен для всех удалённых/sync
  endpoint'ов. Локальные `/api/import-xui*` остаются под
  `database`/`admin`. `XUI_DISABLE_REMOTE=1` отключает удалённые
  источники и cron-режим целиком.

### Совместимость

- Все endpoint'ы, сессии и CLI-команды 1.5.1-beta продолжают работать
  без изменений. Новые endpoint'ы и scope добавляются, не ломая старые.
- Интерфейс `Dialect` поставляется с единственной реализацией
  (`dialect_3xui_mhsanaei`). Импорт из других форков 3x-ui завершается с
  ошибкой `xui_dialect_unknown`; новые адаптеры подключаются без правок
  ядра.
- Отдельный режим импорта поверх Tailscale намеренно не делается: s-ui
  уже собирается с `with_tailscale` и поддерживает
  `endpoints.type=tailscale`, поэтому SSH-источник автоматически работает
  через tailscale-адрес, когда тот настроен.

### Установка / обновление

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.2-beta
```

Или из локального клона:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.2-beta
```

Не публикуйте бэкапы БД, subscription-URL, приватные ключи, сертификаты,
учётные данные администратора, API-токены, Telegram bot-токены, креденшелы
прокси, ключи Telegram-бэкапа, файлы баз 3x-ui (`x-ui.db`), SSH-креденшелы
3x-ui и session-токены 3x-ui в issues, pull requests, CI-логах и чатах
поддержки.

### Верификация

| Команда | Результат |
| --- | --- |
| `go vet ./...` | OK |
| `go build ./...` | OK |
| `go test ./...` | OK (тесты importxui скипаются, если `test-db/` нет) |
| `npm run test:unit` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `go test -race ./...` | требуется CGO и C-компилятор |

`test-db/` намеренно исключён из репозитория (см. `.gitignore`). Тесты
importxui, ssh, xuihttp и CLI/API-импорта аккуратно скипаются, если этих
фикстур нет.

### Откат

1. `systemctl stop s-ui`.
2. Восстановите `s-ui.db` из бэкапа и соответствующие `-wal`/`-shm`-сайдкары.
3. Поставьте предыдущий бинарник или верните предыдущий тег docker-образа.
4. `systemctl start s-ui`.

Сохраняйте `SUI_SECRETBOX_KEY` и (если используется) `XUI_PROFILE_KEY_FILE`
неизменными при апгрейде и откате. Потеря ключей сделает зашифрованные
настройки или sync-профили нечитаемыми.
