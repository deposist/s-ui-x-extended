# S-UI-X Extended v1.0.8-beta8

This beta fixes two failures. Realtime updates could stop after an offline WebSocket token request, and Vitest could fail before registering tests on Windows.

When a WebSocket token request fails because the browser is offline, the client switches to degraded polling and retries after the network returns. Repeated offline and online transitions now end in the `connected` state.

Frontend test commands resolve the Windows path before starting Vitest. This keeps one runner context when the shell and filesystem use different letter case for the same drive.

No database migration or configuration change is required. Test this beta on a non-critical server before upgrading production.

Full release notes: [`docs/releases/v1.0.8-beta8.md`](../docs/releases/v1.0.8-beta8.md).
