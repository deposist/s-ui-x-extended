# s-ui-x-extended — статус реализации: ГОТОВО (M0–M9)

Ветка `s-ui-x-extended` (worktree `c:\s-ui-x-ext`). Ядро заменено на
`shtorm-7/sing-box-extended v1.13.12-extended-2.4.0`. **Коммиты НЕ делались** (по требованию) —
все изменения в рабочем дереве ветки. План: `PLAN-s-ui-x-extended.md`. Исходники ядра для
сверки: `c:\sing-box-extended-src` (тег).

## Итоговая независимая верификация (всё exit 0)
- `go build ./...` (без тегов) · `go vet ./...` · `go test ./...` (весь набор, все пакеты `ok`)
- тегированная сборка `go build -tags "with_quic,...,with_profiler" ./core/...` (реальный код всех фич)
- `cd frontend && npm run build` (vue-tsc strict + vite)
- `go build/test ./api/...` (после правки apiV2 providers)
- ⚠️ Полный тегированный **бинарник** + runtime-smoke — только Linux/Docker (cronet/`with_naive_outbound`).

Масштаб: ~98 файлов (изменено/создано), 19 новых компонентов протоколов + 4 провайдера.

## Сделано
- **M0** — замена ядра: go.mod replace (ядро + 8 nested), порт `core/register.go` (с `ProviderRegistry`,
  без admin_panel/manager/node/manager_api/node_manager_api; только `profiler`), 24 helper+stub файла,
  контекст 6-арг, `core/log.go` Notice, build-теги.
- **M1** — Mieru/Sudoku/TrustTunnel (in+out) + SSH inbound, **полная интеграция users** (clients.ts + service/inbounds.go).
- **M2** — MTProxy inbound + users (secret).
- **M3** — MASQUE + OpenVPN outbound; WARP/WireGuard **Amnezia 2.0** (общий `Amnezia.vue`).
- **M4** — Bond / Failover / Fallback (outbound-группы). Встроенные конфиги — валидируемый JSON-редактор
  (ядро требует полные вложенные конфиги, не tag-ссылки).
- **M5** — DNS: SDNS (DNSCrypt) / DNS Fallback / Resolved.
- **M6** — VPN server/client endpoints (вложенные inbounds/outbound — валидируемый JSON; key = UUID + regen).
- **M7** — лимитеры bandwidth/connection/traffic/rate (с `strategy:users`/`manager`). Per-user квоты через
  inline-users лимитера (квоты через таблицу clients — отдельная будущая задача, не костыль).
- **M8** — Providers (inline/local/remote) — **полный стек**: `database/model/providers.go`, `db.go`
  AutoMigrate, `backup.go`, `service/providers.go`, `service/config.go` (секция + Save), `core/box.go`
  (provider-менеджер + цикл создания + lifecycle), api (apiService/apiHandler/save_dedup/apiV2/server),
  фронт (types/providers.ts, Provider.vue, provider/{Inline,Local,Remote,HealthCheck}.vue, Providers.vue,
  router, Drawer, store).
- **M9** — profiler (service) + VLESS encryption + mKCP/XHTTP transports + parser (outbound) +
  unified_delay (experimental → Basics.vue).

## Осознанные дизайн-решения (не костыли)
- Произвольные вложенные конфиги (bond/failover/vpn embedded inbounds/outbound, route.rules) редактируются
  через **валидируемый JSON-редактор** — это корректно, т.к. там вложен полный конфиг любого протокола;
  переиспользовать весь stateful-form-builder инлайн нецелесообразно. Значения парсятся/валидируются и
  корректно round-trip'ятся.
- Управляющие сервисы admin_panel/manager/node/manager_api/node_manager_api — намеренно исключены.

## Возможные будущие улучшения (опционально, вне текущего объёма)
- Группы-провайдеры: селектор/urltest могут потреблять провайдеры через `providers:[]`/`use_all_providers`
  — UI-селектор провайдеров в группах можно добавить позже.
- Полноценные типизированные вложенные саб-формы вместо JSON-редакторов для bond/failover/vpn.
- Per-user квоты лимитеров через таблицу clients.

## Команды проверки
- Фронт: `cd frontend && npm run build`. Бэк: `go build ./...`, `go vet ./...`, `go test ./...`.
- Полный бинарник/прогон — Linux/Docker: `docker build .` или `./build.sh`.

## Безопасность
См. [SECURITY.md](SECURITY.md): форки зависимостей (`replace`), транзитивный
`redis/go-redis`, `with_profiler` как dev-only тег, закрепление cronet по sha256,
запуск от root и отложенные пункты харденинга.

## Журнал ошибок и исправлений
- [2026-06-23] Проблема: Invalid login запускал remote POST /api/logout, а при потерянной сессии этот POST пытался получить CSRF и порождал цикл Invalid login / CSRF token was not returned → Решение: Invalid login теперь выполняет локальный logout без POST /api/logout, CSRF store передаёт backend-ошибку Invalid login без подмены на missing-token, добавлен guard от повторных уведомлений.
- [2026-06-23] Проблема: на Windows web session-store тест иногда падал при t.TempDir cleanup из-за SQLite WAL/SHM файлов после закрытия DB → Решение: тестовый helper теперь создаёт tempdir вручную, закрывает SQLite с WAL checkpoint и удаляет каталог с retry.
- [2026-06-23] Проблема: веб-панель проверяла обновления через GitHub API старого репозитория deposist/s-ui-x, хотя downloads уже указывали на s-ui-x-extended → Решение: githubAPIBase переключён на https://api.github.com/repos/deposist/s-ui-x-extended и добавлен тест на API/download координаты self-update.
