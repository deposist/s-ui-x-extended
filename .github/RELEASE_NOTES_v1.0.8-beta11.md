# S-UI-X Extended v1.0.8-beta11

This beta fixes version ordering and restores compatibility with the numeric recovery marker written by beta8. The beta channel now treats `beta10` and `beta11` as newer than `beta9`.

A beta8 to beta9 update could leave the service restarting with `cannot unmarshal number into Go value of type service.pendingUpdateMarker`. Beta11 converts that old marker to the current format, verifies the installed and backup binaries by SHA-256, and keeps automatic rollback protection.

Release publication now waits for a newly-created draft to appear through the GitHub API before reconciling assets. This fixes a timing failure found after beta10.

## Upgrade from the console

Servers in the restart loop cannot use the panel updater. Run this command over SSH or from the server console:

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.8-beta11/install.sh \
  | sudo bash -s -- v1.0.8-beta11
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration. No database migration or manual configuration change is required.

Full release notes: [`docs/releases/v1.0.8-beta11.md`](../docs/releases/v1.0.8-beta11.md).

## Обновление через консоль

Если служба попала в цикл перезапусков после обновления с beta8 на beta9, выполните команду по SSH или в консоли сервера:

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.8-beta11/install.sh \
  | sudo bash -s -- v1.0.8-beta11
```

После установки проверьте версию и состояние службы:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу данных и настройки панели. Миграция базы и ручное изменение конфигурации не требуются.