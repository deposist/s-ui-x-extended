# S-UI v1.5.3-beta - aggregated remediation + upstream parity (#1114)

> Aggregates the multi-chat code review remediation passes (P0/P1/P2/P3
> + P4/P5 architecture and logging cleanup) on top of `v1.5.2-beta-hotfix2`
> and ships an upstream parity bug fix for
> [`alireza0/s-ui#1114`](https://github.com/alireza0/s-ui/issues/1114):
> generated TUIC subscription/share links and the Clash export now
> include `udp_relay_mode`. No new schema. The embedded `sing-box`
> runtime is unchanged from the previous beta.
> Full changelog:
> [`CHANGELOG-EN.md`](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-EN.md)
> /
> [`CHANGELOG-RU.md`](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG-RU.md).

## English

### Changes
- **Upstream parity (#1114).** TUIC subscription/share links and the
  Clash export now include `udp_relay_mode`. The generator reads it from
  the inbound's `out_json.udp_relay_mode`, falls back to the inbound's
  root `udp_relay_mode`, and uses a safe default of `quic` when absent.
  Empty/unknown values do not produce empty query parameters. Generated
  TUIC links round-trip through `GetOutbound` preserving the mode, and
  Clash export maps `udp_relay_mode` to Mihomo's `udp-relay-mode`.
- **Multi-chat remediation aggregate (P0..P5).** Aggregates the P0/P1/P2
  hardening, P3 architecture work (restart unification, listen-fallback
  audit, initial DI slice, slog adapter), P4 architecture-debt closure
  (sing-box tracker revalidation policy, formal SemVer/version policy,
  remaining service-runtime globals behind a DI-compatible runtime, slog
  facade), and P5 logging cleanup (deprecated `logger.InitLogger` /
  `logger.GetLogger` removed; `github.com/op/go-logging` fully dropped
  from `go.mod`/`go.sum`).
- **No schema changes, no new endpoints, no new scopes.**

### What this means for operators

- `config/version` is bumped to `1.5.3-beta` and the frontend package
  version follows. The default tag for the release workflow is now
  `v1.5.3-beta`.
- TUIC clients that previously had to set `udp_relay_mode` manually on
  the import side now receive it from the panel directly. Existing
  inbound configurations with an explicit value continue to use that
  value; inbounds without a value get the safe default `quic`.
- External Go integrations that imported `logger.InitLogger` or
  `logger.GetLogger` must migrate to `logger.Init(logger.Level*)`,
  `logger.Slog(source)`, or `slog.Default()`.

### Validation

- `go build ./...` - PASS
- `go test ./...` - PASS
- `go test -race ./util ./sub` - PASS (TUIC link generation, parser
  round-trip, default-mode behavior, Clash conversion).
- The full multi-chat phase validation evidence is in `plans/`:
  - `plans/fix-validation.txt` (P0)
  - `plans/p1-validation.txt` (P1)
  - `plans/p2-validation.txt` (P2)
  - `plans/p3-architecture-validation.txt` (P3)
  - `plans/p4-architecture-debt-validation.txt` (P4)
  - `plans/p5-logging-cleanup-validation.txt` (P5)
  - `plans/upstream-issue-1114-plan.md` (#1114 plan + reusable prompt)

### Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.3-beta
```

Or from a local clone:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.3-beta
```

Drop-in upgrade on top of `v1.5.2-beta-hotfix2`. Full SQLite backup
before upgrade is still recommended.

### Rollback

1. `systemctl stop s-ui`.
2. Restore the backed-up `s-ui.db` and any matching `-wal`/`-shm`
   sidecars.
3. Reinstall `v1.5.2-beta-hotfix2` (or the last working tag).
4. `systemctl start s-ui`.

If rollback crosses the session/CSRF/realtime hardening of P0..P3,
invalidate active sessions after downgrade and rotate admin credentials.

---

## Русский

### Главное

- **Upstream-парити (#1114).** TUIC subscription/share-ссылки и Clash
  export теперь содержат `udp_relay_mode`. Генератор берёт значение из
  `out_json.udp_relay_mode` inbound'а, иначе из корневого
  `udp_relay_mode`, иначе использует безопасный default `quic`. Пустые
  или неизвестные значения не пишутся пустым query-параметром.
  Generated TUIC link проходит round-trip через `GetOutbound` без
  потери `udp_relay_mode`; Clash export маппит `udp_relay_mode` в
  Mihomo-поле `udp-relay-mode`.
- **Агрегат multi-chat remediation (P0..P5).** Включает P0/P1/P2
  hardening, P3 архитектурные пункты (унификация restart-пути, аудит
  listen fallback, initial DI slice, slog-адаптер), закрытие P4
  архитектурного долга (политика revalidation sing-box tracker,
  формальная SemVer/version policy, оставшиеся service-runtime globals
  за DI-совместимым runtime, slog-фасад) и P5 logging cleanup
  (удалены deprecated `logger.InitLogger` / `logger.GetLogger`;
  `github.com/op/go-logging` полностью убран из `go.mod`/`go.sum`).
- **Изменений схемы БД, новых эндпоинтов и новых scope'ов нет.**

### Что это значит для оператора

- `config/version` поднят до `1.5.3-beta`, версия frontend-пакета -
  следом. Default тег release workflow теперь `v1.5.3-beta`.
- TUIC-клиенты, которым раньше приходилось вручную выставлять
  `udp_relay_mode` на стороне импорта, теперь получают его прямо из
  панели. Inbound'ы с явным значением продолжают использовать его;
  Inbounds без значения получают безопасный default `quic`.
- Внешние Go-интеграции, использовавшие `logger.InitLogger` или
  `logger.GetLogger`, должны перейти на `logger.Init(logger.Level*)`,
  `logger.Slog(source)` или `slog.Default()`.

### Валидация

- `go build ./...` - PASS
- `go test ./...` - PASS
- `go test -race ./util ./sub` - PASS (генерация TUIC-ссылки, round-trip
  парсинга, дефолтное поведение, Clash conversion).
- Доказательная база по фазам - в `plans/`:
  - `plans/fix-validation.txt` (P0)
  - `plans/p1-validation.txt` (P1)
  - `plans/p2-validation.txt` (P2)
  - `plans/p3-architecture-validation.txt` (P3)
  - `plans/p4-architecture-debt-validation.txt` (P4)
  - `plans/p5-logging-cleanup-validation.txt` (P5)
  - `plans/upstream-issue-1114-plan.md` (план #1114 + переносимый промт)

### Установка

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.5.3-beta
```

Или из локального клона:

```sh
git clone -b beta https://github.com/deposist/s-ui-rus-inst.git
cd s-ui-rus-inst
sudo bash install.sh v1.5.3-beta
```

Drop-in поверх `v1.5.2-beta-hotfix2`. Полный backup SQLite перед
обновлением по-прежнему рекомендуется.

### Откат

1. `systemctl stop s-ui`.
2. Восстановите `s-ui.db` из бэкапа и соответствующие
   `-wal`/`-shm`-сайдкары.
3. Поставьте `v1.5.2-beta-hotfix2` (или последний рабочий тег).
4. `systemctl start s-ui`.

Если откат пересекает hardening session/CSRF/realtime из P0..P3,
инвалидируйте активные сессии после понижения версии и проведите
ротацию admin credentials.
