# S-UI v1.5.5-beta4-hotfix2

## English

Hotfix release for `v1.5.5-beta4-hotfix1`. It keeps the beta4 feature set and
fixes a backup export failure triggered by databases that contain both the
no-TLS sentinel row and regular TLS configs.

### Fixed

* Backup export no longer copies the `tls.id=0` sentinel through GORM's generic
  auto-increment create path. The exporter now skips that row during the TLS
  table copy and restores it explicitly with `INSERT OR IGNORE`.
* Fixed `UNIQUE constraint failed: tls.id` during backup export when SQLite
  assigned the copied sentinel a generated id that collided with a real TLS row.
* Added regression coverage for a database containing both `tls.id=0` and a
  normal TLS record.

### Validation

* `go test ./database -count=1` - PASS
* `go test ./config ./database ./service -count=1` - PASS

### Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta4-hotfix2
```

## Русский

Hotfix-релиз для `v1.5.5-beta4-hotfix1`. Он сохраняет набор изменений beta4 и
исправляет падение экспорта backup в базах, где одновременно есть служебная
no-TLS строка и обычные TLS-конфиги.

### Исправлено

* Экспорт backup больше не копирует `tls.id=0` через общий GORM-путь создания
  auto-increment записей. Теперь эта строка пропускается при копировании
  таблицы TLS и восстанавливается отдельно через `INSERT OR IGNORE`.
* Исправлена ошибка `UNIQUE constraint failed: tls.id`, возникавшая, когда
  SQLite выдавал скопированной служебной строке сгенерированный id, совпадающий
  с реальной TLS-записью.
* Добавлен regression coverage для базы, где одновременно есть `tls.id=0` и
  обычная TLS-запись.

### Валидация

* `go test ./database -count=1` - PASS
* `go test ./config ./database ./service -count=1` - PASS

### Установка

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta4-hotfix2
```
