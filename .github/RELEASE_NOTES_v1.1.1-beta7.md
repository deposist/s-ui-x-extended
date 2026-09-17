# S-UI-X Extended v1.1.1-beta7

## AmneziaWG 3.1 switches

The WireGuard endpoint form now has two new switches in the Amnezia block: Random trailers and Disable cookies. They turn on the AmneziaWG 3.1 features on top of the existing 3.0 profile (header protection, content padding, rekey timings).

Random trailers appends a random number of bytes to every packet. Handshake messages no longer have a fixed, fingerprintable size in the channel. Disable cookies stops the interface from answering cookie replies and skips the MAC2 check under load, so the cookie exchange disappears as a signature.

Two things to know before you flip them:

- Random trailers must match on the server and on every client. Clients older than AmneziaVPN 5.0.1.5 do not know the key at all, so leave it off while you have such clients. Mismatched flags drop handshakes silently.
- Disable cookies is server-side only. Turn it on without touching clients.

Both switches default to off, and the panel writes them into configs only when they are on. Existing endpoints and device configs keep their exact previous content.

The bundled sing-box-extended core is now `v1.14.0-extended-2.7.5`: it exposes both options in the wireguard and warp amnezia schemas and passes them to the engine. The engine (wireguard-go extended) already carried the implementation. Managed AWG device configs include the flags, so a device QR or `.conf` download carries them too.

No database migration. Existing WireGuard and WARP endpoints, managed AWG devices and their subscriptions continue to work unchanged.

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta7/install.sh \
  | sudo bash -s -- v1.1.1-beta7
```

Check `/usr/local/s-ui/sui -v`, then open a WireGuard endpoint and confirm the two switches in the Amnezia block. On Windows, install the matching archive from this release.

This is a beta. Test the switches on a non-critical endpoint first, and only enable Random trailers once every client is on AmneziaVPN 5.0.1.5 or newer.

## Переключатели AmneziaWG 3.1

В форме WireGuard-эндпоинта, в блоке Amnezia, появились два новых переключателя: «Случайные трейлеры» и «Отключить cookies». Они включают возможности AmneziaWG 3.1 поверх существующего профиля 3.0 (защита заголовков, паддинг содержимого, тайминги пересоздания ключей).

Случайные трейлеры дописывают случайное число байт к каждому пакету. Сообщения рукопожатия больше не имеют фиксированного, отпечатываемого размера в канале. Отключение cookies запрещает интерфейсу отвечать cookie-reply и пропускает проверку MAC2 под нагрузкой, так что обмен cookie исчезает как сигнатура.

Перед включением стоит знать две вещи:

- Случайные трейлеры должны совпадать на сервере и на каждом клиенте. Клиенты старше AmneziaVPN 5.0.1.5 этого ключа не знают вовсе, поэтому при таких клиентах оставьте его выключенным. При несовпадении рукопожатия падают молча.
- Отключение cookies - чисто серверная опция. Включайте, не трогая клиентов.

Оба переключателя по умолчанию выключены, и панель пишет их в конфиги только во включённом состоянии. Существующие эндпоинты и конфиги устройств сохраняют прежнее содержимое без изменений.

Комплектное ядро sing-box-extended обновлено до `v1.14.0-extended-2.7.5`: оно отдаёт обе опции в схемах amnezia для wireguard и warp и передаёт их движку. Сам движок (extended-сборка wireguard-go) реализацию уже содержал. Конфиги управляемых устройств AWG включают флаги, поэтому QR и скачивание `.conf` тоже их переносят.

Миграция базы данных не нужна. Существующие WireGuard- и WARP-эндпоинты, управляемые устройства AWG и их подписки продолжают работать без изменений.

## Обновление

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta7/install.sh \
  | sudo bash -s -- v1.1.1-beta7
```

Проверьте версию через `/usr/local/s-ui/sui -v`, затем откройте WireGuard-эндпоинт и убедитесь, что в блоке Amnezia есть оба переключателя. Для Windows используйте архив этого выпуска под вашу архитектуру.

Это бета. Проверьте переключатели сначала на некритичном эндпоинте и включайте случайные трейлеры, только когда все клиенты обновились до AmneziaVPN 5.0.1.5 или новее.
