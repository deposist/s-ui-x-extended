# Changelog — s-ui-x-extended

Format: [Keep a Changelog](https://keepachangelog.com/) · Versioning: [SemVer](https://semver.org/).

## [1.0.0-beta5] — 2026-06-09

Unifies the navigation menu so the **Classic** and **Nexus** UI environments
always show the same tabs. The two shells previously kept independent,
hand-maintained menu lists that had drifted apart — *Providers* showed only in
Classic, *Paid Subscriptions* only in Nexus. They now read a single shared
source, so neither environment can silently lose a tab again.

### English

#### Changed

- **Single source of truth for the sidebar menu.** New
  `frontend/src/layouts/menu.ts` exports one `appMenu`; both the Classic drawer
  (`layouts/default/Drawer.vue`) and the Nexus sidebar
  (`layouts/nexus/NexusSidebar.vue`) consume it. `nexusMenu.ts` is now a thin
  re-export kept for backward compatibility, and the nexus-only `singBoxSettings`
  metadata lives in the shared list.

#### Fixed

- **Classic and Nexus tabs no longer diverge.** Both environments now expose the
  same 16 tabs in the same order — Classic gains *Paid Subscriptions* and Nexus
  gains *Providers* (previously each was missing one). `/migrate-xui` remains a
  contextual page reached from the Backup dialog, not a top-level tab, as before.

#### Added

- **Menu localization parity.** `pages.providers` and `pages.paidSub` are now
  translated in `zhcn`, `zhtw`, `fa` and `vi` (they fell back to English before).
- **Drift-guard test** (`frontend/src/layouts/menu.test.ts`) pins the menu's path
  set, order, uniqueness and the nexus sing-box surfaces.

### Русский

#### Изменено

- **Единый источник правды для бокового меню.** Новый
  `frontend/src/layouts/menu.ts` экспортирует один `appMenu`; его используют и
  Classic-drawer (`layouts/default/Drawer.vue`), и Nexus-sidebar
  (`layouts/nexus/NexusSidebar.vue`). `nexusMenu.ts` стал тонким ре-экспортом для
  обратной совместимости, а nexus-only метаданные `singBoxSettings` теперь живут
  в общем списке.

#### Исправлено

- **Вкладки Classic и Nexus больше не расходятся.** Оба окружения показывают
  одинаковые 16 вкладок в одном порядке — Classic получил *Paid Subscriptions*,
  Nexus получил *Providers* (раньше у каждого не хватало одной). `/migrate-xui`
  по-прежнему контекстная страница из диалога Backup, а не вкладка верхнего уровня.

#### Добавлено

- **Паритет локализации меню.** `pages.providers` и `pages.paidSub` переведены в
  `zhcn`, `zhtw`, `fa` и `vi` (раньше показывался английский fallback).
- **Тест против дрейфа** (`frontend/src/layouts/menu.test.ts`) фиксирует набор
  путей меню, порядок, уникальность и nexus sing-box-поверхности.

## [1.0.0-beta4] — 2026-06-09

Makes the extended protocols actually usable end-to-end. The panel could already
*configure* ssh / mieru / sudoku / trusttunnel / mtproxy inbounds, but had nothing
to hand the client — the JSON subscription was empty for them and mtproxy couldn't
even start. This release fixes delivery and consolidates protocol knowledge into a
single source of truth.

### English

#### Added

- **Client delivery for ssh / mieru / sudoku / trusttunnel.** These inbounds now
  emit a working client outbound in the JSON subscription (previously the server
  config was generated but the subscription dropped them). Per-user credentials are
  mapped correctly (`name → username` for mieru/trusttunnel, `name → user` for ssh).
- **MTProxy via Telegram.** mtproxy inbounds now produce a `tg://proxy` deep link
  (`clientDelivery: telegram`); they are intentionally excluded from the JSON/Clash
  subscriptions (there is no sing-box mtproxy outbound).
- **Single source of truth for protocol capabilities.** New embedded manifest
  `core/capabilities/protocols.json` drives the backend maps, the frontend lists,
  the `docs/protocol-matrix.md` table, and a new **admin-only `/api/capabilities`**
  endpoint. The inbound-type picker now greys out protocols not compiled into the
  running binary (detected via `//go:build` flags, not by parsing build scripts).
- **C-side out_json editors** for mieru / sudoku / trusttunnel in the inbound modal.

#### Fixed

- **MTProxy could not start.** The panel generated a bare-hex `secret`, but the
  core (`mtglib`) requires a faketls (`ee`) secret — `0xee || 16-byte key ||
  faketls SNI host`. New clients now get a valid `ee` secret (crypto-random key +
  recommended SNI front), so the mtproxy inbound starts and the tg link matches.
- **Empty subscription for extended protocols.** `FillOutJson` previously wiped
  out_json to `{}` for ssh/mieru/sudoku/trusttunnel, so the subscription's `len < 5`
  guard dropped them.

#### Security

- **No server-secret leakage.** Each out_json builder is a strict allow-list
  (ssh copies *nothing* — private host keys never leave the server). A
  forbidden-keys invariant test recursively scans every out_json and subscription
  body and fails on TLS keys, reality private keys, ssh `host_key*`, server
  `fallback`/`handshake_timeout`, etc.; a static test forbids a builder from
  range-copying the whole inbound.
- **`/api/capabilities` is admin-authenticated** and returns only boolean build-tag
  flags and UI capability metadata — no paths, versions, builder names or secrets.

#### Diagnostics

- **ShadowTLS is marked `broken`** (not delivered as working): the panel never
  creates the required backing-shadowsocks detour, and the core inbound is not
  fail-closed without one. Auto-pair is deferred; see `docs/project-context.md`.

### Русский

#### Добавлено

- **Доставка клиенту для ssh / mieru / sudoku / trusttunnel.** Эти inbound'ы теперь
  отдают рабочий клиентский outbound в JSON-подписке (раньше серверный конфиг
  генерился, но подписка их выкидывала). Пользовательские креды маппятся корректно
  (`name → username` для mieru/trusttunnel, `name → user` для ssh).
- **MTProxy через Telegram.** mtproxy-инбаунды теперь дают `tg://proxy`-ссылку
  (`clientDelivery: telegram`); из JSON/Clash-подписок исключены намеренно (sing-box
  mtproxy-outbound не существует).
- **Единый источник правды о возможностях протоколов.** Новый встроенный манифест
  `core/capabilities/protocols.json` питает backend-карты, frontend-списки, таблицу
  `docs/protocol-matrix.md` и новый **admin-only эндпоинт `/api/capabilities`**.
  Селектор типа inbound теперь гасит протоколы, не вкомпилированные в текущий бинарь
  (детект через `//go:build`-флаги, а не парсинг build-скриптов).
- **Редакторы C-side out_json** для mieru / sudoku / trusttunnel в модалке inbound.

#### Исправлено

- **MTProxy не запускался.** Панель генерила bare-hex `secret`, а ядро (`mtglib`)
  требует faketls (`ee`) формат — `0xee || 16-байт ключ || faketls SNI-хост`. Новые
  клиенты получают валидный `ee`-secret (крипто-ключ + рекомендованный SNI), так что
  инбаунд стартует, а tg-ссылка совпадает с серверным секретом.
- **Пустая подписка для расширенных протоколов.** `FillOutJson` раньше затирал
  out_json до `{}` для ssh/mieru/sudoku/trusttunnel, и гард подписки `len < 5` их
  выкидывал.

#### Безопасность

- **Без утечки серверных секретов.** Каждый out_json-билдер — строгий allow-list
  (ssh не копирует *ничего* — приватные host-ключи не покидают сервер).
  Forbidden-keys инвариант-тест рекурсивно сканирует весь out_json и тела подписок и
  падает на TLS-ключах, reality private_key, ssh `host_key*`, серверных
  `fallback`/`handshake_timeout` и т.д.; статик-тест запрещает билдеру копировать
  весь inbound через range.
- **`/api/capabilities` под admin-аутентификацией**, отдаёт только bool-флаги
  build-тегов и UI-метаданные — без путей, версий, имён билдеров и секретов.

#### Диагностика

- **ShadowTLS помечен `broken`** (не отдаётся как рабочий): панель не создаёт нужный
  backing-shadowsocks detour, а core-инбаунд без него не fail-closed. Auto-pair
  отложен; см. `docs/project-context.md`.

## [Unreleased] — planned 1.0.0 GA

> Draft for the eventual 1.0.0 GA — not yet tagged or released. The latest
> released build is `1.0.0-beta5` above.

First stable release — the s-ui-x web panel on the
[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) core
(`shtorm-7/sing-box-extended`, a fork of `SagerNet/sing-box`). Consolidates
beta1–beta3 into one stable build.

### English

#### Added

- **Extended protocol & transport set.** OpenVPN, MASQUE, MTProxy, TrustTunnel,
  WireGuard/AmneziaWG, CCM/OCM, DHCP, QUIC, mKCP/XHTTP, providers, and
  rate/traffic/bandwidth/connection limiters.
- **Recommended defaults across the whole admin panel.** Creating inbounds,
  outbounds, endpoints, DNS servers, services, TLS templates, transports and
  routing rules now opens with security-first, ready-to-use values instead of
  blank fields — `bbr` congestion control for TUIC/Naive/TrustTunnel, `xudp`
  packet encoding for VLESS/VMess, `auto` security for VMess, `AES-256-GCM` /
  `SHA256` for OpenVPN, the core's recommended AEAD / padding for Sudoku, and
  `min_version: 1.3` for new TLS templates. Shared via
  `frontend/src/types/recommended.ts`.
- **Dropdowns for fixed-value fields.** `v-select` for congestion controls, Mieru
  transport / multiplexing, Sudoku AEAD / mask modes, SOCKS version, Tun stack,
  TLS cipher suites, and more — an invalid token can no longer be typed by hand.
- **Editable suggestion comboboxes for free-text fields.** `v-combobox` for Go
  durations, byte-size quotas, bandwidth speeds, listen addresses, time zones,
  NTP servers, DNS resolvers, SSH versions / algorithms, health-check URLs, and
  more — a sensible default that still accepts custom input.

#### Fixed

- **`database` token scope.** API tokens scoped to `database` now actually grant
  database export/import (`getdb`/`importdb`) and x-ui / 3x-ui migration
  (`import-xui`); previously these were unintentionally admin-only because
  `database` was passed only as the audit resource name and never as an allowed
  scope.

#### Build / Packaging

- **Full protocol tag set in every build path.** Prebuilt Linux tarballs and the
  Windows packages now ship the same protocols as the Docker image / `build.sh` —
  WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS and
  CCM/OCM/OOMKiller no longer hit the "not included in this build, rebuild with
  -tags ..." stub at runtime. Only Naive (cronet/CGO) still varies by platform —
  present on Linux amd64/arm64/armv7/armv6/386, Docker and Windows amd64; absent
  on armv5/s390x and Windows arm64.

#### Security

- **TLS 1.3 minimum** (`min_version: 1.3`) for new TLS templates.
- All option tokens were verified against the `sing-box-extended` core; secrets
  are never hard-coded (UUIDs / passwords / keys are still generated); camouflage
  targets (Reality dest / ShadowTLS handshake / SNI) are intentionally left empty.
- Inherited supply-chain and reliability hardening — see
  [`SECURITY.md`](SECURITY.md).

#### Docs

- `docs/scope-matrix.md` corrected to all six token scopes (`admin`, `read`,
  `write`, `database`, `telegram`, `observability`) with per-endpoint gates;
  README restructured and fact-checked (HTTP API, migration, backup, Telegram,
  paid subscriptions, security & hardening, monitoring, transports/TLS, build
  matrix).

### Русский

#### Добавлено

- **Расширенный набор протоколов и транспортов.** OpenVPN, MASQUE, MTProxy,
  TrustTunnel, WireGuard/AmneziaWG, CCM/OCM, DHCP, QUIC, mKCP/XHTTP, провайдеры и
  лимитеры скорости/трафика/полосы/соединений.
- **Рекомендуемые настройки по всей админке.** Создание inbound'ов,
  outbound'ов, endpoint'ов, DNS-серверов, сервисов, TLS-шаблонов, транспортов и
  правил маршрутизации теперь начинается с безопасных, готовых к работе значений,
  а не с пустых полей — congestion control `bbr` для TUIC/Naive/TrustTunnel,
  `xudp` для VLESS/VMess, `auto` для VMess, `AES-256-GCM` / `SHA256` для OpenVPN,
  рекомендованные ядром AEAD / padding для Sudoku и `min_version: 1.3` для новых
  TLS-шаблонов. Реализовано через общий `frontend/src/types/recommended.ts`.
- **Выпадающие списки для полей с фиксированным набором значений.** `v-select` для
  congestion control, Mieru transport / multiplexing, Sudoku AEAD / режимов маски,
  версии SOCKS, стека Tun, cipher suites TLS и др. — невалидный токен ввести
  вручную нельзя.
- **Редактируемые комбобоксы с подсказками для свободного ввода.** `v-combobox`
  для Go-длительностей, квот размера, скоростей, listen-адресов, часовых поясов,
  NTP-серверов, DNS-резолверов, версий / алгоритмов SSH, URL health-check и др. —
  разумное значение по умолчанию, при этом принимает произвольный ввод.

#### Исправлено

- **Scope `database`.** API-токены со scope `database` теперь действительно дают
  экспорт/импорт базы (`getdb`/`importdb`) и миграцию x-ui / 3x-ui
  (`import-xui`); раньше это было доступно только `admin`, потому что `database`
  передавался лишь как имя ресурса аудита, а не как допустимый scope.

#### Сборка / Упаковка

- **Полный набор тегов протоколов во всех путях сборки.** Готовые Linux-tarball'ы
  и пакеты Windows теперь несут те же протоколы, что Docker / `build.sh` —
  WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS и
  CCM/OCM/OOMKiller больше не упираются в заглушку «not included in this build,
  rebuild with -tags ...» в рантайме. По-платформенно различается только Naive
  (cronet/CGO) — есть на Linux amd64/arm64/armv7/armv6/386, в Docker и на Windows
  amd64; отсутствует на armv5/s390x и Windows arm64.

#### Безопасность

- **Минимум TLS 1.3** (`min_version: 1.3`) для новых TLS-шаблонов.
- Все токены значений сверены с ядром `sing-box-extended`; секреты не хардкодятся
  (UUID'ы / пароли / ключи по-прежнему генерируются); camouflage-цели (Reality
  dest / ShadowTLS handshake / SNI) намеренно оставлены пустыми.
- Унаследованный харднинг цепочки поставок и надёжности — см.
  [`SECURITY.md`](SECURITY.md).

#### Документация

- `docs/scope-matrix.md` исправлен до всех шести scope (`admin`, `read`, `write`,
  `database`, `telegram`, `observability`) с пер-эндпоинт гейтами; README
  реструктурирован и выверен (HTTP API, миграции, бэкап, Telegram, платные
  подписки, безопасность и харднинг, мониторинг, транспорты/TLS, матрица сборки).

## [1.0.0-beta3] — 2026-06-08

### English

- **Recommended defaults pre-filled across the admin panel.** Creating inbounds,
  outbounds, endpoints, DNS servers, services, TLS templates, transports and
  routing rules now starts from security-first, ready-to-use values instead of
  blank fields — e.g. TUIC / Naive / TrustTunnel default to `bbr` congestion
  control, VLESS / VMess to `xudp` packet encoding, VMess to `auto` security,
  OpenVPN to `AES-256-GCM` / `SHA256`, and Sudoku to the core's recommended
  AEAD / padding.
- **Dropdowns for fixed-value fields.** Parameters that accept only a fixed set
  of values are now `v-select` dropdowns (congestion controls, Mieru transport /
  multiplexing, Sudoku AEAD / mask modes, SOCKS version, Tun stack, TLS cipher
  suites, …) so an invalid token can no longer be typed by hand.
- **Editable suggestion comboboxes for free-text fields.** Free-text parameters
  with common values now offer an editable `v-combobox` (Go durations, byte-size
  quotas, bandwidth speeds, listen addresses, time zones, NTP servers, DNS
  resolvers, SSH versions / algorithms, health-check URLs, …) with a sensible
  default while still accepting custom input.
- **TLS 1.3 minimum** for new TLS templates (`min_version: 1.3`).
- All option tokens were verified against the `sing-box-extended` core; secrets
  are never hard-coded and camouflage targets (Reality dest / ShadowTLS
  handshake / SNI) are intentionally left empty.

### Русский

- **Рекомендуемые настройки предзаполнены по всей админке.** Создание inbound'ов,
  outbound'ов, endpoint'ов, DNS-серверов, сервисов, TLS-шаблонов, транспортов и
  правил маршрутизации теперь начинается с безопасных, готовых к работе значений,
  а не с пустых полей — например, TUIC / Naive / TrustTunnel по умолчанию
  используют congestion control `bbr`, VLESS / VMess — `xudp`, VMess — `auto`,
  OpenVPN — `AES-256-GCM` / `SHA256`, Sudoku — рекомендованные ядром AEAD / padding.
- **Выпадающие списки для полей с фиксированным набором значений.** Параметры,
  принимающие только определённый набор значений, теперь оформлены как `v-select`
  (congestion control, Mieru transport / multiplexing, Sudoku AEAD / режимы маски,
  версия SOCKS, стек Tun, cipher suites TLS, …) — невалидный токен ввести нельзя.
- **Редактируемые комбобоксы с подсказками для свободного ввода.** Текстовые поля
  с типовыми значениями теперь предлагают редактируемый `v-combobox` (Go-длительности,
  квоты размера, скорости, listen-адреса, часовые пояса, NTP-серверы, DNS-резолверы,
  версии / алгоритмы SSH, URL health-check, …) с разумным значением по умолчанию,
  при этом принимая произвольный ввод.
- **Минимум TLS 1.3** для новых TLS-шаблонов (`min_version: 1.3`).
- Все токены значений сверены с ядром `sing-box-extended`; секреты не хардкодятся,
  а camouflage-цели (Reality dest / ShadowTLS handshake / SNI) намеренно оставлены пустыми.

## [1.0.0-beta2] — 2026-06-08

### English

- **Full protocol tag set in every build path.** Prebuilt Linux tarballs and the
  Windows packages now ship the same protocols as the Docker image / `build.sh` —
  WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS and
  CCM/OCM/OOMKiller no longer hit the "not included in this build" stub at runtime.
  Only Naive (cronet/CGO) still varies by platform.
- **Fix — `database` token scope.** API tokens scoped to `database` now actually
  grant database export/import (`getdb`/`importdb`) and x-ui / 3x-ui migration
  (`import-xui`); previously these were admin-only because `database` was passed
  only as the audit resource name and never as an allowed scope.
- **Docs.** `docs/scope-matrix.md` corrected to document all six token scopes
  (`admin`, `read`, `write`, `database`, `telegram`, `observability`) and the
  per-endpoint gates. The README was restructured and fact-checked, adding HTTP API,
  migration, backup, Telegram, paid-subscription, security & hardening, monitoring,
  and transport/TLS sections.

### Русский

- **Полный набор тегов протоколов во всех путях сборки.** Готовые Linux-tarball'ы и
  пакеты Windows теперь несут те же протоколы, что Docker / `build.sh` — WireGuard/WARP,
  MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS и CCM/OCM/OOMKiller больше не
  упираются в заглушку «not included in this build». По-платформенно различается только
  Naive (cronet/CGO).
- **Исправление — scope `database`.** API-токены со scope `database` теперь
  действительно дают экспорт/импорт базы (`getdb`/`importdb`) и миграцию x-ui / 3x-ui
  (`import-xui`); раньше это было доступно только `admin`, потому что `database`
  передавался лишь как имя ресурса аудита, а не как допустимый scope.
- **Документация.** `docs/scope-matrix.md` исправлен (все шесть scope:
  `admin`, `read`, `write`, `database`, `telegram`, `observability`) и пер-эндпоинт
  гейты. README реструктурирован и выверен: добавлены разделы HTTP API, миграции,
  бэкапа, Telegram, платных подписок, безопасности и харднинга, мониторинга,
  транспортов/TLS.

## [1.0.0-beta1] — 2026-06-08

Initial commit.

### English

First public build of **s-ui-x-extended** — the s-ui-x panel running on the
[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended) core.
Includes extended protocol/transport support (OpenVPN, MASQUE, MTProxy,
TrustTunnel, WireGuard/AmneziaWG, CCM/OCM, DHCP, QUIC, mKCP/XHTTP, providers,
rate/traffic/bandwidth/connection limiters, and more) plus the inherited
security and reliability hardening. See [`SECURITY.md`](SECURITY.md) for
supply-chain and hardening notes.

### Русский

Первая публичная сборка **s-ui-x-extended** — панель s-ui-x на ядре
[`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended).
Включает расширенную поддержку протоколов/транспортов (OpenVPN, MASQUE, MTProxy,
TrustTunnel, WireGuard/AmneziaWG, CCM/OCM, DHCP, QUIC, mKCP/XHTTP, провайдеры,
лимитеры скорости/трафика/полосы/соединений и др.) и унаследованный харденинг
безопасности и надёжности. См. [`SECURITY.md`](SECURITY.md).

[1.0.0]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0
[1.0.0-beta5]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta5
[1.0.0-beta4]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta4
[1.0.0-beta3]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta3
[1.0.0-beta2]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta2
[1.0.0-beta1]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta1
