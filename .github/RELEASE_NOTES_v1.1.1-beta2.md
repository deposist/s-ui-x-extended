# S-UI-X Extended v1.1.1-beta2

This beta fixes a server crash: saving a TrustTunnel inbound without a TLS config could send the restart watchdog into a loop. No database migration.

The TrustTunnel protocol always requires TLS. When you saved an inbound with no TLS template, the panel committed the row, then sing-box failed with `TLS required`. The watchdog retried every ~30 seconds, creating a crash loop.

Now the panel rejects these saves before they hit the database. TrustTunnel is flagged as `onlyTls` in the manifest, so the frontend blocks the save, and the backend API returns an error. This covers the UI and the API/import path.

If you have a broken TrustTunnel inbound from a previous version: edit it, attach a TLS template, save. It starts immediately.

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta2/install.sh \
  | sudo bash -s -- v1.1.1-beta2
```

Verify:
```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta2
systemctl is-active s-ui  # active
```

Full notes: [`docs/releases/v1.1.1-beta2.md`](https://github.com/deposist/s-ui-x-extended/blob/v1.1.1-beta2/docs/releases/v1.1.1-beta2.md)

## Обновление

TrustTunnel всегда требовал TLS. Но при сохранении без TLS-шаблона панель коммитила строку, а sing-box падал с `TLS required`. Watchdog перезапускал ядро каждые ~30 секунд — бесконечный краш-луп.

Теперь сохранение отклоняется до базы. Фронтенд блокирует через `onlyTls`, бэкенд через `TLSRequiredTypes()` — защита работает и в UI, и при API/импорте.

Пострадавший TrustTunnel inbound лечится просто: отредактируйте, привяжите TLS, сохраните.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta2/install.sh \
  | sudo bash -s -- v1.1.1-beta2
```

Проверка:
```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta2
systemctl is-active s-ui  # active
```

Полные заметки: [`docs/releases/v1.1.1-beta2.md`](https://github.com/deposist/s-ui-x-extended/blob/v1.1.1-beta2/docs/releases/v1.1.1-beta2.md)
