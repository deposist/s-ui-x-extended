# Release Notes: v1.5.7-beta1

Release date: 2026-06-04

First beta of the 1.5.7 line. Introduces an **experimental "Paid Subscriptions"
module**: a client-facing Telegram bot through which end users get their
subscription, view usage, self-register with a trial, and pay/renew via several
providers. The feature is **off by default** and fully isolated from the core -
existing setups are unaffected. No core schema migration; the module creates its
own tables on startup.

## What changed

- **Client-facing Telegram bot (separate token).** A new long-polling bot
  (its own token, independent from the admin-notification bot) lets a bound
  client, from inside Telegram:
  - get their subscription link, per-inbound share links (vless/vmess/...), and
    **QR codes** (rendered server-side as images);
  - view **current usage stats** - used/limit with a progress bar, days left
    until expiry, online status, and lifetime traffic.
- **Telegram ID ↔ client binding on a dedicated page.** Admins map a Telegram
  user to a client on the new **Paid Subscriptions** page (a separate left-menu
  item). The existing client card and the core `clients` table are **not**
  touched; bindings live in their own table.
- **Self-registration with a trial.** When enabled, an unknown user who opens
  the bot is auto-registered: a client is created with the admin-selected
  inbounds and a configurable trial period (days + optional traffic), then bound
  to that Telegram user. Guarded by a global cap and a per-user `/start` rate
  limit.
- **Tariff-based payments, multi-provider.** Admins define tariffs
  (name, price, +days, +traffic) on the Paid Subscriptions page. Clients buy/renew
  in the bot; on a confirmed payment the subscription is extended automatically
  (expiry + traffic, re-enabled). Providers are selectable and several can be
  enabled at once: **Telegram Stars (XTR), YooKassa, Stripe, CryptoBot, and an
  external payment link**. Renewals are idempotent (no double-apply on retries
  or races), amounts are verified server-side against the order snapshot, and
  zero-price tariffs cannot grant a renewal.
- **Isolated, removable module.** All of the above lives in a dedicated `paidsub`
  package wired in behind a single `paidSubEnabled` flag, with its own HTTP
  endpoints and its own database tables (created idempotently at startup). The
  admin UI is a lazy-loaded page, marked *experimental*.

## Security

- The client bot uses a **separate, encrypted** bot token; all payment-provider
  tokens are stored encrypted at rest and masked in the API/UI.
- Provider API hosts are pinned and tokens are never logged. The bot resolves a
  client only via its binding (never trusts ids from messages) and renews only
  the bound client.
- For production, set the **`SUI_SECRETBOX_KEY`** environment variable so secrets
  are encrypted with a key kept outside the database (the UI warns when it is
  unset).

## Upgrade

No manual migration; existing data is preserved and the feature is **disabled by
default**. To try it: open **Paid Subscriptions** in the panel, enable the bot,
paste a bot token (from @BotFather), and set the subscription domain
(`subDomain`/`subURI`) so links can be built. This is a beta - test on a
non-critical instance first.

---

# Примечания к релизу: v1.5.7-beta1

Дата релиза: 2026-06-04

Первая бета линейки 1.5.7. Добавлен **экспериментальный модуль «Платные
подписки»**: клиентский Telegram-бот, через который конечные пользователи
получают свою подписку, смотрят статистику, саморегистрируются с пробным
периодом и оплачивают/продлевают подписку через несколько провайдеров. Функция
**выключена по умолчанию** и полностью изолирована от ядра - существующие
установки не затрагиваются. Миграции схемы ядра нет; модуль создаёт свои таблицы
при старте.

## Что изменилось

- **Клиентский Telegram-бот (отдельный токен).** Новый бот на long-polling
  (со своим токеном, независимым от бота админ-уведомлений) позволяет
  привязанному клиенту прямо в Telegram:
  - получить ссылку подписки, ссылки по каждому inbound (vless/vmess/...) и
    **QR-коды** (рендерятся на сервере картинками);
  - смотреть **текущую статистику** - использовано/лимит с прогресс-баром,
    сколько дней до окончания, онлайн-статус и суммарный трафик.
- **Сопоставление Telegram ID ↔ клиент на отдельной странице.** Админ
  сопоставляет Telegram-пользователя с клиентом на новой странице **Платные
  подписки** (отдельный пункт левого меню). Карточка клиента и таблица `clients`
  ядра **не** трогаются; привязки хранятся в отдельной таблице.
- **Саморегистрация с пробным периодом.** Если включено, неизвестный
  пользователь, открывший бота, автоматически регистрируется: создаётся клиент с
  выбранными админом Inboundsи и настраиваемым пробным периодом (дни +
  опционально трафик) и привязывается к этому Telegram-пользователю. Защита -
  глобальный лимит и ограничение частоты `/start` на пользователя.
- **Оплата по тарифам, мультипровайдер.** Админ задаёт тарифы (название, цена,
  +дни, +трафик) на странице «Платные подписки». Клиент покупает/продлевает в
  боте; при подтверждённой оплате подписка продлевается автоматически (срок +
  трафик, повторное включение). Провайдеры выбираемы, можно включить несколько
  сразу: **Telegram Stars (XTR), YooKassa, Stripe, CryptoBot и внешняя ссылка на
  оплату**. Продление идемпотентно (без двойного начисления при повторах/гонках),
  суммы сверяются на сервере со снимком заказа, а нулевые тарифы не дают
  продления.
- **Изолированный, снимаемый модуль.** Всё перечисленное - в отдельном пакете
  `paidsub`, подключённом за единственным флагом `paidSubEnabled`, со своими
  HTTP-эндпоинтами и своими таблицами БД (создаются идемпотентно при старте).
  UI - отдельная lazy-страница, помеченная как *experimental*.

## Безопасность

- Клиентский бот использует **отдельный, шифрованный** токен; все токены
  платёжных провайдеров хранятся в зашифрованном виде и маскируются в API/UI.
- Хосты API провайдеров захардкожены, токены не пишутся в логи. Бот определяет
  клиента только по привязке (никогда не доверяет id из сообщений) и продлевает
  только привязанного клиента.
- Для продакшена задайте переменную окружения **`SUI_SECRETBOX_KEY`**, чтобы
  секреты шифровались ключом вне базы (UI предупреждает, если она не задана).

## Обновление

Ручная миграция не нужна, данные сохраняются, функция **выключена по
умолчанию**. Чтобы попробовать: откройте **Платные подписки** в панели, включите
бота, вставьте токен бота (от @BotFather) и задайте домен подписки
(`subDomain`/`subURI`), чтобы формировались ссылки. Это бета - сначала
протестируйте на некритичном экземпляре.
