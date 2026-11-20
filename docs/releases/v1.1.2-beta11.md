# S-UI-X Extended v1.1.2-beta11

## AnyTLS

Fixed an AnyTLS race between closing a session and writing to a stream. This could occur when the peer disconnects or rejects authentication.

## Upgrade

Back up the panel database before updating. To roll back, reinstall the previous version and restore the backup.

```sh
curl -fLsS https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.2-beta11/install.sh \
  | sudo bash -s -- v1.1.2-beta11
```

On Windows, extract the ZIP for your architecture and run `install-windows.bat` as Administrator. The Docker image is `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta11`.

## По-русски

Исправили гонку между закрытием сеанса AnyTLS и записью в поток. Она могла возникнуть при разрыве соединения или отказе в аутентификации.

Перед обновлением сделайте резервную копию базы данных панели. Для Linux используйте команду выше. В Windows распакуйте ZIP для своей архитектуры и запустите `install-windows.bat` от имени администратора. Для Docker используйте образ `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta11`. Для отката установите предыдущую версию и восстановите копию базы.
