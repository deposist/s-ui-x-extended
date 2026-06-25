# Release Notes: v1.5.10-beta7
Release date: 2026-06-25

This beta fixes the Traffic statistics KPI added in v1.5.10-beta6. The dashboard now uses a dedicated traffic summary endpoint that returns exact bucket sums for all inbound traffic in the selected period.

No manual migration is required. The new endpoint reads the existing `stats` table. Traffic history still depends on Traffic Maximum Age: values above `0` save stats, and `0` disables historical stats.

## Traffic statistics KPI

The first Nexus Dashboard KPI now reads from `/api/stats/traffic` instead of combining per-inbound `/api/stats` responses in the browser.

The backend groups the selected period into fixed buckets and sums bytes inside each bucket. Download and upload totals are calculated from `SUM(traffic)`, so long ranges no longer inherit the old average-downsampling behaviour from `/api/stats`.

The summary covers all historical `resource = inbound` rows for the selected period. This keeps traffic that was already consumed by an inbound even if that inbound was later disabled or removed.

The timeline selector is unchanged: 1 hour, 6 hours, 12 hours, 24 hours, 7 days, and 30 days.

## Backend and database

Added `GET /api/stats/traffic` for dashboard traffic summaries. The existing `/api/stats` endpoint is unchanged for the older stats modal and other callers.

Added an index on `stats(resource, date_time)` so the new all-inbound time-range query can use the resource and time filters efficiently on larger databases.

---

# Примечания к релизу: v1.5.10-beta7
Дата релиза: 2026-06-25

Эта beta исправляет KPI Traffic statistics, добавленный в v1.5.10-beta6. Dashboard теперь использует отдельный endpoint статистики трафика, который возвращает точные суммы по bucket'ам для всего inbound-трафика за выбранный период.

Ручная миграция не требуется. Новый endpoint читает существующую таблицу `stats`. История трафика по-прежнему зависит от Traffic Maximum Age: значения больше `0` включают сохранение stats, а `0` отключает историческую статистику.

## KPI статистики трафика

Первый KPI в Nexus Dashboard теперь читает `/api/stats/traffic`, а не объединяет в браузере ответы `/api/stats` по отдельным inbound'ам.

Backend делит выбранный период на фиксированные bucket'ы и суммирует байты внутри каждого bucket. Totals для download и upload считаются через `SUM(traffic)`, поэтому длинные периоды больше не наследуют старое average-downsampling поведение `/api/stats`.

Сводка учитывает все исторические строки `resource = inbound` за выбранный период. Уже потраченный трафик остаётся в статистике, даже если inbound позже отключили или удалили.

Переключатель периода не изменился: 1 час, 6 часов, 12 часов, 24 часа, 7 дней и 30 дней.

## Backend и база данных

Добавлен `GET /api/stats/traffic` для сводной статистики трафика на Dashboard. Существующий `/api/stats` endpoint не изменён и остаётся доступен для старого stats modal и других вызовов.

Добавлен индекс `stats(resource, date_time)`, чтобы новый запрос по всем inbound'ам за период мог использовать фильтры resource и date_time на больших базах.
