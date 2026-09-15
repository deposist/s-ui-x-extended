# S-UI-X Extended v1.1.1-beta3

The panel runs on a newer sing-box-extended core: OpenVPN endpoints (client and server), the `cloudflared` inbound, USB/IP services, editors for certificate providers, HTTP clients and network namespaces, plus more options for Hysteria, Hysteria 2, TUIC, Mieru, MASQUE and WireGuard.

Saved configurations upgrade themselves on first start. DNS rules that filtered answers by address become an `evaluate` rule followed by a `match_response` rule, a legacy OpenVPN outbound becomes an endpoint, and an inline ACME block becomes a certificate provider. You do not change anything by hand.

Download a database backup from the panel before you upgrade. A rollback means restoring that database file and then the previous binary; the binary alone will not do it, because the schema migration moves in one direction.

This is a beta. Test the new protocols and the automatic migration on a non-critical server before upgrading production.

Full release notes: [`docs/releases/v1.1.1-beta3.md`](../docs/releases/v1.1.1-beta3.md).

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta3/install.sh \
  | sudo bash -s -- v1.1.1-beta3
```

Verify:
```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta3
systemctl is-active s-ui  # active
```

## Обновление

Панель перешла на новое ядро sing-box-extended: эндпоинты OpenVPN (клиент и сервер), входящее подключение `cloudflared`, службы USB/IP, редакторы certificate providers, HTTP clients и network namespaces, а также дополнительные настройки Hysteria, Hysteria 2, TUIC, Mieru, MASQUE и WireGuard.

Сохранённая конфигурация обновляется сама при первом запуске. Правила DNS, которые фильтровали ответы по адресам, превращаются в пару `evaluate` и `match_response`, старый OpenVPN-outbound переезжает в эндпоинт, а встроенный ACME - в certificate provider. Руками ничего менять не нужно.

Перед обновлением выгрузите резервную копию базы данных из панели. Откат означает восстановление файла базы из копии и затем прежнего бинарного файла, один бинарник без базы не откатит, потому что миграция схемы идёт только вперёд.

Это бета. Проверьте новые протоколы и автоматическую миграцию на некритичном сервере, прежде чем обновлять рабочий.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta3/install.sh \
  | sudo bash -s -- v1.1.1-beta3
```

Проверка:
```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta3
systemctl is-active s-ui  # active
```

Полные заметки: [`docs/releases/v1.1.1-beta3.md`](https://github.com/deposist/s-ui-x-extended/blob/v1.1.1-beta3/docs/releases/v1.1.1-beta3.md)