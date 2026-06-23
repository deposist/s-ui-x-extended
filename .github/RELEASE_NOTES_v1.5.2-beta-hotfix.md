# S-UI v1.5.2-beta-hotfix - backup chunking and SPA upgrade safety

> Hotfix on top of `v1.5.2-beta`. No schema changes, no behaviour changes
> in the embedded `sing-box` runtime. Drop the new binary on top of an
> existing 1.5.2-beta install.
> Full changelog:
> [CHANGELOG-EN.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-EN.md)
> · [CHANGELOG-RU.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-RU.md)
> · [CHANGELOG-ZH.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-ZH.md).

---

## English

### What it fixes

- **`too many SQL variables` during database backup and 3x-ui migration.**
  On installs with large `stats`, `client_ips`, `audit_events`, `changes`
  or `clients` tables (≥ ~40k rows in `stats` reproduces it), the backup
  routine in `database/backup.go` emitted a single multi-row
  `INSERT VALUES (...)` that exceeded SQLite's compile-time variable
  limit (`SQLITE_MAX_VARIABLE_NUMBER = 999` in `mattn/go-sqlite3`). This
  also blocked the 3x-ui migration because `WritePreImportBackup` ran the
  same code path before opening the import transaction.
- **Stale `index.html` after upgrade left the Clients tab broken.**
  Browser tabs that still held the previous `index.html` requested old
  hashed chunks like `assets/_5nyNEw12.js`. The Gin static handler
  fell through to the SPA fallback (`index.html`) for those missing
  chunks, so the browser received `text/html` for a JS module request and
  failed with `Failed to load module script` / `Failed to fetch
  dynamically imported module`. The Clients tab is the first dynamic
  import most users hit, so it became the visible symptom.

### What changed

- `database/bulk.go` - new `SafeSQLiteBatchSize`, `CreateInBatchesSafe`
  and `SaveInBatchesSafe` helpers. They parse the GORM model schema to
  count columns and pick a batch size that keeps each generated INSERT
  under the SQLite variable budget (800 placeholders, conservative against
  the 999 hard limit).
- `database/backup.go: copyBackupTable` - now reads the source in pages
  via `FindInBatches` and writes into the backup database with chunked
  `CreateInBatches`, all inside a single transaction. Memory stays bounded
  for arbitrarily large `stats` / `client_ips` tables.
- `database/importxui/history_routing.go` - historical traffic import
  uses the chunked helper too, since `client_traffics`/`outbound_traffics`
  often produce tens of thousands of `stats` rows on production installs.
- `service/client.go` - `addbulk`, `editbulk`, `ResetClients`,
  `DepleteClients` now chunk their bulk `Save`/`Create` calls. Reset and
  deplete jobs no longer fail on installs with thousands of clients.
- `web/web.go` + `web/assets.go` - `/<base>/assets/*` is served by a
  custom handler that returns a real 404 for missing files instead of
  falling through to the SPA fallback. Hashed assets keep
  `Cache-Control: public, max-age=31536000, immutable`. `index.html` is
  served with `Cache-Control: no-cache, no-store, must-revalidate` so an
  upgrade is picked up on the next document load.
- `frontend/src/router/index.ts` - listens for `vite:preloadError` and
  `router.onError`. When a dynamic import fails because of a stale chunk
  hash, the router triggers a single guarded `window.location.reload()`
  (a `sessionStorage` flag prevents reload loops).
- `database/backup_test.go` - regression test creates ~43k `stats` rows
  plus 5k `client_ips` and verifies `GetDb("")` succeeds and round-trips
  the row count.

### Compatibility

- No schema migrations. No new columns, tables, or settings.
- No new endpoints, scopes or environment variables.
- Existing `/api/getdb`, `/apiv2/getdb`, `/api/importdb`, `/api/import-xui*`,
  Telegram backup and the in-app database export keep their request and
  response shapes.

### Install / upgrade

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.2-beta-hotfix
```

Or from a local clone:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.2-beta-hotfix
```

After the new binary starts, browser tabs left over from `v1.5.2-beta`
will reload automatically on the next navigation thanks to the
`vite:preloadError` handler. No manual `Ctrl+F5` is required.

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
| `go test ./...` | OK |
| `go test -race ./...` | OK (CGO + C compiler required) |
| `npm run build` | OK |

### Rollback

1. `systemctl stop s-ui`.
2. Restore the backed-up `s-ui.db` and any matching `-wal`/`-shm` sidecars.
3. Reinstall `v1.5.2-beta` (or the last working tag).
4. `systemctl start s-ui`.

Keep the `SUI_SECRETBOX_KEY` and (if used) `XUI_PROFILE_KEY_FILE` values
stable across upgrade and rollback.

---

## Русский

### Что чинит

- **Ошибка `too many SQL variables` при бэкапе БД и миграции с 3x-ui.**
  На инсталляциях с большими таблицами `stats`, `client_ips`,
  `audit_events`, `changes` или `clients` (воспроизводится начиная
  примерно с 40k строк в `stats`) функция бэкапа в `database/backup.go`
  генерировала один многострочный `INSERT VALUES (...)`, который
  превышал compile-time лимит SQLite
  (`SQLITE_MAX_VARIABLE_NUMBER = 999` в `mattn/go-sqlite3`). Это же
  блокировало миграцию из 3x-ui, так как `WritePreImportBackup`
  вызывал ту же функцию до открытия транзакции импорта.
- **Протухший `index.html` после апгрейда ломал вкладку Clients.**
  Вкладки браузера, в которых остался старый `index.html`, запрашивали
  старые хэшированные чанки вида `assets/_5nyNEw12.js`. Static-хэндлер
  Gin для отсутствующих файлов отдавал SPA-fallback (`index.html`), и
  браузер получал `text/html` на запрос JS-модуля - отсюда
  `Failed to load module script` / `Failed to fetch dynamically
  imported module`. `Clients` - первая страница с динамическим импортом
  на пути большинства пользователей, поэтому она и ломалась видимым
  образом.

### Что изменилось

- `database/bulk.go` - новые helper'ы `SafeSQLiteBatchSize`,
  `CreateInBatchesSafe`, `SaveInBatchesSafe`. Они разбирают схему
  GORM-модели, считают количество колонок и подбирают размер батча,
  при котором каждый INSERT укладывается в бюджет переменных SQLite
  (800 плейсхолдеров - консервативный запас от лимита 999).
- `database/backup.go: copyBackupTable` - теперь читает источник
  страницами через `FindInBatches` и пишет в бэкап-БД чанками через
  `CreateInBatches`, всё внутри одной транзакции. Память не растёт
  на любых размерах `stats` / `client_ips`.
- `database/importxui/history_routing.go` - импорт исторического
  трафика тоже использует чанковый helper: на проде
  `client_traffics`/`outbound_traffics` часто дают десятки тысяч строк
  `stats`.
- `service/client.go` - `addbulk`, `editbulk`, `ResetClients`,
  `DepleteClients` теперь нарезают bulk-`Save`/`Create` на чанки.
  Reset/deplete-задачи больше не падают на инсталлах с тысячами
  клиентов.
- `web/web.go` + `web/assets.go` - `/<base>/assets/*` обслуживается
  кастомным хэндлером, который для отсутствующего файла возвращает
  честный 404, а не SPA-fallback. Хэшированные ассеты остаются с
  `Cache-Control: public, max-age=31536000, immutable`. `index.html`
  отдаётся с `Cache-Control: no-cache, no-store, must-revalidate`, и
  апгрейд подхватывается со следующим запросом документа.
- `frontend/src/router/index.ts` - слушает `vite:preloadError` и
  `router.onError`. Если динамический импорт упал из-за устаревшего
  хэша чанка, роутер делает один защищённый `window.location.reload()`
  (флаг в `sessionStorage` исключает петлю перезагрузок).
- `database/backup_test.go` - регресс создаёт ~43k строк `stats` плюс
  5k `client_ips`, проверяет, что `GetDb("")` отрабатывает и количество
  строк сохраняется.

### Совместимость

- Миграции схемы нет. Новых колонок, таблиц, настроек нет.
- Новых endpoint'ов, scope'ов, переменных окружения нет.
- Существующие `/api/getdb`, `/apiv2/getdb`, `/api/importdb`,
  `/api/import-xui*`, Telegram-бэкап и экспорт БД из UI сохраняют
  формат запросов и ответов.

### Установка / обновление

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.2-beta-hotfix
```

Или из локального клона:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.2-beta-hotfix
```

После старта нового бинарника браузерные вкладки, оставшиеся от
`v1.5.2-beta`, перезагрузятся сами при следующей навигации благодаря
обработчику `vite:preloadError`. Ручной `Ctrl+F5` не требуется.

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
| `go test ./...` | OK |
| `go test -race ./...` | OK (нужен CGO и C-компилятор) |
| `npm run build` | OK |

### Откат

1. `systemctl stop s-ui`.
2. Восстановите `s-ui.db` из бэкапа и соответствующие `-wal`/`-shm`-сайдкары.
3. Поставьте `v1.5.2-beta` (или последний рабочий тег).
4. `systemctl start s-ui`.

Сохраняйте `SUI_SECRETBOX_KEY` и (если используется) `XUI_PROFILE_KEY_FILE`
неизменными при апгрейде и откате.
