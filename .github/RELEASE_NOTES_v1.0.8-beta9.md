# S-UI-X Extended v1.0.8-beta9

This beta fixes payment recovery, core lifecycle failures, malformed subscription input, stale frontend requests, and release and update edge cases.

Paid subscriptions now store every provider charge and retry failed Telegram or CryptoBot work without skipping it. Paid orders can finish after their pending deadline when the provider confirms payment. Refunds restore purchased capacity without rewriting current usage or lifetime traffic counters.

Core startup rolls back partial initialization, and shutdown no longer races. Clash log streaming works again. Certificate renewal retries saved but unapplied files; Telegram backup validates its fixed Argon2 profile. Malformed share links now return errors instead of panicking.

The frontend guards WebSocket, CSRF, loading, and polling work by generation or in-flight state. Old requests cannot overwrite a newer session, and damaged saved JSON or blocked browser storage no longer stops affected views.

Release workflows now validate tag provenance and artifact metadata more strictly. Panel updates preserve a durable rollback snapshot when an update is interrupted or started concurrently.

Normal startup adds the provider-charge ledger. No manual configuration change is required. Test this beta on a non-critical server before upgrading production.

Full release notes: [`docs/releases/v1.0.8-beta9.md`](../docs/releases/v1.0.8-beta9.md).
