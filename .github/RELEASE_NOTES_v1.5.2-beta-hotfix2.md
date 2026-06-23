# S-UI v1.5.2-beta-hotfix2 - drop legacy `client_ips` unique index

> Hotfix on top of `v1.5.2-beta-hotfix`. No schema additions. No
> behaviour changes in the embedded `sing-box` runtime. Drop the new
> binary on top of any 1.5.x install.
> Full changelog:
> [CHANGELOG-EN.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-EN.md)
> · [CHANGELOG-RU.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-RU.md)
> · [CHANGELOG-ZH.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-ZH.md).

---

## English

### What it fixes

- **`UNIQUE constraint failed: client_ips.client_name, client_ips.ip`
  during 3x-ui pre-import backup.** Since 1.5.x `client_ips.ip` is a
  legacy column kept only for backfill and is empty for new rows; the
  canonical unique key is `(client_name, ip_hash)`. The model still
  exposed an obsolete `gorm:"index:idx_client_ips_client_ip,unique"`
  on the `(client_name, ip)` pair, so:
  - `database/backup.go` recreated the live schema in the temporary
    backup database via `AutoMigrate`, including the legacy unique
    index, and the chunked copy of `client_ips` blew up the moment a
    single client owned more than one row with empty `ip`.
  - On hosts already on 1.5.2-beta the index lingered in the live
    `s-ui.db` because the schema migration that created it was already
    marked as applied.

  After this hotfix the unique constraint is purely
  `(client_name, ip_hash)`. Live and pre-import-backup databases share
  the same shape.

### What changed

- `database/model/model.go` - removed `index:idx_client_ips_client_ip,unique`
  from `ClientIP.ClientName` and `ClientIP.IP`. The only unique index
  left on the model is `(client_name, ip_hash)`.
- `cmd/migration/1_5.go` - schema migration now drops the obsolete
  `idx_client_ips_client_ip` and creates a partial non-unique index
  `idx_client_ips_client_legacy_ip ON client_ips(client_name, ip)
  WHERE ip IS NOT NULL AND ip != ''` to keep legacy lookup fast.
  `to1_5` is intentionally idempotent (`DROP INDEX IF EXISTS` /
  `CREATE INDEX IF NOT EXISTS`), so panels already on 1.5.2-beta
  re-run it cleanly when the runner re-evaluates the `1.5` branch
  during the next start.
- `database/db.go: ensureIndexes` - also drops the obsolete unique
  index at every `InitDB`. This is a runtime safety net for installs
  that bypass `MigrateDb` (for example, restoring an older legacy
  backup outside the panel) and means the in-memory backup database
  built by `GetDb("")` no longer carries the bad index either.
- Regression coverage:
  - `cmd/migration/migration_1_5_test.go` - fails if `to1_5` ever
    re-introduces the obsolete index, and inserts two rows with
    `ip=""` for one client to prove the legacy-IP collision is gone.
  - `database/db_test.go: TestInitDBDropsObsoleteClientIPUniqueIndex` -
    boots an old-shape DB with the legacy unique index already in place
    and verifies `InitDB` removes it.
  - `database/backup_test.go: TestGetDbHandlesHashedClientIPsWithEmptyLegacyIP` -
    backs up a DB with multiple `ip_hash` rows and empty `ip` for the
    same client and verifies `GetDb("")` round-trips them.

### Compatibility

- No schema additions. No new columns, tables, settings, endpoints,
  scopes or environment variables.
- Existing `audit_events`, `client_ips` data, `xui_known_hosts` and
  `xui_sync_profiles` rows keep their content.
- Combine with the previous hotfix's chunked-backup helpers; both fixes
  now ship together.

### Install / upgrade

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.2-beta-hotfix2
```

Or from a local clone:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.2-beta-hotfix2
```

After the new binary starts, the migration runner re-evaluates the
`1.5` branch and `to1_5` strips the legacy `idx_client_ips_client_ip`
index. `ensureIndexes` does the same at every `InitDB`, so the live
DB and the temporary backup DB end up with the same shape.

If you want a temporary workaround without rebuilding the binary,
backfill `ip = ip_hash` on rows where `ip` is empty:

```sql
UPDATE client_ips SET ip = ip_hash WHERE ip IS NULL OR ip = '';
```

Restart `s-ui` afterwards. The new binary is still the cleaner fix
because the old code rebuilds the broken index inside the temporary
backup database.

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
| `go test ./cmd/migration ./database ./database/importxui ./api ./ipmonitor` | OK |
| `go test -race ./...` | OK (CGO + C compiler required) |
| `npm run build` | OK |

### Rollback

1. `systemctl stop s-ui`.
2. Restore `s-ui.db` and any matching `-wal`/`-shm` sidecars.
3. Reinstall `v1.5.2-beta-hotfix` (or another working tag).
4. `systemctl start s-ui`.

Keep the `SUI_SECRETBOX_KEY` and (if used) `XUI_PROFILE_KEY_FILE` values
stable across upgrade and rollback.

---

## Русский

### Что чинит

- **Ошибка `UNIQUE constraint failed: client_ips.client_name,
  client_ips.ip` при автобэкапе перед миграцией с 3x-ui.** Начиная с
  1.5.x колонка `client_ips.ip` - legacy-поле только для backfill, для
  новых строк она пустая; настоящий уникальный ключ -
  `(client_name, ip_hash)`. В модели остался устаревший
  `gorm:"index:idx_client_ips_client_ip,unique"` на пару
  `(client_name, ip)`, из-за чего:
  - `database/backup.go` через `AutoMigrate` пересоздавал ту же
    схему во временной backup-БД, включая legacy-индекс, и чанковая
    копия `client_ips` падала, как только у клиента было больше одной
    строки с пустым `ip`.
  - На хостах, уже обновлённых до 1.5.2-beta, индекс жил в самой
    `s-ui.db`, потому что миграция, которая его создавала, числилась
    выполненной.

  После этого хотфикса единственный unique-индекс -
  `(client_name, ip_hash)`. Live-БД и backup-БД получают одинаковую
  схему.

### Что изменилось

- `database/model/model.go` - убран
  `index:idx_client_ips_client_ip,unique` с `ClientIP.ClientName` и
  `ClientIP.IP`. Единственный unique-индекс модели -
  `(client_name, ip_hash)`.
- `cmd/migration/1_5.go` - миграция теперь дропает устаревший
  `idx_client_ips_client_ip` и создаёт partial non-unique индекс
  `idx_client_ips_client_legacy_ip ON client_ips(client_name, ip)
  WHERE ip IS NOT NULL AND ip != ''` для быстрых legacy-lookup.
  `to1_5` идемпотентна (`DROP INDEX IF EXISTS` /
  `CREATE INDEX IF NOT EXISTS`), поэтому уже обновившиеся до
  1.5.2-beta панели чисто прогонят её повторно при следующем старте,
  когда runner снова войдёт в ветку `1.5`.
- `database/db.go: ensureIndexes` - также дропает устаревший
  unique-индекс на каждом `InitDB`. Это рантайм-страховка для случаев,
  когда `MigrateDb` обходится (например, восстановление старого
  бэкапа мимо панели), и заодно гарантирует, что временная backup-БД,
  собранная `GetDb("")`, не получит плохой индекс.
- Регресс:
  - `cmd/migration/migration_1_5_test.go` - падает, если `to1_5` снова
    создаёт устаревший индекс, и вставляет две строки с `ip=""` для
    одного клиента, чтобы проверить, что коллизии больше нет.
  - `database/db_test.go: TestInitDBDropsObsoleteClientIPUniqueIndex` -
    поднимает БД старой формы с уже созданным legacy-индексом и
    проверяет, что `InitDB` его убирает.
  - `database/backup_test.go: TestGetDbHandlesHashedClientIPsWithEmptyLegacyIP` -
    бэкапит БД с несколькими `ip_hash` и пустым `ip` для одного
    клиента и проверяет, что `GetDb("")` это переносит.

### Совместимость

- Новых колонок, таблиц, настроек, endpoint'ов, scope'ов и переменных
  окружения нет.
- Данные `audit_events`, `client_ips`, `xui_known_hosts` и
  `xui_sync_profiles` не меняются.
- Совмещается с чанковыми helper'ами из предыдущего hotfix'а - оба
  фикса теперь идут вместе.

### Установка / обновление

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/db/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/db/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/db/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui

bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.2-beta-hotfix2
```

Или из локального клона:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.2-beta-hotfix2
```

После старта нового бинарника migration runner снова войдёт в ветку
`1.5`, и `to1_5` уберёт legacy-индекс `idx_client_ips_client_ip`.
`ensureIndexes` делает то же при каждом `InitDB`, поэтому live-БД и
временная backup-БД оказываются одинаковыми.

Если нужен временный обход без пересборки бинарника, заполните
`ip = ip_hash` для строк с пустым `ip`:

```sql
UPDATE client_ips SET ip = ip_hash WHERE ip IS NULL OR ip = '';
```

После этого перезапустите `s-ui`. Правильнее всё-таки обновить
бинарник: старый код пересоздаёт сломанный индекс в временной
backup-БД при каждом снэпшоте.

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
| `go test ./cmd/migration ./database ./database/importxui ./api ./ipmonitor` | OK |
| `go test -race ./...` | OK (нужен CGO и C-компилятор) |
| `npm run build` | OK |

### Откат

1. `systemctl stop s-ui`.
2. Восстановите `s-ui.db` из бэкапа и соответствующие `-wal`/`-shm`-сайдкары.
3. Поставьте `v1.5.2-beta-hotfix` (или другой рабочий тег).
4. `systemctl start s-ui`.

Сохраняйте `SUI_SECRETBOX_KEY` и (если используется) `XUI_PROFILE_KEY_FILE`
неизменными при апгрейде и откате.
