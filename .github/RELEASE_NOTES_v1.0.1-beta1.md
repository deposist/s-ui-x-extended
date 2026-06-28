# S-UI-X-Extended v1.0.1-beta1

Hardening release. No database migration is required. All changes are
backward-compatible.

## Shell scripts

- `s-ui.sh` no longer uses `eval` when installing a custom panel version. The
  version string is now validated against a strict format and passed as a
  positional argument to the installer. This eliminates a root-level code
  execution vector that could be reached through menu item 3 (Custom version).

- `s-ui uri` no longer contacts external IP echo services when a web domain or
  listen address is not configured. Instead, it prints local interface
  addresses and an explicit note to configure webDomain or webListen in
  settings. This prevents server IP metadata from being sent to third parties
  without operator awareness.

- All positional arguments in `systemctl`, `journalctl`, and service helper
  functions are now properly quoted. Command arguments in `s-ui setting` and
  `config_after_install` are passed as safe arrays rather than concatenated
  strings.

- `read -r` is used throughout both scripts, preventing unintended backslash
  interpretation on interactive input.

- Hardcoded tilde paths (`~/.acme.sh`) replaced with `${HOME}/.acme.sh` for
  clarity and compatibility.

## Panel and API

- Setting `trafficAge` to `0` is now rejected. Previously, this value
  silently deleted all traffic statistics, which could lead to irreversible
  data loss with a single configuration change.

- Import from 3x-ui: when an inbound is skipped by strategy, its client link
  mappings are now preserved. Previously, skipped inbounds left clients with
  empty inbound references, breaking the client-inbound relationship after
  import.

- SubSecret generation is now atomic. Under concurrent subscription requests,
  all callers converge on the same persisted secret. Previously, two
  simultaneous requests could generate different secrets with only the last
  writer persisted, causing some users to receive a subscription link pointing
  to a secret that was never saved.

- Backend broadcast delivery and the UI recipient count now use the same
  filter: disabled clients and clients with expired subscriptions are excluded
  from both the count and the actual send. Previously they could appear in the
  count but not receive the message.

- Telegram bot: `sendPhoto` and `sendDocument` now parse the JSON response for
  `ok:false` instead of relying on HTTP status alone. Bot token values are no
  longer embedded in logged error messages.

- API error responses now redact sensitive internal details before being sent
  to the client.

- IP monitoring: the initial `installSalt` bootstrap is hardened against
  concurrent access.

- Retries for Cloudflare WARP API requests now respect context cancellation,
  preventing goroutine leaks during shutdown.

- Cron job scheduler now propagates a single context to all jobs and cancels
  it on `Stop()`, ensuring background work terminates cleanly when the panel
  shuts down.

## Validation improvements

- `paidSubCurrency` now accepts the full set of currencies shown in the
  frontend picker: RUB, USD, EUR, GBP, UAH, KZT, BYN, XTR. The previous
  allowlist was narrower than the UI, causing silent failures when a currency
  offered in the dropdown was rejected by the backend.

- `paidSubAutoInbounds`: duplicate inbound IDs and non-existent inbound
  references are now rejected.

- `paidSubExternalUrlTemplate`: validation now blocks local IP addresses,
  control characters, HTTP userinfo credentials, and fragments in template
  URLs.

- `saveBinding`: empty or non-numeric Telegram user ID values are now
  rejected with an error notification instead of silently becoming `0`.

## Frontend

- Nexus is now the only selectable and default UI mode when the Nexus feature
  gate is enabled. Persisted classic selections are normalized back to Nexus,
  and the visible UI mode switch has been removed.

- Authenticated shell selection now resolves to the Nexus shell only for the
  current release scope. Classic shell/shared modal internals remain in the
  codebase and were not deleted in this pass.

- Restored the QUIC transport editor in the web UI as a deprecated-visible
  option for legacy `v2rayquic` compatibility. Operators are warned to prefer
  TUIC or Hysteria2 unless QUIC is explicitly required.

- Bulk outbound save (`OutboundBulk.vue`) now correctly awaits all save
  operations before resetting the loading state. Previously, the loading
  indicator could disappear while saves were still in progress, leaving the
  operator unaware of failures.

- Settings page: the restart button now reloads from the current browser
  location rather than constructing a URL from unsaved form values, which
  could redirect to a wrong address.

- Settings page: the configuration loading poll is bounded by a 10-second
  timeout, preventing an infinite wait if the store never loads.

- Paid subscriptions page: tariff deletion now asks for confirmation before
  proceeding. Loading and status errors are surfaced as notifications.

- TLS drawer: PEM boundary detection uses substring matching instead of exact
  string comparison, making it tolerant of minor formatting variations from
  the backend.

- Rule import: the URL is validated before the fetch request, rejecting
  non-HTTP(S) protocols.

- Inbound drawer: dirty-form detection now includes client user list changes.

- Protocol coverage additions from the prior implementation wave remain part
  of this release.

## Miscellaneous

- `scripts/audit/validate-workflows.js` removed (was not invoked by any CI
  workflow).

- Redundant panic recovery removed from the sing-box startup error path.

- Random name generation for auto-registered clients is more robust: every
  candidate name is verified against the database before use, and the
  function returns an explicit error instead of a potentially conflicting
  name when all retries are exhausted.

---

# S-UI-X-Extended v1.0.1-beta1

Hardening-релиз. Миграция базы не требуется. Все изменения обратно совместимы.

## Shell-скрипты

- `s-ui.sh` больше не использует `eval` при установке произвольной версии
  панели. Строка версии валидируется по строгому формату и передается как
  позиционный аргумент установщику. Это убирает вектор выполнения кода на
  уровне root, доступный через пункт меню 3 (Пользовательская версия).

- `s-ui uri` больше не обращается к внешним IP-сервисам, если не настроен
  домен или адрес прослушивания. Вместо этого команда печатает адреса
  локальных интерфейсов и явную подсказку настроить webDomain или webListen.
  Это предотвращает передачу IP-метаданных сервера третьим сторонам без
  ведома оператора.

- Все позиционные аргументы в `systemctl`, `journalctl` и вспомогательных
  функциях служб теперь закавычены. Аргументы команд `s-ui setting` и
  `config_after_install` передаются безопасными массивами вместо склеенных
  строк.

- `read -r` используется повсеместно в обоих скриптах, предотвращая
  непреднамеренную интерпретацию обратных слешей при интерактивном вводе.

- Захардкоженные пути с тильдой (`~/.acme.sh`) заменены на `${HOME}/.acme.sh`
  для ясности и совместимости.

## Панель и API

- Значение `trafficAge=0` теперь отклоняется. Ранее это значение скрыто
  удаляло всю статистику трафика, что могло привести к необратимой потере
  данных одной сменой настройки.

- Импорт из 3x-ui: при пропуске inbound по стратегии сохраняются его
  привязки к клиентам. Ранее пропущенные inbound оставляли клиентов с
  пустыми ссылками, разрушая связь клиент-inbound после импорта.

- Генерация SubSecret теперь атомарна. При конкурентных запросах на подписку
  все вызывающие стороны сходятся на одном сохраненном секрете. Ранее два
  одновременных запроса могли сгенерировать разные секреты, и только
  последний записавшийся сохранялся, из-за чего часть пользователей получала
  ссылку на секрет, который никогда не был сохранен.

- Backend-доставка broadcast и счетчик получателей в UI используют один
  фильтр: отключенные клиенты и клиенты с истекшей подпиской исключаются и
  из подсчета, и из фактической отправки. Ранее они могли отображаться в
  счетчике, но не получать сообщение.

- Telegram-бот: `sendPhoto` и `sendDocument` теперь разбирают JSON-ответ на
  `ok:false`, а не полагаются только на HTTP-статус. Токены бота больше не
  попадают в логируемые сообщения об ошибках.

- Ответы API с ошибками теперь скрывают чувствительные внутренние детали
  перед отправкой клиенту.

- IP-мониторинг: начальная инициализация `installSalt` защищена от
  конкурентного доступа.

- Повторы запросов к Cloudflare WARP API теперь учитывают отмену контекста,
  предотвращая утечку горутин при завершении работы.

- Планировщик cron-задач пробрасывает единый контекст всем задачам и
  отменяет его при `Stop()`, обеспечивая чистое завершение фоновой работы
  при остановке панели.

## Улучшения валидации

- `paidSubCurrency` теперь принимает полный набор валют из выпадающего списка
  фронтенда: RUB, USD, EUR, GBP, UAH, KZT, BYN, XTR. Предыдущий allowlist
  был уже списка в UI, из-за чего валюта, показанная в выпадающем списке,
  молча отвергалась бэкендом.

- `paidSubAutoInbounds`: повторяющиеся ID inbound и ссылки на несуществующие
  inbound теперь отклоняются.

- `paidSubExternalUrlTemplate`: валидация теперь блокирует локальные
  IP-адреса, управляющие символы, HTTP userinfo и фрагменты в URL шаблона.

- `saveBinding`: пустые и нечисловые значения Telegram user ID теперь
  отклоняются с уведомлением об ошибке, вместо того чтобы молча становиться
  `0`.

## Фронтенд

- Массовое сохранение outbound (`OutboundBulk.vue`) теперь корректно
  дожидается всех операций сохранения перед сбросом состояния загрузки.
  Ранее индикатор загрузки мог исчезнуть, пока сохранения ещё выполнялись,
  оставляя оператора в неведении об ошибках.

- Страница настроек: кнопка перезапуска теперь перезагружает страницу из
  текущего адреса браузера, а не строит URL из несохраненных значений формы,
  которые могли направить на неверный адрес.

- Страница настроек: цикл опроса загрузки конфигурации ограничен 10-секундным
  таймаутом, предотвращая бесконечное ожидание.

- Страница платных подписок: удаление тарифа теперь запрашивает
  подтверждение. Ошибки загрузки настроек и статуса отображаются как
  уведомления.

- Редактор TLS: определение границ PEM использует поиск подстроки вместо
  точного сравнения строк, что делает его устойчивым к небольшим вариациям
  форматирования со стороны бэкенда.

- Импорт правил: URL валидируется перед fetch-запросом, не-HTTP(S) протоколы
  отклоняются.

- Редактор inbound: отслеживание изменений формы теперь включает изменения
  списка пользователей.

## Прочее

- `scripts/audit/validate-workflows.js` удален (не вызывался ни одним CI
  сценарием).

- Убрана избыточная обработка паники в пути ошибки запуска sing-box.

- Генерация имен для авторегистрируемых клиентов стала надежнее: каждое
  имя-кандидат проверяется на уникальность в базе перед использованием, а
  при исчерпании всех попыток функция возвращает явную ошибку вместо
  потенциально конфликтующего имени.
