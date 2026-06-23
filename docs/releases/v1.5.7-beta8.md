# Release Notes: v1.5.7-beta8

Release date: 2026-06-07

This maintenance set hardens database-handle concurrency, improves the
reliability of SQLite-backed tests on Windows, and keeps security scanning
focused on project source code. It contains no user-facing feature changes,
breaking changes, manual migrations, or configuration changes.

## What changed

### Runtime reliability

- **Race-free database-handle publication.** Initialization now builds and
  configures a local GORM handle before publishing it under a read/write mutex.
  `GetDB` uses the same synchronization, removing the detected race when the
  process initializes or replaces the global handle.

### Test and CI reliability

- **Deterministic SQLite teardown on Windows.** Database, import-xui, cron-job,
  and IP-monitor tests now checkpoint WAL files, close active handles, and retry
  temporary-directory cleanup when Windows briefly retains SQLite files.
- **Isolated paid-subscription schema tests.** Test database resets stop the
  asynchronous audit writer and use synchronous audit writes for the test
  lifetime, preventing stale background work from reaching a replaced database.

### Security and quality tooling

- **Focused `gosec` scans.** The Makefile audit target excludes `.gotmp`,
  `.gocache`, and `frontend/node_modules`, so generated caches and vendored
  JavaScript dependencies are not treated as project Go packages.

## Verification

The complete backend and frontend validation gate was run for this release:

- `go test ./...`
- `go test ./... -race -count=1`
- `go build ./...`
- `go vet ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `gosec` across 177 project Go files: 0 issues
- `govulncheck ./...`: no vulnerabilities found
- `npm run lint`
- `npm run test`: 18 files, 88 tests passed
- `npm run build`
- `npm audit`: 0 vulnerabilities

---

# Примечания к релизу: v1.5.7-beta8

Дата релиза: 2026-06-07

Этот набор технических изменений усиливает потокобезопасность работы с
дескриптором базы данных, повышает стабильность SQLite-тестов в Windows и
ограничивает аудит безопасности исходным кодом проекта. Пользовательских
функций, ломающих изменений, ручных миграций и изменений конфигурации нет.

## Что изменилось

### Надёжность приложения

- **Потокобезопасная публикация дескриптора БД.** Инициализация полностью
  настраивает локальный GORM-дескриптор до его публикации под read/write mutex.
  `GetDB` использует ту же синхронизацию, устраняя обнаруженную гонку при
  инициализации или замене глобального дескриптора.

### Надёжность тестов и CI

- **Предсказуемое завершение SQLite-тестов в Windows.** Тесты базы данных,
  import-xui, cron-задач и IP-монитора теперь выполняют checkpoint WAL,
  закрывают активные дескрипторы и повторяют удаление временных каталогов, если
  Windows кратковременно удерживает SQLite-файлы.
- **Изолированные schema-тесты платных подписок.** При сбросе тестовой БД
  асинхронный audit writer останавливается, а на время теста аудит переводится в
  синхронный режим. Это не даёт фоновой записи обратиться к уже заменённой БД.

### Инструменты безопасности и контроля качества

- **Точная область сканирования `gosec`.** Audit-цель Makefile исключает
  `.gotmp`, `.gocache` и `frontend/node_modules`, поэтому сгенерированные кэши и
  сторонние JavaScript-зависимости не считаются Go-пакетами проекта.

## Проверка

Полный набор backend- и frontend-проверок выполнен для этого релиза:

- `go test ./...`
- `go test ./... -race -count=1`
- `go build ./...`
- `go vet ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `gosec` по 177 Go-файлам проекта: 0 проблем
- `govulncheck ./...`: уязвимости не обнаружены
- `npm run lint`
- `npm run test`: 18 файлов, 88 тестов пройдено
- `npm run build`
- `npm audit`: 0 уязвимостей
