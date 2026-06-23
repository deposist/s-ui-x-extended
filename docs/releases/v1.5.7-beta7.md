# Release Notes: v1.5.7-beta7

Release date: 2026-06-06

A hardening beta on top of v1.5.7-beta6-hotfix1. It ships the **removal of the
3x-ui scheduled sync / remote import** (the import feature is now a one-shot local
`.db` upload only) plus a round of **low-risk fixes and hardening** from a fresh
independent code-quality, optimization, and security review of the panel. **No new
features** and **no manual migration**. Paid Subscriptions stays **off by default**.
The 3x-ui removal changes existing behavior - see **Breaking changes**; none of the
review fixes are breaking.

## What changed

### Breaking changes (3x-ui import)

- **Scheduled sync is gone.** The "3x-ui Sync" schedule page, sync profiles, and the
  background cron job no longer exist; any previously configured schedule stops
  running after upgrade. The `xui_sync_profiles` and `xui_known_hosts` tables are
  dropped automatically on first start - no manual step.
- **Remote import is gone.** The `POST /api/import-xui/remote/*` and
  `/api/import-xui/sync/*` endpoints are removed. Import is upload-only:
  `POST /api/import-xui[/plan|/apply|/rollback]` and `GET /api/import-xui/reports`.
- **CLI:** the `s-ui sync-xui` command and the `import-xui --remote` / `--schedule`
  flags are removed; `s-ui import-xui --src <x-ui.db>` (local file) remains.
- **API tokens:** the `xui_remote` token scope is removed and is no longer valid -
  re-issue any such token with an appropriate scope.

### Security & privacy

- **Login no longer reveals whether a username exists.** The "user not found" path now
  performs the same bcrypt work as a wrong-password attempt, closing a timing
  side-channel that allowed admin-username enumeration.
- **URL credentials are masked in logs.** A `user:pass@host` embedded in any logged URL
  is now redacted in free text and under secret-named setting keys.
- **Session cookies are Secure by default** in the session store, so any future
  session-creating code path that forgets to set options still issues a hardened
  cookie. (Production login/CSRF flows already set this explicitly.)
- **Refunds reject corrupted orders.** A paid order with a non-positive amount is never
  processed (defense in depth - orders are created with a positive amount).
- **IP-limit failures are observable.** When the per-client IP-limit check cannot reach
  the database it still fails open (allows the connection), but now logs the event
  (throttled) instead of silently disabling enforcement.

### Reliability & fixes

- **Correct traffic chart.** The per-client statistics graph summed each time bucket
  with a no-op reducer and displayed only the first sample instead of the total; it now
  sums correctly.
- **Safer 1.3 migration.** The anytls / domain-strategy migration now runs inside a
  transaction and checks every write - it previously ignored save errors and carried a
  dead filter clause that loaded every row.
- **IDN panel domains work.** A Unicode panel domain (e.g. `панель.рф`) now matches the
  punycode `Host` header browsers send instead of being rejected with `403`.
- **Bounded public-IP probe.** The `s-ui uri` public-IP lookup caps the response body
  (1 MiB), matching every other outbound reader in the codebase.
- **No drawer thrash.** The default layout's `isMobile` is a pure computed again; the
  drawer's default open state follows the breakpoint through a watcher instead of a
  side effect inside the getter.
- **Clearer core-start log.** A sing-box core that fails to start is logged explicitly;
  the panel intentionally stays up so the config can be fixed from the UI.

### Performance & cleanup

- **Indexed order history.** `payment_orders.telegram_user_id` is now indexed, so a
  user's order / refund history no longer scans the whole table.
- **Lighter frontend install.** Removed three unused dependencies (`core-js`,
  `roboto-fontface`, `material-design-icons-iconfont`).

### Kept

- One-shot local **`.db` upload** import - the UI wizard, the API, and
  `import-xui --src` - including dry-run, conflict strategy, plan/apply, and rollback.

## Upgrade

No manual migration or config change. The deprecated `xui_sync_profiles` and
`xui_known_hosts` tables are dropped automatically on first start. Review the
**Breaking changes** if you used 3x-ui scheduled sync, remote import, the
`s-ui sync-xui` CLI, or an `xui_remote` token.

---

# Примечания к релизу: v1.5.7-beta7

Дата релиза: 2026-06-06

Бета-усиление поверх v1.5.7-beta6-hotfix1. В релиз входит **удаление планового синка
/ удалённого импорта 3x-ui** (импорт теперь - только разовая локальная загрузка `.db`)
плюс набор **малорисковых фиксов и hardening** по итогам свежего независимого аудита
качества кода, оптимизации и безопасности панели. **Новых функций нет** и **ручная
миграция не требуется**. «Платные подписки» по-прежнему **выключены по умолчанию**.
Удаление 3x-ui меняет существующее поведение - см. **Ломающие изменения**; фиксы из
ревью ничего не ломают.

## Что изменилось

### Ломающие изменения (импорт 3x-ui)

- **Плановый синк удалён.** Страница расписания «3x-ui Sync», профили синка и фоновый
  cron-job больше не существуют; любое ранее настроенное расписание перестаёт работать
  после апгрейда. Таблицы `xui_sync_profiles` и `xui_known_hosts` удаляются
  автоматически при первом старте - без ручных действий.
- **Удалённый импорт удалён.** Эндпоинты `POST /api/import-xui/remote/*` и
  `/api/import-xui/sync/*` убраны. Импорт - только через загрузку:
  `POST /api/import-xui[/plan|/apply|/rollback]` и `GET /api/import-xui/reports`.
- **CLI:** команда `s-ui sync-xui` и флаги `import-xui --remote` / `--schedule` удалены;
  `s-ui import-xui --src <x-ui.db>` (локальный файл) остаётся.
- **API-токены:** scope `xui_remote` удалён и больше не действителен - перевыпустите
  такой токен с подходящим scope.

### Безопасность и приватность

- **Логин больше не выдаёт, существует ли имя пользователя.** Путь «пользователь не
  найден» теперь выполняет ту же bcrypt-работу, что и неверный пароль, закрывая
  тайминг-side-channel для перечисления админ-логинов.
- **Креды в URL маскируются в логах.** `user:pass@host` внутри любого логируемого URL
  редактируется в свободном тексте, а не только когда значение лежит под
  секрет-именованным ключом настройки.
- **Cookie сессии - Secure по умолчанию** в session store, чтобы любой будущий путь
  создания сессии, забывший выставить опции, всё равно выдавал защищённую cookie.
  (Боевые login/CSRF-потоки уже выставляют это явно.)
- **Refund отклоняет повреждённые заказы.** Оплаченный заказ с неположительной суммой
  не обрабатывается (защита в глубину - заказы создаются с положительной суммой).
- **Сбои IP-лимита наблюдаемы.** Когда проверка IP-лимита клиента не достучалась до БД,
  она по-прежнему fail-open (пропускает соединение), но теперь логирует событие
  (с троттлингом), а не отключает энфорсмент молча.

### Надёжность и фиксы

- **Корректный график трафика.** График статистики клиента суммировал каждый бакет
  no-op-редьюсером и показывал только первый сэмпл вместо суммы; теперь суммирует
  правильно.
- **Безопаснее миграция 1.3.** Миграция anytls / domain-strategy теперь выполняется в
  транзакции и проверяет каждую запись - раньше она игнорировала ошибки сохранения и
  несла мёртвый фильтр, грузивший все строки.
- **IDN-домены панели работают.** Unicode-домен панели (напр. `панель.рф`) теперь
  совпадает с punycode-`Host`, который шлёт браузер, вместо отказа `403`.
- **Ограниченный public-IP-пробинг.** Запрос публичного IP в `s-ui uri` ограничивает
  тело ответа (1 МиБ), как и все остальные исходящие читатели в коде.
- **Без дёрганья drawer.** `isMobile` в дефолтном лейауте снова чистый computed; дефолт
  открытия drawer следует за брейкпоинтом через watcher, а не через side-effect в
  геттере.
- **Понятный лог старта ядра.** Падение запуска sing-box-ядра логируется явно; панель
  намеренно остаётся поднятой, чтобы конфиг можно было починить из UI.

### Производительность и чистка

- **Индекс истории заказов.** `payment_orders.telegram_user_id` теперь индексирован, и
  история заказов / возвратов пользователя больше не сканирует всю таблицу.
- **Легче установка фронтенда.** Удалены три неиспользуемые зависимости (`core-js`,
  `roboto-fontface`, `material-design-icons-iconfont`).

### Сохранено

- Разовый локальный импорт через загрузку **`.db`** - мастер UI, API и
  `import-xui --src` - включая dry-run, стратегию конфликтов, plan/apply и rollback.

## Обновление

Ручной миграции и правки конфига не нужно. Устаревшие таблицы `xui_sync_profiles` и
`xui_known_hosts` удаляются автоматически при первом старте. Просмотрите **Ломающие
изменения**, если использовали плановый синк 3x-ui, удалённый импорт, CLI
`s-ui sync-xui` или токен `xui_remote`.
