# S-UI-X-Extended v1.0.1-beta3

This release fixes the sing-box 1.11+ legacy inbound field warning. The database migration runs automatically on startup, import, and migration commands. No manual database edit is required.

## sing-box inbound cleanup

- Removed the inbound advanced UI fields for `sniff`, `sniff_override_destination`, and `sniff_timeout`.

- Removed the same fields from frontend inbound types and locale text. Route-rule sniff actions remain available in the route rule editor.

- The Apply inbound recommendations flow no longer writes inbound-level sniff options. If stale form data contains those keys, the recommendation helper deletes them instead of setting defaults.

- Backend inbound JSON output now drops legacy inbound rule-action fields before sing-box sees the config: `sniff`, `sniff_override_destination`, `sniff_timeout`, `domain_strategy`, and `udp_disable_domain_unmapping`.

## Automatic migration

- Existing inbound `sniff: true` settings are moved to scoped route rules with `action: "sniff"`.

- Existing inbound `sniff_timeout` values are copied to the sniff route action as `timeout`.

- Existing inbound `domain_strategy` values are moved to scoped `action: "resolve"` rules with `strategy`.

- Existing inbound `udp_disable_domain_unmapping` values are moved to scoped `action: "route-options"` rules.

- `sniff_override_destination` is removed from inbound options. sing-box does not provide a matching route-action field for it.

- Migrated rules are scoped by inbound tag and inserted before existing routing rules, matching sing-box behavior for deprecated inbound sniff fields. Equivalent existing rules are not duplicated.

- The old `config.json` to database migration path now preserves these settings instead of deleting them while importing pre-database installs.

## Compatibility notes

- The migration is idempotent. Running it more than once does not add the same scoped rule again.

- Current-version databases are also checked, so users already on beta2 get the legacy inbound field migration when beta3 starts.

- Legacy inbound fields submitted by old or external clients are removed after save. The stored inbound options and generated core config no longer keep those keys.

## Verification

- `go test ./cmd/migration ./database/migrateutil ./database/model ./database ./service -run 'TestTo12MovesLegacy|TestMigrateLegacy|TestInboundJSONDrops|TestAdapt|TestConfigRoundTripVLESSInboundLegacyFieldsMigrateToRouteRules'` passed.
- `go test ./core -run TestOptionCoverageNoMissingFields` passed.
- `cd frontend && npm test -- --run` passed.
- `cd frontend && npm run build` passed.
- `cd frontend && npm run lint` passed.

---

# S-UI-X-Extended v1.0.1-beta3

Этот релиз исправляет предупреждение sing-box 1.11+ о legacy inbound fields. Миграция базы запускается автоматически при старте, импорте и через migration command. Ручное редактирование базы не требуется.

## Очистка inbound для sing-box

- Из advanced UI для inbound удалены поля `sniff`, `sniff_override_destination` и `sniff_timeout`.

- Эти же поля удалены из frontend inbound types и локалей. Route-rule sniff actions остаются доступны в редакторе route rules.

- Apply inbound recommendations больше не записывает inbound-level sniff options. Если в старых данных формы есть такие ключи, helper удаляет их вместо установки default values.

- Backend теперь убирает legacy inbound rule-action fields из inbound JSON до передачи конфига в sing-box: `sniff`, `sniff_override_destination`, `sniff_timeout`, `domain_strategy` и `udp_disable_domain_unmapping`.

## Автоматическая миграция

- Существующие inbound `sniff: true` переносятся в scoped route rules с `action: "sniff"`.

- Значения inbound `sniff_timeout` копируются в sniff route action как `timeout`.

- Значения inbound `domain_strategy` переносятся в scoped rules с `action: "resolve"` и `strategy`.

- Значения inbound `udp_disable_domain_unmapping` переносятся в scoped rules с `action: "route-options"`.

- `sniff_override_destination` удаляется из inbound options. У sing-box нет соответствующего route-action поля.

- Мигрированные rules привязаны к inbound tag и вставляются перед существующими routing rules, как это работало для deprecated inbound sniff fields в sing-box. Эквивалентные existing rules не дублируются.

- Старый путь миграции `config.json` в базу теперь сохраняет эти настройки, а не удаляет их при импорте установок до перехода на базу данных.

## Совместимость

- Миграция идемпотентна. Повторный запуск не добавляет один и тот же scoped rule снова.

- Базы с текущей версией тоже проверяются, поэтому пользователи beta2 получают миграцию legacy inbound fields при запуске beta3.

- Legacy inbound fields, отправленные старыми или внешними клиентами, удаляются после сохранения. Stored inbound options и generated core config больше не держат эти ключи.

## Проверка

- `go test ./cmd/migration ./database/migrateutil ./database/model ./database ./service -run 'TestTo12MovesLegacy|TestMigrateLegacy|TestInboundJSONDrops|TestAdapt|TestConfigRoundTripVLESSInboundLegacyFieldsMigrateToRouteRules'` прошел.
- `go test ./core -run TestOptionCoverageNoMissingFields` прошел.
- `cd frontend && npm test -- --run` прошел.
- `cd frontend && npm run build` прошел.
- `cd frontend && npm run lint` прошел.
