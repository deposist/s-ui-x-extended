# s-ui-x-extended v1.0.0-hotfix1

Hotfix for the first stable release. No manual database migration is required.

## Fixes

- Fixed a login loop that could appear after reinstalling the panel in a browser with old cached assets. The login page no longer shows a misleading "Invalid login" message while already unauthenticated, and the frontend reloads once after a successful login if stale assets keep sending unauthenticated requests.
- Fixed user injection for Mieru, TrustTunnel, SSH, and MTProxy inbounds. These protocols now receive `users` from enabled clients before the runtime config is passed to sing-box.
- New Mieru, TrustTunnel, SSH, and MTProxy inbounds now select all existing clients by default and block saving when no client is selected, preventing `users is empty` core start failures.
- Restored the xHTTP transport editor in the inbound UI.
- Restored the mKCP transport editor in the inbound UI.

## Verification

Checked locally with:

- `go test ./...`
- `go vet ./...`
- `cd frontend && npx vitest run`
- `cd frontend && npm run lint`
- `cd frontend && npm run build`
