# S-UI v1.5.4

Stable release of the `1.5.4` line for `deposist/s-ui-x`.

## Changes
- Promotes `v1.5.4-beta1` through `v1.5.4-beta5` to stable `v1.5.4`.
- Adds the opt-in Nexus UI mode while keeping Classic as the default UI.
- Carries the beta hotfixes for canceled duplicate Nexus reads, denser Nexus
  Overview layout, systemd installer `SUI_SECRETBOX_KEY` bootstrap, and
  reserved `/ws` path validation on a path-segment boundary.
- Finishes the release localization pass in Persian, Vietnamese, Simplified
  Chinese, Traditional Chinese, and Russian, including Telegram, Audit,
  maintenance, backup, IP-limit, networking, DNS, TLS, rules, and stats labels.

## Validation

- `go vet ./...` - PASS
- `go test -race -timeout=5m ./...` - PASS
- `go test ./config ./database ./service` - PASS
- `cd frontend && npm run lint` - PASS
- `cd frontend && npm run test` - PASS
- `cd frontend && npm run build` - PASS
- `git diff --check` - PASS

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4
```

From a local clone:

```sh
git clone https://github.com/deposist/s-ui-x.git
cd s-ui-x
sudo bash install.sh v1.5.4
```

## Русский

Стабильный релиз линейки `1.5.4` для `deposist/s-ui-x`.

- `v1.5.4-beta1` - `v1.5.4-beta5` повышены до стабильной `v1.5.4`.
- Добавлен opt-in режим Nexus UI при сохранении Classic как интерфейса по
  умолчанию.
- Включены beta hotfixes для отменённых duplicate Nexus reads, более плотного
  Nexus Overview, bootstrap `SUI_SECRETBOX_KEY` в systemd installer и
  валидации reserved `/ws` path по границе сегмента.
- Завершён релизный проход по Persian, Vietnamese, Simplified/Traditional
  Chinese и Russian локализациям, включая Telegram, Audit, maintenance, backup,
  IP-limit, networking, DNS, TLS, rules и stats.

Установка:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4
```
