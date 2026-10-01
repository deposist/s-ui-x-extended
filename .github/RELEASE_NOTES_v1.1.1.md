# S-UI-X Extended v1.1.1

Version 1.1.1 is the general release incorporating the 1.1.1 beta series after version 1.1.0.

## Main changes

- Bundled core updated to sing-box-extended `v1.14.0-extended-2.7.5`. The panel supports OpenVPN client and server endpoints, the `cloudflared` inbound, USB/IP services, and updated fields for Hysteria, Hysteria 2, TUIC, Mieru, MASQUE and WireGuard.
- AmneziaWG 3.1 controls in the WireGuard endpoint form: Random trailers appends random bytes to packet headers to mask fixed handshake sizes, and Disable cookies drops cookie replies and MAC2 checks under heavy load.
- Hot-reload of managed AWG endpoints now preserves client preshared keys (PSKs).
- Failover job state and member health maps are pruned from memory when groups or members are removed (PR #10).
- Fixed an installer rollback bug where running through a pipe (`curl ... | sudo bash -s`) aborted the installation on the final prompt.
- Structured forms for certificate providers (ACME, Tailscale, Cloudflare Origin CA), reusable HTTP clients, and Linux network namespaces on the Basics tab in Settings.
- Hardened inbound saves: TrustTunnel requires TLS, Mieru and TrustTunnel generate client passwords when empty, and invalid core configurations are rejected before commit.
- Fixed a core shutdown hang when MTProxy was enabled.

The database schema is unchanged. Existing configurations migrate to the 1.14 schema automatically on first start.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1/install.sh \
  | sudo bash -s -- v1.1.1
```

Verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.1.1.md`](../docs/releases/v1.1.1.md).

## Обновление через консоль

Версия 1.1.1 объединяет наработки всей линейки beta после 1.1.0: ядро sing-box 1.14, поддержку AmneziaWG 3.1 со случайными трейлерами, эндпоинты OpenVPN, очистку памяти failover, исправления отката установщика и горячей перезагрузки AWG, а также структурированные формы настроек ядра.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1/install.sh \
  | sudo bash -s -- v1.1.1
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели. Схема базы данных не меняется, конфигурация переносится на схему 1.14 автоматически при первом запуске.
