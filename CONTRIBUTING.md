# Contributing

## Запуск audit-pipeline локально

Минимальный набор перед PR:

```sh
make audit:build
make audit:vet
make audit:test-go
make audit:fe-lint
make audit:fe-build
make audit:test-fe
```

`make audit:fe-build` запускает `vue-tsc --noEmit` внутри `npm run build`; отдельного `typecheck` script сейчас нет.

Soft checks, которые полезны для диагностики, но сейчас не являются merge gates:

```sh
make audit:test-go-race
make audit:lint-go
make audit:gosec
make audit:vuln
go test -tags=chaos ./tests/chaos/... -count=1 -timeout 30m
go test ./... -bench=. -benchmem -run=^$ -benchtime=2s
```

Frontend:
- перед frontend checks выполните `make audit:fe-install` или `npm ci` в `frontend/`;
- Playwright требует `npx playwright install chromium`;
- e2e использует test-only server helper `tests/e2e/run-server.js`.

Fixtures:
- полные importxui fixture-тесты требуют `test-db/x-ui.db` и `test-db/s-ui.db`;
- если fixture нет, эти тесты должны оставаться skipped.

Отчёты собираются в `tests/baseline/phase*/`. Dashboard можно пересобрать так:

```sh
bash scripts/audit/aggregate.sh
```
