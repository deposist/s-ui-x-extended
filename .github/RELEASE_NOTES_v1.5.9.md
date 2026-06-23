# Release Notes: v1.5.9

Release date: 2026-06-22

Stable v1.5.9 consolidates v1.5.9-beta1..beta6. No manual database migration
is required.

## Changes
- **Web self-update from the panel UI.** A **Panel updates** card in
  **Settings → Maintenance** lets administrators check for new releases and
  apply updates in one click. Choose between the stable *Main* and bleeding-edge
  *Beta* channel. Every download is verified against the release SHA-256, and a
  failed update is rolled back automatically. All attempts are logged in the
  audit trail; the admin password is never logged.
- **Automatic outbound failover.** A new **Failover group** outbound type tests
  its members over HTTPS on an operator-defined interval and switches traffic to
  the next healthy backup when the active member stops responding. When the
  primary recovers, traffic returns after a short stabilisation delay. If every
  member is down, the group falls back to a direct connection.
- **Security and supply-chain hardening.** The TLS builder no longer panics on
  mismatched Reality or ECH settings, the installer and self-update verify TLS
  certificates, `libcronet` binaries and Docker images are pinned by hash and
  digest, and GitHub Actions steps are pinned to full commit SHAs.
- **Reliability fixes.** The 1.2→1.3 migration no longer halts on a missing
  raw-config row, listen-address fallback is restricted to loopback, auto-logout
  is now CSRF-protected POST, and a race between deplete-cron hot-reload and
  full core restart was fixed.
- **Settings UX.** Fields show defaults or recommended values as placeholders
  with `(i)` hints for web, subscription, sing-box Basics, and Telegram
  settings. Timezone, Clash API mode, and payment currency are proper controls
  instead of free-text. Login errors show inline. Routing labels correctly
  describe private-IP matching.
- **Performance.** The dashboard stats query uses a matching composite index,
  WebSocket broadcasts serialise payloads once per message, and the subscription
  server compresses responses.
- **Nexus interface polish.** Failover group editor works in the default
  interface, form spacing is consistent across drawers, WireGuard endpoint
  editor loads all fields on first open, and `(i)` hints and field labels are
  readable in the dark theme.

## Upgrade

Upgrade normally. There is no manual database or configuration migration. The
web self-update is available for installations that run the panel as a systemd
service; other setups update through the terminal menu or `s-ui.sh`.

Full per-beta history is in `CHANGELOG-EN.md`, `CHANGELOG-RU.md`, and
`CHANGELOG-ZH.md`.

---

# Примечания к релизу: v1.5.9

Дата релиза: 2026-06-22

Стабильная v1.5.9 объединяет v1.5.9-beta1..beta6. Ручная миграция базы
данных не требуется.

## Главное

- **Веб-обновление из панели.** Карточка **Обновления панели** в разделе
  **Настройки → Обслуживание** позволяет администраторам проверять новые
  релизы и применять обновления в один клик. Выбор между стабильным каналом
  *Main* и bleeding-edge *Beta*. Каждая загрузка сверяется с SHA-256 релиза,
  неудачное обновление автоматически откатывается. Все попытки записываются в
  журнал аудита; пароль администратора не логируется.
- **Автоматическое переключение Outbounds.** Новый тип **Failover group**
  проверяет участников по HTTPS с заданным интервалом и переключает трафик
  на следующий здоровый резерв, когда активный участник перестаёт отвечать.
  При восстановлении primary трафик возвращается после короткой задержки
  стабилизации. Если недоступны все — группа использует direct-соединение.
- **Безопасность и цепочка поставок.** TLS-сборщик больше не паникует на
  несогласованных Reality или ECH настройках, установщик и самообновление
  проверяют TLS-сертификаты, `libcronet` и Docker-образы закреплены по хешу
  и digest, шаги GitHub Actions привязаны к полным commit SHA.
- **Исправления надёжности.** Миграция 1.2→1.3 больше не останавливается на
  отсутствующей raw-config строке, fallback listen-адресов ограничен loopback,
  авто-выход теперь CSRF-защищённый POST, убрана гонка между hot reload из
  deplete-cron и полным рестартом ядра.
- **UX настроек.** Поля показывают default/рекомендуемые значения как
  placeholder с `(i)`-подсказками для web, подписок, sing-box Basics и
  Telegram. Часовой пояс, режим Clash API и валюта платежей — нормальные
  контролы вместо свободного текста. Ошибки входа — inline. Routing labels
  корректно описывают матчинг приватных IP.
- **Производительность.** Запрос статистики дашборда использует составной
  индекс, WebSocket-рассылки сериализуют payload один раз, сервер подписок
  сжимает ответы.
- **Nexus polish.** Редактор failover-групп работает в основном интерфейсе,
  межполевые отступы единообразны, редактор WireGuard-эндпойнта загружает
  все поля сразу, `(i)`-подсказки и подписи полей читаемы в тёмной теме.

## Обновление

Обновляйтесь обычным образом. Ручной миграции базы или конфигурации нет.
Веб-обновление доступно для установок с systemd-службой; остальные обновляются
через терминальное меню или `s-ui.sh`.

Полная история beta-релизов остаётся в `CHANGELOG-EN.md`, `CHANGELOG-RU.md` и
`CHANGELOG-ZH.md`.
