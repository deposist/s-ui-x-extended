# S-UI-X Extended v1.0.9-beta4

Version 1.0.9-beta4 moves the core to sing-box-extended 2.6.5 with AmneziaWG 3.0, adds the Call protocol, and fixes periodic AWG reconcile failing with no reason in the log.

AmneziaWG 3.0 changed the server option set. J1/J2/J3 and itime are gone; managed endpoints and device configs now use the header protection key, content padding addition, rekey/reject/keepalive timings, and max handshake attempts. Existing AWG 2.0 endpoint options migrate to 3.0 automatically on the first start after the upgrade. WARP endpoints get the removed 2.x fields cleaned and timing defaults added. Amnezia 2.x clients keep working with the new core.

The new Call inbound creates a WebRTC bridge room (dion, telemost, vk, wbstream) and clients join through a `join_link`. The `with_call` build tag is enabled in all release builds. The panel also gained VPN client/server options (default gateway, pool size, reconnect and reject delays), AnyTLS `client_metadata`, MASQUE license and private keys, and `rmux` multiplexing.

TrustTunnel no longer exposes `bbr_profile`, which sing-box 2.6.x removed. Re-save a trusttunnel inbound or outbound saved before this release so the field leaves the saved options.

Endpoint-scoped AWG devices now reconcile against their own endpoint tag instead of the legacy global AWG setting. The periodic reconcile no longer fails when that legacy setting is off, and reconcile errors now log the endpoint ID with the sanitized reason.

## Upgrade through the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta4/install.sh \
  | sudo bash -s -- v1.0.9-beta4
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

## Обновление через консоль

Версия 1.0.9-beta4 переводит ядро на sing-box-extended 2.6.5 с AmneziaWG 3.0, добавляет протокол Call и чинит периодический AWG reconcile, который падал в логе без указания причины.

В AmneziaWG 3.0 изменился набор опций сервера. Поля J1/J2/J3 и itime убраны; управляемые эндпоинты и конфиги устройств теперь используют ключ защиты заголовков, дополнение контента, тайминги rekey, reject и keepalive, а также максимум попыток рукопожатия. Существующие опции эндпоинтов AWG 2.0 мигрируют на 3.0 автоматически при первом запуске после обновления. У WARP-эндпоинтов вычищаются старые поля 2.x и добавляются тайминги по умолчанию. Клиенты Amnezia 2.x продолжают работать с новым ядром.

Новый входящий Call создаёт комнату WebRTC-моста (dion, telemost, vk, wbstream), клиенты подключаются по ссылке `join_link`. Сборочный тег `with_call` включён во всех сборках. В панели появились опции VPN client/server (шлюз по умолчанию, размер пула, задержки reconnect и reject), `client_metadata` у AnyTLS, ключи license и private у MASQUE, мультиплексирование `rmux`.

У TrustTunnel убрано поле `bbr_profile`: sing-box 2.6.x его больше не принимает. Входящий или исходящий trusttunnel, сохранённый до этого релиза, откройте в панели и сохраните заново, чтобы поле ушло из опций.

Устройства AWG, привязанные к эндпоинту, теперь сверяются со своим тегом, а не с глобальной legacy-настройкой AWG. Периодический reconcile больше не падает, когда legacy-схема выключена, а ошибки reconcile пишут ID эндпоинта и причину.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.9-beta4/install.sh \
  | sudo bash -s -- v1.0.9-beta4
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели.
