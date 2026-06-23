# Release Notes: v1.5.7-beta4

Release date: 2026-06-04

Two changes on top of v1.5.7-beta3: the experimental **Paid Subscriptions** bot
gains a structured **Payment** section with self-service **refunds** (plus an
admin refund tool), and a panel-wide **duplicate-creation fix** so a single
action can never create two records. Paid Subscriptions remains **off by
default**; no core schema migration.

## What changed

### Paid Subscriptions

- **Bot menu: a "Payment" section.** The single "Buy / Renew" button is replaced
  by a **Payment** menu that opens a submenu with three items: **Buy / Renew**
  (the existing purchase flow), **My purchases**, and **Request a refund**. The
  **Stats** button is renamed to **My subscription** (person icon) - the subscription
  view itself is unchanged.
- **My purchases.** A read-only list of the requesting user's own orders (tariff,
  amount, status, date), scoped strictly by Telegram user id.
- **Refunds.** Telegram Stars are refunded automatically via the Bot API
  (`refundStarPayment`) when the user taps *Request a refund*. Every other
  provider (YooKassa, Stripe, PayMaster, CryptoBot, external link) instead sends
  the administrator a refund request, because the Bot API offers no fiat/crypto
  refund - the money is returned in the provider's own dashboard.
- **Admin refund tool.** The *Orders* tab gains a **Refund** action: for Stars it
  calls `refundStarPayment`; for other providers it marks the order `refunded`.
  A per-refund toggle controls whether the granted days/traffic are revoked.
- **Refund rollback policy.** A new setting `paidSubRefundRevoke` (default on)
  governs the bot's user-initiated Stars refund: on success it also rolls back
  the days and traffic that order granted - `expiry -= addDays` (floored at now),
  `volume -= addTrafficBytes` (floored at 0) - to prevent buy-use-refund abuse.
  The rollback is idempotent and never disables the client. The user does not
  choose this; the admin sets the policy globally and picks per-refund in the panel.
- **Hardening.** The admin Orders API no longer serializes the Telegram charge id
  or the invoice idempotency key to the browser. A concurrent bot/panel refund
  that returns "already refunded" is treated as success (Stars refunds are
  charge-idempotent).

### Reliability

- **Fix: duplicate creation from double-submitted saves.** Saving an entity
  (client, inbound, outbound, ...) synchronously restarts the sing-box core before
  the request responds, so the save is slow; a second submission during that
  window (a re-click on a not-yet-disabled button, or any client resend) created
  a **duplicate row**. Fixed at two layers:
  - **Frontend:** the Save button is disabled while a save is in flight and the
    handler ignores re-entry, across all create/edit modals (Client, Inbound,
    Outbound, Service, and the bulk add/edit dialogs), with the loading flag
    always reset afterwards (also fixes a latent stuck-loading on early return).
  - **Backend (authoritative):** the server skips an identical create (same
    user + object + action + payload) while the first request is still **in
    flight (any duration - it covers a slow core restart)** and for a short
    window after it completes. A failed save is retryable immediately.

  Result: one action creates exactly one record, even under a slow core restart.

## Upgrade

No manual migration; existing data is preserved and Paid Subscriptions stays
**disabled by default**. The order `status` column gains a new value `refunded`;
no schema change is required. The duplicate-create guard is in-memory and needs
no configuration.

---

# Примечания к релизу: v1.5.7-beta4

Дата релиза: 2026-06-04

Два изменения поверх v1.5.7-beta3: в экспериментальном боте **«Платные
подписки»** появляется структурированный раздел **«Оплата»** с самостоятельными
**возвратами** (плюс инструмент возврата для админа), и **общепанельное
исправление задвоения** - одно действие больше не может создать две записи.
«Платные подписки» по-прежнему **выключены по умолчанию**; миграции схемы ядра нет.

## Что изменилось

### Платные подписки

- **Меню бота: раздел «Оплата».** Единственная кнопка «Купить / Продлить»
  заменена разделом **«Оплата»**, открывающим подменю из трёх пунктов: **Купить /
  Продлить** (существующий путь покупки), **Мои покупки** и **Оформить возврат**.
  Кнопка **«Статистика»** переименована в **«Моя подписка»** (иконка профиля) - сам
  экран подписки не изменён.
- **Мои покупки.** Read-only список заказов *самого* пользователя (тариф, сумма,
  статус, дата), строго в рамках его Telegram id.
- **Возвраты.** Telegram Stars возвращаются автоматически через Bot API
  (`refundStarPayment`) по нажатию *Оформить возврат*. Остальные провайдеры
  (YooKassa, Stripe, PayMaster, CryptoBot, внешняя ссылка) вместо этого
  отправляют админу заявку - в Bot API нет возврата для фиата/крипты, деньги
  возвращаются в кабинете провайдера.
- **Инструмент возврата у админа.** Во вкладке *Orders* появилось действие
  **«Возврат»**: для Stars вызывает `refundStarPayment`; для прочих помечает
  заказ `refunded`. Переключатель на каждый возврат определяет, отзывать ли
  выданные дни/трафик.
- **Политика отката возврата.** Новая настройка `paidSubRefundRevoke` (по
  умолчанию вкл.) управляет пользовательским авто-возвратом Stars из бота: при
  успехе он также откатывает выданные этим заказом дни и трафик - `expiry -=
  addDays` (не ниже now), `volume -= addTrafficBytes` (не ниже 0) - против абуза
  «купил → вернул → пользуюсь». Откат идемпотентен и не отключает клиента.
  Пользователь этот выбор не делает; админ задаёт политику глобально и выбирает
  per-refund в панели.
- **Усиление.** Админский Orders API больше не сериализует в браузер Telegram
  charge id и idempotency-ключ инвойса. Параллельный возврат из бота/панели с
  ответом «already refunded» трактуется как успех (возврат Stars идемпотентен на
  уровне charge).

### Надёжность

- **Исправление: задвоение создания при двойной отправке.** Сохранение сущности
  (клиент, Inbound, Outbound, ...) синхронно перезапускает ядро sing-box до ответа,
  поэтому сохранение «медленное»; повторная отправка в это окно (повторный клик
  по ещё не заблокированной кнопке или любой клиентский resend) создавала
  **дубликат**. Исправлено на двух уровнях:
  - **Фронтенд:** кнопка «Сохранить» блокируется на время запроса, а обработчик
    игнорирует повторный вход - во всех модалках создания/редактирования (Client,
    Inbound, Outbound, Service и массовые add/edit), с гарантированным сбросом
    флага загрузки (заодно исправлен латентный баг «залипшей» загрузки при раннем
    возврате).
  - **Бэкенд (авторитетно):** сервер пропускает идентичный create (тот же
    пользователь + объект + действие + payload), пока первый запрос **в полёте
    (любой длительности - покрывает долгий рестарт ядра)** и в коротком окне после
    его завершения. Неуспешное сохранение можно повторить сразу.

  Итог: одно действие создаёт ровно одну запись даже при долгом рестарте ядра.

## Обновление

Ручная миграция не нужна, данные сохраняются, «Платные подписки» **выключены по
умолчанию**. У колонки `status` заказа появляется новое значение `refunded`;
изменение схемы не требуется. Защита от задвоения - in-memory, настройка не нужна.
