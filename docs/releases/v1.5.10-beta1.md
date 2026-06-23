# Release Notes: v1.5.10-beta1
Release date: 2026-06-23

This is the first beta in the 1.5.10 line. It contains the performance fixes from the 2026-06-23 audit: faster stats chart preparation, fewer database reads during a full panel reload, streamed local DB export, safer stats inserts, a short cache for subscription output, and smaller entry chunks in the frontend build.

You do not need to run a database migration or change configuration. The public API and UI behavior stay the same.

## Faster dashboard stats

Stats chart downsampling now sorts the rows once and assigns them to buckets in a single pass. Before this release, the code scanned the full result set for every bucket and traffic direction. Large stats windows should take less CPU when the dashboard or `/api/stats` requests chart data.

Stats writes now use explicit SQLite-safe batches. This avoids oversized insert statements on installations with many clients, inbounds, outbounds, or endpoints.

## Lighter `/api/load` full reloads

The full reload path now reads the settings needed for load data through one request-local snapshot. It no longer repeats several settings queries for the same response.

The backend also reads clients, TLS records, inbounds, outbounds, endpoints, services, and load settings in parallel. These reads are independent, so the response no longer waits for them one by one.

## Streamed unencrypted DB export

Unencrypted local database export now streams the prepared SQLite backup file to the HTTP response. It no longer reads the whole backup into memory before sending it to the browser.

Encrypted backup downloads still use the existing whole-payload envelope. That path still buffers plaintext because the current encrypted envelope seals the complete payload at once.

## Subscription output cache

Base, JSON, and Clash subscription outputs now use a short TTL cache. The cache is cleared after a successful config save. This reduces repeated link generation when many clients poll subscriptions at the same time, for example after a restart.

Only successful responses are cached. Missing clients and errors are not cached.

Subscription headers are cached with the response for up to 45 seconds. During that window, traffic and userinfo headers can lag behind the latest counters on repeated requests.

## Frontend chunks

The frontend build now separates stable vendor chunks for Vue, Vuetify, and HTTP dependencies. The main app entry is smaller, and browsers can reuse vendor chunks more often between releases.

The DateTime picker is lazy-loaded from client modals. Extra Moment locale imports were removed. Moment itself is still present because the current Persian datetime picker depends on `moment-jalaali`; removing it safely means replacing that picker. Chart.js and YAML were reviewed but left in place because replacing them needs chart and Clash config compatibility checks.

## Upgrade

Upgrade normally. No manual database migration or configuration change is needed.

This is a beta release. Publish it as a GitHub pre-release and keep it out of the Latest stable slot.

---

# Примечания к релизу: v1.5.10-beta1
Дата релиза: 2026-06-23

Это первая бета линейки 1.5.10. В ней собраны исправления производительности по аудиту от 2026-06-23: быстрее подготовка графиков статистики, меньше чтений из базы при полном обновлении панели, streaming для локального экспорта БД, безопасные batch-вставки статистики, короткий кеш output подписок и меньший entry chunk во frontend build.

Ручная миграция базы и изменение конфигурации не требуются. Публичное поведение API и UI не менялось.

## Быстрее статистика на dashboard

Downsampling для графиков статистики теперь один раз сортирует строки и за один проход раскладывает их по bucket'ам. Раньше код заново сканировал весь результат для каждого bucket и направления трафика. На больших окнах статистики запросы dashboard или `/api/stats` должны потреблять меньше CPU.

Запись статистики теперь использует явные SQLite-safe batch-вставки. Это защищает крупные установки от слишком больших insert statements при большом числе клиентов, inbounds, outbounds или endpoints.

## Легче полный `/api/load`

Полный reload теперь читает settings, нужные для load data, через один request-local snapshot. Несколько повторных settings-запросов для одного ответа больше не выполняются.

Backend также читает clients, TLS records, inbounds, outbounds, endpoints, services и load settings параллельно. Эти чтения независимы, поэтому ответ больше не ждёт их строго по очереди.

## Streaming для незашифрованного экспорта БД

Незашифрованный локальный экспорт базы теперь стримит подготовленный SQLite backup-файл в HTTP-ответ. Он больше не читает весь backup в память перед отправкой в браузер.

Зашифрованные backup downloads пока используют прежний whole-payload envelope. Этот путь всё ещё буферизует plaintext, потому что текущий encrypted envelope шифрует весь payload целиком.

## Кеш output подписок

Base, JSON и Clash подписки теперь используют короткий TTL cache. Кеш очищается после успешного сохранения конфигурации. Это уменьшает повторную генерацию ссылок, когда много клиентов одновременно запрашивают подписки, например после рестарта.

Кешируются только успешные ответы. Missing clients и ошибки не кешируются.

Headers подписки кешируются вместе с ответом до 45 секунд. В этом окне traffic и userinfo headers при повторных запросах могут немного отставать от текущих счётчиков.

## Frontend chunks

Frontend build теперь отделяет стабильные vendor chunks для Vue, Vuetify и HTTP-зависимостей. Главный app entry стал меньше, а браузеры чаще смогут переиспользовать vendor chunks между релизами.

DateTime picker загружается lazy из client modals. Лишние Moment locale imports удалены. Сам Moment остаётся, потому что текущий Persian datetime picker зависит от `moment-jalaali`; безопасное удаление требует замены picker. Chart.js и YAML проверены, но оставлены: их замена требует проверки графиков и совместимости Clash config.

## Обновление

Обновляйтесь обычным способом. Ручная миграция базы или изменение конфигурации не нужны.

Это beta release. Публикуйте его как GitHub pre-release и не помечайте как Latest stable.
