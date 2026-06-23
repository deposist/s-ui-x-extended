# Release Notes: v1.5.8-beta3

Release date: 2026-06-14

This beta adds **TLS certificates for bare IP addresses** - issue a real Let's
Encrypt certificate for an IP with no domain name, and have the panel renew it
automatically - mirroring 3x-ui's "Get SSL for IP Address (6-day cert,
auto-renews)", but done **entirely in-process** so it works the same on Linux,
Windows and in Docker with no `acme.sh` to install. It also relocates **Config
Doctor** into Settings → Maintenance and streamlines the Nexus Home overview. It
is additive: no database migration, no configuration changes.

## What changed

### IP-address TLS certificates (issue + auto-renew)

- A new **IP certificate** card in **Settings → Maintenance** issues a Let's
  Encrypt TLS certificate for a bare **IP address** (RFC 8738 / the `shortlived`
  certificate profile) - no domain required. Like everything else in s-ui-x the
  ACME exchange runs **in-process** via `go-acme/lego` (the panel never shells
  out to `acme.sh`), so the feature behaves identically across Linux, Windows
  and Docker.
- The ACME challenge is **HTTP-01 standalone** on a configurable port
  (default 80).
- The issued certificate can be applied to either:
  - the **panel's own HTTPS listener** (`webCertFile`/`webKeyFile`); the panel
    restarts to load it, or
  - an **inbound TLS profile**, applied through the hot-reload path so only the
    affected inbounds reload - **no core restart** (recommended).
- **Auto-renew:** a background job checks every 12 hours and re-issues when
  fewer than 72 hours of validity remain (shortlived certs live ~160h ≈
  6.7 days, so a missed run still leaves margin). It also re-issues
  automatically when you point it at a **different target IP**, so the live
  certificate never keeps a stale address.
- The certificate and key are written under `<db>/certs` (key file `0600`). The
  ACME account key is **encrypted at rest** and is never returned by the
  settings API or shown in the UI.
- **Safety:** the target IP is validated against private, loopback, link-local,
  CGNAT, cloud-metadata and reserved ranges (no DNS lookups) on **both** manual
  issue and auto-renew; issuing is a privileged, audited action.

### Config Doctor moves to Settings → Maintenance

- **Config Doctor** now lives in **Settings → Maintenance**, next to Backup and
  the new IP-certificate card, and is shared by both the Nexus and the classic
  layout. It previously appeared in two separate places - the Nexus Home
  overview and a separate inline copy on the classic Home page - which are both
  removed in favour of this single location. Behaviour is unchanged: it is still
  a **read-only dry-check** that assembles the full sing-box config and
  constructs a sing-box instance without starting or restarting the running
  core.

### Nexus Home overview, streamlined

- The Nexus Home overview drops the standalone **System Status** panel and the
  Config Doctor card. The IPv4/IPv6 addresses now appear inside the live-traffic
  KPI tile, and **Recent events** shows 10 entries (was 6). The result is a less
  cluttered Home that leads with KPIs, top clients and recent events.

### Hardening and fixes

- **IP-change re-issue.** Auto-renew now also re-issues when the configured
  target IP differs from the certificate on disk; previously a changed IP could
  keep serving a certificate with the old address until expiry.
- **Stricter issue-time validation.** A direct issue request now validates the
  ACME email and the challenge port with the same rules the settings page uses,
  so an out-of-band issue can no longer persist a malformed value. Email
  validation now uses the Go standard-library mail parser (rejecting `@`-only
  inputs, embedded control characters and display-name forms) instead of a
  bare "contains @" check.
- **Missing icon.** A `shield-check` glyph used by the IP-certificate card was
  not registered in the Lucide icon map and would have rendered blank; it is now
  mapped.

## Verification

- Backend: `go build ./...`, `go vet ./service ./api ./cronjob` and
  `go test ./service ./api ./cronjob` all green. New unit tests cover the renew
  decision, IP/email/port validation, the ACME issue / account-key-reuse /
  renew / issued-but-apply-failed orchestration, the certificate store and PEM
  parsing (including non-leaf and multi-block bundles), the settings round-trip,
  and the TLS server-block patch.
- Frontend: `vue-tsc --noEmit`, `eslint` and `vitest` (25 files / 129 tests,
  including locale parity and the Lucide icon-map scan) all green.

No database migration. Applying a certificate to the panel restarts the panel;
applying to an inbound TLS profile is hot and avoids the restart. Issuance uses
Let's Encrypt production; the `SUI_ACME_DIR_URL` environment variable can point
at staging/Pebble for testing. Binding the challenge port 80 needs privileges
on Linux - use a custom port or a reverse proxy if the panel does not run as
root.

---

# Примечания к релизу: v1.5.8-beta3

Дата релиза: 2026-06-14

Эта бета добавляет **TLS-сертификаты для голых IP-адресов** - выпуск настоящего
сертификата Let's Encrypt для IP без доменного имени и его автоматический
перевыпуск панелью - повторяя пункт 3x-ui «Get SSL for IP Address (6-day cert,
auto-renews)», но **полностью in-process**, поэтому это одинаково работает на
Linux, Windows и в Docker без установки `acme.sh`. Также **Config Doctor**
переезжает в Settings → Maintenance, а обзор главной Nexus становится чище.
Релиз аддитивный: без миграций базы и без изменений конфигурации.

## Что изменилось

### TLS-сертификаты для IP-адреса (выпуск + автоперевыпуск)

- Новая карточка **IP certificate** в **Settings → Maintenance** выпускает
  TLS-сертификат Let's Encrypt для голого **IP-адреса** (RFC 8738 / профиль
  сертификата `shortlived`) - домен не нужен. Как и всё остальное в s-ui-x,
  обмен ACME идёт **in-process** через `go-acme/lego` (панель никогда не шеллит
  `acme.sh`), поэтому фича ведёт себя одинаково на Linux, Windows и в Docker.
- ACME-challenge - **HTTP-01 standalone** на настраиваемом порту (по
  умолчанию 80).
- Выпущенный сертификат можно применить к одному из двух:
  - **HTTPS-листенеру самой панели** (`webCertFile`/`webKeyFile`); панель
    перезапускается, чтобы его загрузить, либо
  - **inbound TLS-профилю** - через путь горячей перезагрузки, при котором
    перезагружаются только затронутые Inbounds, **без перезапуска ядра**
    (рекомендуется).
- **Автоперевыпуск:** фоновая задача проверяет каждые 12 часов и перевыпускает,
  когда остаётся меньше 72 часов срока действия (shortlived-сертификаты живут
  ~160 ч ≈ 6.7 дня, так что пропуск одного прогона всё равно оставляет запас).
  Она также автоматически перевыпускает при смене **целевого IP**, поэтому
  живой сертификат никогда не хранит устаревший адрес.
- Сертификат и ключ пишутся в `<db>/certs` (файл ключа `0600`). Ключ
  ACME-аккаунта **шифруется на диске** и никогда не возвращается API настроек и
  не показывается в UI.
- **Безопасность:** целевой IP проверяется на private-, loopback-, link-local-,
  CGNAT-, cloud-metadata- и зарезервированные диапазоны (без DNS-резолва) - и
  при ручном выпуске, и при автоперевыпуске; выпуск - привилегированное действие
  с аудитом.

### Config Doctor переезжает в Settings → Maintenance

- **Config Doctor** теперь живёт в **Settings → Maintenance**, рядом с Backup и
  новой карточкой IP-сертификата, и общий для Nexus- и классической раскладки.
  Раньше он был в двух разных местах - обзор главной Nexus и отдельная встроенная
  копия на классической главной - оба убраны в пользу этого единого места.
  Поведение прежнее: это по-прежнему **read-only dry-check**, который собирает
  полный конфиг sing-box и создаёт инстанс sing-box без запуска или перезапуска
  работающего ядра.

### Обзор главной Nexus стал чище

- Обзор главной Nexus убирает отдельную панель **System Status** и карточку
  Config Doctor. Адреса IPv4/IPv6 теперь показаны внутри KPI-плитки живого
  трафика, а **Recent events** показывает 10 записей (было 6). В итоге главная
  менее загромождена и ведёт с KPI, топ-клиентов и недавних событий.

### Усиление надёжности и исправления

- **Перевыпуск при смене IP.** Автоперевыпуск теперь перевыпускает и тогда,
  когда заданный целевой IP отличается от сертификата на диске; раньше при смене
  IP мог до самой экспирации отдаваться сертификат со старым адресом.
- **Строже валидация при выпуске.** Прямой запрос выпуска теперь валидирует
  email ACME и порт challenge теми же правилами, что и страница настроек,
  поэтому внеполосный выпуск больше не сохранит некорректное значение. Валидация
  email теперь использует парсер почты из стандартной библиотеки Go (отклоняет
  ввод вида одного `@`, встроенные control-символы и формы с display-name)
  вместо простой проверки «содержит @».
- **Отсутствующая иконка.** Глиф `shield-check`, используемый карточкой
  IP-сертификата, не был зарегистрирован в карте иконок Lucide и отрисовался бы
  пустым; теперь он замаплен.

## Проверка

- Backend: `go build ./...`, `go vet ./service ./api ./cronjob` и
  `go test ./service ./api ./cronjob` - всё зелёное. Новые unit-тесты покрывают
  решение о перевыпуске, валидацию IP/email/port, оркестрацию ACME (выпуск /
  переиспользование ключа аккаунта / перевыпуск / issued-but-apply-failed),
  хранилище сертификатов и парсинг PEM (включая не-leaf и multi-block bundle),
  round-trip настроек и патч server-блока TLS.
- Frontend: `vue-tsc --noEmit`, `eslint` и `vitest` (25 файлов / 129 тестов,
  включая locale parity и скан карты иконок Lucide) - всё зелёное.

Миграций базы нет. Применение сертификата к панели перезапускает панель;
применение к inbound TLS-профилю - горячее и перезапуска избегает. Выпуск
использует Let's Encrypt production; переменная окружения `SUI_ACME_DIR_URL`
может указывать на staging/Pebble для тестов. Бинд порта 80 для challenge на
Linux требует привилегий - используйте свой порт или reverse-proxy, если панель
работает не от root.
