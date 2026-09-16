# S-UI-X Extended v1.1.1-beta6

## MTProxy restart fix

This release fixes a core shutdown hang when MTProxy is enabled. The panel could show the core as stopped while its MTProxy port remained open. New connections then failed with `use of closed network connection`, and further core start or restart requests could not finish.

The bundled sing-box-extended core is now `v1.14.0-extended-2.7.4`. It closes the MTProxy listener before waiting for connections to finish. Proxy settings and existing Telegram links do not need to change. The database schema is unchanged from beta5.

## Upgrade

If the old process is already stuck, restart the entire service before updating:

```sh
sudo systemctl restart s-ui
```

The old process may need to reach systemd's stop timeout before the restart completes. Restarting the service disconnects active clients.

Install the beta explicitly:

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta6/install.sh \
  | sudo bash -s -- v1.1.1-beta6
```

Check `/usr/local/s-ui/sui -v`, then confirm that the panel reports the core as running. On Windows, install the matching archive from this release.

## Исправление перезапуска MTProxy

Выпуск исправляет зависание остановки ядра при включённом MTProxy. Панель могла показывать остановленное ядро, хотя порт MTProxy оставался открыт. Новые соединения завершались с ошибкой `use of closed network connection`, а повторный запуск или перезапуск ядра зависал.

Комплектное ядро sing-box-extended обновлено до `v1.14.0-extended-2.7.4`. Теперь MTProxy закрывает слушающий сокет перед ожиданием завершения соединений. Настройки прокси и ссылки Telegram менять не нужно. Схема базы данных осталась прежней, как в beta5.

## Обновление

Если старый процесс уже завис, сначала перезапустите всю службу командой `sudo systemctl restart s-ui`. Возможно, придётся дождаться таймаута остановки systemd. Перезапуск разорвёт активные подключения.

Затем выполните команду установки выше с явным тегом `v1.1.1-beta6`. Проверьте версию через `/usr/local/s-ui/sui -v` и убедитесь в панели, что ядро работает. Для Windows используйте архив этого выпуска под вашу архитектуру.
