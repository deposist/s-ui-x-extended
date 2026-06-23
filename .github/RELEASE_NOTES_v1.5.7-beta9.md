# Release Notes: v1.5.7-beta9

Release date: 2026-06-10

This release brings the panel's default (Nexus) interface in line with the dark
"technical" reference design: an exact colour palette, a single typography stack,
clear status badges, a compact sidebar, and a topbar that carries each page's
title, counts and search. It is a frontend-only release - there are no backend,
breaking, manual-migration, or configuration changes.

## What changed

### Interface redesign (Nexus)

- **Exact dark "technical" palette.** Surfaces, borders (`#2a2a2a`), the cyan
  accent (`#00d4ff`), and the status and text colours now match the reference
  design exactly - including Vuetify component borders and the primary button
  (cyan background with dark text).
- **Unified typography.** The whole UI renders in one system font stack (Segoe UI
  on Windows). Table and menu body text is secondary grey with white emphasis,
  and IPs, ports and UUIDs use a monospace font.
- **Section header in the topbar.** Each page now shows its title, a stats
  subtitle (for example "8 inbounds • 3 online") and the search box in the
  topbar, with the global controls on the right; Add and filters stay in the
  content area. The header is driven by shared state so it stays consistent
  across navigation.
- **Status and TLS cells.** Online / Offline / Disabled render as dot status
  badges; TLS shows On / Off pills. The status column header is now "Status".
- **Compact sidebar.** A flat, full-width active row with a 3px cyan marker,
  40px rows, tertiary-grey group labels, and the cyan "S" brand mark.
- **Lucide icons** for the visible UI (sidebar, topbar, toolbars, row actions,
  drawers, empty states); Vuetify's internal icons are unchanged.
- **Dashboard.** The first menu entry was renamed from "Home" to "Dashboard".

### Fixes

- **Bulk client dialogs.** The bulk Add and Edit client dialogs could leave
  dropdown menus open on top of each other; they now open as proper drawers that
  close as expected.
- **Bulk Edit "Save".** The Save button in bulk client Edit no longer stays
  disabled when "All clients" is selected.
- **Paid Subscriptions labels.** The "Refresh" and "Cancel" buttons on the Paid
  Subscriptions screen are now translated instead of showing raw keys.

### Localization

- **Paid Subscriptions is now localized** (English and Russian); the remaining
  locales fall back to English. Page subtitles are localized with natural,
  per-language phrasing.

### Tests

- Added a Lucide icon-set source-scan test (every `lucide:` icon used in source
  must be mapped) and an English/Russian locale key-parity test.

## Verification

This is a frontend-only release; it was validated with:

- `npm run lint`
- `npm run test`: 25 files, 128 tests passed
- `npm run build`
- Playwright end-to-end specs for the Nexus screens
- A multi-agent regression review of the diff: no regressions found

No backend sources changed, so the Go build, tests, and security gates are
unchanged from v1.5.7-beta8.

---

# Примечания к релизу: v1.5.7-beta9

Дата релиза: 2026-06-10

Этот релиз приводит интерфейс панели по умолчанию (Nexus) к тёмному
«техническому» эталонному дизайну: точная цветовая палитра, единый набор
шрифтов, понятные бейджи статусов, компактный сайдбар и верхняя панель, в которой
размещены заголовок раздела, счётчики и поиск. Релиз затрагивает только фронтенд -
изменений в backend, ломающих изменений, ручных миграций и изменений конфигурации
нет.

## Что изменилось

### Редизайн интерфейса (Nexus)

- **Точная тёмная «техническая» палитра.** Поверхности, границы (`#2a2a2a`),
  cyan-акцент (`#00d4ff`), а также цвета статусов и текста теперь точно
  соответствуют эталону - включая границы компонентов Vuetify и основную кнопку
  (cyan-фон с тёмным текстом).
- **Единая типографика.** Весь интерфейс использует один системный стек шрифтов
  (Segoe UI в Windows). Основной текст таблиц и меню - вторичный серый с белыми
  акцентами; IP-адреса, порты и UUID выводятся моноширинным шрифтом.
- **Заголовок раздела в верхней панели.** Каждая страница теперь показывает свой
  заголовок, подзаголовок-статистику (например, «Inbounds: 8 • Онлайн: 3») и поле
  поиска в верхней панели; глобальные элементы управления - справа, а кнопка
  «Добавить» и фильтры остаются в области контента. Заголовок управляется общим
  состоянием и поэтому стабилен при переходах между разделами.
- **Ячейки статуса и TLS.** Online / Offline / Disabled показываются бейджами с
  точкой; TLS - пилюлями On / Off. Заголовок колонки статуса теперь «Status».
- **Компактный сайдбар.** Плоская активная строка во всю ширину с 3px cyan-меткой,
  строки 40px, серые (tertiary) подписи групп и cyan-логотип «S».
- **Иконки Lucide** для видимого интерфейса (сайдбар, верхняя панель, тулбары,
  действия строк, дроверы, пустые состояния); внутренние иконки Vuetify не
  изменены.
- **Dashboard.** Первый пункт меню переименован с «Home» на «Dashboard».

### Исправления

- **Диалоги массовых операций с клиентами.** Диалоги массового добавления и
  редактирования клиентов могли оставлять выпадающие меню открытыми друг поверх
  друга; теперь они открываются как полноценные дроверы и закрываются как
  ожидается.
- **Кнопка «Сохранить» в массовом редактировании.** Кнопка «Сохранить» в
  массовом редактировании клиентов больше не остаётся недоступной при выбранном
  варианте «Все клиенты».
- **Подписи на экране «Платные подписки».** Кнопки «Обновить» и «Отмена» на
  экране «Платные подписки» теперь переведены, а не показывают сырые ключи.

### Локализация

- **Экран «Платные подписки» теперь локализован** (английский и русский);
  остальные локали откатываются на английский. Подзаголовки страниц локализованы
  с естественными для каждого языка формулировками.

### Тесты

- Добавлены тест-скан набора иконок Lucide (каждая используемая в исходниках
  иконка `lucide:` должна быть в карте) и тест паритета ключей локалей
  английский/русский.

## Проверка

Это релиз только для фронтенда; он проверен следующими командами:

- `npm run lint`
- `npm run test`: 25 файлов, 128 тестов пройдено
- `npm run build`
- end-to-end-тесты Playwright для экранов Nexus
- многоагентное ревью диффа на регрессии: регрессий не обнаружено

Исходный код backend не менялся, поэтому сборка, тесты и проверки безопасности Go
остаются такими же, как в v1.5.7-beta8.
