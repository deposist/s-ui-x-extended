# S-UI-X Extended v1.0.9-beta3

Version 1.0.9-beta3 fixes a database restore deadlock. An open dashboard WebSocket held a shared read lease on the database for its whole lifetime, and restore waits for every reader before swapping SQLite. The restore never started: the panel froze, API requests queued behind the lock, and the maintenance job logged `cron: skip` every two seconds. A manual service restart closed the socket, which is why it appeared to help.

The WebSocket now holds the lease only during the handshake, and maintenance jobs skip ticks silently while a restore is in progress. Restore completes with the dashboard open. No database migration or manual configuration change is required.

## Upgrade through the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta3/install.sh \
  | sudo bash -s -- v1.0.9-beta3
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

## Обновление через консоль

Версия 1.0.9-beta3 исправляет зависание восстановления базы данных. Пока открыта панель, её веб-сокет (WebSocket) держит общую блокировку чтения базы всё время сессии. Восстановление перед заменой SQLite берёт исключительную блокировку и ждёт всех читателей, поэтому при открытом сокете оно не начиналось вовсе. Панель замирала, запросы к API вставали в очередь, а служебное задание каждые две секунды писало в лог `cron: skip`. Ручной перезапуск службы закрывал сокет, поэтому казалось, что он помогает.

Теперь сокет держит блокировку только во время рукопожатия, а служебные задания молча пропускают такт, пока идёт восстановление. База восстанавливается при открытой панели, без перезапуска. Миграция базы и ручное изменение конфигурации не требуются.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta3/install.sh \
  | sudo bash -s -- v1.0.9-beta3
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели.
