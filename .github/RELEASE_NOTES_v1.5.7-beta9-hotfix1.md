# Release Notes: v1.5.7-beta9-hotfix1

Release date: 2026-06-10

Hotfix for v1.5.7-beta9. It fixes a blank panel that could appear on a fresh
load when the build happened to emit an asset name beginning with an underscore.
There are no backend logic, breaking, manual-migration, or configuration
changes.

## What changed

### Fixes

- **Blank panel / "Failed to fetch dynamically imported module" (404 on a JS
  chunk).** Rolldown names hashed assets with URL-safe base64 (`A-Za-z0-9-_`),
  so a chunk filename occasionally started with `_` (for example
  `assets/_l6q6ELT2.js`). Go's `//go:embed` excludes files whose names begin
  with `_` or `.` unless the `all:` prefix is used, so such a chunk was silently
  left out of the binary and 404'd at runtime, leaving the panel blank. Fixed on
  both sides: the embed now uses `//go:embed all:*`, and the frontend prefixes
  asset names (`app-`/`chunk-`/`style-[hash]`) so they can never start with `_`.

## Verification

- Frontend production build (`npm run build`): 0 underscore-leading assets.
- Go build of the web package with the new embed directive succeeds.
- A runtime test confirmed `//go:embed all:*` embeds an underscore-named asset
  (a plain `*` dropped it).

Upgrading from v1.5.7-beta9 is a drop-in binary replacement.

---

# Примечания к релизу: v1.5.7-beta9-hotfix1

Дата релиза: 2026-06-10

Хотфикс к v1.5.7-beta9. Исправляет пустой экран панели, который мог появляться
при загрузке, если сборка случайно сгенерировала имя ассета, начинающееся с
подчёркивания. Изменений в логике backend, ломающих изменений, ручных миграций
и изменений конфигурации нет.

## Что изменилось

### Исправления

- **Пустой экран / «Failed to fetch dynamically imported module» (404 на
  JS-чанке).** Rolldown именует хешированные ассеты в URL-safe base64
  (`A-Za-z0-9-_`), поэтому имя чанка иногда начиналось с `_` (например,
  `assets/_l6q6ELT2.js`). Go `//go:embed` исключает файлы, чьи имена начинаются
  с `_` или `.`, если не использован префикс `all:` - такой чанк молча не
  попадал в бинарь, отдавал 404 во время выполнения, и панель оставалась пустой.
  Исправлено с обеих сторон: embed теперь использует `//go:embed all:*`, а
  фронтенд добавляет префиксы к именам ассетов (`app-`/`chunk-`/`style-[hash]`),
  чтобы они никогда не начинались с `_`.

## Проверка

- Production-сборка фронтенда (`npm run build`): 0 ассетов на `_`.
- Go-сборка веб-пакета с новой embed-директивой проходит.
- Runtime-тест подтвердил, что `//go:embed all:*` встраивает ассет с именем на
  `_` (обычный `*` его терял).

Обновление с v1.5.7-beta9 - простая замена бинаря.
