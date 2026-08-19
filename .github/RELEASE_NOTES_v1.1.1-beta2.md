# S-UI-X Extended v1.1.1-beta2

This beta fixes a server crash: saving a TrustTunnel inbound without a TLS configuration stopped sing-box from starting and sent the panel into a restart loop. No database migration is required.

The TrustTunnel protocol requires TLS — the core rejects an inbound with `TLS required` when no TLS config is attached. The panel validation did not check this, so a save with no TLS template passed, committed the row, and then sing-box failed to start. The watchdog kept retrying, creating a crash loop.

The panel now blocks the save before it reaches the database — on the frontend (the protocol picker flags TrustTunnel as TLS-required) and on the backend (the API path rejects the save and returns an error). If you have an affected TrustTunnel inbound from a previous version, edit it, attach a TLS configuration, and save.

## Fixed

- TrustTunnel inbound saves without a TLS template are now rejected before commit, preventing a core restart loop. The protocol manifest now marks TrustTunnel as `onlyTls`, the frontend blocks the save, and the server-side save path returns an error for API and import callers.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta2/install.sh \
  | sudo bash -s -- v1.1.1-beta2
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.1.1-beta2.md`](../docs/releases/v1.1.1-beta2.md).

## Обновление через консоль

Эта beta исправляет падение сервера: сохранение inbound TrustTunnel без TLS-конфигурации останавливало sing-box и отправляло панель в цикл перезапусков. Миграция базы не требуется.

Протокол TrustTunnel требует TLS — ядро возвращает `TLS required`, если TLS-конфигурация не привязана. Проверка в панели этого не делала, поэтому сохранение без TLS-шаблона проходило, строка коммитилась, а sing-box не запускался. Watchdog перезапускал ядро снова и снова — получался краш-луп.

Теперь панель блокирует сохранение до базы данных — на фронтенде (TrustTunnel помечен как TLS-обязательный) и на бэкенде (API-путь возвращает ошибку). Если у вас есть затронутый inbound TrustTunnel с предыдущей версии, отредактируйте его, привяжите TLS-конфигурацию и сохраните.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta2/install.sh \
  | sudo bash -s -- v1.1.1-beta2
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели.

Полные заметки о релизе: [`docs/releases/v1.1.1-beta2.md`](../docs/releases/v1.1.1-beta2.md).
