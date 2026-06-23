# Release Notes: v1.5.7-beta6-hotfix1

Release date: 2026-06-05

An emergency **build** hotfix for **v1.5.7-beta6**. It carries **no code, config,
or data changes** beyond the version string - everything shipped in beta6
(security, reliability, performance, accessibility) is unchanged. It exists solely
to fix a broken frontend build in the beta6 release artifacts.

## The problem in beta6

After installing v1.5.7-beta6 the web panel failed to load: a **black screen**,
with the browser console reporting `404 (Not Found)` for a JavaScript chunk
(e.g. `assets/_WJiVkoC.js`). The application shell loaded, but a lazily-imported
view chunk was missing from the server, so the UI never rendered.

## Root cause

`frontend/package-lock.json` had drifted out of sync with `frontend/package.json`
(the icon dependency was switched from the `@mdi/font` webfont to `@mdi/js` SVG
paths without regenerating the lock). The strict CI gates (`npm ci`) correctly
went **red**, but the **release workflow built the frontend with the lenient
`npm install`**, which silently resolved a different, unvalidated dependency tree
and produced an internally-inconsistent bundle - one whose entry chunks reference
a chunk that was never emitted. That broken bundle was embedded into the beta6
binaries and shipped.

## The fix

- **`package-lock.json` regenerated in sync** with `package.json` via a clean,
  cross-platform resolution. The frontend now builds a fully consistent bundle -
  verified locally: lint clean, **88/88** unit tests pass, type-check passes, and
  every referenced asset chunk is present in the build output (no dangling 404).
- **The release pipeline is now fail-closed.** It builds the frontend with
  `npm ci` and runs the same lint + unit-test gates as CI **before** producing any
  artifact, so a desynced lockfile - or any frontend that CI would reject - can
  never be shipped again.

## Upgrade

If your panel shows a black screen on v1.5.7-beta6, install this build and
hard-refresh the browser (Ctrl/Cmd+Shift+R) once. No migration, no config change.

---

# Примечания к релизу: v1.5.7-beta6-hotfix1

Дата релиза: 2026-06-05

Экстренный **сборочный** хотфикс для **v1.5.7-beta6**. **Изменений кода,
конфигурации или данных нет** - кроме строки версии; всё содержимое beta6
(безопасность, надёжность, производительность, доступность) не изменилось. Он
исправляет только сломанную сборку фронтенда в артефактах beta6.

## Проблема в beta6

После установки v1.5.7-beta6 веб-панель не открывалась: **чёрный экран**, а в
консоли браузера - `404 (Not Found)` на JavaScript-чанк (например
`assets/_WJiVkoC.js`). Оболочка приложения загружалась, но лениво подгружаемый
чанк-вью отсутствовал на сервере, и интерфейс не отрисовывался.

## Корневая причина

`frontend/package-lock.json` рассинхронизировался с `frontend/package.json`
(зависимость иконок переключили с веб-шрифта `@mdi/font` на SVG-пути `@mdi/js`, не
перегенерировав lock). Строгие гейты CI (`npm ci`) корректно стали **красными**,
но **релизный workflow собирал фронтенд нестрогим `npm install`**, который молча
разрешал другое, невалидированное дерево зависимостей и собирал внутренне
несогласованный бандл - где входные чанки ссылаются на чанк, который так и не был
сгенерирован. Этот битый бандл вшили в бинарники beta6 и опубликовали.

## Исправление

- **`package-lock.json` перегенерирован в синхрон** с `package.json` чистым
  кросс-платформенным разрешением. Теперь фронтенд собирает полностью
  согласованный бандл - проверено локально: линт чист, **88/88** юнит-тестов
  проходят, проверка типов проходит, и каждый упоминаемый чанк присутствует в
  выводе сборки (никаких висячих 404).
- **Релизный пайплайн теперь fail-closed.** Он собирает фронтенд через `npm ci` и
  прогоняет те же линт и юнит-тесты, что и CI, **до** создания артефактов - так
  что рассинхронизированный lock (или любой фронтенд, который CI отклонил бы)
  больше не сможет уехать в релиз.

## Обновление

Если на v1.5.7-beta6 панель показывает чёрный экран - установите эту сборку и
один раз сделайте жёсткое обновление страницы (Ctrl/Cmd+Shift+R). Миграция и
правка конфигурации не нужны.
