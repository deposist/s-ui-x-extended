# S-UI v1.5.5-beta4

## English

This prerelease is a broad hardening release for migration, audit, backups,
runtime stability, and realtime recovery. The notes below group the fixes by
area and describe both the user-visible issue and the effect of the fix.

### 1. Security, authentication, and audit

* **Forced password reset during import**
  * **Problem:** the UI exposed the `reset_required` mode for x-ui
    administrator migration, but the backend did not have durable state for a
    mandatory password change and could fall back to generating a new password.
  * **Effect:** imported administrators can now be marked with
    `force_password_reset`, the API contract matches the UI, and users imported
    with `reset_required` must change their password before normal panel use.
    No temporary password is generated or written to the import report for this
    mode.
* **Token hardening**
  * **Problem:** WebSocket token checks had measurable timing differences, the
    legacy `Token` authorization header had no enforced sunset, and migration
    of legacy API tokens could re-enable tokens that had been disabled.
  * **Effect:** WebSocket token consumption now uses a safer match-and-delete
    path, the legacy `Token` header is rejected after its sunset, and legacy
    token migration preserves the original enabled/disabled state.
* **Reducing sensitive data exposure**
  * **Problem:** system information could expose private or link-local server
    addresses, Telegram backup secrets needed clearer memory ownership, and
    generated MigrateXui admin passwords were too visible on screen.
  * **Effect:** private addresses are filtered from system info, Telegram
    backup payloads and passphrases are wiped after use, and generated admin
    passwords stay hidden until explicit reveal and are cleared automatically.
* **Audit quality and prioritization**
  * **Problem:** warning/security events could be pushed out of a
    full audit queue by ordinary `info` events, successful legacy decrypt
    fallback created noise, stats commit failures lacked a complete audit trail,
    and optional URL settings accepted unsafe control characters.
  * **Effect:** warning/security audit events keep priority, secretbox fallback
    noise is removed, stats commit failures are audited, and optional URL
    settings reject control characters and unsafe input forms.

### 2. X-UI import, sync, and frontend recovery

* **User import policy is honored**
  * **Problem:** background X-UI cron sync used hard-coded behavior and could
    ignore stored profile fields such as `OnlyNew`, settings import, history,
    routing, and administrator handling mode.
  * **Effect:** the scheduler now passes the saved import policy into planning
    and apply steps, so cron sync follows the administrator's selected options.
* **Large imports are safer**
  * **Problem:** migration plans were read as ordinary multipart fields with an
    8 MiB limit, and aborted uploads could leave temporary directories behind.
  * **Effect:** the `plan` multipart field is streamed through temporary
    storage under the 200 MiB request limit, reducing memory pressure and
    supporting larger plans. Old `xui-import-*` temporary directories are
    cleaned up by age.
* **Import isolation and reporting**
  * **Problem:** failed TLS cleanup in replace mode could be ignored before new
    rows were created, and skipped WireGuard endpoints were counted as skipped
    inbounds.
  * **Effect:** TLS delete failures now abort the transaction and roll back
    safely, and the import report has a separate skipped endpoints counter.
* **Apply/rollback UX**
  * **Problem:** apply errors could send the user back to the previous step
    without a clear explanation, rollback used a fixed one-second wait before
    reload, and other sessions did not receive realtime invalidation.
  * **Effect:** MigrateXui shows apply errors inline, rollback waits for a
    health check before reload, and the backend publishes `config_invalidated`
    after successful rollback.

### 3. Database, backups, and fault tolerance

* **Backup and database migration safety**
  * **Problem:** SIGHUP timeout was fixed at three seconds, WAL checkpoint could
    fatally break backup on a locked SQLite database, missing `settings.config`
    blocked versioned restore entirely, and post-migration adapt errors during
    startup were warning-only.
  * **Effect:** SIGHUP timeout is configurable, WAL checkpoint falls back from
    `TRUNCATE` to `FULL`, backups without `settings.config` can restore with a
    warning, and broken post-migration adapt now stops startup instead of
    continuing with a potentially inconsistent schema.
* **Database scalability and startup races**
  * **Problem:** SQLite pool limits were fixed, and parallel first-start could
    create duplicate default settings.
  * **Effect:** SQLite pool limits are configurable through environment
    variables, and default settings are inserted through a DB-level idempotent
    path.
* **IP monitor fail-closed behavior**
  * **Problem:** a transient DB read error in the IP monitor path could allow an
    unknown address through in enforce mode.
  * **Effect:** `client_ips` cache entries are treated as unreliable after read
    errors, and enforcement fails closed.

### 4. Runtime stability, races, and networking

* **OOM protection and realtime self-healing**
  * **Problem:** import-xui rate-limit state could grow without a bound under a
    stream of unique IPs, and the frontend could remain in degraded polling mode
    forever after a network failure.
  * **Effect:** the rate-limit cache now evicts expired buckets within bounds,
    and the WebSocket runtime keeps healing reconnect attempts alive from
    fallback mode.
* **Data race fixes**
  * **Problem:** concurrent access to core restart timers, Telegram HTTP
    client state, and token-use flush lifecycle could trip the race detector,
    panic, or write through a stale DB handle.
  * **Effect:** critical paths are protected with mutex, single-flight, and
    lifecycle barrier mechanisms, and token-use flush is synchronized with DB
    reset and API test lifecycle.
* **Backoff, storm protection, and update checks**
  * **Problem:** cron sync used short fixed retries, token-use write failures
    had no backoff circuit, update checks hit GitHub without ETag, sync failure
    summaries lost useful error details, and WARP authorization headers were
    scattered through the code.
  * **Effect:** retry policies now use exponential backoff, token-use flush has
    a circuit breaker, release checks send `If-None-Match`, sync failure
    summaries include sanitized error class/detail, and WARP authorized headers
    are centralized.
* **IPv6 handling and shared API route registry**
  * **Problem:** system info could panic on short interface flag/address data in
    uncommon IPv6-only environments, and import-xui routes diverged between v1
    and v2 APIs.
  * **Effect:** network interface data is validated by length and contents, and
    import-xui endpoints are registered from a shared route spec for `/api` and
    `/apiv2`.

### Validation

* `go build ./...` - PASS
* `go vet ./...` - PASS
* `go test ./...` - PASS
* `go test -race ./... -timeout 900s` - PASS
* `govulncheck ./...` - PASS, no vulnerabilities found
* `gosec ./...` - known classified baseline, 55 issues

### Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta4
```

## Русский

Ниже представлен самодостаточный обзор исправлений, вошедших в `v1.5.5-beta4`.
Изменения сгруппированы по логическим блокам: для каждой группы описана суть
закрытых проблем и влияние исправлений на работу панели.

---

## 1. Безопасность, аутентификация и аудит

* **Принудительный сброс пароля при импорте**
  * **Суть проблемы:** интерфейс предлагал режим `reset_required` при миграции
    администраторов из x-ui, но backend не имел отдельного durable-состояния
    для обязательной смены пароля и фактически уходил в сценарий генерации
    нового пароля.
  * **Влияние:** в модель пользователя добавлено состояние
    `force_password_reset`, API-контракт синхронизирован с интерфейсом, а
    импортированные администраторы с `reset_required` должны сменить пароль
    перед нормальной работой в панели. Временный пароль больше не генерируется
    и не попадает в отчёт импорта для этого режима.
* **Защита токенов от атак и устаревания**
  * **Суть проблемы:** WebSocket-токены проверялись с измеримой разницей во
    времени, устаревший заголовок авторизации `Token` не имел жёсткой даты
    отключения, а миграция legacy API-токенов могла включить ранее отключённые
    токены.
  * **Влияние:** потребление WebSocket-токенов переведено на безопасный
    match-and-delete путь, legacy `Token` header получает отказ после Sunset,
    а миграция старых токенов сохраняет их исходный enabled/disabled статус.
* **Защита от утечек системных данных**
  * **Суть проблемы:** системная информация могла раскрывать private и
    link-local IP-адреса сервера, секреты Telegram backup требовали более
    явного владения памятью, а сгенерированные admin-пароли в MigrateXui были
    слишком легко видимы на экране.
  * **Влияние:** внутренние IP фильтруются из ответа system info, payload и
    passphrase Telegram backup зануляются после использования, а generated
    admin passwords скрыты до явного reveal и автоматически очищаются.
* **Приоритезация и качество аудита**
  * **Суть проблемы:** при переполнении audit queue важные warn/security
    события могли вытесняться обычными `info`, успешная legacy-расшифровка
    создавала лишний audit noise, ошибки сохранения статистики не оставляли
    полноценного следа, а URL-настройки принимали опасные управляющие символы.
  * **Влияние:** audit writer теперь сохраняет приоритет warning/security
    событий, лишний secretbox fallback noise убран, failures при commit
    статистики фиксируются в audit, а optional URL settings отклоняют control
    characters и небезопасные формы ввода.

---

## 2. Импорт, синхронизация X-UI и интерфейс

* **Точное следование настройкам пользователя**
  * **Суть проблемы:** фоновая cron-синхронизация с X-UI использовала
    жёстко заданные правила и могла игнорировать пользовательские поля профиля:
    `OnlyNew`, импорт настроек, истории, routing и режим обработки
    администраторов.
  * **Влияние:** scheduler backend теперь передаёт сохранённую import policy в
    планирование и применение импорта, поэтому cron sync исполняет настройки,
    выбранные администратором.
* **Работа с крупными импортами**
  * **Суть проблемы:** JSON-план миграции читался как обычное multipart-поле с
    лимитом 8 MiB, из-за чего крупные панели нельзя было применить тем же
    контрактом. Оборванные загрузки также могли оставлять временные директории
    на диске.
  * **Влияние:** multipart-поле `plan` теперь читается потоково из временного
    хранилища под общим лимитом запроса 200 MiB, что снижает memory pressure и
    позволяет применять большие планы. Старые `xui-import-*` temp directories
    очищаются автоматически по безопасному возрастному правилу.
* **Изоляция и точность импорта**
  * **Суть проблемы:** ошибка удаления старых TLS-записей в replace-сценарии
    могла быть проигнорирована перед созданием новых записей, а пропущенные
    WireGuard endpoints попадали в счётчик skipped inbounds.
  * **Влияние:** ошибки удаления TLS теперь прерывают транзакцию и приводят к
    безопасному rollback, а отчёт импорта получил корректный отдельный счётчик
    skipped endpoints.
* **UX восстановления и откатов**
  * **Суть проблемы:** ошибки apply могли возвращать пользователя на прошлый
    шаг без понятного объяснения, rollback ждал фиксированную секунду перед
    reload, а другие активные сессии не получали realtime-сигнал об изменении
    конфигурации.
  * **Влияние:** MigrateXui показывает apply error inline, rollback ждёт
    подтверждения health-check перед reload, а backend публикует
    `config_invalidated` после успешного rollback.

---

## 3. База данных, резервное копирование и отказоустойчивость

* **Защита backup и процесса миграции БД**
  * **Суть проблемы:** SIGHUP timeout был жёстко зафиксирован на 3 секундах,
    WAL checkpoint мог фатально сорвать backup на заблокированной SQLite DB,
    отсутствие `settings.config` полностью блокировало versioned restore, а
    ошибки post-migration adapt при запуске оставались warning-only.
  * **Влияние:** timeout вынесен в env-настройку, WAL checkpoint получил
    fallback `TRUNCATE -> FULL`, backup без `settings.config` восстанавливается
    с предупреждением, а повреждённый post-migration adapt теперь останавливает
    startup вместо продолжения с потенциально неконсистентной схемой.
* **Масштабируемость БД и гонки при старте**
  * **Суть проблемы:** SQLite pool имел фиксированные лимиты, а параллельный
    first-start мог создать дубликаты настроек по умолчанию.
  * **Влияние:** добавлены env-переменные для настройки SQLite pool, а default
    settings создаются через DB-level idempotent insert path, исключающий
    дубликаты при конкурентном старте.
* **Отказоустойчивость IP-монитора**
  * **Суть проблемы:** transient DB read error в IP-monitor path мог привести к
    пропуску неизвестного адреса в enforce-mode.
  * **Влияние:** при ошибке чтения `client_ips` cache entry считается
    недостоверным, и enforcement переходит в fail-closed поведение.

---

## 4. Сеть, гонки данных и стабильность ядра

* **Защита от OOM и самовосстановление realtime**
  * **Суть проблемы:** import-xui rate-limit state мог расти без верхней
    границы при потоке запросов с уникальных IP, а после сетевого сбоя frontend
    мог навсегда остаться в degraded polling mode.
  * **Влияние:** rate-limit cache получил bounded eviction и очистку expired
    buckets, а WebSocket runtime теперь выполняет healing reconnect attempts из
    fallback-режима.
* **Устранение data races**
  * **Суть проблемы:** конкурентный доступ к таймерам перезапуска ядра,
    Telegram HTTP client и token-use flush мог приводить к race detector
    failures, паникам или записи через устаревший DB handle.
  * **Влияние:** критичные участки защищены mutex/single-flight/barrier
    механизмами, а token-use flush lifecycle синхронизирован с DB reset и API
    test lifecycle.
* **Умные повторы и защита от штормов**
  * **Суть проблемы:** cron sync использовал слишком короткие fixed retries,
    token-use write failures не имели backoff circuit, update-check ходил на
    GitHub без ETag, причины ошибок sync терялись, а WARP auth headers были
    хрупко распределены по коду.
  * **Влияние:** retry-политики получили exponential backoff, token-use flush
    получил circuit breaker, release checks используют `If-None-Match`,
    sync-fail summaries включают sanitized error class/detail, а WARP
    authorized headers централизованы.
* **Работа с IPv6 и единый API route registry**
  * **Суть проблемы:** system info path мог паниковать на коротких interface
    flags/address данных, включая нестандартные IPv6-only окружения, а
    import-xui routes расходились между v1 и v2 API.
  * **Влияние:** сетевые интерфейсы проверяются по содержимому и длине данных,
    а import-xui endpoints регистрируются из общего route spec для `/api` и
    `/apiv2`.

---

## Validation

* `go build ./...` - PASS
* `go vet ./...` - PASS
* `go test ./...` - PASS
* `go test -race ./... -timeout 900s` - PASS
* `govulncheck ./...` - PASS, no vulnerabilities found
* `gosec ./...` - known classified baseline, 55 issues

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta4
```
