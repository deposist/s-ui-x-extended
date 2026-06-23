# Release Notes: v1.5.6

Release date: 2026-06-04

First stable release of the 1.5.6 line. It consolidates the 1.5.6-beta1..beta9
series. Its main item is the **3x-ui (x-ui) → s-ui-x migration**, and it adds the
import-correctness fixes below. No schema migrations; existing data is preserved.

## What changed

- **3x-ui / x-ui migration.** Import a 3x-ui or x-ui SQLite
  database into s-ui-x: inbounds and clients (with generated subscription links),
  proxy / WARP / system outbounds, routing rules and matchers, the DNS block, and
  inline TLS certificates are converted to their sing-box equivalents and merged
  into the live config. `geosite`/`geoip` matches become remote rule sets
  (sing-box 1.12 removed the inline fields); re-import and scheduled sync are
  idempotent and never clobber operator-edited entries.

- **Import-correctness fixes (new in this stable release):**
  - An Xray **`blackhole`** outbound now migrates to a sing-box `reject` rule
    action instead of a dangling `outbound: "block"` reference. sing-box 1.11+ has
    no `block` outbound, so the old reference made the imported config fail at
    route time with *"outbound not found: block"* - the whole config would not
    apply. (Supersedes the `blackhole`→`block` mapping shipped in beta7/beta8.)
  - The migration **preserves a DNS-only source**: a config whose only migratable
    content was DNS (no routing rules, no proxy Outbounds, no endpoints) was
    silently skipped and its DNS dropped - it now imports.
  - A built-in **`direct`** outbound is ensured whenever migrated routing routes
    to it (a rule, or a remote rule-set download detour); the check consults the
    database, so the default `direct` outbound is no longer re-reported as a
    skipped duplicate.
  - A user proxy legitimately tagged `block`/`blocked`/`dns` keeps routing to
    itself instead of being turned into a reject / hijack-dns action.

- **Panel recovery & certificates (terminal menu).** A new *Clear panel domain
  and address* item (and `s-ui setting -clearDomain`) restores access when a wrong
  domain or an unbindable listen address locks you out. *Get SSL* can re-issue a
  certificate acme.sh already holds (`--issue --force` + `--installcert`) instead
  of dead-ending.

See the `1.5.6-beta1`..`1.5.6-beta9` entries in `CHANGELOG-EN.md` for the full
per-step history.

## Upgrade

No manual migration; existing data is preserved. To migrate from a 3x-ui / x-ui
panel use *Migrate from 3x-ui* (Backup & Restore) or the import CLI - a
pre-import backup is written automatically.

---

# Примечания к релизу: v1.5.6

Дата релиза: 2026-06-04

Первый стабильный релиз линейки 1.5.6. Он объединяет серию 1.5.6-beta1..beta9 -
главная тема которой **миграция 3x-ui (x-ui) → s-ui-x** - и добавляет
исправления корректности импорта ниже. Без миграций схемы; существующие данные
сохраняются.

## Что изменилось

- **Миграция 3x-ui / x-ui (главное в 1.5.6).** Импорт базы 3x-ui или x-ui (SQLite)
  в s-ui-x: Inbounds и клиенты (со сгенерированными ссылками подписки),
  proxy- / WARP- / системные Outbounds, правила маршрутизации и матчеры, блок
  DNS и встроенные TLS-сертификаты конвертируются в эквиваленты sing-box и
  сливаются в живой конфиг. Совпадения `geosite`/`geoip` становятся удалёнными
  rule-set'ами (sing-box 1.12 убрал встроенные поля); повторный импорт и плановая
  синхронизация идемпотентны и не затирают отредактированные оператором записи.

- **Исправления корректности импорта (новое в этом стабильном релизе):**
  - Xray-outbound **`blackhole`** теперь мигрирует в действие правила `reject`, а
    не в висячую ссылку `outbound: "block"`. В sing-box 1.11+ нет outbound
    `block`, поэтому прежняя ссылка ломала импортированный конфиг во время
    маршрутизации с ошибкой *«outbound not found: block»* - конфиг не применялся
    целиком. (Заменяет отображение `blackhole`→`block` из beta7/beta8.)
  - Миграция **сохраняет источник только с DNS**: конфиг, у которого мигрируемым
    был только DNS (без правил маршрутизации, proxy-Outbounds и endpoint'ов),
    раньше тихо пропускался, а его DNS терялся - теперь импортируется.
  - Встроенный outbound **`direct`** гарантируется всякий раз, когда мигрированная
    маршрутизация ссылается на него (правило или download-detour rule-set'а);
    проверка обращается к базе, поэтому стандартный `direct` больше не отмечается
    как пропущенный дубликат.
  - Пользовательский прокси, легитимно названный `block`/`blocked`/`dns`,
    продолжает маршрутизироваться сам в себя, а не превращается в действие
    reject / hijack-dns.

- **Восстановление панели и сертификаты (терминальное меню).** Новый пункт
  *Очистить домен и адрес панели* (и `s-ui setting -clearDomain`) возвращает
  доступ, когда неверный домен или неназначаемый адрес прослушивания заблокировали
  вход. *Получить SSL* умеет перевыпустить сертификат, уже имеющийся в acme.sh
  (`--issue --force` + `--installcert`), а не упираться в тупик.

Полную пошаговую историю см. в записях `1.5.6-beta1`..`1.5.6-beta9` в
`CHANGELOG-RU.md`.

## Обновление

Ручная миграция не нужна, данные сохраняются. Для переезда с панели 3x-ui / x-ui
используйте *Migrate from 3x-ui* (Backup & Restore) или CLI импорта - резервная
копия перед импортом создаётся автоматически.
