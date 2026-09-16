# S-UI-X Extended v1.1.1-beta5

The Additional collections section is now visible on the Basics tab in Settings. Beta4 added the structured forms but mounted them on a page the panel no longer renders, so the new fields never appeared. This release puts them in the tab users actually see.

Certificate providers (ACME, Tailscale, Cloudflare Origin CA), reusable HTTP clients and Linux network namespaces are editable as cards with add, edit and delete actions. Forms validate input before saving. The panel Save button writes all three collections to the panel database in one request; this was verified end to end on a live panel.

The section appears in both panel layouts, the nexus grid and the classic expansion panels.

The database schema is unchanged from beta4. No migration is required.

Full release notes: [`docs/releases/v1.1.1-beta5.md`](../docs/releases/v1.1.1-beta5.md).

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta5/install.sh \
  | sudo bash -s -- v1.1.1-beta5
```

Verify:

```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta5
systemctl is-active s-ui  # active
```

## Обновление

Раздел «Additional collections» теперь виден на вкладке «Basics» в настройках. Beta4 добавил структурированные формы, но разместил их на странице, которую панель больше не рендерит, поэтому новые поля не появлялись. Этот релиз помещает их во вкладку, которую видит пользователь.

Провайдеры сертификатов (ACME, Tailscale, Cloudflare Origin CA), общие HTTP-клиенты и сетевые пространства имён Linux редактируются как карточки с действиями добавления, правки и удаления. Формы проверяют ввод до сохранения. Кнопка Save панели записывает все три коллекции в базу одним запросом; это проверено на живой панели.

Раздел появляется в обеих раскладках панели: в nexus-гриде и в классических expansion-панелях.

Схема базы данных не менялась со времён beta4. Миграция не требуется.

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta5/install.sh \
  | sudo bash -s -- v1.1.1-beta5
```

Проверка:

```sh
/usr/local/s-ui/sui -v   # 1.1.1-beta5
systemctl is-active s-ui  # active
```

Полные заметки: [`docs/releases/v1.1.1-beta5.md`](https://github.com/deposist/s-ui-x-extended/blob/v1.1.1-beta5/docs/releases/v1.1.1-beta5.md)
