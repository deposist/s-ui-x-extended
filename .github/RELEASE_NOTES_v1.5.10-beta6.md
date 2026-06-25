# Release Notes: v1.5.10-beta6
Release date: 2026-06-24

This beta continues the Nexus dashboard work from v1.5.10-beta5. It makes the main dashboard cards more even and replaces the local live-traffic sparkline with traffic history from the existing stats API.

No database, API, or sing-box configuration migration is required. Traffic history still uses the existing Traffic Maximum Age setting: values above `0` save stats, and `0` disables historical stats.

## Nexus dashboard cards

Top clients now shows up to 10 clients instead of 5. Its height matches System status, and the client table scrolls inside the card, so a longer list no longer stretches the dashboard row.

Recent events now uses the same height as System status. The event list scrolls inside the card when there are more rows than fit in the visible area.

System status is shorter and sized around the 8 values it currently shows: S-UI-X, sing-box, host uptime, sing-box uptime, CPU, memory, disk, and realtime status.

## Traffic statistics KPI

The first KPI now shows Traffic statistics instead of Live traffic. The chart uses the existing `/api/stats` endpoint and aggregates enabled inbound tags for the selected timeline.

The timeline selector covers 1 hour, 6 hours, 12 hours, 24 hours, 7 days, and 30 days. The chart plots download and upload as separate curves and shows selected-period totals in the KPI.

No mock data was added. If Traffic Maximum Age is `0`, the backend does not save historical stats, so the chart has no history to show.

---

# Примечания к релизу: v1.5.10-beta6
Дата релиза: 2026-06-24

Эта beta продолжает работу над Nexus Dashboard из v1.5.10-beta5. Основные карточки Dashboard стали ровнее по высоте, а локальная live-traffic кривая заменена историей трафика из существующего stats API.

Миграция базы данных, API или конфигурации sing-box не требуется. История трафика по-прежнему использует существующую настройку Traffic Maximum Age: значения больше `0` включают сохранение stats, а `0` отключает историческую статистику.

## Карточки Nexus Dashboard

Top clients теперь показывает до 10 клиентов вместо 5. Высота карточки совпадает с System status, а таблица клиентов прокручивается внутри карточки, поэтому длинный список не растягивает строку Dashboard.

Recent events теперь использует ту же высоту, что и System status. Список событий прокручивается внутри карточки, если строк больше, чем помещается в видимую область.

System status стал ниже и рассчитан под 8 текущих значений: S-UI-X, sing-box, host uptime, sing-box uptime, CPU, memory, disk и realtime status.

## KPI статистики трафика

Первый KPI теперь показывает Traffic statistics вместо Live traffic. График использует существующий `/api/stats` endpoint и агрегирует включённые inbound tags для выбранного периода.

Переключатель периода поддерживает 1 час, 6 часов, 12 часов, 24 часа, 7 дней и 30 дней. График показывает download и upload отдельными кривыми, а KPI выводит totals за выбранный период.

Mock-данные не добавлялись. Если Traffic Maximum Age равен `0`, backend не сохраняет исторические stats, поэтому графику нечего показывать.
