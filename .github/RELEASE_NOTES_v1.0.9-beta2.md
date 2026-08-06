# S-UI-X Extended v1.0.9-beta2

This beta fixes WARP Endpoint creation. Registration previously demanded a bare IP peer address with a real port. Cloudflare returns a domain in `endpoint.host` and a placeholder `:0` port in the `v4`/`v6` fields, so genuine responses kept failing validation. The panel now accepts the actual endpoint shape, substitutes a usable port when it sees `:0`, and prefers a literal IP address so the core does not resolve its own peer through DNS at startup.

No database migration or manual configuration change is required.

## Upgrade through the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta2/install.sh \
  | sudo bash -s -- v1.0.9-beta2
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

## Обновление через консоль

Эта бета-версия исправляет создание WARP Endpoint. Модуль регистрации ждал peer-адрес в виде голого IP с настоящим портом, а в ответе Cloudflare домен лежит в `endpoint.host`, а в `v4`/`v6` порт равен `0`. Поэтому валидные ответы не проходили проверку. Теперь панель принимает реальную форму endpoint, подставляет рабочий порт вместо `0` и предпочитает литеральный IP-адрес, чтобы core не резолвил свой домен через DNS при старте.

Миграция базы и ручное изменение конфигурации не требуются.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta2/install.sh \
  | sudo bash -s -- v1.0.9-beta2
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели.