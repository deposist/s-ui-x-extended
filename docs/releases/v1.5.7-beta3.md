# Release Notes: v1.5.7-beta3

Release date: 2026-06-04

Bug-fix beta for the experimental **Paid Subscriptions** module. Most
it also fixes the admin write path (bindings, tariffs and broadcast did
not actually save in beta2), plus several reliability fixes. Feature remains
**off by default**; no core schema migration.

## What changed

- **Fix: the Paid Subscriptions admin page now works end-to-end.** Two issues
  made it non-functional before: (1) write requests (`/api/paidsub/*`: bindings,
  tariffs, broadcast) were sent as `x-www-form-urlencoded` while the backend
  parsed JSON, and (2) every paidsub response omitted empty `msg`/`obj` keys,
  which the frontend rejected as "unknown data" - so even reads (bindings,
  tariffs, orders) came back empty. Requests now send JSON and responses always
  include the `success`/`msg`/`obj` envelope, so the page loads and saves
  correctly.
- **Fix: no spurious auto-registration on a transient DB error.** `/start` only
  auto-registers a new trial client on a genuine "not found"; a transient
  database error no longer risks creating-and-rebinding a new client over an
  existing subscription.
- **Fix: connection leak in the bot poll loop.** The bot rebuilt its HTTP client
  each poll cycle; discarded proxy/outbound transports now have their idle
  connections closed, preventing a slow socket leak when a proxy or outbound is
  configured.
- **Hardening:** the bot command rate-limiter refuses new keys when saturated
  (bounded memory under a burst); CryptoBot invoice ids are URL-escaped; very
  long link lists are hard-split below Telegram's message limit; the custom
  greeting is defensively truncated.
- **Payments: PayMaster provider** added (alongside YooKassa, Stripe, Telegram
  Stars, CryptoBot, external link). It uses Telegram-native invoicing with a
  `provider_token` from @BotFather; configure it on the Payments tab.
- **Fix:** the Orders table now shows Telegram Stars (XTR) amounts as whole units
  (a 1-Star order was shown as "0.01 XTR" because every amount was divided by 100).

## Upgrade

No manual migration; existing data is preserved and the feature stays **disabled
by default**. If you were on v1.5.7-beta2, upgrade to use the Paid Subscriptions
admin page (bindings/tariffs/broadcast) - those actions only work from beta3.

---

# Примечания к релизу: v1.5.7-beta3

Дата релиза: 2026-06-04

Багфикс-бета экспериментального модуля **«Платные подписки»**. Главное -
исправлен путь записи в админке (в beta2 привязки, тарифы и рассылка фактически
не сохранялись), плюс несколько правок надёжности. Функция по-прежнему
**выключена по умолчанию**; миграции схемы ядра нет.

## Что изменилось

- **Исправление: страница «Платные подписки» теперь работает целиком.** Раньше
  её ломали две причины: (1) запросы записи (`/api/paidsub/*`: привязки, тарифы,
  рассылка) слались как `x-www-form-urlencoded`, а бэкенд парсит JSON, и (2) все
  paidsub-ответы опускали пустые ключи `msg`/`obj`, которые фронтенд отбраковывал
  как «unknown data» - поэтому даже чтение (привязки, тарифы, заказы) возвращалось
  пустым. Теперь запросы шлются как JSON, а ответы всегда содержат конверт
  `success`/`msg`/`obj` - страница загружается и сохраняет корректно.
- **Исправление: нет ложной авто-регистрации при транзиентной ошибке БД.**
  `/start` авто-регистрирует пробного клиента только при реальном «не найдено»;
  временная ошибка БД больше не приводит к созданию и перепривязке нового
  клиента поверх существующей подписки.
- **Исправление: утечка соединений в цикле опроса бота.** Бот пересоздавал
  HTTP-клиент каждый цикл; у выброшенных proxy/Outbound transports теперь
  закрываются idle-соединения - нет медленной утечки сокетов при настроенном
  прокси/Outbound.
- **Усиление:** rate-лимитер команд бота отказывает новым ключам при
  переполнении (ограниченная память при всплеске); `invoice_ids` CryptoBot
  экранируются в URL; длинные списки ссылок жёстко режутся под лимит сообщения
  Telegram; кастомное приветствие защитно усечено.
- **Оплата: добавлен провайдер PayMaster** (рядом с YooKassa, Stripe, Telegram
  Stars, CryptoBot, внешней ссылкой). Работает через Telegram-инвойсы с
  `provider_token` из @BotFather; настраивается на вкладке «Оплата».
- **Исправление:** в таблице Orders суммы Telegram Stars (XTR) показываются
  целыми (заказ на 1 звезду отображался как «0.01 XTR», т.к. сумма делилась на 100).

## Обновление

Ручная миграция не нужна, данные сохраняются, функция **выключена по умолчанию**.
Если вы были на v1.5.7-beta2 - обновитесь, чтобы пользоваться страницей «Платные
подписки» (привязки/тарифы/рассылка): эти действия работают только с beta3.
