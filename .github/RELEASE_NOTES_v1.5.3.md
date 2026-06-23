# S-UI v1.5.3

Stable release of the `1.5.3` line for `deposist/s-ui-x`.

## Changes
- Promotes `1.5.3-beta` to stable `1.5.3`.
- Keeps runtime compatibility with existing `s-ui` installs: binary name,
  service name, data paths, and database layout remain unchanged.
- Includes the `1.5.3-beta` remediation aggregate and upstream parity fix for
  [`alireza0/s-ui#1114`](https://github.com/alireza0/s-ui/issues/1114).
- Adds a friendlier Telegram database backup frequency UI with presets,
  custom minute/hour intervals, and Advanced cron compatibility. The scheduler
  still stores and reads the existing `telegramBackupCron` setting.
- Uses the `deposist/s-ui-x` release/download coordinates and
  `ghcr.io/deposist/s-ui-x` container image identity.

## Validation

- `cd frontend && npm run test` - PASS
- `cd frontend && npm run lint` - PASS
- `cd frontend && npm run build` - PASS
- `go test ./service ./cronjob` - PASS

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.3
```

From a local clone:

```sh
git clone https://github.com/deposist/s-ui-x.git
cd s-ui-x
sudo bash install.sh v1.5.3
```

## Русский

Стабильный релиз линейки `1.5.3` для `deposist/s-ui-x`.

- `1.5.3-beta` повышен до стабильного `1.5.3`.
- Runtime-совместимость сохранена: бинарник, service, пути данных и схема БД
  остаются прежними.
- Включены исправления `1.5.3-beta` и upstream-парити-фикс
  [`alireza0/s-ui#1114`](https://github.com/alireza0/s-ui/issues/1114).
- Периодичность Telegram backup теперь настраивается через понятные пресеты,
  свой интервал в минутах/часах и Advanced cron. Под капотом по-прежнему
  используется существующий setting `telegramBackupCron`.

Установка:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.3
```
