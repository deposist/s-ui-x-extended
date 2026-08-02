# S-UI-X Extended v1.0.9-beta1

This beta fixes `update manifest is invalid` when a beta installation selects a newer stable release without changing its saved channel. Stable releases now validate against their `main` manifest, while prerelease artifacts still require a beta manifest. Version, platform, archive name, and SHA-256 checks remain in place.

Release builds also stop downloading Debian signing keys from a public keyserver. Cronet jobs use an official pinned Debian keyring package verified by SHA-256, with regression checks that prevent the old flaky path from returning.

No database migration or manual configuration change is required.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta1/install.sh \
  | sudo bash -s -- v1.0.9-beta1
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.0.9-beta1.md`](../docs/releases/v1.0.9-beta1.md).

## Обновление через консоль

Эта бета-версия исправляет ошибку `update manifest is invalid`, которая возникала, когда установка с beta-каналом выбирала более новый стабильный релиз. Стабильный релиз теперь проверяется по своему манифесту `main`, а для предварительной версии по-прежнему нужен манифест beta. Проверки версии, платформы, имени архива и SHA-256 сохранены.

Сборка Cronet больше не загружает ключи Debian с публичного сервера. Релизное задание использует официальный закреплённый пакет ключей с проверкой SHA-256. Регрессионная проверка запрещает возврат старой ненадёжной схемы.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta1/install.sh \
  | sudo bash -s -- v1.0.9-beta1
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели. Миграция базы и ручное изменение конфигурации не требуются.
