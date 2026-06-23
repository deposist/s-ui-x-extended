# Release Notes: v1.5.8

Release date: 2026-06-15

Stable v1.5.8 consolidates v1.5.8-beta1..beta4 plus the final Nexus polish.
No manual database migration is required.

## Changes
- **No-restart apply for all managed sing-box objects.** Inbounds, outbounds,
  endpoints, and services hot-replace in the running core. Captured adapter
  references still use a full restart; route-rule and `route.final` edits stay
  hot. Unchanged sing-box settings no longer restart the core, and apply/restart
  paths are serialized.
- **Referenced-tag safety.** Deleting or renaming a referenced outbound,
  endpoint, or managed inbound is now blocked with an error listing the
  references.
- **Personal Ops Pack.** Adds read-only Config Doctor dry-checks, client
  diagnosis, RU/ZH routing and DNS preset galleries, and the platform-aware
  subscription Delivery dialog.
- **IP-address TLS certificates.** The panel can issue and auto-renew Let's
  Encrypt shortlived certificates for bare IPs via in-process ACME, with target
  IP validation, audit logging, and apply-to-panel or apply-to-inbound options.
- **IP-certificate issuance moved to terminal/CLI.** The web card and API routes
  were removed. Use `s-ui.sh` or `sui ip-cert <issue|renew|status|disable>`.
  CSR generation now uses an empty Common Name and IP-only SubjectAltName,
  fixing Let's Encrypt `badCSR` failures and auto-renew.
- **Embedded sing-box v1.13.13.** Updated from v1.13.12 and pinned to the fixed
  release commit after the upstream tag move.
- **Performance and code health.** `GET /load` is about 3.7x faster by skipping
  redundant settings seed writes. Dead code was removed; generation paths gained
  determinism, round-trip, golden, and benchmark coverage.
- **Stable Nexus polish.** Mobile topbar actions collapse into a compact menu,
  dense tables scroll horizontally on mobile, overview panels share primitives,
  Settings/Rules/DNS/Audit/Login/Admins received layout and accessibility fixes,
  and WARN logs now show warning toasts instead of false core-error toasts.

## Breaking Changes Vs v1.5.7

- Deleting or renaming a referenced outbound, endpoint, or managed inbound is
  blocked until the reference is removed or pointed elsewhere.
- IP-certificate issuance is no longer available from the web UI; use the
  terminal menu or `sui ip-cert`.

## Upgrade

Upgrade normally. There is no manual database or configuration migration. If
you use IP certificates, issue or re-issue them from the terminal menu or
`sui ip-cert`. If automation deletes or renames tags, clear references first.

Full per-beta history is in `CHANGELOG-EN.md`, `CHANGELOG-RU.md`, and
`CHANGELOG-ZH.md`.

---

# Примечания к релизу: v1.5.8

Дата релиза: 2026-06-15

Стабильная v1.5.8 объединяет v1.5.8-beta1..beta4 и финальный polish Nexus.
Ручная миграция базы данных не требуется.

## Главное

- **Применение без рестарта для всех управляемых объектов sing-box.** Inbound'ы,
  Outbounds, endpoint'ы и сервисы горячо заменяются в работающем ядре.
  Захваченные adapter-ссылки всё ещё применяются через полный рестарт;
  route-правила и `route.final` остаются горячими. Неизменённые настройки
  sing-box больше не рестартят ядро, пути apply/restart сериализованы.
- **Защита referenced tags.** Удаление или переименование outbound'а,
  endpoint'а или managed inbound'а со ссылками блокируется ошибкой со списком
  этих ссылок.
- **Personal Ops Pack.** Добавлены read-only Config Doctor, диагностика клиента,
  RU/ZH галереи routing/DNS пресетов и platform-aware окно Delivery для
  подписок.
- **TLS-сертификаты для IP-адресов.** Панель выпускает и автоматически
  перевыпускает Let's Encrypt shortlived-сертификаты для голых IP через
  in-process ACME, с валидацией IP, аудитом и применением к панели или inbound
  TLS-профилю.
- **Выпуск IP-сертификата перенесён в терминал/CLI.** Web-карточка и API
  удалены. Используйте `s-ui.sh` или
  `sui ip-cert <issue|renew|status|disable>`. CSR теперь с пустым Common Name и
  IP только в SubjectAltName, поэтому исправлены `badCSR` и автоперевыпуск.
- **Встроенный sing-box v1.13.13.** Обновление с v1.13.12 и pin на исправленный
  release commit после перемещения upstream tag.
- **Производительность и чистка кода.** `GET /load` быстрее примерно в 3.7x за
  счёт пропуска лишнего settings seed-write. Удалён мёртвый код; генерация
  закреплена determinism, round-trip, golden и benchmark тестами.
- **Stable polish Nexus.** На мобильных topbar сворачивает действия в компактное
  меню, dense tables получают горизонтальный scroll, overview-панели используют
  общие primitives, Settings/Rules/DNS/Audit/Login/Admins получили правки layout
  и accessibility, WARN-логи показываются warning toast вместо ложных core-error.

## Ломающие Изменения Относительно v1.5.7

- Удаление или переименование referenced outbound, endpoint или managed inbound
  блокируется, пока ссылка не удалена или не переведена на другой объект.
- Выпуск IP-сертификата больше не доступен из web UI; используйте терминальное
  меню или `sui ip-cert`.

## Обновление

Обновляйтесь обычным образом. Ручной миграции базы или конфигурации нет. Если
используете IP-сертификаты, выпускайте или перевыпускайте их из терминального
меню или через `sui ip-cert`. Если автоматизация удаляет или переименовывает
теги, сначала очищайте ссылки.

Полная история beta-релизов остаётся в `CHANGELOG-EN.md`, `CHANGELOG-RU.md` и
`CHANGELOG-ZH.md`.
