# S-UI-X Extended v1.0.9

Version 1.0.9 contains all changes from the 1.0.9 beta series after version 1.0.8.

## Main changes

- AmneziaWG 3.0. The bundled core moves to sing-box-extended 2.6.5. Managed AWG endpoints and device configs use the 3.0 option set (header protection key, content padding addition, rekey/reject/keepalive timings, max handshake attempts); J1/J2/J3 and itime are gone. Existing 2.0 endpoints migrate automatically on the first start after the upgrade, and Amnezia 2.x clients keep working.
- New Call protocol. A server-side WebRTC bridge (dion, telemost, vk, wbstream) works as an inbound or outbound. The inbound creates a call room and clients join through a `join_link`.
- New protocol options. VPN client/server gains the default gateway, pool size, and reconnect and reject delays. AnyTLS gains `client_metadata`, MASQUE gains license and private keys, and multiplex gains `rmux`.
- Fixed periodic AWG reconcile failing with a bare `AWG reconcile failed` log. Endpoint-scoped devices now reconcile against their own endpoint tag, and reconcile errors log the endpoint ID with the sanitized cause.
- Database restore no longer deadlocks behind an open realtime WebSocket, and cron jobs stop spamming `cron: skip` during a restore.
- WARP Endpoint creation accepts the peer endpoint Cloudflare returns, including a domain host and a placeholder port of `0`.
- A stable release selected while tracking the beta channel validates against the release's `main` manifest. This fixes `update manifest is invalid` when a beta installation graduates to stable without changing its saved channel.
- TrustTunnel no longer exposes `bbr_profile` (removed in sing-box 2.6.x). Re-save a trusttunnel entry saved before this release.
- Built with Go 1.26.6, which fixes seven Go standard library vulnerabilities published on 2026-08-13.

Existing AWG 2.0 endpoint options migrate automatically on first start. No manual configuration change is required.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9/install.sh \
  | sudo bash -s -- v1.0.9
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.0.9.md`](../docs/releases/v1.0.9.md).

## Обновление через консоль

Версия 1.0.9 включает все изменения линейки beta после 1.0.8: AmneziaWG 3.0 на новом ядре, протокол Call, новые опции VPN/AnyTLS/MASQUE, а также исправления восстановления базы, создания WARP Endpoint, перехода с beta на stable и периодического AWG reconcile.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9/install.sh \
  | sudo bash -s -- v1.0.9
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели. Существующие опции эндпоинтов AWG 2.0 мигрируют автоматически при первом запуске. Ручное изменение конфигурации не требуется.
