# Release Notes: v1.5.7-hotfix1

Release date: 2026-06-11

Hotfix for v1.5.7. It fixes a "record not found" error shown after deleting a
client whose database row was already gone, and adds a `SUI_COOKIE_KEY`
generator to the `s-ui` menu and the installer. There are no breaking,
manual-migration, or configuration changes.

## What changed

### Fixes

- **"failed - save: record not found" after deleting a client.** Deleting a
  client whose database row was already gone - a stale list row, a concurrent
  delete from another session or tab, or a resubmitted request - failed with
  "record not found", even though the intended end state (client absent)
  already held. Client deletion (single and bulk) is now idempotent: deleting
  an absent client is a no-op success, and a bulk delete skips already-gone
  ids while deleting the rest. Real database errors still fail the request.

### Management script and installer

- **New `s-ui` menu item 23: generate the session cookie key
  (`SUI_COOKIE_KEY`).** The key is stored in `/etc/s-ui/secretbox.env` (the
  same root-only file the secretbox key already uses) and loaded through the
  existing systemd drop-in. If a key already exists, the menu shows it masked
  and offers a **rotation with rollover**: the new key signs new session
  cookies while the previous keys (up to two) stay accepted, so rotating the
  key does not sign anybody out.
- **The installer now generates `SUI_COOKIE_KEY` automatically - only when it
  is absent.** An existing key is never touched and the installer never
  prompts, so non-interactive installs and updates keep working. Key rotation
  is an explicit operator action via the menu item. One consequence on the
  first upgrade that introduces the key: sessions signed with the previous
  built-in fallback key are signed out once - log in again afterwards.

## Verification

- New regression tests: deleting the same client twice succeeds; a bulk delete
  containing a missing id succeeds and removes the existing clients.
- The delete paths of inbounds, outbounds, endpoints, services, and TLS were
  verified to already be idempotent (they never produced this error) and are
  now locked in with regression tests of their own.
- Shell scripts pass `bash -n`; the env-file read/write/rotation logic and the
  installer's generate-only-when-absent behavior were exercised with
  functional tests (coexistence with `SUI_SECRETBOX_KEY`, single key line,
  rollover order, list cap, silent no-op on re-run, 32-byte key material).
- Full backend gates pass: `go test ./service`, `go vet`, `staticcheck`,
  `golangci-lint`, `go build`.

Upgrading from v1.5.7 is a drop-in binary replacement.

---

# Примечания к релизу: v1.5.7-hotfix1

Дата релиза: 2026-06-11

Хотфикс к v1.5.7. Исправляет ошибку «record not found», появлявшуюся после
удаления клиента, строка которого уже отсутствовала в базе, и добавляет
генератор `SUI_COOKIE_KEY` в меню `s-ui` и установщик. Ломающих изменений,
ручных миграций и изменений конфигурации нет.

## Что изменилось

### Исправления

- **«failed - save: record not found» после удаления клиента.** Удаление
  клиента, строки которого уже не было в базе - устаревшая строка списка,
  параллельное удаление из другой сессии или вкладки, повторно отправленный
  запрос - завершалось ошибкой «record not found», хотя желаемое конечное
  состояние (клиента нет) уже было достигнуто. Удаление клиентов (одиночное и
  массовое) теперь идемпотентно: удаление отсутствующего клиента - успешный
  no-op, а массовое удаление пропускает уже отсутствующие id и удаляет
  остальные. Настоящие ошибки базы данных по-прежнему завершают запрос ошибкой.

### Скрипт управления и установщик

- **Новый пункт 23 меню `s-ui`: генерация ключа сессионных cookie
  (`SUI_COOKIE_KEY`).** Ключ хранится в `/etc/s-ui/secretbox.env` (тот же
  root-only файл, что уже используется для secretbox-ключа) и подключается
  через существующий systemd drop-in. Если ключ уже существует, меню
  показывает его маскированно и предлагает **ротацию с плавным переходом**:
  новый ключ подписывает новые сессионные cookie, а прежние ключи (до двух)
  продолжают приниматься - ротация никого не разлогинивает.
- **Установщик теперь генерирует `SUI_COOKIE_KEY` автоматически - только при
  его отсутствии.** Существующий ключ никогда не перезаписывается, и
  установщик не задает вопросов - неинтерактивные установки и обновления
  продолжают работать. Ротация ключа - явное действие оператора через пункт
  меню. Следствие первого обновления, добавляющего ключ: сессии, подписанные
  прежним встроенным fallback-ключом, будут разлогинены один раз - войдите
  заново.

## Проверка

- Новые регрессионные тесты: повторное удаление того же клиента проходит
  успешно; массовое удаление с отсутствующим id проходит и удаляет существующих
  клиентов.
- Пути удаления Inbounds, Outbounds, endpoint-ов, сервисов и TLS проверены:
  они уже были идемпотентными (этой ошибки не давали) и теперь закреплены
  собственными регрессионными тестами.
- Shell-скрипты проходят `bash -n`; логика чтения/записи env-файла и ротации,
  а также поведение установщика «генерировать только при отсутствии» проверены
  функциональными тестами (сосуществование с `SUI_SECRETBOX_KEY`, единственная
  строка ключа, порядок rollover, ограничение списка, тихий no-op при повторном
  запуске, 32 байта ключевого материала).
- Полные backend-проверки пройдены: `go test ./service`, `go vet`,
  `staticcheck`, `golangci-lint`, `go build`.

Обновление с v1.5.7 - простая замена бинаря.
