# Release Notes: v1.5.7

Release date: 2026-06-11

First stable release of the 1.5.7 line. It consolidates the
1.5.7-beta1..beta10 series: the experimental **Paid Subscriptions** Telegram
bot, the refreshed Nexus interface, and two rounds of independent security
hardening. This stable release also adds **no-restart client/TLS apply**. One
additive database column is created automatically on first start; no manual
migration is needed.

## What changed

- **Paid Subscriptions (experimental, off by default).**
  A client-facing Telegram bot on its own encrypted token: a bound client gets
  their subscription link, per-protocol share links and QR codes, and sees live
  usage (used vs. limit, days left, online status, total traffic). Unknown
  users can self-register with a configurable free trial (global cap +
  per-user rate limit). Admins define tariffs (price, +days, +traffic) and
  clients pay or renew right in the bot across six providers - **Telegram
  Stars, YooKassa, Stripe, CryptoBot, PayMaster, and external links** - with
  idempotent, server-verified renewals (no double-charging on retries). The
  bot's Payment menu covers *Buy / Renew*, *My purchases*, and *Request a
  refund* (Stars refund automatically; other providers route the request to
  the admin); the admin Orders tab can refund any order with an optional
  claw-back of the granted days/traffic. Includes broadcasts to all bound
  clients, an editable /start greeting, and per-module Telegram egress via a
  proxy or one of your sing-box outbounds. Existing setups are unaffected
  until the module is switched on; for production set `SUI_SECRETBOX_KEY` so
  bot and payment tokens are sealed with a key kept outside the database.

- **New in this stable release: client and TLS edits apply without restarting
  the core.** Adding, editing, enabling/disabling, or deleting a client no
  longer restarts sing-box - only the affected inbounds are hot-reloaded in
  the running core. Connections on those inbounds are closed, so revoked or
  disabled credentials stop working immediately, while connections on
  unrelated inbounds survive the change. IP-limit changes (`limitIp`, mode)
  apply at once instead of after a 30-second cache TTL. Editing a TLS
  certificate hot-reloads only the inbounds and services that reference it,
  after the database transaction commits; creating or deleting a TLS entry no
  longer touches the core at all. If a hot reload fails, the panel falls back
  to a full core restart so the core never serves a stale configuration. Also
  fixed: editing a TLS entry referenced by a service used to fail with a
  database scan error and roll back the whole edit.

- **Refreshed Nexus interface.** The default interface now matches the dark
  "technical" reference design end to end: exact palette with a cyan accent, a
  single system font, monospace IP / port / UUID cells, coloured status badges
  and TLS pills, a page header with live counts and search in the top bar, and
  a compact flat sidebar. Bulk client Add/Edit open as proper drawers, the
  Paid Subscriptions screen is localized (EN/RU), and unsaved-changes
  confirmation covers every entity form in both interface modes.

- **Security hardening (independent reviews in beta6, beta7, beta10).**
  Updated to Go 1.26.4 (closes `GO-2026-5037`, `GO-2026-5039`). API-token
  scopes are enforced on every `apiv2` action. Login no longer reveals whether
  a username exists (timing oracle closed); brute force is slowed by a
  per-username escalating delay (tarpit) on top of the per-IP block; the
  session ID rotates on login, and changing the admin password signs out all
  other sessions and WebSocket tokens. With `SUI_SECRETBOX_KEY` set, stored
  secrets are re-sealed once at startup under the out-of-database key. URL
  credentials are masked in logs, session cookies are Secure by default, HSTS
  honors `X-Forwarded-Proto` only from a trusted proxy, and x-ui import
  validates geoip/geosite codes before building rule-set URLs. New audit
  signals (`/sub` enumeration, login from a new IP, cross-user order access,
  IP-limit enforcement, audit-pipeline drop marker) with real-time alerts on
  account lockout and database export.

- **Fixes for everyone.** Double-submitting a save can no longer create
  duplicate clients/inbounds/outbounds (the button locks and the server
  rejects the duplicate). The per-client traffic chart sums correctly instead
  of showing the first sample. A client name with a quote no longer corrupts
  the admin Changes feed. Bulk-editing clients with different inbound sets
  regenerates each client's links from its own inbounds. Unicode (IDN) panel
  domains work. A sing-box config that fails to parse surfaces the error
  instead of silently reporting a healthy empty core. Subscription builders
  skip malformed fields instead of panicking.

- **Performance.** Frontend bundle cut from 6.2 MB to 2.5 MB (-60%; lazy
  `moment`/date-picker, SVG icons). The IP monitor writes pending records in a
  single batched upsert; the subscription hot path caches display settings
  (~8 fewer queries per request); order history is indexed.

- **Release-pipeline and supply-chain guards.** The release workflow builds
  the frontend fail-closed (`npm ci` + the same lint/test gates as CI), the
  binary embeds assets with `go:embed all:` and the build prefixes asset names
  so a hashed chunk can never be silently dropped (the beta6/beta9 blank-panel
  class of bug). Third-party and Docker GitHub Actions are pinned to commit
  SHAs with Dependabot keeping them current.

See the `1.5.7-beta1`..`1.5.7-beta10` entries in `CHANGELOG-EN.md` for the
full per-step history.

## Breaking changes (vs v1.5.6)

- **Scoped API tokens lose access they should never have had.** An integration
  using a `read`/`observability`/`telegram`/`database` token to write config,
  restart the panel, or read settings is now rejected - use an `admin` or
  appropriately scoped `write` token.
- **3x-ui scheduled sync and remote import are removed.** Import is now the
  one-shot local `.db` upload only (UI wizard, `POST /api/import-xui[/plan|
  /apply|/rollback]`, `s-ui import-xui --src`). The `s-ui sync-xui` command,
  the `--remote`/`--schedule` flags, the remote/sync API routes, and the
  `xui_remote` token scope are gone; the `xui_sync_profiles` and
  `xui_known_hosts` tables are dropped automatically on first start.
- **Behavior changes:** changing the admin password signs out all other
  sessions; the per-username login lockout is a delay (tarpit), not a hard
  block.

## Upgrade

No manual migration. One additive database column and a `settings` unique
index are created automatically on first start; with `SUI_SECRETBOX_KEY` set,
the one-time secret re-seal runs at startup. Review the breaking changes above
if you use scoped API tokens or relied on 3x-ui scheduled sync / remote
import. Paid Subscriptions stays off until you enable it - try it on a
non-critical instance first.

---

# Примечания к релизу: v1.5.7

Дата релиза: 2026-06-11

Первый стабильный релиз линейки 1.5.7. Он объединяет серию
1.5.7-beta1..beta10 - главная тема которой экспериментальный Telegram-бот
**«Платные подписки»** - вместе с обновлённым интерфейсом Nexus и двумя
раундами независимого аудита безопасности, и добавляет **применение изменений
клиентов/TLS без перезапуска ядра** (новое в этом стабильном релизе). Одна
добавочная колонка БД создаётся автоматически при первом старте; ручная
миграция не нужна.

## Что изменилось

- **«Платные подписки» (главное в 1.5.7; экспериментально, выключено по
  умолчанию).** Клиентский Telegram-бот на отдельном зашифрованном токене:
  привязанный клиент получает ссылку подписки, ссылки по протоколам и QR-коды
  и видит живую статистику (израсходовано/лимит, дни, онлайн, суммарный
  трафик). Незнакомый пользователь может зарегистрироваться сам с настраиваемым
  пробным периодом (глобальный лимит + ограничение частоты). Админ задаёт
  тарифы (цена, +дни, +трафик), клиенты платят и продлевают прямо в боте через
  шесть провайдеров - **Telegram Stars, YooKassa, Stripe, CryptoBot, PayMaster
  и внешние ссылки** - с идемпотентными, проверяемыми на сервере продлениями
  (без двойных списаний при повторах). Меню «Оплата» в боте: *Купить /
  Продлить*, *Мои покупки*, *Оформить возврат* (Stars возвращаются
  автоматически; по остальным провайдерам заявка уходит админу); вкладка
  *Orders* в панели возвращает любой заказ с опциональным отзывом выданных
  дней/трафика. Плюс рассылки всем привязанным клиентам, редактируемое
  приветствие /start и эгресс Telegram через прокси или ваш sing-box-outbound,
  отдельно для бота и админ-уведомлений. Существующие установки не затронуты,
  пока модуль не включён; для продакшена задайте `SUI_SECRETBOX_KEY`, чтобы
  токены бота и платёжных провайдеров шифровались ключом вне базы.

- **Новое в этом стабильном релизе: изменения клиентов и TLS применяются без
  перезапуска ядра.** Добавление, редактирование, включение/выключение и
  удаление клиента больше не перезапускают sing-box - в работающем ядре горячо
  перезагружаются только затронутые Inbounds. Соединения на них закрываются,
  поэтому отозванные или отключённые учётные данные перестают работать сразу,
  а соединения на остальных Inbounds переживают изменение. Изменения
  IP-лимита (`limitIp`, режим) применяются немедленно, а не после 30-секундного
  TTL кэша. Редактирование TLS-сертификата горячо перезагружает только
  ссылающиеся на него Inbounds и сервисы - после фиксации транзакции БД;
  создание и удаление TLS-записи ядро вообще не трогают. Если горячая
  перезагрузка не удалась, панель откатывается к полному перезапуску ядра,
  чтобы оно не работало на устаревшей конфигурации. Также исправлено:
  редактирование TLS-записи, на которую ссылается сервис, падало с ошибкой
  чтения из БД и целиком откатывалось.

- **Обновлённый интерфейс Nexus.** Интерфейс по умолчанию теперь полностью
  соответствует тёмному «техническому» референс-дизайну: точная палитра с
  cyan-акцентом, единый системный шрифт, моноширинные ячейки IP / порт / UUID,
  цветные бейджи статусов и пилюли TLS, заголовок страницы с живыми счётчиками
  и поиском в верхней панели, компактный плоский сайдбар. Массовые
  добавление/правка клиентов открываются как полноценные drawer'ы, экран
  «Платных подписок» локализован (EN/RU), подтверждение несохранённых
  изменений покрывает каждую форму в обоих режимах интерфейса.

- **Усиление безопасности (независимые ревью в beta6, beta7, beta10).**
  Обновление до Go 1.26.4 (закрывает `GO-2026-5037`, `GO-2026-5039`). Скоупы
  API-токенов проверяются на каждом действии `apiv2`. Логин больше не выдаёт,
  существует ли имя пользователя (закрыт timing-оракул); перебор замедляется
  нарастающей задержкой (tarpit) по имени поверх блокировки по IP; ID сессии
  ротируется при входе, а смена пароля администратора завершает все остальные
  сессии и WebSocket-токены. При заданном `SUI_SECRETBOX_KEY` сохранённые
  секреты однократно перешифровываются при старте ключом вне базы. Учётные
  данные в URL маскируются в логах, cookie сессии Secure по умолчанию, HSTS
  доверяет `X-Forwarded-Proto` только от доверенного прокси, импорт x-ui
  валидирует коды geoip/geosite до построения URL rule-set'ов. Новые
  аудит-сигналы (перебор `/sub`, вход с нового IP, чужие заказы в клиентском
  боте, срабатывание IP-лимита, маркер потерь аудит-конвейера) и алерты в
  реальном времени на блокировку аккаунта и экспорт базы.

- **Исправления для всех.** Двойная отправка сохранения больше не создаёт
  дубликаты клиентов/Inbounds/Outbounds (кнопка блокируется, сервер
  отклоняет повтор). График трафика клиента суммирует корректно, а не
  показывает первую выборку. Имя клиента с кавычкой больше не ломает ленту
  изменений. Массовая правка клиентов с разными наборами Inbounds
  регенерирует ссылки каждого клиента из его собственных Inbounds. Работают
  Unicode-домены панели (IDN). Ошибка разбора конфига sing-box показывается
  явно, вместо тихо «здорового» пустого ядра. Генераторы подписок пропускают
  битые поля вместо паники.

- **Производительность.** Фронтенд-бандл уменьшен с 6,2 МБ до 2,5 МБ (-60%;
  ленивые `moment`/date-picker, SVG-иконки). IP-монитор пишет накопленные
  записи одним батч-upsert'ом; горячий путь подписок кэширует настройки
  отображения (~8 запросов меньше на запрос); история заказов
  проиндексирована.

- **Защита релизного конвейера и цепочки поставок.** Релизный workflow
  собирает фронтенд fail-closed (`npm ci` + те же lint/test-гейты, что в CI),
  бинарник встраивает ассеты через `go:embed all:`, а сборка префиксует имена
  ассетов, чтобы хэшированный чанк не мог быть молча выброшен (класс багов
  «пустая панель» из beta6/beta9). Сторонние и Docker GitHub Actions
  запинены на commit SHA, Dependabot поддерживает их актуальность.

Полную пошаговую историю см. в записях `1.5.7-beta1`..`1.5.7-beta10` в
`CHANGELOG-RU.md`.

## Ломающие изменения (относительно v1.5.6)

- **Скоупированные API-токены теряют доступ, которого не должны были иметь.**
  Интеграция, использующая токен `read`/`observability`/`telegram`/`database`
  для записи конфига, перезапуска панели или чтения настроек, теперь
  отклоняется - используйте `admin` или подходящий `write`-токен.
- **Удалены плановая синхронизация и удалённый импорт 3x-ui.** Импорт - только
  разовая загрузка локального `.db` (мастер в UI, `POST /api/import-xui[/plan|
  /apply|/rollback]`, `s-ui import-xui --src`). Команда `s-ui sync-xui`, флаги
  `--remote`/`--schedule`, remote/sync API-маршруты и скоуп токена `xui_remote`
  удалены; таблицы `xui_sync_profiles` и `xui_known_hosts` удаляются
  автоматически при первом старте.
- **Изменения поведения:** смена пароля администратора завершает все остальные
  сессии; блокировка логина по имени пользователя - это задержка (tarpit), а
  не жёсткий блок.

## Обновление

Ручная миграция не нужна. Добавочная колонка БД и уникальный индекс `settings`
создаются автоматически при первом старте; при заданном `SUI_SECRETBOX_KEY`
однократное перешифрование секретов выполняется на старте. Просмотрите
ломающие изменения выше, если используете скоупированные API-токены или
полагались на плановую синхронизацию / удалённый импорт 3x-ui. «Платные
подписки» выключены, пока вы их не включите - сначала попробуйте на
некритичном экземпляре.
