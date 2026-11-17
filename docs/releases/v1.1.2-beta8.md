# S-UI-X Extended v1.1.2-beta8

This beta adds fleet node management and moves the core to sing-box 1.14.4. Stable remains `v1.1.1`.

## Fleet and core

- Added the Nodes section in beta for remote servers. It shows whether a server is online and which inbounds it reported. Client assignments and limiter outbounds are managed there too.
- Client subscriptions include links for assigned nodes. A server without a client-facing address contributes no links.
- Each client can have a shared or separate download/upload speed limit and a connection limit. Zero means unlimited.
- Updated sing-box-extended to `v1.14.4-extended-2.7.10`, up from `v1.14.3-extended-2.7.8` in beta7. Release builds use the deposist forks of sing-box, sing, tailscale and sing-openvpn so the extended changes reach the published packages.

## Reliability and data

- Network reset closes tracked connections. Flow trackers release their state once when a flow closes or resets, so old flows cannot keep changing counters after a core restart.
- Panel update recovery resumes interrupted cleanup and rollback with binary hash checks. Completed updates and rollbacks no longer depend on a final cleanup call succeeding on the first try.
- Backups now include nodes, client assignments, client limits and paid-subscription charge records. Composite-key charge records are copied in full.
- Legacy migrations handle missing and null sections. DNS migration preserves HTTPS/H3 paths and existing response filters, including filters nested in logical rules.
- Subscription caches no longer keep stale entries after client changes. JSON subscriptions retain base TLS settings when an address overrides only part of the TLS block.
- Nested inbound groups and provider changes now trigger the required core reload or restart. Renaming or deleting a referenced provider is rejected instead of leaving a broken outbound reference.
- Panel and subscription HTTP servers close their listeners during shutdown and restart. Deleting a user also removes the user's traffic counter.
- Paid-subscription registration checks database errors before choosing a client name. Sudoku `keys` values are masked in configuration change records, and missing Sudoku `options` no longer break master-key generation.

## Interface and imports

- Form editors ignore responses that arrive after a newer request or after the edited object changed. New clients, inbounds, outbounds, endpoints, providers and services no longer share nested default objects.
- Bulk client editing works on copies and does not change rows before the request succeeds. Node client selection keeps existing assignments and ignores late responses from another server.
- Dashboard polling no longer overlaps requests or drops unrelated status fields. Offline and closed pages ignore late status, audit and traffic responses. The traffic time-zone setting tolerates unavailable browser storage.
- WebSocket Host editing preserves other headers. Custom header maps safely handle special names such as `__proto__`.
- SSH keeps the username field visible for key authentication. VPN pool sizes stay numeric, and new VPN users receive the first free address instead of a duplicate after deletions.
- Amnezia obfuscation results apply only to the endpoint that requested them. ACME default SNI uses the configured domain. TLS default arrays and objects are copied before editing.
- Rule editors initialize missing route and DNS sections. Removing an absent rule-set reference no longer deletes the last rule. Mieru and MASQUE forms no longer show duplicate or unsupported transport/TLS blocks. The obsolete OOM Killer `checks_before_limit` field was removed.
- Client search treats regular-expression characters as ordinary text. Compressed IPv6 addresses are masked correctly, and literal table labels such as IP, ACME, ECH and Reality no longer depend on missing translations.
- VMess import rejects missing or invalid ports and reads `allowInsecure` only from explicit true values. Empty transport blocks are omitted from VMess, VLESS and Trojan. Invalid WireGuard private keys return an error. Automatic HTTPS redirects preserve escaped paths.

## Installation and certificates

- Failed remote installation or update commands now return an error instead of reporting success. Archive path validation handles backslashes consistently.
- Certificate reissue keeps the existing certificate when issuing a new one fails. Domain names are validated before use in paths, and self-signed certificate generation passes key algorithm arguments correctly.

## Upgrade

Back up the panel database before installing this beta.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.2-beta8/install.sh \
  | sudo bash -s -- v1.1.2-beta8
```

Windows: extract the ZIP for your architecture and run `install-windows.bat` as Administrator.

Docker image: `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta8`.

Existing settings, clients and outbounds are retained. To roll back, reinstall the previous version and restore the database copy made before upgrading.

## По-русски

### Ноды и ядро

- Раздел «Ноды» находится в бета-статусе. В нём можно добавлять удалённые серверы, проверять их состояние и список входящих, назначать клиентов и выбирать outbound для ограничений.
- Подписка клиента включает подключения к назначенным серверам. Если адрес сервера для клиентов не задан, его подключения в подписку не попадают.
- Для каждого клиента задаются общий или раздельный лимит скорости на приём и отдачу, а также лимит соединений. Нулевое значение снимает ограничение.
- Ядро обновлено до sing-box-extended `v1.14.4-extended-2.7.10`; в beta7 стояла версия `v1.14.3-extended-2.7.8`. Релизные сборки используют форки `deposist/sing-box-extended`, `deposist/sing`, `deposist/tailscale` и `deposist/sing-openvpn`, чтобы доработки extended попали в готовые пакеты.

### Стабильность и данные

- Сброс сети закрывает отслеживаемые соединения. Обработчики потоков освобождают состояние один раз при закрытии или сбросе, поэтому старые потоки не меняют счётчики после перезапуска ядра.
- Восстановление после прерванного обновления продолжает очистку и откат с проверкой хешей исполняемых файлов. Завершённые обновления и откаты больше не зависят от одной удачной попытки финальной очистки.
- Резервные копии включают серверы, назначения клиентов, лимиты и записи о списаниях по платным подпискам. Записи списаний с составным ключом копируются полностью.
- Миграции старых конфигураций обрабатывают отсутствующие и пустые секции. Миграция DNS сохраняет пути HTTPS/H3 и существующие фильтры ответов, в том числе вложенные в логические правила.
- Кэш подписок не сохраняет устаревший результат после изменения клиента. JSON-подписки сохраняют базовые TLS-настройки, когда адрес переопределяет только часть TLS-блока.
- Изменения вложенных групп inbound и provider запускают нужную перезагрузку или перезапуск ядра. Переименование и удаление связанного provider отклоняются, чтобы в outbound не оставалась битая ссылка.
- HTTP-серверы панели и подписок закрывают сетевые сокеты при остановке и перезапуске. Удаление пользователя также удаляет его счётчик трафика.
- Регистрация платной подписки проверяет ошибки базы перед выбором имени клиента. Ключи Sudoku `keys` маскируются в записях изменений конфигурации. Отсутствующий объект `options` больше не вызывает ошибку при генерации master-ключа Sudoku.

### Интерфейс и импорт

- Формы игнорируют ответы, которые пришли после более нового запроса или смены редактируемого объекта. Новые клиенты, inbound, outbound, эндпоинты, провайдеры и сервисы не делят вложенные объекты по умолчанию.
- Массовое редактирование клиентов работает с копиями и не меняет строки до успешного запроса. Выбор клиентов ноды сохраняет текущие назначения и игнорирует поздние ответы от другого сервера.
- Опросы дашборда не пересекаются и не удаляют остальные поля состояния. После закрытия страницы или потери соединения поздние ответы о статусе, аудите и трафике игнорируются. Настройка часового пояса графика работает даже при недоступном хранилище браузера.
- Изменение Host в WebSocket сохраняет остальные заголовки. Пользовательские карты заголовков корректно обрабатывают специальные имена вроде `__proto__`.
- Поле имени пользователя SSH остаётся видимым при аутентификации ключом. Размер пула VPN остаётся числом, а новый VPN-пользователь получает первый свободный адрес, а не дубликат после удалений.
- Параметры обфускации Amnezia применяются только к эндпоинту, который их запросил. SNI по умолчанию в ACME использует настроенный домен. Массивы и объекты TLS по умолчанию копируются перед редактированием.
- Редакторы правил создают отсутствующие секции `route` и DNS. Удаление отсутствующей ссылки `rule_set` больше не удаляет последнее правило. В формах Mieru и MASQUE убраны дублирующиеся или неподдерживаемые transport/TLS-блоки. Устаревшее поле OOM Killer `checks_before_limit` удалено.
- Поиск клиентов воспринимает символы регулярных выражений как обычный текст. Сжатые IPv6-адреса маскируются корректно, а подписи столбцов IP, ACME, ECH и Reality не зависят от отсутствующих переводов.
- Импорт VMess отклоняет отсутствующий или некорректный порт и учитывает `allowInsecure` только при явном значении true. Пустые transport-блоки не добавляются в VMess, VLESS и Trojan. Некорректный закрытый ключ WireGuard возвращает ошибку. Автоматическое перенаправление на HTTPS сохраняет экранированные пути.

### Установка и сертификаты

- Ошибки удалённой установки и обновления больше не выглядят как успешное завершение. Проверка путей архива единообразно обрабатывает обратные слеши.
- При неудачном перевыпуске сертификата прежний сохраняется. Доменные имена проверяются до использования в путях, а генерация самоподписанного сертификата правильно передаёт аргументы алгоритма ключа.

Перед установкой сохраните резервную копию базы. Команда обновления приведена выше. Для Windows скачайте ZIP своей архитектуры и запустите `install-windows.bat` от имени администратора. Docker-образ: `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta8`.

Настройки, клиенты и outbound сохраняются. Для отката установите предыдущую версию и восстановите копию базы, сделанную до обновления. Стабильная версия остаётся `v1.1.1`.
