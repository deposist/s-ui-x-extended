# S-UI-X Extended v1.1.1-beta8

## AmneziaWG 3.1 random trailers fixed

Random trailers, introduced in beta7, broke handshakes: with the switch on, no client could connect, and the panel showed no handshakes at all. The engine wrote handshake messages into a buffer sized for the message plus the random trailer, but the marshaling code demands the exact message size, returned a length error, and the caller ignored it. The packet went out with padding, zeros and the trailer - no handshake body, a zero type field outside every configured header range, and MACs shifted into the trailer region. Neither side could classify or authenticate the packet.

The engine is now pinned to `deposist/wireguard-go v0.0.5-extended-1.6.2` (commit `8f4b19e`, a fork of `shtorm-7/wireguard-go v0.0.5-extended-1.6.1`). Handshake initiation, response and cookie reply are marshaled into an exact-size slice, and a marshaling error now aborts the send instead of shipping an empty body. A loopback test in the engine checks the wire format with trailers on; the panel's integration test (`core/awg31_wire_test.go`) checks it against the real core.

After upgrading, endpoints with Random trailers enabled work as advertised. Disable cookies is unaffected - it worked in beta7. Clients on AmneziaVPN 5.0.1.5 or newer connect as before; the flag still has to match on the server and every client.

No database migration. The panel and sing-box core configuration format did not change.

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta8/install.sh \
  | sudo bash -s -- v1.1.1-beta8
```

Check the version with `/usr/local/s-ui/sui -v`, then toggle the connection in your AmneziaWG client: handshakes complete and traffic flows.

## AmneziaWG 3.1: исправлены случайные трейлеры

Переключатель «Случайные трейлеры», появившийся в beta7, ломал рукопожатия: при включённом флаге ни один клиент не подключался, а панель не показывала ни одного рукопожатия. Движок записывал сообщение рукопожатия в буфер размером «сообщение плюс трейлер», а кодировщик требует точный размер сообщения - он возвращал ошибку длины, и вызывающий код её игнорировал. На провод уходили паддинг, нули и трейлер: тела рукопожатия не было, поле типа равнялось нулю и не попадало ни в один настроенный диапазон заголовков, а MAC уезжали в область трейлера. Стороны не могли ни опознать, ни проверить пакет.

Движок перпиннен на `deposist/wireguard-go v0.0.5-extended-1.6.2` (коммит `8f4b19e`, форк `shtorm-7/wireguard-go v0.0.5-extended-1.6.1`). Инициация, ответ и cookie-reply теперь кодируются в срез точного размера, а ошибка кодирования прерывает отправку вместо пустого тела. Loopback-тест в движке проверяет формат пакета с включёнными трейлерами; интеграционный тест панели (`core/awg31_wire_test.go`) проверяет то же на реальном ядре.

После обновления эндпоинты с включёнными случайными трейлерами работают как задумано. Отключение cookies не задето - оно работало и в beta7. Клиенты AmneziaVPN 5.0.1.5 и новее подключаются как раньше; флаг по-прежнему должен совпадать на сервере и всех клиентах.

Миграция базы данных не нужна. Формат конфигурации панели и ядра sing-box не менялся.

## Обновление

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta8/install.sh \
  | sudo bash -s -- v1.1.1-beta8
```

Проверьте версию через `/usr/local/s-ui/sui -v`, затем переключите подключение в клиенте AmneziaWG: рукопожатия завершаются, трафик идёт.
