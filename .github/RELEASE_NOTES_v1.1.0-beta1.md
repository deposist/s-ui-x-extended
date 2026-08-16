# S-UI-X Extended v1.1.0-beta1

This beta fixes a blank page in browsers that block storage, and tightens two paths in the self-updater. No database migration or manual configuration change is required.

In Chrome with "Block all cookies" enabled, any read of `localStorage` throws a `SecurityError`. The panel read stored preferences during startup, so the first read aborted boot and the page stayed white. All preference access now goes through a wrapper that catches the failure and keeps values in memory for the session. Theme, language, table page sizes, and sidebar state work again in those browsers, and a write that hits a quota error no longer loses the value.

The page `lang` attribute now follows the selected interface language instead of a hardcoded `zh-CN` tag. Screen readers pick the right pronunciation; the attribute starts as `en` until the app applies the saved locale.

On the update side, the artifact downloader refuses a redirect that would move the transfer from HTTPS to plain HTTP. A release tag from the GitHub API must also normalize to a valid version before it is placed in a download URL; a tag that fails validation is reported as no release. The checksum and manifest checks are unchanged. These two changes are defense in depth, not a response to a known attack.

## Upgrade from the console

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.0-beta1/install.sh \
  | sudo bash -s -- v1.1.0-beta1
```

Then verify the installation:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

The installer preserves the panel database and configuration.

Full release notes: [`docs/releases/v1.1.0-beta1.md`](../docs/releases/v1.1.0-beta1.md).

## Обновление через консоль

Эта бета исправляет пустой экран в браузерах, которые блокируют хранилище, и закрывает два пути в модуле самообновления. Миграция базы и ручное изменение конфигурации не требуются.

В Chrome с включённой настройкой «Блокировать все файлы cookie» любое чтение `localStorage` вызывает `SecurityError`. Панель читала сохранённые настройки во время запуска, поэтому первое же чтение прерывало загрузку и страница оставалась белой. Теперь весь доступ к настройкам идёт через обёртку, которая перехватывает сбой и хранит значения в памяти до конца сессии. Тема, язык, размер страниц таблиц и состояние боковой панели снова работают в таких браузерах, а запись, упёршаяся в лимит хранилища, больше не теряет значение.

Атрибут `lang` страницы теперь следует выбранному языку интерфейса, а не жёстко заданному `zh-CN`. Скринридеры выбирают правильное произношение; до применения сохранённого языка атрибут равен `en`.

В модуле обновления загрузчик артефактов отклоняет redirect, который переводит передачу с HTTPS на открытый HTTP. Тег релиза из API GitHub обязан нормализоваться в корректную версию, прежде чем попадёт в URL загрузки; тег, не прошедший проверку, трактуется как отсутствие релиза. Проверки контрольной суммы и манифеста не менялись. Обе правки нужны для глубины защиты, а не как ответ на известную атаку.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.0-beta1/install.sh \
  | sudo bash -s -- v1.1.0-beta1
```

После установки проверьте версию и службу:

```sh
/usr/local/s-ui/sui -v
systemctl is-active s-ui
journalctl -u s-ui -n 50 --no-pager
```

Установщик сохраняет базу и настройки панели.
