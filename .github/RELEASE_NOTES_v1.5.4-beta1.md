# S-UI v1.5.4-beta1

Opt-in beta for the new Nexus UI mode in `deposist/s-ui-x`.

## Changes
- Adds `nexus` as an opt-in UI mode next to the existing `classic`
  interface. Classic remains the default.
- Adds the UI mode contract, localStorage persistence, `VITE_ENABLE_NEXUS`
  kill switch, CSP-safe pre-mount bootstrap and authenticated layout host.
- Adds the Nexus shell, responsive sidebar/topbar behavior, RTL `fa`
  support, Nexus themes/tokens and the fixed Nexus Overview dashboard.
- Keeps stage 1 frontend-only: no backend endpoints, no API/CSRF/CSP
  changes, no new WebSocket connections and no new package dependencies.

## Validation

- `cd frontend && npm run test` - PASS
- `cd frontend && npm run lint` - PASS
- `cd frontend && npm run build` - PASS
- External-origin primary gate over Nexus source - PASS, zero matches
- Built `dist/index.html` inline-script/event-handler gate - PASS, zero matches
- Supply-chain gate, `git diff -- package.json` dependency blocks - PASS
- Nexus viewport verification - PASS for LTR `en` and RTL `fa` at
  `1440x900`, `1180x800`, `834x1112` and `390x844`
- No backend/API/CSRF/CSP drift detected

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4-beta1
```

From a local clone:

```sh
git clone https://github.com/deposist/s-ui-x.git
cd s-ui-x
sudo bash install.sh v1.5.4-beta1
```

## Notes

- Nexus is an opt-in beta. Existing installations keep rendering Classic
  until an admin switches their own browser to Nexus.
- To disable the Nexus path at build time, set `VITE_ENABLE_NEXUS=false`.
- A full SQLite backup before upgrading is still recommended.

## Русский

Opt-in beta нового режима интерфейса Nexus для `deposist/s-ui-x`.

- `classic` остаётся дефолтом; `nexus` включается отдельно в браузере.
- Добавлены UI mode contract, localStorage persistence, kill switch
  `VITE_ENABLE_NEXUS`, CSP-safe bootstrap и authenticated layout host.
- Добавлены Nexus shell, responsive sidebar/topbar, RTL `fa`, Nexus
  themes/tokens и фиксированный Overview dashboard.
- Stage 1 остаётся frontend-only: без новых backend endpoint'ов,
  API/CSRF/CSP изменений, WebSocket-потоков и package dependencies.

Установка:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4-beta1
```
