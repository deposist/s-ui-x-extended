# s-ui-x-extended v1.0.0-hotfix3

Hotfix for the first stable release. No manual database migration is required.

## Fixes

- Fixed Mieru and other extended protocol inbounds failing with `users is empty` when existing clients did not have per-protocol credentials. The backend now backfills missing protocol config blocks (passwords, UUIDs, mtproxy secrets) when clients are linked to an inbound, covering all three paths: new inbound, edit inbound, and bulk client edit.
- Hardened error handling in the credential backfill: UUID generation, shadowsocks password generation, and mtproxy secret generation now propagate errors instead of silently using empty values.

## Verification

Checked locally with:

- `go test ./... -count=1`
- `go vet ./...`
- `golangci-lint run ./service`
- `cd frontend && npm run test`
- `cd frontend && npm run lint`
- `cd frontend && npm run build`
