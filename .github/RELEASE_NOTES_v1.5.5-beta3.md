# S-UI v1.5.5-beta3

Prerelease hotfix for backup restores where sing-box DNS and routing rules
were missing after import.

## Fixed

- DNS settings and routing rules are covered by restore regression tests as
  part of `settings.config`, the shared sing-box config row used by the panel.
- Saving the sing-box config recreates `settings.config` when that row is
  missing instead of silently updating zero rows.
- Restore rejects versioned S-UI database backups that already lost
  `settings.config` before they can replace the live database. Such a backup
  cannot restore DNS servers or route rules safely.

## Added

- Regression coverage for exporting and restoring a backup with custom DNS and
  route rules intact.
- Regression coverage for rejecting a versioned backup without
  `settings.config` and for recreating that row on config save.

## Validation

- `go test ./config ./cmd/migration ./database ./service ./web ./api -count=1` - PASS.
- `cd frontend && npm run build` - PASS.
- `git diff --check` - PASS.

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta3
```

## Русский

Prerelease hotfix для restore, после которого пропадали sing-box DNS и
routing rules.

- DNS и routing rules хранятся в общей строке `settings.config`; restore
  regression теперь проверяет перенос этого config через backup export/import.
- Сохранение sing-box config пересоздаёт `settings.config`, если строка
  отсутствует, вместо успешного `UPDATE` без изменённых строк.
- Restore отклоняет versioned S-UI backup без `settings.config` до подмены
  live DB. Из такого файла нельзя корректно вернуть DNS-серверы и правила.
