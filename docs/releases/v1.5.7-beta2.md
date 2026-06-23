# Release Notes: v1.5.7-beta2

Release date: 2026-06-04

Second beta of the 1.5.7 line. Builds on the experimental **Paid Subscriptions**
module from beta1: adds a Telegram **transport selector** (proxy or sing-box
outbound) for each module independently, **broadcast announcements** to all
clients, an **editable /start greeting**, and fixes the admin UI for inbound
selection and bindings. The feature remains **off by default** and isolated from
the core. No core schema migration.

## What changed

- **Telegram transport selector - per module.** Both the Paid Subscriptions bot
  and the admin Telegram module (notifications/backups) can now egress either
  through a **proxy** (http/https/socks5, with its own credentials) or through a
  configured **sing-box outbound** (routes Telegram traffic via a VPN/proxy
  outbound; requires the core to be running). The two modules are configured
  **independently** - e.g. the client bot via one outbound and admin alerts via
  a different outbound or proxy.
- **Broadcast to all clients.** A new *Messages* tab on the Paid Subscriptions
  page sends a one-off announcement to every bound Telegram user (throttled,
  with a sent/failed report and a confirmation step).
- **Editable greeting.** The message shown on `/start` to a bound client is now
  editable on the *Messages* tab; empty falls back to the built-in greeting.
- **Fixes (beta1 UI):** the *Auto-registration → Inbounds for new clients*
  dropdown now lists inbounds (it was reading the API response incorrectly), and
  the *Bindings* tab gained an explicit **Add binding** action (pick a client +
  Telegram ID) with a clear empty state.

## Security

- The Paid Subscriptions module keeps its own encrypted proxy credentials,
  separate from the admin Telegram module. Outbound transport dials through the
  running core's outbound by tag; provider/proxy tokens remain encrypted at rest
  and are never logged. Broadcast and binding endpoints are admin-only
  (session + CSRF) and audited.

## Upgrade

No manual migration; existing data is preserved and the feature stays **disabled
by default**. If you use outbound transport, ensure the core is running and the
selected outbound tag exists. This is a beta - test on a non-critical instance.

---

# Примечания к релизу: v1.5.7-beta2

Дата релиза: 2026-06-04

Вторая бета линейки 1.5.7. Развивает экспериментальный модуль **«Платные
подписки»** из beta1: добавлен **выбор транспорта** Telegram (прокси или
sing-box Outbound) независимо для каждого модуля, **рассылка** всем клиентам,
**редактируемое приветствие** `/start`, и исправлен админ-UI выбора Inbounds и
привязок. Функция по-прежнему **выключена по умолчанию** и изолирована от ядра.
Миграции схемы ядра нет.

## Что изменилось

- **Выбор транспорта Telegram - для каждого модуля.** И бот «Платных подписок»,
  и админский модуль Telegram (уведомления/бэкапы) теперь могут выходить в сеть
  либо через **прокси** (http/https/socks5, со своими реквизитами), либо через
  настроенный **sing-box Outbound** (трафик Telegram идёт через VPN/proxy Outbound;
  требуется запущенное ядро). Модули настраиваются **независимо** - например, бот
  через один Outbound, а админ-уведомления через другой Outbound или прокси.
- **Рассылка всем клиентам.** Новая вкладка *Messages* на странице «Платные
  подписки» отправляет разовое объявление всем привязанным Telegram-пользователям
  (с троттлингом, отчётом sent/failed и подтверждением).
- **Редактируемое приветствие.** Сообщение, показываемое привязанному клиенту по
  `/start`, теперь редактируется на вкладке *Messages*; пусто - используется
  встроенное приветствие.
- **Исправления (UI beta1):** выпадающий список *Auto-registration → Inbounds for
  new clients* теперь показывает Inbounds (раньше неверно читался ответ API), а на
  вкладке *Bindings* появилась явная кнопка **Add binding** (выбор клиента +
  Telegram ID) и понятное пустое состояние.

## Безопасность

- У модуля «Платные подписки» свои зашифрованные реквизиты прокси, отдельные от
  админского Telegram-модуля. Outbound-транспорт дозванивается через Outbound
  запущенного ядра по тегу; токены провайдеров/прокси хранятся в зашифрованном
  виде и не пишутся в логи. Эндпоинты рассылки и привязок доступны только
  админу (session + CSRF) и аудируются.

## Обновление

Ручная миграция не нужна, данные сохраняются, функция **выключена по умолчанию**.
Если используете Outbound transport - убедитесь, что ядро запущено и выбранный
тег Outbound существует. Это бета - сначала протестируйте на некритичном
экземпляре.
