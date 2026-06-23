# API Token Scope Matrix

Status: `1.5.1-beta`.

S-UI API tokens support five scopes: `admin`, `read`, `write`, `observability`,
and `telegram`.
An empty scope is normalized to `admin` when a token is created. Browser cookie
sessions are treated as `admin` in the current single-admin model.

Use `Authorization: Bearer <token>` for `/apiv2/*`. The legacy `Token` header is
accepted during the sunset window only and returns `Deprecation` and `Sunset`
headers. Do not put API tokens into URLs.

## Scope Rules

| Endpoint or channel | Cookie session | `admin` | `telegram` | `read` | `write` | `observability` | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `GET /api/security/audit` | allowed | allowed | denied | denied | denied | denied | Cursor pagination plus `event`, `severity`, `since`, and `until` filters. Denials write `audit_scope_denied`. |
| `GET /apiv2/security/audit` | n/a | allowed | denied | denied | denied | denied | Bearer/API-token flow for audit reads. |
| `GET /api/observability/history` | allowed | allowed | denied | denied | denied | allowed | Validates bucket, metric, and since query values. |
| `GET /api/observability/core-history` | allowed | allowed | denied | denied | denied | allowed | Same scope policy as observability history. |
| `POST /api/telegram/test` | allowed | allowed | denied | denied | denied | denied | Telegram remains off by default; proxy/token fields stay secret. |
| `POST /api/telegram/backup` | allowed | allowed | allowed | denied | denied | denied | Legacy tg_backup_run route; requires `telegramBackupEnabled=true`, returns no `backupKey`, and shares the manual Telegram backup rate-limit bucket. |
| `POST /api/telegram/backup/run` | allowed | allowed | allowed | denied | denied | denied | tg_backup_run manual trigger for browser sessions. |
| `POST /apiv2/telegram/backup` | n/a | allowed | allowed | denied | denied | denied | Legacy tg_backup_run Bearer/API-token route; same behavior as `/apiv2/telegram/backup/run`. |
| `POST /apiv2/telegram/backup/run` | n/a | allowed | allowed | denied | denied | denied | tg_backup_run manual trigger for Bearer/API-token clients. |
| `GET /api/getdb` and `GET /apiv2/getdb` | allowed | allowed | denied | denied | denied | denied | Database exports are audited; `encryptTelegramBackup=true` optionally returns a Telegram backup envelope using the stored Telegram backup passphrase. |
| `POST /api/importdb` and `POST /apiv2/importdb` | allowed | allowed | denied | denied | denied | denied | Database imports are capped, integrity-checked, audited, and can restore Telegram backup envelopes with a supplied passphrase. |
| `POST /api/rotateSubSecret` and `POST /apiv2/rotateSubSecret` | allowed | allowed | denied | denied | allowed | denied | Rotates per-client subscription secrets and audits the action without logging the secret. |
| `/api/realtime/ws-token` + `/api/realtime/ws` | allowed | allowed | connected, filtered | connected, filtered | connected, filtered | connected, filtered | Current browser flow is session-based. If a scoped context is present, `security_event` is delivered only to `admin`; other realtime topics follow the existing topic policy. |

For endpoints not listed above, `/api/*` still requires a browser session and
CSRF protection on mutating requests. `/apiv2/*` still requires a valid API token,
but not every legacy action has a dedicated per-action scope gate yet; use
`admin` unless the endpoint is explicitly listed in this matrix.

## Security Invariants

- Secret values are not returned by list/get endpoints; only marker or prefix
  fields are exposed where needed.
- Secret values must not be written to logs, audit details, config change
  history, or Telegram captions.
- API tokens must be sent in headers, not query strings.
- Browser session/CSRF cookies enable `Secure` when any of these is true:
  `SUI_FORCE_COOKIE_SECURE=true`, configured `webURI` starts with `https://`,
  configured `webDomain` starts with `https://`, or request HTTPS/proxy
  detection marks the request as HTTPS.
- `SUI_COOKIE_KEY` accepts one or more base64-encoded raw keys of at least
  32 bytes separated by commas or semicolons. The first key signs new session
  cookies; later keys are accepted for rollover.
- `SUI_SECRETBOX_KEY` accepts a base64-encoded raw key of at least 32 bytes for
  encrypted settings. Without it, settings encryption uses a domain-separated
  HKDF key derived from `settings.secret` and can still read legacy ciphertexts
  with an audit event.
- Security-relevant denials and state changes are audited without including raw
  tokens, subscription secrets, Telegram backup passphrases, proxy credentials,
  or Telegram bot tokens.
