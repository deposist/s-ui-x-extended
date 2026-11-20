# S-UI-X Extended v1.1.2-beta12

- The Clients page loads one page at a time. Search, filters, and sorting still cover the full list. The overview fetches only the clients it displays.
- Traffic charts calculate their points in SQLite. Paired nodes reuse subscription data while assignments and access rules stay current.
- Slave traffic accounting reuses its database statement. A client with only delayed start activates once, on first traffic.
- AWG skips database updates when device state and counters are unchanged. Backups copy history tables inside SQLite.

## Upgrade

Back up the panel database before upgrading. On Linux, run:

```bash
curl -fLsS https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.2-beta12/install.sh \
  | sudo bash -s -- v1.1.2-beta12
```

On Windows, extract the ZIP for your architecture and run `install-windows.bat` as Administrator. For Docker, use `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta12`. To roll back, reinstall the previous version and restore your backup.

## По-русски

- Панель загружает клиентов по страницам. Поиск, фильтры и сортировка работают по всему списку. На главную страницу попадают только клиенты из сводки.
- Панель рассчитывает точки графиков в SQLite. При выдаче подписок нода повторно использует данные, пока назначения и правила доступа актуальны.
- Slave повторно использует запрос для учёта трафика. Если для клиента задан только отложенный старт, он срабатывает при первом трафике.
- AWG пропускает обновление базы, когда состояние устройства и счётчики не изменились. При резервном копировании таблицы истории копируются средствами SQLite.

Перед обновлением сохраните копию базы панели. Для Linux используйте команду выше. В Windows распакуйте ZIP для своей архитектуры и запустите `install-windows.bat` от имени администратора. Образ Docker: `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta12`. Для отката установите предыдущую версию и восстановите копию базы.
