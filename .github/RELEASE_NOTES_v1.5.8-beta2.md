# Release Notes: v1.5.8-beta2

Release date: 2026-06-12

This beta adds a **Personal Ops Pack** - an operations and diagnostics layer
aimed at RU/ZH single-panel admins. It introduces a Config Doctor on the Home
page, per-client "why doesn't it work" diagnosis, a named RU/ZH routing/DNS
preset gallery, and a reworked subscription delivery dialog, plus the security
and correctness hardening found while reviewing the feature. It is additive:
new read-only diagnostic endpoints and frontend surfaces, with no database
migration and no configuration changes.

## What changed

### Config Doctor (Home)

- A new **Run Doctor** panel on the Home page - in both the Nexus and the
  classic layout - assembles the full sing-box config from the database and
  dry-checks it **without starting or restarting the running core**: it parses
  the JSON/options and constructs a sing-box instance via `NewBox` (no listener
  binds, no outbound dials, no rule-set downloads), then closes it. The report
  lists `ok`/`warn`/`error` items for: config build, the dry-check, core
  running state, DNS `final`/rule server references, route `final`/rule outbound
  references, remote rule-set URL shape, subscription URI/formats/secret health,
  recent core warnings, and outbound reachability. It never mutates the
  configuration - it is report-only.

### Client diagnosis ("why doesn't this client work")

- A **Diagnose** action on each client row opens a report covering: enabled /
  expired / traffic-limit state, inbound membership (including stale inbound
  ids), stored delivery links, subscription secret and format availability,
  core running state, the current online signal, and outbound reachability. An
  optional connectivity target is validated as an SSRF-safe HTTPS URL before any
  probe (see Hardening).

### RU/ZH routing & DNS preset gallery

- The Rules and DNS pages gain a **preset gallery** with named `.srs`
  sources (SagerNet `sing-geosite`/`sing-geoip`, runetfreedom
  `russia-blocked-geoip`). Presets add or update `route.rule_set`,
  `route.rules`, `dns.servers`, `dns.rules`, and enable
  `experimental.cache_file` - applied only to the **local, unsaved** config with
  a preview diff. You still pick your own proxy and direct Outbound tags; no tags
  are hardcoded. The gallery warns when the proxy and direct Outbound are the
  same, since that disables the split the preset exists for.

### Subscription delivery UX

- The client QR dialog is reworked into a **Delivery** dialog with per-platform
  tabs (sing-box, Clash/Mihomo, Hiddify-compatible, raw links), per-platform
  copy/QR, the visible subscription URL, and a reachability "Test URL" action.
- Settings now exposes the previously hidden subscription controls: `subTitle`,
  `subSupportUrl`, `subProfileUrl`, `subAnnounce`, the per-format enable
  toggles, `subSecretRequired`, the per-IP rate limit, and name-in-remark. These
  keys already existed in the backend; only the UI was missing.

### Hardening and fixes

- **SSRF guard on the client-diagnose probe.** The diagnosis endpoint accepted a
  caller-supplied connectivity target and probed it through every outbound
  (including the always-present direct dialer) without validation, while the
  sibling outbound-check endpoints already reject private/loopback/metadata
  targets. The target is now validated with the same guard - HTTPS only, no
  userinfo, no internal/metadata addresses - before any probe.
- **More accurate diagnostics.** Subscription-format checks no longer report
  "all formats disabled" when the settings read itself failed (they warn
  instead); outbound probes cancelled by the doctor's own time budget are now
  reported as "not tested" rather than counted as failures.
- **Preset safety.** Each preset is bound to its apply function in the catalog,
  so a preset can never be listed yet silently do nothing; the "Test URL"
  success message now states that it checks reachability only, not subscription
  validity.

## Verification

- Backend: `go test ./api ./service ./sub ./core`, full `go test ./...` (the
  only failures are known Windows-only test-teardown flakes that pass in
  isolation), with `go vet` and `staticcheck` clean on the changed packages.
- Frontend: `npm run lint`, `npm run test`, `npm run build`, and the
  `personal-ops-pack` Playwright e2e - all green.
- New tests pin the doctor's read-only dry-check contract (no listener bind, no
  rule-set download), the client traffic-limit boundaries, and per-side preset
  routing.

No database migration. The Config Doctor and client diagnosis are read-only and
do not change your configuration.

---

# Примечания к релизу: v1.5.8-beta2

Дата релиза: 2026-06-12

Эта бета добавляет **Personal Ops Pack** - слой эксплуатации и диагностики,
ориентированный на RU/ZH-админов одиночной панели. Появляются Config Doctor на
главной странице, диагностика клиента «почему не работает», галерея именованных
RU/ZH-пресетов маршрутизации и DNS и переработанное окно доставки подписок, а
также исправления по безопасности и корректности, найденные при ревью фичи.
Релиз аддитивный: новые read-only диагностические эндпоинты и экраны фронтенда,
без миграций базы и без изменений конфигурации.

## Что изменилось

### Config Doctor (главная)

- Новый блок **Run Doctor** на главной странице - и в Nexus-, и в классической
  раскладке - собирает полный конфиг sing-box из базы и проверяет его **без
  запуска или перезапуска работающего ядра**: парсит JSON/опции и собирает
  инстанс sing-box через `NewBox` (без bind'а слушателей, без дозвона
  Outbounds, без загрузки rule-set'ов), затем закрывает его. Отчёт перечисляет
  пункты `ok`/`warn`/`error` для: сборки конфига, dry-check, состояния ядра,
  ссылок DNS `final`/server в правилах, ссылок route `final`/outbound в правилах,
  формы URL удалённых rule-set'ов, здоровья URI/форматов/секрета подписки,
  недавних предупреждений ядра и доступности Outbounds. Конфиг при этом не
  меняется - только отчёт.

### Диагностика клиента («почему этот клиент не работает»)

- Действие **Diagnose** в строке клиента открывает отчёт: включён / истёк /
  лимит трафика, привязка к Inbounds (включая устаревшие id), сохранённые
  ссылки доставки, наличие секрета и форматов подписки, состояние ядра, текущий
  сигнал online и доступность Outbounds. Опциональная цель проверки
  связности валидируется как SSRF-безопасный HTTPS-URL до любого зондирования
  (см. «Усиление надёжности»).

### Галерея RU/ZH пресетов маршрутизации и DNS

- На страницах Rules и DNS появляется **галерея пресетов** с именованными
  `.srs`-источниками (SagerNet `sing-geosite`/`sing-geoip`, runetfreedom
  `russia-blocked-geoip`). Пресеты добавляют/обновляют `route.rule_set`,
  `route.rules`, `dns.servers`, `dns.rules` и включают `experimental.cache_file`
  - применяются только к **локальной несохранённой** конфигурации с preview diff.
  Proxy- и direct Outbound теги выбираешь сам, ничего не зашито. Галерея
  предупреждает, когда proxy и direct Outbound совпадают, потому что это
  отключает разделение, ради которого пресет и существует.

### UX доставки подписок

- Окно QR клиента переработано в окно **Delivery** с вкладками по платформам
  (sing-box, Clash/Mihomo, Hiddify-совместимый, raw links), копированием/QR по
  каждой платформе, видимым URL подписки и проверкой доступности «Test URL».
- В настройках теперь показаны ранее скрытые элементы подписки: `subTitle`,
  `subSupportUrl`, `subProfileUrl`, `subAnnounce`, переключатели включения
  форматов, `subSecretRequired`, лимит запросов на IP и имя в remark. Эти ключи
  уже были в backend - не хватало только UI.

### Усиление надёжности и исправления

- **SSRF-защита проверки в диагностике клиента.** Эндпоинт диагностики принимал
  заданную вызывающим цель проверки связности и зондировал её через каждый
  outbound (включая всегда присутствующий direct-диалер) без валидации, тогда
  как соседние эндпоинты проверки Outbounds уже отклоняют
  private/loopback/metadata-цели. Теперь цель валидируется тем же guard'ом -
  только HTTPS, без userinfo, без внутренних/metadata-адресов - до любого
  зондирования.
- **Точнее диагностика.** Проверки форматов подписки больше не сообщают «все
  форматы отключены», когда упало само чтение настроек (вместо этого - warning);
  проверки Outbounds, отменённые собственным таймбюджетом доктора, теперь
  помечаются как «не проверено», а не считаются провалом.
- **Безопасность пресетов.** Каждый пресет привязан к своей apply-функции в
  каталоге, поэтому пресет не может быть в списке и при этом молча ничего не
  делать; сообщение об успехе «Test URL» теперь явно говорит, что проверяется
  только доступность, а не валидность подписки.

## Проверка

- Backend: `go test ./api ./service ./sub ./core`, полный `go test ./...`
  (единственные падения - известные Windows-флейки на teardown тестов, которые
  проходят изолированно), `go vet` и `staticcheck` чисто на изменённых пакетах.
- Frontend: `npm run lint`, `npm run test`, `npm run build` и Playwright e2e
  `personal-ops-pack` - всё зелёное.
- Новые тесты пинят read-only-контракт dry-check доктора (без bind'а слушателя
  и без загрузки rule-set'а), границы лимита трафика клиента и пораздельную
  маршрутизацию пресетов.

Миграций базы нет. Config Doctor и диагностика клиента работают только на
чтение и не меняют конфигурацию.
