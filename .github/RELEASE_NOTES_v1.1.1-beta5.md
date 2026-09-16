# S-UI-X Extended v1.1.1-beta5

The Additional collections section is now visible on the Basics tab in Settings. Beta4 added the structured forms but mounted them on a page the panel no longer renders, so the new fields never appeared. This release puts them in the tab users actually see.

Where to look: open Settings, switch to the Basics tab and scroll to the bottom. Advanced Collections is the last section, after Experimental settings. It holds three editors, each with an add button: certificate providers (ACME, Tailscale, Cloudflare Origin CA), reusable HTTP clients, and Linux network namespaces.

What the collections do:

- Certificate providers. The core obtains certificates on its own. An ACME provider issues and renews Let's Encrypt certificates through HTTP-01, TLS-ALPN-01 or DNS-01, so an inbound or endpoint with TLS gets a real certificate without manual steps. A Tailscale provider takes certificates from your existing Tailscale endpoint; a Cloudflare Origin CA provider covers servers behind Cloudflare.
- HTTP clients. A named HTTP transport configured once: engine, HTTP versions 1 through 3, headers, receive windows, TLS and dial settings. ACME providers, rule-sets and providers reference it by tag instead of repeating the settings.
- Network namespaces (Linux only). Default opens an existing namespace by path; unshare creates an isolated one when the core starts. Saving the form changes only the configuration, the panel creates nothing.

How this differs from the terminal menu certificates: the terminal menu (items 20 and 21) installs the external acme.sh tool and issues certificates for the panel's own web interface, stopping the panel during issuance and leaving you to wire the file paths into TLS settings by hand. Certificate providers work inside the sing-box core, issue certificates for inbound protocols such as VLESS with TLS or Hysteria 2, run the ACME challenge and renewals themselves, and are referenced by tag with no file paths to maintain. Tailscale and Cloudflare Origin CA providers have no terminal-menu counterpart. In short: the terminal menu protects the panel's own HTTPS, certificate providers protect the proxy protocols the core serves.

The beta3 protocol fields live in the protocol forms on the Inbounds and Outbounds pages, most of them after the related toggle is on: QUIC options for Hysteria, Hysteria 2 and TUIC appear when the QUIC settings switch is enabled; Mieru has MTU and handshake mode; MASQUE has a local address and port; WireGuard has UDP mapping, UDP filtering and UDP NAT limits. OpenVPN endpoints, the cloudflared inbound and the usbip services are separate entries on the Endpoints, Inbounds and Services pages.

What USB/IP does: it forwards USB devices over the network, so a device plugged into one machine appears on another as a local USB device with its normal drivers. The usbip-server service runs on the machine with the physical port: set a listen address, then export devices from a hand-written list (default provider) or automatically as they appear (dynamic provider). The usbip-client service runs on the machine that needs the devices: point it at the server and pick devices by bus ID, vendor ID, product ID or serial number, and the operating system sees them as local USB. Linux needs the vhci driver and device access rights, so the service may require root or group membership; the core reports this at startup.

Forms validate input before saving. The panel Save button writes all three collections to the panel database in one request; this was verified end to end on a live panel.

The database schema is unchanged from beta4. No migration is required.

Full release notes: [`docs/releases/v1.1.1-beta5.md`](../docs/releases/v1.1.1-beta5.md).

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta5/install.sh \
  | sudo bash -s -- v1.1.1-beta5
```

Verify:

```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta5
systemctl is-active s-ui  # active
```

## Обновление

Раздел «Additional collections» теперь виден на вкладке «Basics» в настройках. Beta4 добавил структурированные формы, но разместил их на странице, которую панель больше не рендерит, поэтому новые поля не появлялись. Этот релиз помещает их во вкладку, которую видит пользователь.

Где искать: откройте настройки, перейдите на вкладку «Basics» и прокрутите вниз. Advanced collections - последний раздел, после Experimental settings. В нём три редактора с кнопкой добавления: провайдеры сертификатов (ACME, Tailscale, Cloudflare Origin CA), общие HTTP-клиенты и сетевые пространства имён Linux.

Что делают коллекции:

- Провайдеры сертификатов. Ядро получает сертификаты само. ACME выпускает и перевыпускает сертификаты Let's Encrypt через HTTP-01, TLS-ALPN-01 или DNS-01, поэтому inbound или endpoint с TLS получает настоящий сертификат без ручных шагов. Провайдер Tailscale берёт сертификаты у существующего Tailscale-эндпоинта; Cloudflare Origin CA - для серверов за Cloudflare.
- HTTP-клиенты. Именованный HTTP-транспорт, который настраивают один раз: движок, версии HTTP с первой по третью, заголовки, окна приёма, TLS и настройки исходящего соединения. ACME-провайдеры, rule-set и провайдеры ссылаются на него по тегу, а не повторяют настройки.
- Сетевые пространства имён (только Linux). Режим default открывает существующее пространство по пути; unshare создаёт изолированное при запуске ядра. Сохранение формы меняет только конфигурацию, панель ничего не создаёт.

Чем это отличается от сертификатов из терминального меню: терминальное меню (пункты 20 и 21) ставит внешний инструмент acme.sh и выдаёт сертификаты для самого веб-интерфейса панели, останавливая панель на время выпуска, а пути к файлам вы вписываете в настройки TLS вручную. Провайдеры сертификатов работают внутри ядра sing-box, обслуживают inbound-протоколы вроде VLESS с TLS или Hysteria 2, сами проводят ACME-проверку и продление, и ссылаются на них по тегу без путей к файлам. Провайдеров Tailscale и Cloudflare Origin CA в терминальном меню нет. Коротко: терминальное меню защищает HTTPS самой панели, провайдеры сертификатов защищают прокси-протоколы ядра.

Поля протоколов из beta3 живут в формах протоколов на страницах Inbounds и Outbounds, большинство появляется после включения соответствующего переключателя: QUIC-опции у Hysteria, Hysteria 2 и TUIC видны при включённом переключателе QUIC-настроек; у Mieru есть MTU и режим рукопожатия; у MASQUE - локальный адрес и порт; у WireGuard - UDP mapping, UDP filtering и лимиты UDP NAT. Эндпоинты OpenVPN, inbound cloudflared и службы usbip - отдельные записи на страницах Endpoints, Inbounds и Services.

Что делает USB/IP: передаёт USB-устройства по сети, так что устройство, воткнутое в один компьютер, появляется на другом как локальное USB со штатными драйверами. Служба usbip-server запускается на машине с физическим портом: задаётся адрес прослушивания, затем устройства экспортируются из списка, составленного вручную (провайдер default), или автоматически по мере появления (провайдер dynamic). Служба usbip-client нужна машине, которая пользуется удалёнными устройствами: укажите сервер и выберите устройства по bus ID, vendor ID, product ID или серийному номеру, и система увидит их как локальные USB. Linux требует драйвер vhci и прав доступа к устройствам, поэтому службе может понадобиться root или группа; ядро сообщит об этом при старте.

Формы проверяют ввод до сохранения. Кнопка Save панели записывает все три коллекции в базу одним запросом; это проверено на живой панели.

Схема базы данных не менялась со времён beta4. Миграция не требуется.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta5/install.sh \
  | sudo bash -s -- v1.1.1-beta5
```

Проверка:

```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta5
systemctl is-active s-ui  # active
```

Полные заметки: [`docs/releases/v1.1.1-beta5.md`](https://github.com/deposist/s-ui-x-extended/blob/v1.1.1-beta5/docs/releases/v1.1.1-beta5.md)
