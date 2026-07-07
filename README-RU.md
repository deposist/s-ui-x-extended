## S-UI-X Extended

[English](README.md) | [Русский](README-RU.md)

<p align="center">
  <img width="492" height="450" alt="Логотип S-UI-X Extended" src="https://raw.githubusercontent.com/deposist/s-ui-x-extended/refs/heads/main/docs/592996937-cfc9da97-f8ea-4c68-961c-2bf164932272.png" />
</p>
<p align="center">
  <a href="https://github.com/deposist/s-ui-x-extended/releases/latest">
    <img src="https://img.shields.io/github/v/release/deposist/s-ui-x-extended?style=for-the-badge&label=release" alt="Релиз">
  </a>
  <a href="https://github.com/deposist/s-ui-x-extended/releases">
    <img src="https://img.shields.io/github/downloads/deposist/s-ui-x-extended/total?style=for-the-badge&label=downloads" alt="Всего загрузок">
  </a>
  <a href="https://github.com/deposist/s-ui-x-extended/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/deposist/s-ui-x-extended?style=for-the-badge" alt="Лицензия">
  </a>
  <a href="https://github.com/deposist/s-ui-x-extended/stargazers">
    <img src="https://img.shields.io/github/stars/deposist/s-ui-x-extended?style=for-the-badge" alt="Звезды">
  </a>
</p>

<p align="center">
  <img width="1024" alt="Скриншоты панели S-UI-X Extended" src="https://github.com/deposist/s-ui-x-extended/blob/main/docs/screen1.png" />
</p>

<p align="center">
  <img width="1024" alt="Поддержать разработку S-UI-X Extended" src="docs/support-s-ui-x.png" />
</p>

## Поддержать S-UI-X Extended

S-UI-X Extended это проект с открытым исходным кодом. Пожертвования помогают оплачивать разработку, работу над безопасностью, тестирование и подготовку релизов.

- WEB: [https://web.tribute.tg/d/LRJ](https://web.tribute.tg/d/LRJ)
- Telegram: [https://t.me/tribute/app?startapp=dLRJ](https://t.me/tribute/app?startapp=dLRJ)

| Сеть | Адрес |
| ---- | ----- |
| TON | `UQB5-DZ3q5vjXGf3_tVUeOHPNuXMLh8lfY0MPW3uGzjdOzke` |
| ETH | `0x0e67e1b4363a163c36943Ef4F9227c3126bB952B` |
| SOL | `BtFm5E1BrUjpoaDNwv3emc2qbyvqkn6ECnDzwhgRn7Df` |
| TRX | `TFqEbp1Z82ZQebzDdsW1MbytMvVsHJGpPd` |
| BTC | `bc1qn86mfmsnackfwvjd4czjaalv75sh830fws7xc9` |

S-UI-X Extended это веб-панель на базе [`sing-box-extended`](https://github.com/shtorm-7/sing-box-extended), форка `SagerNet/sing-box` от shtorm-7.

Этот репозиторий основан на `alireza0/s-ui`, начиная с `v1.4.1`. Он сохраняет паритет с upstream s-ui-x `v1.5.10-beta7` и работает на ядре `sing-box-extended`.

Можно использовать опубликованные скрипты установки или сделать форк репозитория и собрать проект самостоятельно.

> Отказ от ответственности: проект предназначен только для личного обучения и обмена знаниями. Не используйте его в незаконных целях.

## Релизы

Руководства по настройке и использованию:

- Руководство на английском: [`docs/INSTRUCTIONS-EN.md`](docs/INSTRUCTIONS-EN.md)
- Руководство на русском: [`docs/INSTRUCTIONS-RU.md`](docs/INSTRUCTIONS-RU.md)

История релизов и заметки по обновлению:

- Английский changelog: [`CHANGELOG-EN.md`](CHANGELOG-EN.md)
- Русский changelog: [`CHANGELOG-RU.md`](CHANGELOG-RU.md)
- Changelog на упрощенном китайском: [`CHANGELOG-ZH.md`](CHANGELOG-ZH.md)
- Заметки последнего стабильного релиза: [`docs/releases/v1.0.1.md`](docs/releases/v1.0.1.md)
- Справка по паритету с upstream: [`docs/releases/v1.5.10-beta7.md`](docs/releases/v1.5.10-beta7.md)

## Чем отличается от `alireza0/s-ui`

<details>
  <summary>Показать подробности</summary>

Форк рассчитан на совместимость с существующими установками 1.x. Можно заменить бинарный файл на уже установленном сервере, а панель выполнит миграции базы при первом запуске. Поведение протоколов остается близким к upstream; основные изменения относятся к безопасности, эксплуатации, наблюдаемости, обновлениям и интерфейсу администратора.

- При свежей установке создается случайный первый пароль администратора. Пароли хранятся через bcrypt, браузерные сессии используют усиленные cookie, изменяющие API-запросы из браузера требуют CSRF-защиты, а API-токены хранятся в виде хэшей с правами `admin`, `read`, `write` или `observability`.
- Учетные данные Telegram, прокси-учетные данные, install salt и другие чувствительные настройки шифруются при хранении через secretbox. Сообщения Telegram, детали аудита, подписи резервных копий и история изменений редактируются перед выходом из панели.
- Сетевые входные данные проверяются строже. `X-Forwarded-For` игнорируется без настроенных доверенных прокси. Загрузка внешних подписок проверяет URL и resolved IP, по умолчанию блокирует private и loopback цели, ограничивает ответ 4 MiB и повторно проверяет IP во время dial.
- Подписки поддерживают отдельные секреты для каждого клиента в форматах link, JSON и Clash. Старые URL по имени клиента работают, пока `subSecretRequired=false`. Ответы подписок очищают headers, применяют лимиты на IP, поддерживают gzip и недолго кэшируют успешный вывод.
- sing-box запускается как встроенная Go-библиотека, а не как subprocess. Изменения clients, TLS, inbounds, outbounds, endpoints и services применяются через hot apply к затронутому объекту, когда это возможно. Полный restart используется только если изменение требует его или hot apply не может безопасно сохранить running config.
- Маршрутизация включает outbound-группы под управлением панели и участников от providers. Группы Selector, URLTest, fallback и failover используют проверку tag-ссылок, предварительный просмотр и capability metadata. Тип `failover` под управлением панели проверяет участников по HTTPS, переводит новые подключения с отказавшего активного участника и поддерживает явные all-down policies.
- Панель умеет обновляться из Settings. Администратор может проверить stable или beta-релизы, сверить скачанный бинарный файл с release SHA-256, применить обновление и автоматически откатиться, если новый бинарный файл несколько раз не стартует. Действие записывается в audit и требует повторного ввода пароля.
- Backup и import стали осторожнее. Import ограничен 64 MiB, проверяет SQLite magic, использует staging, read-only integrity checks, schema migrations и rollback к предыдущей базе при ошибке. Локальный незашифрованный экспорт базы стримит подготовленный SQLite-файл вместо буферизации всего backup в памяти.
- Audit и observability встроены в панель. Панель хранит audit events с retention cleanup, предоставляет scoped и paginated audit API, хранит bounded logs, собирает bounded observability buckets и публикует realtime events через защищенный WebSocket path с одноразовыми tokens и Origin checks.
- История IP клиентов по умолчанию хранится как salted hashes. Raw IP display включается отдельно, retention настраивается, а enforce mode отклоняет только новые подключения сверх лимита и не закрывает активные.
- HTTP-серверы панели и подписок используют read, write, header и idle timeouts. TLS использует `MinVersion = 1.2`. Security headers включены, а ответы подписок помечены no-store. Если сохраненный listen IP больше не существует на хосте, fallback остается ограниченным и не расширяет доступ молча.
- Nexus это основной интерфейс панели. Общие страницы переиспользуют существующие компоненты там, где это практично, настройки показывают defaults и help text, release notes обновлений отображаются как безопасный Markdown, а route-based code splitting не добавляет тяжелые views в initial page load.
- Работа над производительностью покрывает ежедневные операции: оптимизированные stats queries и chart downsampling, пакетная запись stats, меньше повторных settings reads в `/api/load`, параллельное чтение независимых данных загрузки, одна сериализация на WebSocket broadcast и разделенные frontend vendor chunks для browser cache.
- Панель, install script и terminal menu включают локализацию на английском, русском и китайском.

</details>

## Поддерживаемые протоколы

<details>
  <summary>Показать поддерживаемые протоколы</summary>

Список следует capability matrix репозитория. Для части протоколов, endpoints и services нужна сборка с соответствующими core tags.

### Входящие протоколы

| Протокол | Примечания |
| :--- | :--- |
| Socks | |
| HTTP | |
| Mixed | Socks + HTTP на одном порту |
| Shadowsocks | AEAD-шифры, методы 2022 |
| VMess | UUID-авторизация, xudp/gRPC/WS |
| VLESS | Reality / TLS-транспорт |
| Trojan | HTTPS-маскировка, fallback |
| Naive | Сетевой стек Chromium |
| Hysteria | QUIC-транспорт |
| Hysteria2 | QUIC-транспорт |
| TUIC | QUIC-транспорт |
| AnyTLS | |
| ShadowTLS | Detour к скрытому протоколу |
| Mieru | Stealth-протокол |
| Sudoku | HTTP-маскировка |
| TrustTunnel | QUIC-туннель |
| SSH | Эмуляция SSH-сервера |
| MTProxy | Telegram-прокси, FakeTLS, отдается как `tg://proxy` |
| Direct | Проброс порта / relay |
| Tun | Виртуальный сетевой интерфейс |
| Redirect | Прозрачный redirect proxy в Linux |
| TProxy | Прозрачный proxy в Linux |
| Bond | Нативная агрегация inbound в core |
| Core failover | Нативный failover inbound в core |

### Исходящие протоколы

| Протокол | Примечания |
| :--- | :--- |
| Direct | Прямой трафик без прокси |
| Block | Тихий сброс трафика |
| Socks | |
| HTTP | |
| Shadowsocks | |
| VMess | |
| VLESS | |
| Trojan | |
| Naive | |
| Tor | SOCKS5 к Tor daemon |
| SSH | |
| ShadowTLS | |
| AnyTLS | |
| Mieru | |
| TrustTunnel | |
| Sudoku | |
| MASQUE | |
| OpenVPN | |
| Hysteria | |
| Hysteria2 | |
| TUIC | |
| Core failover | Нативный dial-time failover в core |

### Исходящие группы

| Тип | Управляется | Примечания |
| :--- | :--- | :--- |
| Selector | Core | Ручное переключение, участника выбирает оператор |
| URLTest | Core | Автовыбор участника с минимальной latency |
| Fallback | Core | Failover во время подключения |
| Failover | Panel | Периодические health checks и all-down policies; влияет на новые подключения |

### Провайдеры

| Тип | Примечания |
| :--- | :--- |
| Inline | Участники задаются вручную в панели |
| Local | Локальный file provider |
| Remote | Удаленный subscription provider |

### Конечные точки

| Тип | Примечания |
| :--- | :--- |
| WireGuard | Нужна core-сборка с WireGuard |
| Tailscale | Нужна core-сборка с Tailscale |
| VPN | WARP/VPN endpoint для клиента и сервера |

### Сервисы ядра

| Сервис | Примечания |
| :--- | :--- |
| resolved | DNS resolver service |
| ssm-api | SSM API service |
| derp | DERP service |
| ccm | Нужна core-сборка с CCM |
| ocm | Нужна core-сборка с OCM |
| oom-killer | Нужна core-сборка с oom-killer |
| profiler | Сервис профилирования для разработки |

</details>

## Поддерживаемые платформы

| Платформа | Архитектура | Статус |
|----------|--------------|---------|
| Linux | amd64, arm64, armv7, armv6, armv5, 386, s390x | Поддерживается |
| Windows | amd64, 386, arm64 | Поддерживается |
| macOS | amd64, arm64 | Экспериментальная поддержка |

## Информация об установке по умолчанию

- Порт панели: 2095
- Путь панели: /app/
- Порт подписки: 2096
- Путь подписки: /sub/
- Изменения лимита подписок на IP (`subRateLimitPerIP`) применяются в течение 1 минуты после сохранения.
- Имя пользователя: admin
- Пароль для свежей установки: при первом запуске генерируется случайная строка из 24 символов, которая выводится в журнал приложения. Найдите `created initial admin user. username=admin password=...` в `journalctl -u s-ui` на Linux или в журнале панели при первом запуске. После входа смените пароль в панели.

## Установка или обновление

Для обычных установок используйте stable. Beta нужна только для проверки изменений до стабильного релиза.

| Канал | Версия | Заметки |
|---|---|---|
| Stable | `v1.0.1` | Рекомендуется для production. Включает hardening-линейку v1.0.1 и исправление настройки Telegram Chat ID. Release notes: [`docs/releases/v1.0.1.md`](docs/releases/v1.0.1.md). |

### Linux/macOS, stable

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.1/install.sh)
```

Эта команда ставит последний stable release. Укажите тег версии явно, если нужна конкретная beta или старая сборка.

### Локальный clone

```sh
git clone --branch v1.0.1 --depth 1 https://github.com/deposist/s-ui-x-extended.git
cd s-ui-x-extended
sudo bash install.sh v1.0.1
```

### Windows

- Stable: скачайте `v1.0.1` на [странице релиза](https://github.com/deposist/s-ui-x-extended/releases/tag/v1.0.1), распакуйте ZIP и запустите `install-windows.bat` от имени администратора.

Существующие установки сохраняют settings, users, inbounds, outbounds, clients, TLS, services и tokens. Миграции базы запускаются автоматически при первом старте. Заметки по обновлению и откату находятся в changelog: [EN](CHANGELOG-EN.md), [RU](CHANGELOG-RU.md), [中文](CHANGELOG-ZH.md).

## Ручная установка

<details>
  <summary>Показать шаги ручной установки</summary>

### Linux/macOS

1. Скачайте последнюю версию S-UI-X Extended для вашей системы и архитектуры из GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. Необязательно: скачайте последнюю версию `s-ui.sh`: [https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.1/s-ui.sh](https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.0.1/s-ui.sh)
3. Необязательно: скопируйте `s-ui.sh` в `/usr/bin/` и выполните `chmod +x /usr/bin/s-ui`.
4. Распакуйте tar.gz-архив S-UI-X Extended в выбранный каталог и перейдите в распакованную папку.
5. Скопируйте файлы `*.service` в `/etc/systemd/system/`, затем выполните `systemctl daemon-reload`.
6. Выполните `systemctl enable s-ui --now`, чтобы включить автозапуск и запустить службу `s-ui`, используемую S-UI-X Extended.
7. Выполните `systemctl enable sing-box --now`, чтобы запустить службу sing-box.

### Windows

1. Скачайте последнюю версию для Windows из GitHub: [https://github.com/deposist/s-ui-x-extended/releases/latest](https://github.com/deposist/s-ui-x-extended/releases/latest)
2. Скачайте подходящий пакет для Windows, например `s-ui-windows-amd64.zip`.
3. Распакуйте ZIP-файл в выбранный каталог.
4. Запустите `install-windows.bat` от имени администратора.
5. Следуйте инструкциям мастера установки.
6. Откройте панель: http://localhost:2095/app

</details>

## Удаление S-UI-X Extended

```sh
sudo -i

systemctl disable s-ui  --now

rm -f /etc/systemd/system/sing-box.service
systemctl daemon-reload

rm -fr /usr/local/s-ui
rm /usr/bin/s-ui
```

## Установка с помощью Docker

<details>
   <summary>Показать подробности</summary>

### Использование

Шаг 1: установите Docker

```shell
curl -fsSL https://get.docker.com | sh
```

Шаг 2: установите S-UI-X Extended

Вариант с Docker Compose:

```shell
services:
  s-ui:
    image: ghcr.io/deposist/s-ui-x-extended
    container_name: s-ui-x-extended
    hostname: "s-ui-x-extended"
    network_mode: host
    volumes:
      - "./db:/app/db"
      - "./cert:/app/cert"
    tty: true
    restart: unless-stopped
    entrypoint: "./entrypoint.sh"
```

`docker compose up -d`

Прямой запуск через Docker:

```shell
mkdir s-ui-x-extended && cd s-ui-x-extended

docker run -itd \
    --network host \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name s-ui-x-extended \
    --restart=unless-stopped \
    ghcr.io/deposist/s-ui-x-extended
```

Самостоятельная сборка образа:

```shell
git clone https://github.com/deposist/s-ui-x-extended
docker build -t s-ui-x-extended .
```

</details>

## Ручной запуск для разработки и участия в проекте

<details>
   <summary>Показать подробности</summary>

### Сборка и запуск полного проекта

```shell
./runSUI.sh
```

### Клонирование репозитория

```shell
git clone https://github.com/deposist/s-ui-x-extended
```

### Фронтенд

Код фронтенда находится в каталоге [frontend](frontend).

### Бэкенд

Перед сборкой бэкенда нужно хотя бы один раз собрать фронтенд.

Сборка бэкенда:

```shell
rm -fr web/html/*
cp -R frontend/dist/ web/html/
go build -o sui main.go
```

Запуск бэкенда из корня репозитория:

```shell
./sui
```

</details>

## Языки

- Английский
- Персидский
- Вьетнамский
- Упрощенный китайский
- Традиционный китайский
- Русский

## Переменные окружения

<details>
  <summary>Показать подробности</summary>

### Использование

| Переменная | Тип | Значение по умолчанию |
| -------------- | :--------------------------------------------: | :------------ |
| SUI_LOG_LEVEL | `"debug"` \| `"info"` \| `"warn"` \| `"error"` | `"info"` |
| SUI_DEBUG | `boolean` | `false` |
| SUI_BIN_FOLDER | `string` | `"bin"` |
| SUI_DB_FOLDER | `string` | `"db"` |
| SINGBOX_API | `string` | - |
| SUI_TRUSTED_PROXIES | список CIDR/IP через запятую | нет, XFF игнорируется |
| SUI_ALLOW_PRIVATE_SUB_URLS | `boolean` | `false` |
| SUI_SECRETBOX_KEY | `string` | нет, fallback на `settings.secret` |

Для systemd-установок через `install.sh` S-UI-X Extended один раз генерирует стабильный `SUI_SECRETBOX_KEY` в `/etc/s-ui/secretbox.env`, один раз показывает сгенерированное значение и подключает файл через systemd drop-in. Держите этот файл в секрете и сохраняйте тот же ключ при обновлениях и восстановлении.

</details>

## SSL-сертификаты

<details>
  <summary>Показать подробности</summary>

### Certbot

```bash
snap install core; snap refresh core
snap install --classic certbot
ln -s /snap/bin/certbot /usr/bin/certbot

certbot certonly --standalone --register-unsafely-without-email --non-interactive --agree-tos -d <ваш домен>
```

</details>

#### Благодарности

- Автор оригинальной панели: [alireza0/s-ui](https://github.com/alireza0/s-ui)
- Extended-ядро: [shtorm-7/sing-box-extended](https://github.com/shtorm-7/sing-box-extended)

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date&theme=dark" />
  <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
  <img alt="График истории звезд" src="https://api.star-history.com/chart?repos=deposist/s-ui-x-extended&type=date" />
</picture>
