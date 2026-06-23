# S-UI v1.5.5-beta1

Prerelease addressing two upstream subscription bugs reported on
`alireza0/s-ui` against the same code paths in s-ui-x.

## Fixed

- VLESS `xtls-rprx-vision` flow no longer leaks onto non-TCP transports
  when a single client UUID is reused across multiple inbounds. The flow
  is now stripped on `grpc`, `ws`, `http`, `httpupgrade`, ... transports
  in three places that build per-user payloads: `fetchUsersByCondition`
  (panel-served sing-box config), JSON subscription rendering, and
  shareable `vless://` link generation. This matches Xray-core's TCP-only
  contract for vision and lets the same UUID serve a TCP+REALITY inbound
  and a gRPC+TLS inbound side by side
  (alireza0/s-ui#1127).
- Clash subscription `ws-opts.headers` now carries `Host` again. The
  previous `[]interface{}` cast against the map-shaped header silently
  dropped the header, which broke Mihomo's WebSocket handshake through
  strict CDN / Nginx upstreams. When no explicit Host header is set,
  the exporter now falls back to the TLS `server_name` so the upstream
  always sees a Host that matches the SNI
  (alireza0/s-ui#1126).

## Added

- Regression tests:
  - `service/inbounds_vless_flow_test.go` - vless+TCP keeps flow,
    vless+grpc/ws/no-tls strips, vmess unaffected.
  - `util/genLink_vless_flow_test.go` - `vless://` links emit `flow`
    only when transport is TCP.
  - `sub/clashService_ws_host_test.go` - explicit Host header survives
    YAML round-trip; SNI fallback populates `ws-opts.headers.Host`.

## Validation

- `go test ./...` - all 26 packages PASS.
- `cd frontend && npm run build` - PASS.
- `git diff --check` - PASS.

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta1
```

## Русский

Prerelease с двумя точечными исправлениями subscription, поднятыми в
upstream `alireza0/s-ui` и затрагивающими ту же логику в s-ui-x.

- VLESS-флаг `xtls-rprx-vision` больше не утекает на не-TCP транспорты
  при использовании одного UUID на нескольких inbound. Флаг снимается
  для `grpc`, `ws`, `http`, `httpupgrade` и т.п. в трёх местах, где
  собирается per-user payload: `fetchUsersByCondition` (panel-served
  sing-box config), JSON-подписка, генерация `vless://` ссылки. Один
  UUID теперь корректно работает одновременно на TCP+REALITY и
  gRPC+TLS inbound (alireza0/s-ui#1127).
- В Clash-подписке `ws-opts.headers` снова содержит `Host`. Предыдущий
  cast в `[]interface{}` молча отбрасывал map-структуру header, из-за
  чего Mihomo получал 400 Bad Request за строгими CDN / Nginx. Если
  явный Host не задан, экспортёр теперь берёт его из TLS `server_name`,
  чтобы upstream всегда видел Host, совпадающий с SNI
  (alireza0/s-ui#1126).
