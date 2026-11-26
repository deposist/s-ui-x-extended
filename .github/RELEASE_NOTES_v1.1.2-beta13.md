# S-UI-X Extended v1.1.2-beta13

- During an upgrade, the panel reads and restores missing invoice references for older CryptoBot orders in the same database transaction.
- The Sudoku WebSocket transport now limits memory allocations for incoming frames.

## Upgrade

Back up the panel database before upgrading. On Linux, run the installer for this version.

```bash
curl -fLsS https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.2-beta13/install.sh \
  | sudo bash -s -- v1.1.2-beta13
```

On Windows, extract the ZIP for your architecture and run `install-windows.bat` as Administrator. For Docker, use `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta13`. To roll back, reinstall the previous version and restore your backup.

## По-русски

- При обновлении панели чтение старых заказов CryptoBot и восстановление недостающих номеров счетов проходят в одной транзакции.
- В Sudoku ограничили выделение памяти при чтении входящих кадров WebSocket.

Перед обновлением сохраните копию базы панели. Для Linux используйте команду выше. В Windows распакуйте ZIP для своей архитектуры и запустите `install-windows.bat` от имени администратора. Для Docker используйте образ `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta13`. Для отката установите предыдущую версию и восстановите копию базы.
