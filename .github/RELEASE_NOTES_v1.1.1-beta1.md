# S-UI-X Extended v1.1.1-beta1

This beta fixes two problems reported against 1.1.0: TrustTunnel clients that stayed "connected" while no site opened, and inbound saves that could crash-loop sing-box. No database migration is required.

Clients assigned to a TrustTunnel or Mieru inbound received a subscription with a username but no password, so the official TrustTunnel client connected, showed Connected, and then every request failed server-side with `authorization failed`. The panel now generates the missing password when it creates the client credentials block, and fills in an empty password when the block already exists. To repair a client hit by the old behavior, open the inbound in the panel, save it once, and re-download the subscription.

Saving certain inbounds could make sing-box restart in a loop with `json: unknown field`. The panel no longer emits fields the inbound option struct does not declare, validates the generated inbound before committing a save, and validates a restored core config before restarting sing-box. The `github.com/pion/dtls/v3` dependency was also updated from 3.1.2 to 3.1.4 (GO-2026-6165); the panel does not call the affected symbols.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta1/install.sh \
  | sudo bash -s -- v1.1.1-beta1
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.1.1-beta1.md`](../docs/releases/v1.1.1-beta1.md).

## Обновление через консоль

Эта beta исправляет две проблемы 1.1.0: клиенты TrustTunnel оставались «подключёнными», при этом сайты не открывались, и сохранение inbound'ов могло отправить sing-box в цикл перезапусков. Миграция базы не требуется.

Клиенты на inbound TrustTunnel или Mieru получали подписку с именем пользователя, но без пароля: официальный клиент подключался, показывал Connected, и каждый запрос падал на сервере с `authorization failed`. Теперь панель генерирует недостающий пароль при создании блока credentials и заполняет пустой пароль, если блок уже существует. Чтобы починить клиента, откройте inbound в панели, сохраните его один раз и заново скачайте подписку.

Сохранение некоторых inbound'ов могло заставить sing-box перезапускаться по кругу с `json: unknown field`. Панель больше не передаёт лишние поля, проверяет сгенерированный inbound до коммита и конфиг ядра до перезапуска. Зависимость `github.com/pion/dtls/v3` обновлена с 3.1.2 до 3.1.4 (GO-2026-6165); панель затронутые символы не вызывает.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta1/install.sh \
  | sudo bash -s -- v1.1.1-beta1
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели.

Полные заметки о релизе: [`docs/releases/v1.1.1-beta1.md`](../docs/releases/v1.1.1-beta1.md).
