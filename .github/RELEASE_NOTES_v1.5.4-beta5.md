# S-UI v1.5.4-beta5

Prerelease hotfix for reserved path validation.

## Fixed

- Custom panel and subscription paths such as `/wsub/` no longer fail with
  `reserved path prefix: /ws`.
- Slashless reserved framework routes now match on a path-segment boundary:
  `/ws`, `/ws/` and descendants under `/ws/` stay reserved, while unrelated
  string prefixes do not.
- Added regression coverage for accepted `/wsub/` paths and rejected `/ws/`
  descendants.

## Validation

- `go test ./config ./util ./service` - PASS
- `go test ./middleware/... -run TestAdminSecurityHeaders` - PASS
- `cd frontend && npm run build` - PASS
- `git diff --check` - PASS

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4-beta5
```

## Русский

Prerelease hotfix для reserved path validation.

- Пользовательские panel и subscription paths вроде `/wsub/` больше не
  падают с `reserved path prefix: /ws`.
- Framework route `/ws` теперь проверяется по границе сегмента пути:
  `/ws`, `/ws/` и дочерние пути под `/ws/` остаются зарезервированными,
  а несвязанные строковые prefix matches не блокируются.
- Добавлено regression coverage для разрешённого `/wsub/` и запрещённых
  дочерних `/ws/` paths.
