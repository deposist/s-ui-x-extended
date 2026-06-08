# Changelog — s-ui-x-extended

Format: [Keep a Changelog](https://keepachangelog.com/) · Versioning: [SemVer](https://semver.org/).

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

[1.0.0-beta3]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta3
[1.0.0-beta2]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta2
[1.0.0-beta1]: https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.0-beta1
