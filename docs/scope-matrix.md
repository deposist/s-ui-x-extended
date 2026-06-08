# API Token Scope Matrix

Status: `1.0.0-beta1`.

S-UI API tokens support **six** scopes: `admin`, `read`, `write`, `database`,
`telegram`, and `observability`. An empty scope is normalized to `admin` when a
token is created. Browser cookie sessions carry no token scope and are treated
as full access (single-admin model); mutating `/api/*` requests still require a
CSRF token.

Send tokens with `Authorization: Bearer <token>` on `/apiv2/*`. The legacy
`Token` header is accepted only during the sunset window and returns
`Deprecation` and `Sunset` response headers (hard cutoff
`Sat, 15 Aug 2026 00:00:00 GMT`). Never put API tokens in URLs.

A token carries exactly one scope. When a token's scope is insufficient for an
action, the request is rejected with `403 insufficient scope` and a
`scope_denied` audit event is recorded.

## What each scope grants

| Scope | Grants |
| --- | --- |
| `admin` | Everything, including the admin-only endpoints below. The default scope. |
| `read` | Read-only config/identity reads (`load`, `inbounds`, `outbounds`, `endpoints`, `providers`, `services`, `tls`, `clients`, `config`, `users`, `settings`, `changes`, `keypairs`), link/sub conversion (`linkConvert`, `subConvert`), and operational metrics (`stats`, `status`, `onlines`, `logs`). |
| `write` | Everything `read` allows, plus mutations and probes: `save`, `restartApp`, `restartSb`, `checkOutbound`, and subscription-secret rotation (`rotateSubSecret`). |
| `database` | Database export (`getdb`), import (`importdb`), and x-ui / 3x-ui migration (`import-xui` plan / apply / rollback / reports). |
| `telegram` | Manual Telegram backup trigger (`telegram/backup`, `telegram/backup/run`). |
| `observability` | Operational metrics (`stats`, `status`, `onlines`, `logs`) plus history (`observability/history`, `observability/core-history`). |

## Specially-gated endpoints

`admin` is always allowed in addition to the scopes listed. Cookie sessions
carry no scope, so they pass the scope gate but still require CSRF on mutating
requests.

| Endpoint | Allowed token scopes | Notes |
| --- | --- | --- |
| `GET /api/security/audit`, `/apiv2/security/audit` | `admin` | Cursor pagination + `event` / `severity` / `since` / `until` filters; rate-limited; denials write `scope_denied`. |
| `GET /api/getdb`, `/apiv2/getdb` | `database`, `admin` | Database export; `encryptTelegramBackup=true` returns an encrypted Telegram envelope using the stored backup passphrase. Audited. |
| `POST /api/importdb`, `/apiv2/importdb` | `database`, `admin` | Database import; 64 MiB cap, SQLite-magic + read-only integrity check, audited, can restore an encrypted Telegram envelope. |
| `/apiv2/import-xui/{plan,apply,rollback,reports}` (and `/apiv2/import-xui`) | `database`, `admin` | x-ui / 3x-ui migration; rate-limited, audited, automatic pre-import backup with rollback. |
| `POST /api/telegram/test` | `admin` | Telegram is off by default; proxy/token fields stay secret. |
| `POST /api/telegram/backup`, `/backup/run` (and `/apiv2/...`) | `telegram`, `admin` | Manual Telegram backup; requires `telegramBackupEnabled=true`; shares the manual-backup rate-limit bucket. |
| `GET /api/observability/history`, `/core-history` | `observability`, `admin` | Validates `bucket` / `metric` / `since` query values. |
| `POST /api/rotateSubSecret`, `/apiv2/rotateSubSecret` | `write`, `admin` | Rotates a client's subscription secret and audits the action without logging the secret. |
| `stats`, `status`, `onlines`, `logs` (`/api/*` and `/apiv2/*`) | `read`, `write`, `observability`, `admin` | Operational metrics. |
| `/api/realtime/ws-token` + `/api/realtime/ws` | session-based | If a scoped context is present, `security_event` is delivered only to `admin`; other realtime topics follow the existing topic policy. |

For actions not listed above: `/api/*` requires a browser session (with CSRF on
mutating requests), and `/apiv2/*` requires a valid token. Actions covered by
the per-action scope map ([`api/apiV2Handler.go`](../api/apiV2Handler.go))
enforce `read` / `write` / `observability` exactly as in the scope table; any
other action accepts any valid token but is effectively `admin`-only in
practice — prefer an `admin` token unless the action is listed above.

## Security invariants

- Secret values are not returned by list/get endpoints; only marker or prefix
  fields are exposed where needed.
- Secret values must not be written to logs, audit details, config change
  history, or Telegram captions.
- API tokens must be sent in headers, not query strings. The stored form is a
  salted SHA-256 hash plus an 8-character display prefix; the plaintext is shown
  once at creation.
- Browser session/CSRF cookies enable `Secure` when any of these is true:
  `SUI_FORCE_COOKIE_SECURE=true`, configured `webURI` starts with `https://`,
  configured `webDomain` starts with `https://`, or request HTTPS/proxy
  detection marks the request as HTTPS.
- `SUI_COOKIE_KEY` accepts one or more base64-encoded raw keys of at least
  32 bytes, separated by commas, semicolons, or newlines. The first key signs
  new session cookies; later keys are accepted for rollover.
- `SUI_SECRETBOX_KEY` accepts a base64-encoded raw key of at least 32 bytes for
  encrypted settings. Without it, settings encryption uses a domain-separated
  HKDF key derived from `settings.secret` and can still read legacy ciphertexts
  with an audit event.
- Security-relevant denials and state changes are audited without including raw
  tokens, subscription secrets, Telegram backup passphrases, proxy credentials,
  or Telegram bot tokens.
