# S-UI v1.5.1-beta - remediation hardening and UI completion

> Pre-release. Closes the remaining security, privacy, realtime, Telegram,
> observability, frontend and test gaps from the `1.5.0` remediation cycle.
> Supersedes [`v1.4.3`](https://github.com/deposist/s-ui-rus-inst/releases/tag/v1.4.3).
> Full changelog and upgrade guide:
> [CHANGELOG-EN.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-EN.md)
> · [CHANGELOG-RU.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-RU.md)
> · [CHANGELOG-ZH.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-ZH.md).

The embedded `sing-box` runtime behaviour is unchanged from `v1.4.3`.

---

## English

This release closes the remaining security, privacy, realtime, Telegram,
observability, frontend and test gaps from the `1.5.0` remediation cycle.

You can drop the new binary on top of an existing 1.x install or restore an
older `.db` backup from the panel: schema migrations and the post-migration
adapter run automatically. **No data loss, no manual steps.**

### Changes
- **Async Telegram pipeline.** Notifications go through a bounded queue with
  retry/backoff and audited overflow/failure events; login and other handlers
  never block on Telegram network failures.
- **Encrypted Telegram secrets.** Bot tokens, proxy credentials, API tokens
  and backup keys are encrypted at rest, masked from settings responses,
  redacted from audit/change history and never put into Telegram captions.
- **Hardened realtime WebSocket.** Origin allow-listing, per-IP handshake
  rate limit, single-use replay-protected tokens, ping/pong heartbeat, idle
  close and close-all on session rotation.
- **Admin-scoped audit and Telegram test.** `GET /api/security/audit` and
  `POST /api/telegram/test` require `admin` scope for API tokens, are rate
  limited, and the audit list supports cursor pagination plus validated
  `event`/`severity` filters.
- **Privacy-by-default IP history.** Client IPs are stored as salted SHA-256
  hashes by default, raw display is opt-in, retention is handled by cron GC
  and `enforce` mode rejects only new over-limit connections.
- **Subscription completion.** Per-client subscription secrets, validated
  paths, sanitized headers, configurable per-IP rate limit, and 404 on legacy
  name URLs when `subSecretRequired=true`.
- **Telegram proxy egress and reports.** Validated HTTP/HTTPS/SOCKS5 proxy,
  normalized error classes, CPU hysteresis alerts, scheduled reports and
  encrypted DB backup export - all opt-in.
- **Bounded observability and logs.** Bounded buckets sampled by cron with
  validated API parameters; bounded `GET /api/logs` and fail-soft 1h-cached
  `GET /api/version`.
- **Frontend completion.** Realtime store with reconnect/degraded states and
  polling fallback, secret-aware fields, masked IP history modal, Telegram
  settings view and Audit view with cursor pagination and server-side
  filters.
- **Migration reliability.** SQLite WAL checkpoint runs only after a
  successful commit, fixing `database table is locked` failures when
  upgrading from `1.4.x`.
- **Server-side admin sessions.** Browser cookie carries only a signed
  session ID; session data lives in the local SQLite `sessions` table.
- **Verified install.** `install.sh` downloads the matching `*.sha256`
  artifact and runs `sha256sum -c` before extraction.

### Compatibility

- Existing subscription `name` URLs keep working while
  `subSecretRequired=false`.
- Legacy `/apiv2/*` `Token` header still works, but responses include
  `Deprecation` and `Sunset` headers. Legacy header sunset date:
  **`Sat, 15 Aug 2026 00:00:00 GMT`**. Move integrations to
  `Authorization: Bearer <token>` before that date.
- All new features stay off by default, except realtime websocket support
  with frontend polling fallback and monitor-only IP tracking.

### Install / upgrade

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.1-beta
```

Or from a local clone:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.1-beta
```

Do not publish database backups, subscription URLs, private keys,
certificates, admin credentials, API tokens, Telegram bot tokens, proxy
credentials or Telegram backup keys in issues, pull requests, CI logs or
support chats.

### Verification

| Command | Result |
| --- | --- |
| `go vet ./...` | OK |
| `go test ./...` | OK |
| `npm run test:unit` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `go test -race ./...` | requires CGO and a C compiler |

`go test -race ./...` could not run in this Windows workspace because the Go
race detector requires CGO and no C compiler is available:

```text
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
```

Install GCC (for example via MSYS2/UCRT64) and rerun:

```powershell
$env:CGO_ENABLED='1'
go test -race ./...
```

### Rollback

1. `systemctl stop s-ui`.
2. Restore the backed-up `s-ui.db` and any matching `-wal`/`-shm` sidecars.
3. Reinstall the previous binary or pin the previous Docker image tag.
4. `systemctl start s-ui`.

Keep the `SUI_SECRETBOX_KEY` value stable across upgrade and rollback.
Losing that key can make encrypted settings unreadable.

---

## Русский

Этот релиз закрывает оставшиеся пробелы по безопасности, приватности,
realtime, Telegram, observability, фронтенду и тестам из remediation-цикла
`1.5.0`.

Можно положить новый бинарник поверх существующей установки 1.x или
восстановить старый `.db`-бэкап через панель: миграции схемы и
post-migration-адаптер выполнятся автоматически. **Данные не теряются,
ручных действий не требуется.**

### Главное

- **Асинхронный Telegram-pipeline.** Уведомления идут через bounded-очередь
  с retry/backoff и audit-событиями переполнения/неудачи; handler логина и
  другие пути не блокируются сетевыми сбоями Telegram.
- **Шифрование Telegram-секретов.** Bot-токены, креденшелы прокси,
  API-токены и ключи бэкапа шифруются at rest, маскируются в ответах
  настроек, редактируются в audit/change history и не попадают в captions.
- **Защищённый realtime WebSocket.** Origin allow-list, per-IP rate-limit на
  handshake, одноразовые токены с защитой от replay, ping/pong heartbeat,
  idle-close и close-all при ротации сессий.
- **Admin-scope для audit и Telegram-test.** `GET /api/security/audit` и
  `POST /api/telegram/test` для API-токенов требуют scope `admin`,
  ограничены rate-limit, audit-список поддерживает cursor pagination и
  валидированные фильтры `event`/`severity`.
- **Privacy-by-default для IP-истории.** Клиентские IP по умолчанию хранятся
  как salt+SHA-256 хэш, показ raw-IP - opt-in, retention обслуживается
  cron-GC, режим `enforce` отбрасывает только новые сверхлимитные
  подключения.
- **Завершение подписок.** Per-client subscription-секреты, валидируемые
  пути, санитизированные заголовки, настраиваемый per-IP rate-limit и
  404 на legacy name-URL при `subSecretRequired=true`.
- **Telegram-прокси и отчёты.** Валидируемые HTTP/HTTPS/SOCKS5-прокси,
  нормализованные классы ошибок, CPU-hysteresis алерты, scheduled отчёты и
  зашифрованный экспорт БД-бэкапа в Telegram - всё opt-in.
- **Bounded observability и логи.** Bounded buckets сэмплируются cron-job с
  валидируемыми API-параметрами; bounded `GET /api/logs` и fail-soft
  1h-cached `GET /api/version`.
- **Завершение фронтенда.** Realtime-store с reconnect/degraded-стейтами и
  polling-fallback, secret-aware-поля, IP-history modal с маской,
  Telegram-settings view и Audit view с cursor pagination и server-side
  фильтрами.
- **Надёжность миграций.** SQLite WAL checkpoint выполняется только после
  успешного commit, что исправляет `database table is locked` при
  обновлении с `1.4.x`.
- **Server-side admin-сессии.** Cookie в браузере содержит только
  подписанный session ID, данные сессии живут в локальной SQLite-таблице
  `sessions`.
- **Проверенная установка.** `install.sh` скачивает соответствующий
  `*.sha256`-артефакт и проверяет тарбол через `sha256sum -c` до
  распаковки.

### Совместимость

- Существующие subscription-URL по `name` продолжают работать, пока
  `subSecretRequired=false`.
- Legacy `/apiv2/*` `Token`-заголовок ещё работает, но ответы содержат
  заголовки `Deprecation` и `Sunset`. Sunset-дата legacy-заголовка:
  **`Sat, 15 Aug 2026 00:00:00 GMT`**. Переведите интеграции на
  `Authorization: Bearer <token>` до этой даты.
- Все новые фичи остаются off by default, кроме realtime WebSocket с
  polling-фолбэком и monitor-only IP-tracking.

### Установка / обновление

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.1-beta
```

Или из локального клона:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.1-beta
```

Не публикуйте бэкапы БД, subscription-URL, приватные ключи, сертификаты,
учётные данные администратора, API-токены, Telegram bot-токены, креденшелы
прокси и ключи Telegram-бэкапа в issues, pull requests, CI-логах и чатах
поддержки.

### Верификация

| Команда | Результат |
| --- | --- |
| `go vet ./...` | OK |
| `go test ./...` | OK |
| `npm run test:unit` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `go test -race ./...` | требуется CGO и C-компилятор |

`go test -race ./...` не запускался в этом Windows-workspace: race-детектору
Go нужен CGO, а C-компилятор отсутствует:

```text
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
```

Установите GCC (например, через MSYS2/UCRT64) и повторите:

```powershell
$env:CGO_ENABLED='1'
go test -race ./...
```

### Откат

1. `systemctl stop s-ui`.
2. Восстановите `s-ui.db` из бэкапа и соответствующие `-wal`/`-shm`-сайдкары.
3. Поставьте предыдущий бинарник или верните предыдущий тег docker-образа.
4. `systemctl start s-ui`.

Сохраняйте значение `SUI_SECRETBOX_KEY` неизменным при апгрейде и откате.
Потеря ключа сделает зашифрованные настройки нечитаемыми.
