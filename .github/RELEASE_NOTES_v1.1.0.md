# S-UI-X Extended v1.1.0

Version 1.1.0 contains all changes from the 1.1.0 beta series after version 1.0.9.

## Main changes

- Managed AmneziaWG no longer depends on the Paid Subscriptions page: the status line moved to a new "AmneziaWG 3.0" tab in panel Settings and is served from `api/awg/status`. The panel provisions the device encryption key automatically on start, the old single-endpoint mode converts to managed endpoints on first start, managed endpoints with devices can be edited again, and the endpoint form gains obfuscation presets and a public endpoint suggestion.
- Call outbounds can no longer crash-loop the core. The form no longer saves the unsupported `server` field and marks `join_link` required, and the panel rejects both mistakes server-side. If your core is already restart-looping, delete the call outbound on the Outbounds page.
- Saving a new telegram backup passphrase no longer fails with `save: secret setting decrypt failed` when the stored value cannot be decrypted. The new passphrase overwrites the broken value.
- The panel loads in browsers that block storage, the page `lang` attribute follows the selected interface language, and the self-updater refuses HTTPS-to-HTTP redirects and invalid release tags.
- Fixed the core start cooldown log printing `15ns seconds`.

The database schema is unchanged. Existing AWG setups convert automatically on first start.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.0/install.sh \
  | sudo bash -s -- v1.1.0
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.1.0.md`](../docs/releases/v1.1.0.md).

## Обновление через консоль

Версия 1.1.0 включает все изменения линейки beta после 1.0.9: управляемый AmneziaWG с настройками в панели, автоматическим ключом устройств и пресетами обфускации, исправления call-outbound и passphrase telegram-бэкапа, а также исправления запуска панели и самообновления.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.0/install.sh \
  | sudo bash -s -- v1.1.0
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели. Схема базы данных не меняется, существующие настройки AWG конвертируются автоматически при первом запуске.
