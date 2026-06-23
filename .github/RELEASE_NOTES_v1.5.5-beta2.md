# S-UI v1.5.5-beta2

Prerelease hotfix for backup restore safety after the `v1.5.5-beta1`
restore report.

## Fixed

- Backup export now preserves the no-TLS `tls(id=0)` sentinel explicitly.
  No-TLS inbounds store `tls_id=0`; without that parent row a copied backup
  can fail restore with `Foreign key check failed: inbounds=1`.
- Restore inserts the no-TLS parent before migration foreign-key checks, so
  backups created before this prerelease can restore without a manual SQLite
  edit for that sentinel row.
- Failed database imports reopen the rolled-back live database. The running
  panel no longer keeps a closed DB handle after a rejected restore.
- SQLite-backed admin sessions follow the current live DB handle after an
  import swap. Settings reads also return an error instead of panicking if
  the global DB is briefly unavailable.

## Added

- Regression tests for no-TLS backup FK validity, no-TLS migration sentinel
  repair, and rollback/reopen after an import rejected by foreign-key checks.

## Validation

- `go test ./cmd/migration ./database ./service ./web ./api` - PASS.
- `cd frontend && npm run build` - PASS.
- `git diff --check` - PASS.

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta2
```

## Русский

Prerelease hotfix для восстановления резервной копии после отчёта по
`v1.5.5-beta1`.

- Backup export явно сохраняет служебную строку no-TLS `tls(id=0)`.
  No-TLS inbound хранит `tls_id=0`; без parent row скопированный backup
  может быть отклонён с `Foreign key check failed: inbounds=1`.
- Restore восстанавливает этого no-TLS parent до migration foreign-key
  check, поэтому backup, созданный до этого prerelease, не требует ручной
  SQLite-правки для этой строки.
- При ошибке database import rollback переоткрывает live DB. Работающая
  панель больше не остаётся с закрытым DB handle после отклонённого restore.
- SQLite-backed admin sessions после import swap используют актуальный DB
  handle, а чтение settings при краткой недоступности DB возвращает ошибку
  вместо panic.
