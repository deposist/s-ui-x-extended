# Release Notes: v1.5.7-beta5

Release date: 2026-06-04

A UI-only beta on top of v1.5.7-beta4: it polishes the experimental **Paid
Subscriptions** admin page. The **Bindings** tab becomes the first (default) tab
and the **Bot** tab moves last, unbinding a Telegram id now asks for
**confirmation**, and both the **Bindings** and **Orders** tables gain client
columns. Paid Subscriptions remains **off by default**; no schema migration.

## What changed

### Paid Subscriptions (admin UI)

- **Tab order + Bindings columns.** The Paid Subscriptions tabs are reordered so
  **Bindings** is first (and opens by default) and **Bot** is last (after
  *Orders*). The Bindings table gains three columns: **Client ID**,
  **Description**, and **Expiry**. Expiry shows the date/time plus a
  remaining-days chip (green = unlimited, red = expired), reusing the same
  formatter as the Clients page.
- **Unbind confirmation.** Removing a client's Telegram binding now opens a
  **confirmation dialog** ("Unbind Telegram from ...?") instead of unbinding on the
  first click; the link is cleared only after you confirm. The client itself stays
  in the panel - only the binding is removed.
- **Orders columns.** The Orders table now shows **Client name** (replacing the
  bare numeric client id), **Telegram ID**, and **Description**. The name and
  description are joined server-side from the clients table with a LEFT JOIN (a
  deleted client renders a dash). The Orders API still never serializes the
  provider charge id, the invoice idempotency key, or the provider payload.

## Upgrade

No manual migration. The new columns are read from existing client fields via a
read-only join - there is no schema change - and Paid Subscriptions stays
**disabled by default**.

---

# Примечания к релизу: v1.5.7-beta5

Дата релиза: 2026-06-04

UI-бета поверх v1.5.7-beta4: шлифует админскую страницу экспериментальных
**«Платных подписок»**. Вкладка **Bindings** становится первой (по умолчанию), а
вкладка **Bot** уезжает в конец; отвязка Telegram id теперь спрашивает
**подтверждение**; в таблицы **Bindings** и **Orders** добавлены колонки клиента.
«Платные подписки» по-прежнему **выключены по умолчанию**; миграции схемы нет.

## Что изменилось

### Платные подписки (интерфейс админки)

- **Порядок вкладок + колонки Bindings.** Вкладки «Платных подписок»
  переупорядочены: **Bindings** теперь первая (и открывается по умолчанию), а
  **Bot** - последняя (после *Orders*). В таблицу Bindings добавлены три колонки:
  **Client ID**, **Description** и **Expiry**. Expiry показывает дату/время плюс
  чип с остатком дней (зелёный = безлимит, красный = истёк), переиспользуя тот же
  форматтер, что и страница Clients.
- **Подтверждение отвязки.** Снятие Telegram-привязки клиента теперь открывает
  **диалог подтверждения** («Unbind Telegram from ...?») вместо мгновенной отвязки
  по первому клику; привязка снимается только после подтверждения. Сам клиент
  остаётся в панели - удаляется только привязка.
- **Колонки Orders.** В таблице Orders теперь показываются **имя клиента**
  (вместо голого числового id), **Telegram ID** и **описание**. Имя и описание
  подтягиваются на сервере из таблицы клиентов через LEFT JOIN (удалённый клиент
  отображается как прочерк). Orders API по-прежнему не отдаёт в браузер provider
  charge id, idempotency-ключ инвойса и provider payload.

## Обновление

Ручная миграция не нужна. Новые колонки читаются из существующих полей клиента
через read-only join - изменения схемы нет, - а «Платные подписки» остаются
**выключенными по умолчанию**.
