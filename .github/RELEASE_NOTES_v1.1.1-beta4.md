# S-UI-X Extended v1.1.1-beta4

The Additional collections section on the Basics page now has structured forms for certificate providers, HTTP clients and network namespaces. Certificate providers cover ACME, Tailscale and Cloudflare Origin CA. The HTTP client form includes HTTP versions 1 through 3, headers, TLS and dial settings. Network namespace forms cover existing namespaces and isolated namespaces created by the core.

Forms validate tags and field values. Closing without saving leaves the stored object unchanged, while normal edits retain unknown fields, explicit `false` and `0` values, lists and secrets.

The QUIC settings switch now opens an empty options block for Hysteria, Hysteria 2 and TUIC. Opening the block does not write defaults; turning it off removes only QUIC fields.

This update does not change the database schema. Download a database backup before upgrading a production server.

Full release notes: [`docs/releases/v1.1.1-beta4.md`](../docs/releases/v1.1.1-beta4.md).

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta4/install.sh \
  | sudo bash -s -- v1.1.1-beta4
```

Verify:

```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta4
systemctl is-active s-ui  # active
```

## Обновление

В блоке «Дополнительные коллекции» на странице «Основы» появились формы для провайдеров сертификатов, HTTP-клиентов и сетевых пространств имён. Провайдеры сертификатов поддерживают ACME, Tailscale и Cloudflare Origin CA. В форме HTTP-клиента есть версии HTTP с первой по третью, заголовки, TLS и настройки исходящего соединения. Для сетевых пространств доступны существующее пространство и изолированное пространство, которое ядро создаёт при запуске.

Формы проверяют теги и значения полей. Закрытие без сохранения не меняет сохранённый объект, а обычная правка сохраняет неизвестные поля, явные значения `false` и `0`, списки и секреты.

Переключатель параметров QUIC теперь открывает пустой блок у Hysteria, Hysteria 2 и TUIC. Открытие не записывает значения по умолчанию, а выключение удаляет только поля QUIC.

Схема базы данных не меняется. Перед обновлением рабочего сервера выгрузите резервную копию базы из панели.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta4/install.sh \
  | sudo bash -s -- v1.1.1-beta4
```

Проверка:

```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta4
systemctl is-active s-ui  # active
```

Полные заметки: [`docs/releases/v1.1.1-beta4.md`](https://github.com/deposist/s-ui-x-extended/blob/v1.1.1-beta4/docs/releases/v1.1.1-beta4.md)
