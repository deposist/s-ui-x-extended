# S-UI-X-Extended v1.0.1

Stable v1.0.1 promotes the v1.0.1 beta line and adds one Telegram setup fix on top. No manual database or configuration migration is required.

## Telegram setup

Telegram settings now have a Detect Chat ID action. Enter a bot token, send `/start` or any message to the bot, then let the panel fill the Chat ID from Telegram updates.

Detection works with a newly typed token or with the encrypted token already saved in the panel. Saved bot tokens stay encrypted at rest and are not returned to the browser.

The Telegram Test button now saves changed Telegram settings before sending the test message. This avoids the common `missing_chat` result after entering or detecting Chat ID but not pressing Save.

## Included from the v1.0.1 beta line

- API token delete and enable/disable actions require token ownership.
- Fresh installs default `subSecretRequired` to `true`; existing installs keep their saved value.
- Admins with `force_password_reset=true` receive a restricted session and must change the password before using the panel.
- API tokens owned by admins who must reset a password are rejected until the password changes.
- The login page supports the forced password reset flow.
- WARP and WireGuard endpoint output strips unsupported legacy `peers[].reserved` fields before config generation.
- Legacy inbound sniff settings migrate to scoped route rules for sing-box 1.11+ compatibility.
- Frontend guidance, Dashboard timezone selection, protocol-aware inbound recommendations, and extended documentation from the beta line are included.

## Upgrade notes

Upgrade normally. Existing Telegram, subscription, admin, and protocol settings are kept. If Detect Chat ID returns `no_updates`, open the bot in Telegram, send `/start`, and try again. For groups, add the bot to the group and send a message there first.

---

# S-UI-X-Extended v1.0.1

Стабильная v1.0.1 повышает beta-линейку v1.0.1 до stable и добавляет исправление настройки Telegram. Ручная миграция базы данных или конфигурации не требуется.

## Настройка Telegram

В настройках Telegram появилась кнопка «Определить Chat ID». Введите токен бота, отправьте боту `/start` или любое сообщение, затем панель заполнит Chat ID из Telegram updates.

Определение работает с только что введённым токеном и с уже сохранённым encrypted token. Сохранённые bot tokens остаются зашифрованными at rest и не возвращаются в браузер.

Кнопка Telegram Test теперь сохраняет изменённые Telegram-настройки перед отправкой тестового сообщения. Это убирает частый `missing_chat` после ввода или определения Chat ID без нажатия Save.

## Что вошло из beta-линейки v1.0.1

- Удаление и включение/выключение API-токена требуют владения токеном.
- Новые установки используют `subSecretRequired=true` по умолчанию; существующие установки сохраняют своё значение.
- Админы с `force_password_reset=true` получают ограниченную сессию и должны сменить пароль перед работой с панелью.
- API-токены таких админов отклоняются до смены пароля.
- Страница логина поддерживает forced password reset flow.
- WARP и WireGuard endpoint output удаляет неподдерживаемые legacy `peers[].reserved` fields перед генерацией config.
- Legacy inbound sniff settings мигрируют в scoped route rules для совместимости с sing-box 1.11+.
- В stable входят frontend guidance, выбор timezone на Dashboard, protocol-aware inbound recommendations и расширенная документация из beta-линейки.

## Обновление

Обновляйтесь обычным способом. Существующие настройки Telegram, подписок, админов и протоколов сохраняются. Если «Определить Chat ID» возвращает `no_updates`, откройте бота в Telegram, отправьте `/start` и попробуйте снова. Для группы сначала добавьте бота в группу и отправьте сообщение там.
