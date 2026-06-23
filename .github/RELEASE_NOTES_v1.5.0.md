# S-UI v1.5.0 - security foundation and realtime platform

## Upgrade notes

Before upgrading production, create a protected backup of the database files:

```sh
sudo systemctl stop s-ui
sudo install -d -m 0700 /root/s-ui-backups
sudo cp -a /usr/local/s-ui/s-ui.db /root/s-ui-backups/s-ui.db.$(date +%Y%m%d%H%M%S)
sudo cp -a /usr/local/s-ui/s-ui.db-wal /root/s-ui-backups/ 2>/dev/null || true
sudo cp -a /usr/local/s-ui/s-ui.db-shm /root/s-ui-backups/ 2>/dev/null || true
sudo systemctl start s-ui
```

Do not publish database backups, subscription URLs, private keys,
certificates, admin credentials, or API tokens in pull requests, issues, CI
logs, or support chats.

## Changes
- Admins can invalidate all active web sessions from the Admins panel. This
  rotates the web session generation and clears the initiator cookie. API
  tokens are not revoked by this action.
- Secret-aware settings now use an AES-GCM/HKDF secretbox helper. Set
  `SUI_SECRETBOX_KEY` in production; otherwise the panel falls back to
  `settings.secret` for compatibility and logs a warning.
- Secret values are not returned by `api/settings`; the response exposes
  `<key>HasSecret` markers instead, and saving an empty secret field keeps the
  existing encrypted value.
- Security audit events are stored in the new `audit_events` table and exposed
  through `/api/security/audit`. Event details are redacted before storage.
  The default retention setting is `30` days.
- Browser API mutations now require a session-bound CSRF token from
  `GET /api/csrf` and reject missing, wrong, or expired tokens with HTTP 403.
  `/apiv2/*` token API requests are not affected.
- API tokens are migrated from plaintext to salted SHA-256 hashes on load.
  New tokens are shown once, then stored only as hash and prefix metadata.
  Token enable/disable controls are available in the Admins API token dialog.
- Use `Authorization: Bearer <token>` for `/apiv2/*`. The legacy `Token`
  header remains temporarily supported, writes an audit event, and returns
  `Deprecation: true` plus `Sunset: Sat, 15 Aug 2026 00:00:00 GMT`.
- Subscription URLs now have per-client secrets. Legacy `/sub/<name>` links
  keep working by default, while `/sub/<secret>`, `/sub/json/<secret>`,
  `/sub/clash/<secret>`, `/json/<secret>`, and `/clash/<secret>` are available
  for hardened clients. Set `subSecretRequired=true` after rotating published
  links if name-based subscription URLs must be disabled.
- Subscription responses sanitize profile headers and enforce a per-IP rate
  limit.
- Observability endpoints now expose bounded in-memory system/core history and
  `GET /api/version`. `POST /api/checkOutbounds` checks all outbounds with
  concurrency and timeout bounds and rejects non-HTTPS or private-IP targets.
- Telegram notifications are disabled by default. `POST /api/telegram/test`
  returns `{success, errorClass}` without raw Telegram responses, and event
  delivery is best-effort only after Telegram is explicitly enabled.
- Realtime WebSocket foundation is available under `/api/realtime/*` and uses
  one-time `wsToken` authentication. Slow clients are dropped, connection
  counts are limited per user/IP, and `logoutAllAdmins` closes active sockets
  with code `4401`. The frontend has a polling fallback.
- Client IP monitoring stores observed IPs through an in-memory batch
  aggregator and periodic flush. The default mode is `monitor`; `enforce`
  rejects only new over-limit connections and does not close active sessions.
  The Clients page shows IP counts, recent IP history, clear history, and an
  enforcement warning.
- Grouped API routes were added as the compatibility layer for upcoming
  security, notification, observability, and bulk outbound-check features.
  Existing `/api/<action>` URLs remain supported.
- The installer and `s-ui` management menu now include Chinese as language
  option `3`. Non-interactive installs can use `SUI_LANG=zh`.
- The embedded `sing-box` runtime remains `v1.13.11` from the `v1.4.3`
  runtime update.

## Security defaults

- Telegram notifications are opt-in. This release does not add external
  analytics or telemetry.
- Production deployments that enable encrypted secret storage should set a
  stable `SUI_SECRETBOX_KEY` value and keep it outside the repository and CI
  logs.
- Legacy `Token` header sunset date: `2026-08-15`. Move integrations to
  `Authorization: Bearer <token>` before that date.

## Rollback

The current database schema remains compatible with the previous `v1.4.3`
binary. If rollback is required, deploy the full previous archive or image,
including runtime sidecar libraries such as `libcronet`, so they
remain in sync with the binary.
