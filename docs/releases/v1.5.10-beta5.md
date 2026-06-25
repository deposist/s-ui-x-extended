# Release Notes: v1.5.10-beta5
Release date: 2026-06-24

This beta is a frontend release for the Nexus interface. It tightens the dashboard layout, adds two palette families, and moves sing-box Basics into Settings.

No database, API, or sing-box configuration migration is required.

## Nexus dashboard

The dashboard layout is denser and more even across the main cards. KPI cards, overview panels, protocol summaries, recent events, system status, and dense tables now use tighter spacing and more consistent alignment.

Top clients no longer looks empty when there are only a few rows. It now shows real summary totals, online count, total client count, compact traffic columns, and a link to the Clients page.

The live traffic KPI now has a local time-window selector for 1 minute, 5 minutes, 30 minutes, 60 minutes, 5 hours, 12 hours, and 24 hours. It uses the existing realtime samples; no mock data was added.

## Sidebar status

The Nexus sidebar now uses the S-UI-X name consistently, with a shorter collapsed mark.

The lower-left server status block now shows CPU and RAM percentages when `/api/status?r=cpu,mem` returns them. If those values are not available, the connection status remains visible without showing stale metrics.

## Palettes and light theme

Two Nexus palette families were added: Emerald and Dracula. Both have dark and light variants.

Light Nexus palettes now use darker text and lighter surfaces for better contrast. Dracula Light warning and info colors were adjusted so warning text does not sit on a matching yellow background.

## Settings and Basics

Nexus Settings now use card-based sections with more predictable spacing. The Maintenance tab has a two-column layout, and the Config Doctor and panel update cards have more even headers, dividers, and empty states.

The sing-box Basics page is now available as a Settings tab: Basics (Singbox). The old `/basics` route redirects to `/settings?tab=basics`, and the Basics menu item was removed from both Nexus and Classic navigation to avoid two entry points for the same settings.

The migrated Basics tab keeps the original Log, NTP, Certificate Trust, Cache File, Debug, Clash API, and V2Ray API controls. The Save button is enabled only when the sing-box config has changed, and the loading state is cleared even if a save fails.

---

# Примечания к релизу: v1.5.10-beta5
Дата релиза: 2026-06-24

Эта бета относится к фронтенду Nexus. В ней уплотнён Dashboard, добавлены две группы палитр, а sing-box Basics перенесён в Settings.

Миграция базы данных, API или конфигурации sing-box не требуется.

## Nexus Dashboard

Dashboard стал плотнее и ровнее по основным карточкам. KPI cards, overview panels, protocol summaries, recent events, system status и dense tables теперь используют более аккуратные отступы и одинаковое выравнивание.

Карточка Top clients больше не выглядит пустой при малом числе строк. В ней появились реальные сводные значения, число online-клиентов, общее число клиентов, компактные traffic-колонки и ссылка на страницу Clients.

В KPI live traffic добавлен выбор локального окна: 1 минута, 5 минут, 30 минут, 60 минут, 5 часов, 12 часов и 24 часа. Используются уже доступные realtime samples; mock-данные не добавлялись.

## Статус в sidebar

В Nexus sidebar теперь везде используется имя S-UI-X, а collapsed-состояние показывает короткую метку.

Нижний блок server status показывает CPU и RAM в процентах, когда `/api/status?r=cpu,mem` возвращает эти значения. Если метрики недоступны, статус соединения остаётся видимым без устаревших цифр.

## Палитры и светлая тема

Добавлены две группы палитр Nexus: Emerald и Dracula. У каждой есть тёмный и светлый вариант.

Светлые палитры Nexus получили более тёмный текст и более светлые поверхности для лучшей читаемости. В Dracula Light исправлены warning и info цвета, чтобы warning-текст не попадал на похожий жёлтый фон.

## Settings и Basics

Nexus Settings теперь разбиты на карточки с более предсказуемыми отступами. Вкладка Maintenance получила двухколоночную раскладку, а карточки Config Doctor и Panel updates получили ровные заголовки, разделители и empty states.

sing-box Basics теперь доступен как вкладка Settings: Basics (Singbox). Старый маршрут `/basics` ведёт на `/settings?tab=basics`, а пункт Basics удалён из Nexus и Classic navigation, чтобы у одной страницы не было двух разных входов.

Перенесённая вкладка Basics сохраняет исходные группы Log, NTP, Certificate Trust, Cache File, Debug, Clash API и V2Ray API. Кнопка Save включается только при изменении sing-box config, а loading state сбрасывается даже при неудачном сохранении.
