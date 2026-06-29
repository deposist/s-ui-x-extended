# S-UI-X-Extended v1.0.1-beta2

This release updates frontend guidance and documentation. No database migration is required.

## Dashboard

- Traffic statistics now has a timezone selector in the KPI controls. The first value comes from the browser timezone.

- The selected timezone is stored in `localStorage`, so it survives page reloads and browser restarts.

- The timezone menu opens with a short default list. The search field filters the full IANA timezone list only while text is entered.

- Traffic chart labels use `YYYY-MM-DD HH:MM` in the selected timezone. The active timezone is shown next to the range selector.

## Configuration forms

- Migrated configuration forms no longer use the generic `RecommendedValues` block. They now use field-level help and explicit preset buttons where a safe preset exists.

- Presets run only after the operator clicks the apply button in create or add mode. Edit forms are not auto-filled, and existing empty values stay empty.

- Outbound, Service, Endpoint, TLS, DNS server, DNS rule, and route rule forms now show field help. Settings, Subscription JSON, and Subscription Clash editors show field help without apply buttons.

- Presets are type-aware. They cover cases such as port `443` for common TLS or QUIC outbounds, TLS 1.3 and uTLS client defaults where the form supports them, WireGuard MTU `1420`, VPN server seed addresses, common DNS transport ports, the standard DoH path, logical OR mode for grouped rules, and `5m` UDP timeout for route options.

- Forms that depend on deployment layout no longer show broad “depends on deployment” alerts. Guidance is kept at the field level.

## Inbound setup

- Inbound forms use the same pattern with protocol-aware help. Create-mode presets are available only where the form can recommend values without guessing deployment-specific choices.

- VLESS inbound has a dedicated preset. It sets `decryption: none`, enables sniffing, enables `sniff_override_destination`, sets `sniff_timeout: 300ms`, resets transport settings, and uses `xudp` packet encoding. It does not choose TLS, Reality, or a transport type.

- Other supported inbound protocols use presets only when the defaults are safe. ShadowTLS, MTProxy, Tun, Bond, Core Failover, Redirect, Mixed, SOCKS, and HTTP show guidance without a preset button.

- New Shadowsocks and Sudoku inbounds generate credentials at creation time. ShadowTLS generates a password only for version 2, matching the version 3 form behavior.

- TLS template selection is filtered for inbound protocols. Reality templates are shown only for protocols that support them, currently VLESS and Trojan.

## Locales and tests

- New recommendation and field-help text was added to English, Russian, Persian, Vietnamese, Simplified Chinese, and Traditional Chinese locale files.

- Frontend tests cover locale coverage, recommendation helpers, inbound credential generation, TLS template compatibility, and the Dashboard timezone selector.

## Documentation

- README was reorganized into a shorter English entry point with links to guides, changelogs, and release notes.

- Added `README-RU.md` with the Russian project overview.

- The README supported-protocol list was refreshed from the repository capability matrix and configuration docs.

- README beta links now point to `v1.0.1-beta2`.

## Verification

- `cd frontend && npm run build` passed.
- `cd frontend && npm run lint` passed.
- `cd frontend && npm test` passed with 37 test files and 186 tests.
- `cd frontend && npm test -- src/components/nexus/overview/selectors/overviewSelectors.test.ts` passed before the overview commit.

## Operator notes

- No database or configuration migration is required.
- Clearing browser site storage resets the Dashboard traffic timezone selector to the browser timezone.
- Recommendation presets affect a form only after the operator clicks the apply button.
- Existing saved forms are preserved when opened for editing.

---

# S-UI-X-Extended v1.0.1-beta2

Этот релиз обновляет фронтенд и документацию. Миграция базы не требуется.

## Dashboard

- В KPI Traffic statistics добавлен выбор часового пояса. Начальное значение берется из часового пояса браузера.

- Выбранный часовой пояс сохраняется в `localStorage`, поэтому он переживает перезагрузку страницы и перезапуск браузера.

- Меню часовых поясов открывается с коротким списком. Полный список IANA фильтруется только при вводе текста в поле поиска.

- Подписи графика трафика используют формат `YYYY-MM-DD HH:MM` в выбранном часовом поясе. Активная зона показана рядом с выбором диапазона.

## Configuration forms

- Мигрированные формы больше не используют общий блок `RecommendedValues`. Вместо него используются подсказки у полей и явные кнопки preset там, где есть безопасный preset.

- Preset применяется только после клика оператора в режиме create или add. Формы редактирования не заполняются автоматически, а существующие пустые значения остаются пустыми.

- Формы Outbound, Service, Endpoint, TLS, DNS server, DNS rule и route rule теперь показывают подсказки у полей. Settings, Subscription JSON и Subscription Clash показывают подсказки без apply-кнопок.

- Preset учитывает тип формы. Он покрывает такие случаи, как порт `443` для распространенных TLS или QUIC outbounds, TLS 1.3 и uTLS client defaults там, где форма их поддерживает, MTU `1420` для WireGuard, стартовые адреса VPN server, стандартные порты DNS transport, обычный DoH path, logical OR для grouped rules и UDP timeout `5m` для route options.

- Формы, зависящие от схемы развертывания, больше не показывают широкие alerts в стиле “depends on deployment”. Подсказки оставлены на уровне конкретных полей.

## Inbound setup

- Inbound forms используют ту же схему с подсказками по протоколу. Create-mode presets доступны только там, где форма может рекомендовать значения без угадывания deployment-specific choices.

- Для VLESS inbound есть отдельный preset. Он задает `decryption: none`, включает sniffing, включает `sniff_override_destination`, ставит `sniff_timeout: 300ms`, сбрасывает transport settings и использует packet encoding `xudp`. TLS, Reality и transport type оператор выбирает сам.

- Другие поддерживаемые inbound-протоколы получают presets только там, где defaults безопасны. ShadowTLS, MTProxy, Tun, Bond, Core Failover, Redirect, Mixed, SOCKS и HTTP показывают подсказки без preset-кнопки.

- Новые Shadowsocks и Sudoku inbounds получают credentials при создании. ShadowTLS генерирует password только для version 2, что соответствует поведению формы для version 3.

- Выбор TLS template для inbound фильтруется по протоколу. Reality templates показываются только для протоколов, которые их поддерживают: сейчас это VLESS и Trojan.

## Локали и тесты

- Новый текст рекомендаций и подсказок добавлен в английскую, русскую, персидскую, вьетнамскую, упрощенную китайскую и традиционную китайскую локали.

- Frontend tests покрывают locale coverage, recommendation helpers, генерацию inbound credentials, совместимость TLS template и selector часового пояса на Dashboard.

## Документация

- README переработан в короткую английскую стартовую страницу со ссылками на guides, changelogs и release notes.

- Добавлен `README-RU.md` с русским обзором проекта.

- Список поддерживаемых протоколов в README обновлен по capability matrix и configuration docs репозитория.

- Beta-ссылки в README теперь указывают на `v1.0.1-beta2`.

## Проверка

- `cd frontend && npm run build` прошел.
- `cd frontend && npm run lint` прошел.
- `cd frontend && npm test` прошел: 37 test files, 186 tests.
- `cd frontend && npm test -- src/components/nexus/overview/selectors/overviewSelectors.test.ts` прошел перед overview commit.

## Заметки для операторов

- Миграция базы или конфигурации не требуется.
- Очистка browser site storage сбрасывает selector часового пояса Traffic statistics к часовому поясу браузера.
- Recommendation presets влияют на форму только после клика по apply button.
- Существующие сохраненные формы не меняются при открытии на редактирование.
