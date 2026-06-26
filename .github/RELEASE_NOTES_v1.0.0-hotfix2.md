# s-ui-x-extended v1.0.0-hotfix2

Hotfix for the first stable release. No manual database migration is required.

## Fixes

- Fixed the stale frontend cache loop after reinstalling the panel. The login page now sends `Clear-Site-Data: "cache"`, and the frontend does a cache-busting reload if it receives `Invalid login` right after a successful login.
- Fixed Mieru inbound generation when some enabled clients do not have `config.mieru`. Missing per-protocol client config is skipped instead of crashing on a SQL `NULL` scan.
- Kept generated inbound `users` as an empty array when no valid per-protocol users are found, rather than serializing `users` as `null`.
- Added missing English and Russian labels for extended protocol editors, including Sudoku, Mieru, MTProxy, TrustTunnel, MASQUE, OpenVPN, providers, VPN, xHTTP, and mKCP fields.
- Restored the frontend asset filename convention checked by the release tests: entry files use `entry-`, chunks use `chunk-`, styles use `style-`, and copied assets use `asset-` prefixes.
- Cleared the service linter findings in Doctor, IP certificate storage, and panel self-update code. Managed certificate files, downloaded update artifacts, and pending update markers now use private file permissions.

## Verification

Checked locally with:

- `go test ./... -count=1`
- `go vet ./...`
- `golangci-lint run ./service`
- `cd frontend && npm run test`
- `cd frontend && npm run lint`
- `cd frontend && npm run build`
