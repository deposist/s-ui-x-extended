# S-UI v1.5.6-beta2

Prerelease for sing-box 1.13.12 settings coverage across Classic and Nexus UI
in `deposist/s-ui-x`.

## Changes
- Extends the sing-box 1.13.12 settings UI across Classic and Nexus by sharing
  the same advanced editor surfaces for basics, rules, DNS, TLS, inbounds,
  outbounds, endpoints and services.
- Adds `DomainResolveOptions` editing, route network presets, Dial/Listen/TUN
  advanced settings, top-level certificate trust editing, rule route-options TLS
  fragmentation controls and the rule `client` matcher.
- Adds HTTP/Mixed system proxy controls plus Shadowsocks plugin, Trojan
  fallback, ShadowTLS strict mode, Hysteria v1 port hopping/bandwidth,
  Hysteria2 brutal debug and SSM API cache path fields.
- Preserves top-level `certificate` and unknown top-level sing-box config fields
  through backend round-trips, so runtime config generation keeps certificate
  trust settings.
- Keeps no-op/default JSON clean: `Off` removes fields, default delays and zero
  marks are not written, empty app/package selections are blocked, and
  `tls_record_fragment` remains mutually exclusive with `tls_fragment`.

## Validation

- `npm run build`
- `npm run test`
- `npm run lint`
- `go test ./...`
- `go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" ./core`

## Install

```bash
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.6-beta2
```

# S-UI v1.5.6-beta2

Prerelease для покрытия настроек sing-box 1.13.12 в Classic и Nexus UI в
`deposist/s-ui-x`.

## Главное

- Расширено покрытие настроек sing-box 1.13.12 в Classic и Nexus: одни и те же
  advanced editor surfaces используются для basics, rules, DNS, TLS, inbounds,
  outbounds, endpoints и services.
- Добавлены редактирование `DomainResolveOptions`, route network presets,
  advanced settings Dial/Listen/TUN, top-level certificate trust editor, TLS
  fragmentation в rule route-options и matcher `client`.
- Добавлены HTTP/Mixed system proxy controls, Shadowsocks plugin, Trojan
  fallback, ShadowTLS strict mode, Hysteria v1 port hopping/bandwidth,
  Hysteria2 brutal debug и SSM API cache path fields.
- Backend сохраняет top-level `certificate` и unknown top-level поля sing-box
  config при round-trip, поэтому runtime config сохраняет certificate trust
  settings.
- JSON не засоряется no-op/default значениями: `Off` удаляет поля, default
  delays и zero marks не пишутся, пустые app/package selections блокируются, а
  `tls_record_fragment` остаётся взаимоисключающим с `tls_fragment`.

## Проверки

- `npm run build`
- `npm run test`
- `npm run lint`
- `go test ./...`
- `go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" ./core`

## Установка

```bash
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.6-beta2
```
