# S-UI v1.5.6-beta1

Beta for sing-box 1.13 UI parity in `deposist/s-ui-x`.

## Changes
- Adds first-class UI controls for sing-box 1.13 TLS advanced fields, including
  curve preferences, client authentication/certificates, certificate public key
  pins and outbound kTLS.
- Fixes route/DNS interface-address JSON shapes and extends route, DNS and
  inline/source headless rule-set editors with network/Wi-Fi state matchers.
- Adds route `bypass` serialization, route reject `reply`, Naive receive
  windows and UoT version selection, TUN reset mark/NFQUEUE, Tailscale
  advertise tags, OCM/CCM headers and the `oom-killer` service.
- Adds representative sing-box 1.13 option-unmarshal coverage and verifies the
  OOM service registry entry.

## Validation

- `npm --prefix frontend run build`
- `npm --prefix frontend run test`
- `npm --prefix frontend run lint`
- `go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" ./...`

## Install

```bash
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.6-beta1
```

# S-UI v1.5.6-beta1

Beta для UI-паритета sing-box 1.13 в `deposist/s-ui-x`.

## Главное

- Добавлены first-class UI controls для advanced TLS-полей sing-box 1.13:
  curve preferences, client authentication/certificates, certificate public key
  pins и outbound kTLS.
- Исправлены JSON shapes для `interface_address` в route/DNS rules, а route,
  DNS и inline/source headless rule-set editors получили network/Wi-Fi state
  matchers.
- Добавлены сериализация route `bypass`, route reject `reply`, Naive receive
  windows и выбор UoT version, TUN reset mark/NFQUEUE, Tailscale advertise
  tags, OCM/CCM headers и сервис `oom-killer`.
- Добавлены representative sing-box 1.13 option-unmarshal coverage и проверка
  регистрации OOM service.

## Проверки

- `npm --prefix frontend run build`
- `npm --prefix frontend run test`
- `npm --prefix frontend run lint`
- `go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" ./...`

## Установка

```bash
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.6-beta1
```
