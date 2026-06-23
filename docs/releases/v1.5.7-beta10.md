# Release Notes: v1.5.7-beta10

Release date: 2026-06-10

This release applies the remediation from a full code-quality, optimization,
security, and supply-chain review. It fixes financial-correctness issues in the
Paid Subscriptions module, hardens the external attack surface, adds new
abuse-detection audit signals, fixes frontend correctness and safety issues, and
hardens the CI supply chain. There is no manual migration (one additive database
column is created automatically). Two changes alter behavior - see **Behavior
changes** below. This is a beta - test first.

## What changed

### Paid Subscriptions (payments) - financial correctness

- **A CryptoBot payment confirmed after the local order timeout is no longer
  lost.** The poll now confirms out-of-band payments before any expiry pass, and
  CryptoBot orders are reaped only after a long grace window, so a late but valid
  payment is still applied (money taken is no longer left without a grant).
- **Refunds restore usage counters symmetrically.** A renewal that refills
  traffic records a snapshot of the prior usage counters; a refund now restores
  them, instead of rolling back only the volume.
- **Stars refunds return the money first, then finalize.** A transient Telegram
  failure now leaves the order paid and retryable rather than revoked-with-money-
  not-returned, mirroring the admin refund path.
- **CryptoBot polling re-validates the paid amount and currency** against the
  server-side order snapshot before granting (mirrors the Telegram-native path).
- **Tariffs reject negative values** (price, stars, days, traffic, sort) on both
  the server and client side.

### Security

- **Changing the admin password now rotates the session generation.** All other
  web sessions and every WebSocket token are invalidated; only the session that
  performed the change stays signed in. See **Behavior changes**.
- **The per-username login throttle is now a tarpit, not a hard block.** After
  repeated failures a username incurs an escalating, capped delay instead of a
  total lockout, so a distributed attempt cannot lock a known admin out of their
  own panel. The per-IP hard block is unchanged and remains the primary
  brute-force defense.
- **x-ui import validates geoip/geosite codes** from the imported database before
  using them in a remote rule-set URL (charset allowlist).
- **The subscription link service no longer panics** on a malformed vmess `ps`
  field.
- **A database-import race** on the shared connection handle is fixed (the swap
  now takes the same lock as open).

### Detection and audit

- **New audit signals:** `/sub` enumeration (repeated invalid subscription IDs
  from one source), login from a new source IP, cross-user order access on the
  client bot, IP-limit enforcement, and an audit-pipeline drop marker.
- **Real-time alerts** are now sent on account lockout and on full database
  export - two of the highest-signal admin-compromise events.

### Interface (Nexus)

- **Unsaved-changes confirmation now covers every entity form** - all drawers and
  the classic dialogs (clients, DNS, endpoints, rules, rule sets, bulk) - in both
  interface modes, so closing a form with edits asks before discarding.
- **Assorted correctness and safety fixes:** the client list search is now
  case-insensitive (consistent with the other lists); TLS option defaults no
  longer leak between forms; the inbound form's Save is disabled until the config
  is valid; the per-row selection checkbox has a correct screen-reader label; and
  unused UI code was removed.
- **The dashboard status poll pauses while the browser tab is hidden** and
  refreshes immediately when it returns.

### Supply chain (CI)

- **Third-party and Docker GitHub Actions are pinned to commit SHAs**, and a new
  Dependabot configuration keeps them - and the Go, npm, and Docker dependencies -
  current.
- **The Docker frontend build uses `npm ci`** so the image is built from the
  exact, audited lockfile.

### Behavior changes

- **Changing the admin password signs out all other sessions** (web and
  WebSocket). This is intentional hardening; re-log in elsewhere after a change.
- **The per-username login lockout is now a delay (tarpit), not a block.** Legit
  admins are never locked out by failures from other sources; attempts are
  slowed.
- One additive database column (`payment_orders.granted_up` / `granted_down`) is
  created automatically at startup. No manual migration; existing databases are
  upgraded in place.

## Verification

- Go: `go build`, `go vet`, `staticcheck`, `golangci-lint` clean; `gosec` 0
  issues; `govulncheck` reports no vulnerabilities.
- Go tests: every package passes `go test` standalone (api, service, paidsub,
  sub, ipmonitor, database, importxui, and the rest), with new tests for the
  CryptoBot timeout, refund counters, tariff validation, and the login tarpit.
- Frontend: `vue-tsc --noEmit`, `vite build`, and `eslint` clean; `vitest` passes
  (123 tests).
- CI: every changed workflow and the Dependabot config parse as valid YAML; each
  pinned action SHA was verified against the GitHub API.
- The changes were validated by the deterministic toolchain above plus a
  multi-agent code review of the diff with adversarial verification.

---

# Примечания к релизу: v1.5.7-beta10

Дата релиза: 2026-06-10

Этот релиз применяет исправления по итогам полного ревью качества кода,
оптимизации, безопасности и цепочки поставок. Он чинит проблемы финансовой
корректности в модуле «Платные подписки», усиливает внешнюю поверхность атаки,
добавляет новые сигналы аудита для детектирования злоупотреблений, исправляет
корректность и безопасность фронтенда и усиливает цепочку поставок CI. Ручная
миграция не нужна (одна добавочная колонка БД создаётся автоматически). Два
изменения меняют поведение - см. **Изменения поведения** ниже. Это бета -
сначала протестируйте.

## Что изменилось

### Платные подписки (платежи) - финансовая корректность

- **Платёж CryptoBot, подтверждённый после локального таймаута заказа, больше не
  теряется.** Опрос теперь подтверждает out-of-band платежи до прохода истечения,
  а заказы CryptoBot истекают только после длинного грейс-окна, поэтому поздний,
  но валидный платёж всё равно применяется (списанные деньги больше не остаются
  без выдачи услуги).
- **Возвраты симметрично восстанавливают счётчики использования.** Продление с
  пополнением трафика сохраняет снимок прежних счётчиков; возврат теперь их
  восстанавливает, а не откатывает только объём.
- **Возврат Stars сначала возвращает деньги, затем финализирует.** Транзиентный
  сбой Telegram теперь оставляет заказ оплаченным и доступным для повтора, а не
  отзывает грант с невозвращёнными деньгами - как в админском пути возврата.
- **Опрос CryptoBot пере-проверяет сумму и валюту платежа** против серверного
  снимка заказа перед выдачей (как в Telegram-native пути).
- **Тарифы отвергают отрицательные значения** (цена, stars, дни, трафик, сортировка)
  на сервере, а не только на клиенте.

### Безопасность

- **Смена пароля администратора теперь ротирует поколение сессии.** Все прочие
  web-сессии и каждый WebSocket-токен инвалидируются; в системе остаётся только та
  сессия, что выполнила смену. См. **Изменения поведения**.
- **Per-username троттлинг логина теперь tarpit, а не жёсткая блокировка.** После
  повторных неудач для логина применяется нарастающая ограниченная задержка вместо
  полной блокировки, поэтому распределённая попытка не может заблокировать
  известный admin-логин в его же панели. Per-IP блокировка не изменилась и
  остаётся основной защитой от brute-force.
- **Импорт x-ui валидирует geoip/geosite-коды** из импортируемой БД перед
  использованием в URL удалённого rule-set (allowlist символов).
- **Сервис ссылок подписки больше не паникует** на некорректном поле `ps` в vmess.
- **Гонка при импорте БД** на общем хэндле соединения исправлена (смена указателя
  теперь под той же блокировкой, что и открытие).

### Детектирование и аудит

- **Новые сигналы аудита:** перебор `/sub` (повторные невалидные ID подписки с
  одного источника), вход с нового IP, доступ к чужому заказу в клиент-боте,
  срабатывание IP-лимита и маркер сброса событий в пайплайне аудита.
- **Оповещения в реальном времени** теперь отправляются при блокировке аккаунта и
  при полном экспорте БД - двух самых сигнальных событиях компрометации админа.

### Интерфейс (Nexus)

- **Подтверждение несохранённых изменений теперь во всех формах сущностей** - все
  дроверы и классические диалоги (клиенты, DNS, endpoint'ы, правила, rule-set'ы,
  массовые) - в обоих режимах интерфейса: закрытие формы с правками спрашивает
  перед сбросом.
- **Набор исправлений корректности и безопасности:** поиск в списке клиентов
  теперь регистронезависимый (как в остальных списках); дефолты TLS-опций больше
  не «протекают» между формами; кнопка «Сохранить» в Inbounds недоступна, пока
  конфиг невалиден; чекбокс выбора строки имеет корректную метку для скринридеров;
  удалён неиспользуемый UI-код.
- **Опрос статуса на дашборде приостанавливается, пока вкладка скрыта**, и
  обновляется сразу при возврате.

### Цепочка поставок (CI)

- **Сторонние и Docker GitHub Actions запинены на commit-SHA**, а новая
  конфигурация Dependabot держит их - и зависимости Go, npm, Docker - в актуальном
  состоянии.
- **Docker-сборка фронтенда использует `npm ci`**, чтобы образ собирался из
  точного проверенного lockfile.

### Изменения поведения

- **Смена пароля администратора разлогинивает все прочие сессии** (web и
  WebSocket). Это намеренное усиление; войдите заново в других местах после смены.
- **Per-username блокировка логина теперь задержка (tarpit), а не блок.** Легитимные
  админы никогда не блокируются из-за неудач из других источников; попытки
  замедляются.
- Одна добавочная колонка БД (`payment_orders.granted_up` / `granted_down`)
  создаётся автоматически при старте. Ручная миграция не нужна; существующие базы
  обновляются на месте.

## Проверка

- Go: `go build`, `go vet`, `staticcheck`, `golangci-lint` - чисто; `gosec` 0
  замечаний; `govulncheck` - уязвимостей нет.
- Go-тесты: каждый пакет проходит `go test` по отдельности (api, service, paidsub,
  sub, ipmonitor, database, importxui и остальные), с новыми тестами для таймаута
  CryptoBot, счётчиков возврата, валидации тарифа и tarpit-логина.
- Фронтенд: `vue-tsc --noEmit`, `vite build` и `eslint` - чисто; `vitest` проходит
  (123 теста).
- CI: каждый изменённый workflow и конфиг Dependabot парсятся как валидный YAML;
  каждый запиненный SHA экшена проверен через GitHub API.
- Изменения проверены детерминированным тулчейном выше плюс многоагентным ревью
  диффа с адверсариальной верификацией.
