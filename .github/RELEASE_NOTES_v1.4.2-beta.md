# S-UI v1.4.2-beta - security and reliability hardening

> Pre-release. Binary artifacts to follow.
> Supersedes [`v1.4.1`](https://github.com/deposist/s-ui-rus-inst/releases/tag/v1.4.1).
> Full changelog and upgrade guide: [CHANGELOG.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG.md).

---

## English

This release rewrites large parts of the auth, transaction, and runtime
control flow, hardens the external-subscription fetcher against SSRF, and
renames the Go module to `github.com/deposist/s-ui-rus-inst`.

You can drop the new binary on top of an existing 1.x install or restore
an older `.db` backup from the panel: schema migrations and the new
post-migration adapter run automatically. **No data loss, no manual steps.**

### Changes
- **bcrypt passwords** with lazy plaintext-to-bcrypt migration on the next successful login.
- **No more `admin/admin`** default - first-run admin password is randomly generated and printed once to the application log.
- **Login rate limiter**: 5 failures / 15 minutes / 15 minutes block per source IP, with bounded memory.
- **Hardened cookies**: `HttpOnly` + `SameSite=Lax` + HTTPS-aware `Secure`.
- **`X-Forwarded-For`** is ignored unless `SUI_TRUSTED_PROXIES` is set, and the chain is now walked right-to-left so the leftmost (spoofable) value cannot bypass IP-based logic.
- **Parameterised SQL** everywhere; the inbound user-fetch query enforces a static identifier allow-list.
- **External subscription fetcher** now refuses `localhost`/private/link-local/multicast targets, caps responses at 4 MiB, and re-validates the resolved IP at dial time, defeating DNS-rebinding attacks. Opt back in for trusted private origins with `SUI_ALLOW_PRIVATE_SUB_URLS=true`.
- **Race-clean** core lifecycle, online-stats and last-update bookkeeping, and v2 token store; `go test -race` is green.
- **Reliable saves**: configuration changes update sing-box runtime only after a successful DB commit; user-driven restarts bypass the cron cooldown so the API reflects the real start status.
- **Backup includes `services` and `tokens`** tables now.
- **Frontend**: `v-html` removed from logs/IP lists/rule-import surfaces and the gauge tile, code splitting re-enabled, deprecated `axios.CancelToken` replaced with `AbortController`, ESLint flat config, 0 `npm audit` findings.
- **HTTP servers** for the panel and subscription endpoint gain `Read/Write/Header/Idle` timeouts and `tls.MinVersion = 1.2`.

### Legacy backup auto-adapt
- `cmd/migration.MigrateDb` now returns errors instead of calling `log.Fatal`, so an incompatible import no longer kills the panel process.
- `ImportDB` rolls back to the previous database when migration fails.
- New `database.AdaptToCurrentVersion` runs after every `InitDB` and import. It rehashes legacy plaintext passwords with bcrypt, refreshes the new indexes, and bumps the `settings.version` row.
- `app.Init` runs migration **before** opening the DB, so dropping in the new binary on top of an existing 1.x database upgrades it automatically on first start.

### Install / upgrade
```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.4.2-beta
```
Or from a local clone:
```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.4.2-beta
```

After the upgrade:
- Behind a reverse proxy? Set `SUI_TRUSTED_PROXIES=...CIDRs...` so the panel sees real client IPs in audit logs.
- Pulling subscriptions from `http://127.0.0.1`? Set `SUI_ALLOW_PRIVATE_SUB_URLS=true`.

### Breaking / behaviour changes
- Go module path renamed: `github.com/alireza0/s-ui` → `github.com/deposist/s-ui-rus-inst`. Source consumers must update imports; binary users are unaffected.
- Subscription fetch over self-signed TLS no longer succeeds without a trusted CA (the implicit `InsecureSkipVerify` was removed).
- The leftmost `X-Forwarded-For` value is no longer used for client identity; configure `SUI_TRUSTED_PROXIES`.
- Five failed logins from the same IP within 15 minutes now block that IP for 15 minutes.

### Verification
| Command | Result |
| --- | --- |
| `go vet ./...` | OK |
| `go test -count=1 ./...` | OK |
| `go test -count=1 -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale" ./...` | OK |
| `go test -race -count=1 ./...` | OK (CGO + C compiler required) |
| `npm ci` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `npm audit --audit-level=high` | OK (0 vulnerabilities) |

### Rollback
1. `systemctl stop s-ui`
2. Restore `s-ui.db` from your backup (the panel **Backup → Backup** button or a copy of `/usr/local/s-ui/db/s-ui.db`).
3. Reinstall the previous binary or pin the previous Docker image tag.
4. `systemctl start s-ui`.
5. If the previous binary cannot read the bcrypt-hashed admin password (it cannot), reset it once with `s-ui admin -reset`. All other data is preserved.

---

## Русский

Этот релиз переписывает значительную часть логики аутентификации,
транзакций и запуска ядра, защищает загрузчик внешних подписок от SSRF и
переименовывает Go-модуль в `github.com/deposist/s-ui-rus-inst`.

Можно просто положить новый бинарник поверх существующей установки 1.x
или восстановить старый `.db`-бэкап через панель: миграции схемы и новый
адаптер выполнятся автоматически. **Данные не теряются, ручных действий
не требуется.**

### Главное
- **bcrypt-пароли** с автоматической миграцией plaintext → bcrypt при первом успешном входе.
- **Никаких `admin/admin`** по умолчанию - пароль администратора при первой установке генерируется случайно и однократно выводится в журнал.
- **Лимит входа**: 5 неуспешных попыток с одного IP за 15 минут блокируют IP на 15 минут, с ограниченным потреблением памяти.
- **Защищённые cookie сессии**: `HttpOnly` + `SameSite=Lax` + `Secure` для HTTPS.
- **`X-Forwarded-For`** игнорируется без переменной `SUI_TRUSTED_PROXIES`, и теперь цепочка обходится справа налево - крайнее левое (подделываемое) значение нельзя использовать для обхода IP-логики.
- **Параметризованный SQL** во всех путях; запрос пользователей по inbound теперь использует жёсткий список разрешённых типов.
- **Загрузчик подписок** отклоняет `localhost`/частные/link-local/multicast адреса, ограничивает размер ответа 4 МиБ и заново проверяет IP перед dial-ом - защита от DNS rebinding. Для своих локальных адресов используйте `SUI_ALLOW_PRIVATE_SUB_URLS=true`.
- **Race-free** жизненный цикл core, online-статистика и last-update, хранилище токенов v2; `go test -race` зелёный.
- **Надёжные сохранения**: изменения core применяются только после успешного коммита БД; пользовательские рестарты обходят cooldown крона и реальный статус старта возвращается в API.
- **Бэкап** теперь включает таблицы `services` и `tokens`.
- **Фронтенд**: убран `v-html` из логов, IP-листов, импорта правил и gauge-плитки, включён code splitting, заменён устаревший `axios.CancelToken` на `AbortController`, ESLint flat config, 0 находок в `npm audit`.
- **HTTP-серверы** панели и подписки получили таймауты `Read/Write/Header/Idle` и `tls.MinVersion = 1.2`.

### Автоадаптация старых бэкапов
- `cmd/migration.MigrateDb` возвращает ошибку вместо `log.Fatal` - несовместимый импорт больше не убивает процесс панели.
- `ImportDB` откатывает БД к предыдущей при ошибке миграции.
- Новый `database.AdaptToCurrentVersion` запускается после каждого `InitDB` и импорта: перешивает plaintext-пароли в bcrypt, обновляет индексы, поднимает строку `settings.version`.
- `app.Init` запускает миграции **до** открытия БД, поэтому новый бинарник поверх существующей базы 1.x обновляет её автоматически при первом старте.

### Установка / обновление
```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.4.2-beta
```
Или из локального клона:
```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.4.2-beta
```

После обновления:
- Если панель за reverse-proxy и важно видеть реальный IP клиента в журналах входа, выставьте `SUI_TRUSTED_PROXIES=...CIDR...`.
- Если внешние подписки забираются с `http://127.0.0.1`, выставьте `SUI_ALLOW_PRIVATE_SUB_URLS=true`.

### Что меняет поведение
- Go-модуль переименован: `github.com/alireza0/s-ui` → `github.com/deposist/s-ui-rus-inst`. Это касается только тех, кто собирает из исходников; готовые бинарники и docker-образ работают без изменений.
- Загрузка подписки с самоподписанным TLS больше не проходит без доверенного CA - неявный `InsecureSkipVerify` удалён.
- Крайнее левое значение `X-Forwarded-For` больше не используется как identity клиента; настройте `SUI_TRUSTED_PROXIES`.
- 5 неуспешных входов с одного IP за 15 минут блокируют IP на 15 минут.

### Верификация
| Команда | Результат |
| --- | --- |
| `go vet ./...` | OK |
| `go test -count=1 ./...` | OK |
| `go test -count=1 -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale" ./...` | OK |
| `go test -race -count=1 ./...` | OK (нужны CGO и C-компилятор) |
| `npm ci` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `npm audit --audit-level=high` | OK (0 уязвимостей) |

### Откат
1. `systemctl stop s-ui`
2. Восстановите `s-ui.db` из бэкапа (через **Backup → Backup** в панели или из копии `/usr/local/s-ui/db/s-ui.db`).
3. Поставьте предыдущий бинарник или верните предыдущий тег docker-образа.
4. `systemctl start s-ui`.
5. Старый бинарник не сможет прочитать bcrypt-хеш админа, поэтому однократно сделайте `s-ui admin -reset`. Все остальные данные сохранятся.
