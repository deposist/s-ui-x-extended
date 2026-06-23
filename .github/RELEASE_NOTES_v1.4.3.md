# S-UI v1.4.3 - sing-box runtime update

> Supersedes [`v1.4.2-beta`](https://github.com/deposist/s-ui-rus-inst/releases/tag/v1.4.2-beta).
> Full changelog and upgrade guide: [CHANGELOG.md](https://github.com/deposist/s-ui-rus-inst/blob/beta/CHANGELOG.md).

---

## English

This release updates the embedded sing-box runtime from `v1.13.4` to
`v1.13.11`. The panel API, frontend forms, and SQLite schema are unchanged.

### Changes
- **sing-box `v1.13.11`** with upstream fixes from the `1.13.x` line.
- **NaiveProxy runtime refreshed** through the matching `cronet-go` dependency set.
- **fake-ip DNS fix** from upstream `1.13.8`.
- **process searcher regression fixed** by upstream `1.13.10`/`1.13.11`.
- **No database migration** and no manual config rewrite required.

### Security and Operations
- Linux release builds pin `cronet-go` to full commit `e4926ba205fae5351e3d3eeafff7e7029654424a`.
- Deploy the full archive or rebuilt image; do not copy only the `sui` binary over an older `libcronet.so`/`libcronet.dll`.
- Back up `s-ui.db`, `s-ui.db-wal`, and `s-ui.db-shm` before production rollout.
- Keep production logging at `info` after smoke testing.

### Upgrade
```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.4.3
```

Manual installs should replace the full extracted archive for the target
architecture, then restart the service.

---

## Русский

Этот релиз обновляет встроенный sing-box с `v1.13.4` до `v1.13.11`. API
панели, frontend-формы и схема SQLite не меняются.

### Основное
- **sing-box `v1.13.11`** с исправлениями upstream-ветки `1.13.x`.
- **Обновлён runtime NaiveProxy** через согласованный набор зависимостей `cronet-go`.
- **Исправление fake-ip DNS** из upstream `1.13.8`.
- **Исправление регрессии process searcher** из upstream `1.13.10`/`1.13.11`.
- **Миграция БД не нужна**, ручная правка конфигов не требуется.

### Безопасность и эксплуатация
- Linux release-сборки фиксируют `cronet-go` на полном коммите `e4926ba205fae5351e3d3eeafff7e7029654424a`.
- Разворачивайте полный архив или пересобранный образ; не копируйте только `sui` поверх старого `libcronet.so`/`libcronet.dll`.
- Перед production-обновлением сохраните `s-ui.db`, `s-ui.db-wal` и `s-ui.db-shm`.
- После smoke-теста верните production-логирование на `info`.

### Обновление
```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-rus-inst/beta/install.sh) v1.4.3
```

При ручной установке заменяйте полный распакованный архив для нужной
архитектуры, затем перезапускайте сервис.
