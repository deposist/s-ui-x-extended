# Release Notes: v1.5.8-beta1

Release date: 2026-06-11

This beta extends no-restart apply from clients and TLS to **every object the
panel manages**: saving inbounds, outbounds, endpoints, and services now
hot-replaces the affected object inside the running sing-box core instead of
restarting it. Active connections on unrelated objects survive every save. It
is a backend-only release - no manual migration and no configuration changes.

One intentional behavior change: deleting or renaming an outbound, endpoint, or
managed inbound that is still referenced elsewhere is now rejected with an
error that lists every referencing site. Previously the save was accepted and
the next core start failed on the dangling reference, taking the whole proxy
down.

## What changed

### Hot reload for inbounds, outbounds, endpoints, and services

- **Saving an inbound, outbound, endpoint, or service no longer restarts the
  core.** The changed object is removed and re-added inside the running core -
  the same mechanism client and TLS edits have used since v1.5.7. Renames
  remove the old tag first; deletes close the object's tracked connections.
  A failed hot apply still falls back to a full core restart, so the core
  never keeps serving a stale configuration.
- **Conservative escalation for captured references.** sing-box captures some
  adapter references at construction time; replacing the target under such a
  reference would leave the dependent holding a closed adapter. When the saved
  outbound or endpoint tag is referenced by another outbound's `detour`, a
  `selector`/`urltest` member list or `default`, a service dial detour,
  `dns`/`ntp` detours, a rule-set `download_detour`, or the Clash API UI
  download detour, the panel applies the change via a full restart instead.
  References from route rules and `route.final` are resolved per connection,
  so those edits stay hot - including the common case of editing the proxy
  outbound your routing points at.
- **ssm-api cascade.** Editing a managed-shadowsocks inbound also recreates
  the `ssm-api` service bound to it, so the service keeps tracking the fresh
  inbound adapter.

### Referenced-tag delete and rename guard

- Deleting or renaming an outbound, endpoint, or managed inbound whose tag is
  still referenced anywhere - including lazily-resolved route rules and
  `route.final` - is now blocked. The error enumerates every referencing site,
  for example: `outbound "proxy-a" is still referenced by: route rule #3
  (outbound), selector "auto" (outbounds list) - remove the reference or point
  it to another outbound (e.g. direct) first`.

### Fixes and hardening

- **No-op config saves no longer restart the core.** Re-saving the sing-box
  settings blob without changes used to drop every active connection; the
  panel now compares against the stored blob and skips the restart when
  nothing changed (the audit trail still records the save).
- **Core synchronization is serialized.** The post-save apply, user-initiated
  restarts, and the cron core starter now run under one exclusive section: a
  save can no longer interleave with a restart and silently leave the running
  core out of sync with the database.
- **Fixed a latent data race** in the panel-restart scheduler (the restart
  timer callback read a variable the scheduler wrote without
  synchronization). Found by the race detector through the new tests; the
  race predates this release.

## Verification

- `go build ./...`, `go vet ./...`
- Full `go test ./service` suite, including ~30 new regression tests: per-type
  hot reload (stub-based and against a real in-process sing-box core),
  rename/delete guards with exact error-message assertions, the ssm-api
  cascade order, reference-scanner fixtures for every reference site, apply
  serialization under a concurrent restart, and fallback-to-restart on a
  failed hot apply
- Full `go test ./... -race -count=1`: all 23 packages pass with zero data
  races

No frontend sources changed: guard errors surface through the existing save
error toast, and hot saves return the same partial-reload payload as before.

---

# Примечания к релизу: v1.5.8-beta1

Дата релиза: 2026-06-11

Эта бета расширяет применение без рестарта с клиентов и TLS на **все объекты,
которыми управляет панель**: сохранение Inbounds, Outbounds, endpoint'ов и
сервисов теперь горячо заменяет изменённый объект в работающем ядре sing-box
вместо его перезапуска. Активные соединения через незатронутые объекты
переживают каждое сохранение. Релиз затрагивает только backend - ручных
миграций и изменений конфигурации нет.

Одно намеренное изменение поведения: удаление или переименование outbound'а,
endpoint'а или managed inbound'а, на который ещё есть ссылки, теперь
отклоняется ошибкой с перечислением всех ссылающихся мест. Раньше сохранение
проходило, а следующий старт ядра падал на повисшей ссылке, роняя весь прокси.

## Что изменилось

### Горячая перезагрузка Inbounds, Outbounds, endpoint'ов и сервисов

- **Сохранение inbound'а, outbound'а, endpoint'а или сервиса больше не
  перезапускает ядро.** Изменённый объект удаляется и добавляется заново в
  работающем ядре - тем же механизмом, которым с v1.5.7 применяются правки
  клиентов и TLS. При переименовании сначала удаляется старый тег; при
  удалении закрываются отслеживаемые соединения объекта. Неудачное горячее
  применение по-прежнему откатывается на полный рестарт - ядро никогда не
  обслуживает устаревшую конфигурацию.
- **Консервативная эскалация для захваченных ссылок.** Часть ссылок sing-box
  захватывает при создании адаптера; горячая замена цели такой ссылки оставила
  бы у зависимого объекта закрытый адаптер. Если на тег сохраняемого
  outbound'а/endpoint'а ссылаются `detour` другого outbound'а, список
  участников или `default` у `selector`/`urltest`, detour сервиса, detour в
  `dns`/`ntp`, `download_detour` rule-set'а или detour загрузки Clash UI -
  панель применяет изменение через полный рестарт. Ссылки из route-правил и
  `route.final` резолвятся на каждое соединение, поэтому такие правки остаются
  горячими - включая типовой случай правки прокси-outbound'а, на который
  указывает маршрутизация.
- **Каскад ssm-api.** Правка managed-shadowsocks inbound'а пересоздаёт и
  привязанный к нему сервис `ssm-api`, чтобы тот продолжал отслеживать свежий
  адаптер inbound'а.

### Защита от удаления и переименования тега со ссылками

- Удаление или переименование outbound'а, endpoint'а или managed inbound'а,
  на чей тег ещё есть ссылки - включая лениво резолвящиеся route-правила и
  `route.final` - теперь блокируется. Ошибка перечисляет все ссылающиеся
  места, например: `outbound "proxy-a" is still referenced by: route rule #3
  (outbound), selector "auto" (outbounds list) - remove the reference or point
  it to another outbound (e.g. direct) first`.

### Исправления и усиление надёжности

- **Сохранение конфига без изменений больше не перезапускает ядро.** Повторное
  сохранение блока настроек sing-box без правок раньше рвало все активные
  соединения; теперь панель сравнивает с сохранённым блоком и пропускает
  рестарт, когда ничего не изменилось (запись в аудит при этом остаётся).
- **Синхронизация ядра сериализована.** Применение после сохранения,
  пользовательский рестарт и cron-запуск ядра теперь выполняются в одной
  эксклюзивной секции: сохранение не может переплестись с рестартом и молча
  оставить работающее ядро рассинхронизированным с базой.
- **Исправлена латентная гонка данных** в планировщике рестарта панели
  (callback таймера читал переменную, которую планировщик писал без
  синхронизации). Найдена race-детектором через новые тесты; гонка
  существовала и до этого релиза.

## Проверка

- `go build ./...`, `go vet ./...`
- Полный набор `go test ./service`, включая ~30 новых регрессионных тестов:
  горячая перезагрузка каждого типа (на стабах и против настоящего in-process
  ядра sing-box), защита del/rename с точными проверками текста ошибок,
  порядок каскада ssm-api, фикстуры сканера для каждого вида ссылок,
  сериализация применения при конкурентном рестарте и откат на рестарт при
  неудачном горячем применении
- Полный `go test ./... -race -count=1`: все 23 пакета проходят без единой
  гонки данных

Исходники фронтенда не менялись: ошибки защиты показываются существующим
тостом ошибки сохранения, а горячие сохранения возвращают тот же ответ
частичной перезагрузки, что и раньше.
