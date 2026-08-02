# S-UI-X Extended v1.0.8

Version 1.0.8 contains all changes from the 1.0.8 beta series after version 1.0.7.

## Main changes

- Self-update verifies release metadata, checksums, size limits, archive contents, and rollback state before replacing the binary. It also recovers the numeric pending marker written by beta8.
- Compact `betaN` and `rcN` suffixes are compared numerically. This fixes beta10 offering beta9 instead of beta11.
- Regional routing and DNS presets use verified local `.srs` files. The panel refreshes them daily and protects direct downloads against private-address redirects and DNS rebinding.
- Route and DNS rules support nested `and` and `or` groups with bounded backend validation.
- Paid subscriptions keep a provider-charge ledger, recover confirmed orders after pending expiry, retry unfinished provider work, and restore purchased capacity correctly on refunds.
- Database restore drains active work before replacing SQLite. Failed traffic, audit, IP-monitor, and token-use writes remain queued for retry.
- The panel improves narrow-screen layouts, empty states, Save explanations, accessibility, WebSocket recovery, stale-request handling, CSRF refresh, and browser-storage recovery.
- Core lifecycle, Clash logs, certificate renewal, WireGuard IPC removal, subscription parsing, generated links, sessions, realtime delivery, redirects, and cache headers handle their recorded edge cases.
- The bundled core is `sing-box-extended v1.13.14-extended-2.5.4`.
- Release jobs verify tag provenance, checksums, archive inventories, SBOMs, provenance, vulnerability scans, Windows packages, and Docker platform digests.

Normal startup creates the provider-charge ledger. No manual configuration change is required.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.8/install.sh \
  | sudo bash -s -- v1.0.8
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Servers caught in the beta8 to beta9 restart loop must use the console command. Beta10 and beta11 users should switch the panel selector to **Main (stable)** before applying 1.0.8; applying stable 1.0.8 while `beta` remains selected fails with `update manifest is invalid`. If beta10 offers beta9, do not apply that downgrade.

Full release notes: [`docs/releases/v1.0.8.md`](../docs/releases/v1.0.8.md).

## Обновление через консоль

Версия 1.0.8 включает все изменения линейки beta после 1.0.7: защищённое самообновление и восстановление, локальные наборы правил, вложенные условия маршрутов и DNS, исправления платежей и базы, улучшения панели и обновлённый core.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.8/install.sh \
  | sudo bash -s -- v1.0.8
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели. При обычном запуске создаётся таблица списаний провайдера. Ручное изменение конфигурации не требуется.

Если сервер постоянно перезапускается после перехода с beta8 на beta9, используйте команду из консоли. В beta10 и beta11 перед установкой 1.0.8 через панель переключите канал на **Main (стабильные)**: при выбранном `beta` установка завершается ошибкой `update manifest is invalid`. Если beta10 предлагает beta9, не устанавливайте это понижение.
