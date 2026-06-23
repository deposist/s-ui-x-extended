# S-UI v1.5.5

Stable release of the `1.5.5` line for `deposist/s-ui-x`.

## Changes
- Promotes `v1.5.5-beta1` through `v1.5.5-beta4-hotfix2` to stable `v1.5.5`.
- Fixes subscription correctness for shared VLESS UUIDs and Clash WebSocket
  Host headers: `xtls-rprx-vision` is no longer exported for non-TCP
  transports, and Clash/Mihomo exports keep a usable `ws-opts.headers.Host`.
- Hardens backup export, restore, and import rollback paths. The no-TLS
  `tls.id=0` sentinel is preserved safely, failed imports reopen the live DB,
  DNS/routing config in `settings.config` is validated, and the TLS sentinel
  no longer collides with real TLS rows during backup export.
- Carries the beta4 security and reliability work: forced password-reset state
  for imported administrators, safer token handling, audit prioritization,
  large streamed X-UI import plans, config rollback realtime invalidation,
  SQLite pool configurability, fail-closed IP-monitor reads, bounded rate-limit
  state, realtime self-healing, retry/backoff improvements, and data-race fixes.
- Includes frontend release hardening from the hotfixes: synchronized npm
  lockfile, more stable Playwright/Vite e2e runs, safer reconnect chaos tests,
  and a longer accessibility baseline timeout.
- Updates the build/runtime toolchain to Go `1.26.3` and the embedded
  `github.com/sagernet/sing-box` runtime to `v1.13.12`.

## Validation

- `go vet ./...` - PASS
- `go test -race -timeout=10m ./...` - PASS
- `go build -ldflags="-w -s" -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" -o sui main.go` - PASS
- `git diff --check` - PASS
- Docker build was not run in the local workspace because Docker CLI was not
  installed; the tag push will trigger the repository release and Docker
  workflows.

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5
```

From a local clone:

```sh
git clone https://github.com/deposist/s-ui-x.git
cd s-ui-x
sudo bash install.sh v1.5.5
```

## Русский

Стабильный релиз линейки `1.5.5` для `deposist/s-ui-x`.

- `v1.5.5-beta1` - `v1.5.5-beta4-hotfix2` повышены до стабильной `v1.5.5`.
- Исправлена корректность подписок: VLESS `xtls-rprx-vision` больше не
  попадает в не-TCP транспорты при общем UUID, а Clash/Mihomo export сохраняет
  рабочий `ws-opts.headers.Host`.
- Усилены backup/restore/import paths: no-TLS sentinel `tls.id=0` сохраняется
  безопасно, failed import переоткрывает live DB, DNS/routing config в
  `settings.config` проверяется, а TLS sentinel больше не конфликтует с
  реальными TLS-записями при backup export.
- Включён большой beta4 hardening: forced password reset для импортированных
  администраторов, более безопасные токены, приоритет audit-событий,
  потоковая обработка больших X-UI import plans, realtime invalidation после
  rollback, настраиваемые SQLite pool limits, fail-closed IP monitor,
  bounded rate-limit state, self-healing realtime, retry/backoff и исправления
  data races.
- Включены frontend hotfixes: синхронизированный npm lockfile, стабильнее
  Playwright/Vite e2e, более устойчивый reconnect chaos test и увеличенный
  timeout accessibility baseline.
- Сборка обновлена до Go `1.26.3`, embedded `github.com/sagernet/sing-box` -
  до `v1.13.12`.

Установка:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5
```
