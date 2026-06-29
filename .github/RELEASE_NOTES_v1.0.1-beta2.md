# S-UI-X-Extended v1.0.1-beta2

Frontend and documentation update. No database migration is required.

## Dashboard

- Dashboard Traffic statistics now has a timezone selector. The default value
  comes from the browser timezone.

- The selected timezone is saved in `localStorage`, so the Dashboard keeps the
  same statistics timezone after reloads and across browser sessions.

- Traffic chart labels now use `YYYY-MM-DD HH:MM`. The selected timezone is
  shown in the KPI metadata next to the range selector, which makes bucket
  times easier to read when the browser and server use different timezones.

## Guided inbound setup

- Add inbound forms now show protocol-aware recommendations as explicit
  actions. The UI does not apply these values automatically. Operators choose
  when to apply the suggested settings.

- The VLESS inbound form has a dedicated recommended setup action. It sets a
  safe baseline for new VLESS inbounds: `decryption: none`, sniffing enabled,
  `sniff_override_destination` enabled, `sniff_timeout: 300ms`, clean transport
  settings, and `xudp` packet encoding. It does not choose TLS, Reality, or a
  transport type for the operator.

- Other supported inbound protocols now have the same create-mode pattern:
  field hints plus an Apply inbound recommendations button where a safe preset
  exists. Protocols that need an operator decision, including ShadowTLS,
  MTProxy, Tun, Bond, and Core Failover, show guidance only.

- Existing inbound configs are not rewritten in edit mode. Empty advanced
  fields stay empty unless the operator changes them.

- New Shadowsocks and Sudoku inbounds now generate credentials at creation
  time. ShadowTLS generates a password only for version 2, matching the current
  version 3 form behavior.

- TLS template selection is filtered for inbounds. Reality templates are shown
  only where the protocol supports them, currently VLESS and Trojan.

- Field help now includes recommended values or clear guidance to leave a field
  empty or disabled. The new inbound help text is available in all supported UI
  locales.

## Documentation

- README now documents the Dashboard Traffic statistics timezone selector,
  including the browser default, `localStorage` persistence,
  `YYYY-MM-DD HH:MM` chart labels, and the KPI timezone label.

- The README supported-protocol list was refreshed from the repository
  capability and configuration docs.

## Verification

- Frontend build, lint, and the full frontend test suite passed before this
  release draft was updated.

## Operator notes

- No configuration or database migration is required for this release.
- Clearing site storage resets the Dashboard timezone selector to the browser
  timezone.
- Recommendation buttons affect only newly created inbound forms and only after
  the operator clicks them.
- Existing inbound configurations are preserved.

---

# S-UI-X-Extended v1.0.1-beta2

Обновление фронтенда и документации. Миграция базы не требуется.

## Dashboard

- В Dashboard Traffic statistics добавлен выбор часового пояса. По умолчанию
  используется часовой пояс браузера.

- Выбранный часовой пояс сохраняется в `localStorage`, поэтому Dashboard
  сохраняет его после перезагрузки страницы и между сессиями браузера.

- Подписи на графиках трафика теперь используют формат `YYYY-MM-DD HH:MM`.
  Выбранный часовой пояс показан в метаданных KPI рядом с выбором диапазона,
  поэтому интервалы проще читать, если браузер и сервер работают в разных
  часовых поясах.

## Настройка inbound

- Формы добавления inbound теперь показывают рекомендации с учетом протокола.
  Они применяются только явным действием. UI не меняет значения сам.

- Для VLESS inbound добавлено отдельное действие с рекомендуемой настройкой.
  Оно задает безопасную базу для нового inbound: `decryption: none`, включенный
  sniffing, включенный `sniff_override_destination`, `sniff_timeout: 300ms`,
  чистые настройки transport и packet encoding `xudp`. TLS, Reality и тип
  transport оператор выбирает сам.

- Для других поддержанных inbound-протоколов используется та же схема в режиме
  создания: подсказки у полей и кнопка применения рекомендаций там, где есть
  безопасный preset. Протоколы, где нужен выбор оператора, включая ShadowTLS,
  MTProxy, Tun, Bond и Core Failover, получают только подсказки.

- Существующие inbound-конфиги не переписываются при редактировании. Пустые
  advanced-поля остаются пустыми, пока оператор сам их не изменит.

- Новые Shadowsocks и Sudoku inbound теперь получают credentials при создании.
  ShadowTLS генерирует password только для version 2, что соответствует текущему
  поведению формы для version 3.

- Выбор TLS template для inbound теперь фильтруется. Reality templates
  показываются только там, где протокол их поддерживает: сейчас это VLESS и
  Trojan.

- Подсказки у полей теперь показывают рекомендуемое значение или прямое
  указание оставить поле пустым или выключенным. Новый текст подсказок для
  inbound добавлен во все поддерживаемые локали UI.

## Документация

- README теперь описывает выбор часового пояса для Dashboard Traffic statistics:
  дефолт из браузера, сохранение в `localStorage`, подписи
  `YYYY-MM-DD HH:MM` и отображение часового пояса в KPI.

- Список поддерживаемых протоколов в README обновлен по capability и
  configuration docs репозитория.

## Проверка

- Frontend build, lint и полный frontend test suite прошли перед обновлением
  этого release draft.

## Заметки для операторов

- Для этого релиза не требуется менять конфигурацию или выполнять миграцию базы.
- Очистка site storage сбрасывает selector часового пояса Dashboard к часовому
  поясу браузера.
- Кнопки рекомендаций влияют только на новые формы inbound и только после
  явного клика оператора.
- Существующие inbound-конфиги сохраняются без изменений.
