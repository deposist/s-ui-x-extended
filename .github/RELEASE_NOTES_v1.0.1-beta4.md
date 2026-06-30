# S-UI-X-Extended v1.0.1-beta4

This beta fixes WARP endpoint saves with the sing-box-extended core used by this build. No manual database edit is required.

## WARP endpoint fix

- WARP registration no longer writes `reserved` inside `peers[]`. The current endpoint schema rejects that field and returned `endpoints[0].peers[0].reserved: json: unknown field "reserved"` after save.

- Core config generation now strips unsupported `peers[].reserved` fields from WARP and WireGuard endpoints. Existing stored rows with that field no longer break sing-box startup.

- The WARP form now displays reserved bytes safely for old rows without requiring the field on the peer object.

## Compatibility notes

- This release does not add a database migration. The filter runs when endpoint JSON is stored or generated for sing-box.

- `v1.0.1-beta3` remains the inbound sniff migration release. `v1.0.1-beta4` adds the WARP reserved-field fix on top of it.

## Verification

- `go test ./database/model ./service -run 'TestEndpointMarshalDropsUnsupportedWireGuardPeerReserved|TestWarpUnmarshalDropsReservedBeforeStore|TestConfigRoundTripWarpEndpointDropsUnsupportedReservedFields|TestSetWarp|TestConfigRoundTrip' -count=1` passed.
- `go test ./database/model ./service -count=1` passed.
- `go test ./...` passed.
- `cd frontend && npm run build` passed.
- `cd frontend && npm run lint` passed.
- `cd frontend && npm test -- --run` passed.

---

# S-UI-X-Extended v1.0.1-beta4

Эта beta исправляет сохранение WARP endpoint с sing-box-extended core, который используется в этой сборке. Ручное редактирование базы не требуется.

## Исправление WARP endpoint

- Регистрация WARP больше не записывает `reserved` внутри `peers[]`. Текущая schema endpoint отвергает это поле и возвращала `endpoints[0].peers[0].reserved: json: unknown field "reserved"` после сохранения.

- Генерация core config теперь удаляет неподдерживаемые `peers[].reserved` поля из WARP и WireGuard endpoints. Существующие сохраненные rows с этим полем больше не ломают запуск sing-box.

- Форма WARP теперь безопасно показывает reserved bytes для старых записей и не требует это поле внутри peer object.

## Совместимость

- Этот релиз не добавляет миграцию базы. Фильтр работает при сохранении endpoint JSON и при генерации JSON для sing-box.

- `v1.0.1-beta3` остается релизом с миграцией inbound sniff. `v1.0.1-beta4` добавляет к нему исправление WARP reserved-field.

## Проверка

- `go test ./database/model ./service -run 'TestEndpointMarshalDropsUnsupportedWireGuardPeerReserved|TestWarpUnmarshalDropsReservedBeforeStore|TestConfigRoundTripWarpEndpointDropsUnsupportedReservedFields|TestSetWarp|TestConfigRoundTrip' -count=1` прошел.
- `go test ./database/model ./service -count=1` прошел.
- `go test ./...` прошел.
- `cd frontend && npm run build` прошел.
- `cd frontend && npm run lint` прошел.
- `cd frontend && npm test -- --run` прошел.
