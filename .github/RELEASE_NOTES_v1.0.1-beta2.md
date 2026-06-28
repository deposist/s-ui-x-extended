# S-UI-X-Extended v1.0.1-beta2

Frontend and documentation update. No database migration is required.

## Frontend

- Dashboard Traffic statistics now has a timezone selector. The default value
  comes from the browser timezone.

- The selected timezone is saved in `localStorage`, so the Dashboard keeps the
  same statistics timezone after reloads and across browser sessions.

- Traffic chart labels now use `YYYY-MM-DD HH:MM`. The selected timezone is
  shown in the KPI metadata next to the range selector, which makes the bucket
  times clear when the browser and server use different timezones.

## Documentation

- README now documents the Dashboard Traffic statistics timezone selector,
  including the browser default, `localStorage` persistence,
  `YYYY-MM-DD HH:MM` chart labels, and the KPI timezone label.

- The README supported-protocol list was refreshed from the repository
  capability and configuration docs.

## Operator notes

- No configuration or database migration is required for this release.
- Clearing site storage resets the Dashboard timezone selector to the browser
  timezone.

---

# S-UI-X-Extended v1.0.1-beta2

Обновление фронтенда и документации. Миграция базы не требуется.

## Фронтенд

- В Dashboard Traffic statistics добавлен выбор часового пояса. По умолчанию
  используется часовой пояс браузера.

- Выбранный часовой пояс сохраняется в `localStorage`, поэтому Dashboard
  сохраняет его после перезагрузки страницы и между сессиями браузера.

- Подписи на графиках трафика теперь используют формат `YYYY-MM-DD HH:MM`.
  Выбранный часовой пояс показан в метаданных KPI рядом с выбором диапазона,
  поэтому интервалы понятны даже если браузер и сервер используют разные
  часовые пояса.

## Документация

- README теперь описывает выбор часового пояса для Dashboard Traffic statistics:
  дефолт из браузера, сохранение в `localStorage`, подписи
  `YYYY-MM-DD HH:MM` и отображение часового пояса в KPI.

- Список поддерживаемых протоколов в README обновлен по capability и
  configuration docs репозитория.

## Заметки для операторов

- Для этого релиза не требуется менять конфигурацию или выполнять миграцию
  базы данных.
- Очистка site storage сбрасывает selector часового пояса Dashboard к часовому
  поясу браузера.
