# S-UI-X-Extended v1.0.1-beta5

Security hardening release for admin sessions, API tokens, and public subscription URLs.

## Security fixes

- API token delete and enable/disable actions now require token ownership. An admin can no longer change another admin's token by posting its numeric id.
- New installs now require subscription secrets by default. Legacy name-based subscription lookup still works only when `subSecretRequired=false` is set explicitly.
- Admins imported with `force_password_reset=true` now get a restricted session after entering the old password. Until they change their password, only CSRF, password change, and logout endpoints are available.
- API tokens owned by an admin who must reset a password are rejected until the password is changed.

## UI

- The login page now handles forced password reset. It asks for a new username and password after the old password is verified, then opens the panel after the change succeeds.

## Compatibility notes

- Existing installs keep their current `subSecretRequired` setting. The safer default applies to fresh installs.
- No manual database migration is required.

## Verification

- `go test -count=1 ./api ./service ./sub ./database` passed.
- `go test -p 1 -count=1 ./...` passed.
- `go vet ./...` passed.
- `cd frontend && npm run lint` passed.
- `cd frontend && npm run build` passed.
- `cd frontend && npm test -- --run` passed.

---

# S-UI-X-Extended v1.0.1-beta5

Релиз с усилением безопасности админ-сессий, API-токенов и публичных subscription URL.

## Исправления безопасности

- Удаление и включение/выключение API-токена теперь требуют владения токеном. Админ больше не может изменить токен другого админа, передав его numeric id.
- Новые установки теперь требуют subscription secret по умолчанию. Legacy lookup по имени клиента работает только если явно задано `subSecretRequired=false`.
- Админы, импортированные с `force_password_reset=true`, после ввода старого пароля получают ограниченную сессию. До смены пароля доступны только CSRF, смена пароля и logout.
- API-токены админа с обязательной сменой пароля отклоняются до смены пароля.

## UI

- Страница логина теперь поддерживает forced password reset. После проверки старого пароля она просит новый username и пароль, затем открывает панель после успешной смены.

## Совместимость

- Существующие установки сохраняют текущее значение `subSecretRequired`. Более безопасный дефолт применяется к новым установкам.
- Ручная миграция базы не нужна.

## Проверка

- `go test -count=1 ./api ./service ./sub ./database` прошел.
- `go test -p 1 -count=1 ./...` прошел.
- `go vet ./...` прошел.
- `cd frontend && npm run lint` прошел.
- `cd frontend && npm run build` прошел.
- `cd frontend && npm test -- --run` прошел.
