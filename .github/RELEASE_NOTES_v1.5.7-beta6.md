# Release Notes: v1.5.7-beta6

Release date: 2026-06-05

A hardening beta on top of v1.5.7-beta5, driven by a full code-quality,
optimization, and security audit of the panel. There are **no new features** and
**no manual migration**: it closes several security gaps, removes silent-failure
and panic risks, trims the frontend bundle by ~60%, and fixes a few
data-integrity bugs. Paid Subscriptions stays **off by default**. Two items change
existing behavior - see **Breaking changes**.

## What changed

### Security

- **Go 1.26.4** - closes two reachable Go standard-library vulnerabilities
  (`GO-2026-5037` in `crypto/x509`, `GO-2026-5039` in `net/textproto`).
- **API-token scopes are enforced.** The `apiv2` action endpoints previously ran
  every action regardless of token scope, so a `read`/`observability`/`telegram`/
  `database` token could write config, restart the panel, or read settings. Each
  action is now gated - writes/restarts require `write`/`admin`, config and
  identity reads require `read`/`write`/`admin`, metrics also allow
  `observability`. Browser (admin-session) access is unchanged.
- **Remote x-ui import/sync hardened against SSRF.** Remote imports validate the
  target URL and re-check the resolved IP at connection time (defeating
  DNS-rebinding), with bounded redirects; cloud-metadata, loopback, and private
  ranges are blocked for untrusted (scoped-token) callers. `file` and `ssh` import
  sources are admin-only, and scheduled sync runs a `file`/`ssh` source only from
  an admin-saved profile.
- **Secrets at rest.** With `SUI_SECRETBOX_KEY` set, stored secrets are re-sealed
  once at startup under that out-of-database key, so a value written before you
  adopted the key is no longer recoverable from the database alone. Remote-panel
  credential encryption now derives its key from a random per-install secret
  instead of a predictable default.
- **Login brute-force** gains a per-username throttle on top of the per-IP limit.
- **Session fixation** is closed: the session ID is rotated on login.
- **Transport & headers** - HSTS honors `X-Forwarded-Proto` only from a trusted
  proxy; the CSRF cookie honors a strict `SameSite` policy; `s-ui admin -reset`
  generates a random password instead of a fixed default.
- **Telegram payments** verify the payer's Telegram id; proxy URLs carrying
  embedded credentials are masked in logs.

### Reliability and fixes

- The core no longer reports **running** when the generated config fails to parse
  (it surfaces the error instead of silently starting an empty instance).
- **Background jobs can't crash the panel** - cron jobs are panic-isolated and
  skip-if-still-running; the WAL-checkpoint job is nil-guarded at startup.
- The link, Clash, and JSON **subscription builders no longer panic** on malformed
  inbound/client configuration.
- A client name with a quote/JSON metacharacter no longer corrupts the change log,
  which previously made the admin **Changes** view return an empty response.
- **Bulk client edits** with differing inbound sets now regenerate each client's
  links from its own inbounds.
- API errors are consistent and redact internal details.

### Performance

- **Backend** - IP-monitor writes are batched into a single upsert; the
  subscription hot path caches its display settings (~8 fewer queries per request)
  and `settings` reads use an index.
- **Frontend bundle 6.2 MB → 2.5 MB (-60%)** - `moment` and the date-picker load
  lazily, and icons moved from the full Material Design webfont to inline SVG.

### Accessibility

- Icon-only admin action buttons (edit / changes / delete) now have accessible
  names for screen readers.

## Breaking changes

- **Scoped API tokens** that wrote config, restarted the panel, or read settings
  (only possible because of the enforcement gap above) are now rejected - use an
  `admin` or appropriately scoped `write` token.
- **`file`/`ssh` x-ui sync profiles** must be admin-saved: after upgrading, a
  scheduled profile whose source is a local `file`/`ssh` target will not run until
  an admin re-saves it.

## Upgrade

No manual migration or config change. The `settings` table gains a unique index
automatically on first start, and (if `SUI_SECRETBOX_KEY` is set) the one-time
secret re-seal runs at startup. Review the two **Breaking changes** if you use
scoped API tokens or `file`/`ssh` scheduled sync.

---

# Примечания к релизу: v1.5.7-beta6

Дата релиза: 2026-06-05

Бета-усиление поверх v1.5.7-beta5 по итогам полного аудита качества кода,
оптимизации и безопасности панели. **Новых функций нет** и **ручная миграция не
требуется**: закрыты несколько брешей в безопасности, убраны риски тихих отказов
и паник, фронтенд-бандл урезан на ~60%, исправлено несколько багов целостности
данных. «Платные подписки» по-прежнему **выключены по умолчанию**. Два пункта
меняют существующее поведение - см. **Ломающие изменения**.

## Что изменилось

### Безопасность

- **Go 1.26.4** - закрывает две достижимые уязвимости стандартной библиотеки Go
  (`GO-2026-5037` в `crypto/x509`, `GO-2026-5039` в `net/textproto`).
- **Scope API-токенов энфорсится.** Раньше action-эндпоинты `apiv2` выполняли
  любое действие независимо от scope, так что токен
  `read`/`observability`/`telegram`/`database` мог писать конфиг, перезапускать
  панель и читать настройки. Теперь каждое действие гейтится - запись/перезапуск
  требуют `write`/`admin`, чтение конфига/учёток - `read`/`write`/`admin`, метрики
  допускают и `observability`. Доступ через браузерную сессию админа не изменился.
- **Удалённый импорт/синк x-ui усилен против SSRF.** Удалённые импорты валидируют
  целевой URL и повторно проверяют разрешённый IP в момент соединения (защита от
  DNS-rebinding), с ограничением редиректов; cloud-metadata, loopback и приватные
  диапазоны блокируются для недоверенных (scoped-токен) вызовов. Источники `file`
  и `ssh` - только для админа, а плановый синк выполняет `file`/`ssh`-источник
  только из профиля, сохранённого админом.
- **Секреты at-rest.** При заданном `SUI_SECRETBOX_KEY` секреты одноразово
  ре-шифруются на старте под этим внедатабазным ключом - значение, записанное до
  включения ключа, больше не восстановить из одной только БД. Шифрование учёток
  удалённых панелей выводит ключ из случайного per-install секрета.
- **Брутфорс логина** - добавлен throttle по имени пользователя поверх лимита по
  IP.
- **Session fixation** закрыт: ID сессии ротируется при логине.
- **Транспорт и заголовки** - HSTS доверяет `X-Forwarded-Proto` только от
  доверенного прокси; CSRF-cookie соблюдает строгий `SameSite`; `s-ui admin
  -reset` генерирует случайный пароль вместо фиксированного дефолта.
- **Telegram-платежи** сверяют Telegram-id плательщика; proxy-URL со встроенными
  кредами маскируются в логах.

### Надёжность и фиксы

- Ядро больше не рапортует **running**, если конфиг не парсится (возвращает
  ошибку вместо тихого запуска пустого инстанса).
- **Фоновые джобы не роняют панель** - изоляция паник и skip-if-still-running;
  WAL-checkpoint защищён от nil на старте.
- Генераторы link/Clash/JSON-подписок **больше не падают** на малформном конфиге
  inbound/клиента.
- Имя клиента с кавычкой/JSON-метасимволом больше не портит журнал изменений (из-за
  чего страница **Changes** отдавала пустой ответ).
- **Bulk-правка** клиентов с разными наборами Inbounds пересобирает ссылки
  каждого клиента из его собственных Inbounds.
- Ошибки API единообразны и редактируют внутренние детали.

### Производительность

- **Бэкенд** - записи IP-монитора батчатся в один upsert; горячий путь подписки
  кэширует display-настройки (≈8 запросов меньше на запрос), чтения `settings`
  идут по индексу.
- **Фронтенд-бандл 6.2 МБ → 2.5 МБ (-60%)** - `moment` и date-picker грузятся
  лениво, иконки переехали с веб-шрифта на инлайн-SVG.

### Доступность

- Иконочные кнопки действий админа (редактировать / изменения / удалить) получили
  доступные имена для скринридеров.

## Ломающие изменения

- **Scoped API-токены**, писавшие конфиг, перезапускавшие панель или читавшие
  настройки (что работало лишь из-за бреши выше), теперь отклоняются - используйте
  токен `admin` или подходящий `write`.
- **`file`/`ssh`-профили синка x-ui** должны быть сохранены админом: после
  апгрейда плановый профиль с локальным `file`/`ssh`-источником не запустится, пока
  админ не пере-сохранит его.

## Обновление

Ручной миграции и правки конфига не нужно. Таблица `settings` получает уникальный
индекс автоматически при первом старте, а при использовании `SUI_SECRETBOX_KEY`
одноразовый ре-seal секретов выполняется на старте. Просмотрите два **Ломающих
изменения**, если используете scoped API-токены или `file`/`ssh` плановый синк.
