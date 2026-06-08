# Security notes — s-ui-x (extended)

Этот документ фиксирует осознанные решения по безопасности и цепочке поставок
в расширенной сборке. Он дополняет [EXTENDED-PROGRESS.md](EXTENDED-PROGRESS.md).

## Форки зависимостей (`replace` в go.mod)

Ядро и ряд сетевых/крипто-библиотек заменены на форки `shtorm-7` через `replace`.
Все версии **закреплены** и проверяются через `go.sum` (минимальный порог
целостности Go), но это **неаудированные сторонние форки** библиотек, отвечающих
за криптографию и маршрутизацию трафика. Это осознанное доверительное решение;
при обновлении сверяйте diff с апстримом на предмет security-релевантных изменений.

| Оригинал | Форк |
|---|---|
| `github.com/sagernet/sing-box` | `shtorm-7/sing-box-extended` (pinned) |
| `github.com/sagernet/sing` | `shtorm-7/sing` |
| `github.com/sagernet/wireguard-go` | `shtorm-7/wireguard-go` |
| `github.com/sagernet/tailscale` | `shtorm-7/tailscale` |
| `github.com/sagernet/sing-mux` | `shtorm-7/sing-mux` |
| `github.com/sagernet/sing-vmess` | `shtorm-7/sing-vmess` |
| `github.com/ameshkov/dnscrypt/v2` | `shtorm-7/dnscrypt/v2` |
| `github.com/dolonet/mtg-multi` | `shtorm-7/mtg-multi` |
| `github.com/Diniboy1123/connect-ip-go` | `shtorm-7/connect-ip-go` |
| `github.com/shtorm-7/go-cache/v2` | `shtorm-7/go-cache/v2` |

(`quic-go` также имеет `replace` на конкретную версию — не форк.)

## Транзитивные зависимости

- `github.com/redis/go-redis/v9` тянется **транзитивно** (core → sing-box
  `protocol/limiter/rate` → `AliRizaAynaci/gorl/v2` → redis storage) и **напрямую
  в коде проекта не используется**. Redis-бэкенд лимитера задействуется только
  если он явно сконфигурирован.

## Build-теги и профайлер

- **`with_profiler` убран из сборок по умолчанию** (`build.sh`, `Dockerfile`).
  Сервис profiler регистрирует `/debug/pprof/*` (включая `cmdline`) **без
  аутентификации**. Это тег **только для разработки**: при сборке с
  `-tags with_profiler` НИКОГДА не привязывайте `listen` к не-loopback адресу.
  Без тега попытка сконфигурировать profiler-сервис вернёт явную ошибку (stub).
- **`with_oomkiller`** теперь явный тег (как остальные чувствительные сервисы):
  без него регистрируется stub с понятной ошибкой; сборки по умолчанию его
  включают, сохраняя текущее поведение.

## cronet (нативная библиотека)

`Dockerfile` скачивает `libcronet-linux-<arch>.so` из **закреплённого релиза**
(`CRONET_GO_RELEASE`, по умолчанию `v148.0.7778.96-1`) и **проверяет sha256**
(дайджесты из GitHub release API), вместо плавающего `releases/latest`. При
обновлении cronet поменяйте `CRONET_GO_RELEASE`/`CRONET_GO_VERSION` и обновите
суммы в `Dockerfile`.

## Запуск от root

Контейнер работает от root намеренно — сетевые операции (`nftables`, TUN/WireGuard)
требуют повышенных привилегий. Если потребуется non-root, понадобится выдать
`CAP_NET_ADMIN`/`CAP_NET_RAW`, `chown` рабочих директорий и проверить TUN.

## Отложенные улучшения харденинга (документированы, вне текущего объёма)

- **Digest-пиннинг базовых образов** (`node:alpine`, `golang:*-alpine`, `alpine`):
  закрепить по `@sha256:`. Отложено — в окружении сборки нет docker/skopeo для
  получения и проверки дайджестов.
- **HEALTHCHECK** в финальном образе: порт панели задаётся динамически
  (в настройках/БД), поэтому корректный healthcheck требует знания порта —
  отложено во избежание ложных «unhealthy».
- **Полная типизация Vue-форм через `PropType<Interface>`**: сейчас пропсы
  приведены к объектной форме (`{ type: Object, required: true }`); строгая
  типизация `data` по интерфейсам из `types/*.ts` — отдельный проход
  (взаимодействует с дефолтной инициализацией полей).
- **Стабильные ключи `v-for`** в списках с удалением (Bond/Failover/OpenVPN/
  VpnServer): «правильный» ключ требует id, который попал бы в сериализуемый
  конфиг sing-box, поэтому индекс-ключ оставлен намеренно.
