# S-UI-X Extended v1.0.8-beta10

This beta repairs the release pipeline failures found after beta9. CI now invokes tracked shell scripts through Bash, and release jobs use validators staged before the requested tag is checked out.

Release preflight permits a new release or one existing draft. It rejects published tag reuse, duplicate matches, and malformed release state before build jobs run. SHA-256 asset verification remains fail-closed, so a different build must use a new SemVer tag instead of replacing published files.

Docker smoke checks now resolve and run each architecture by its immutable platform manifest digest. Regression tests cover script invocation, trusted guard staging, release-state decisions, asset verification, and platform digest selection.

No database migration or manual configuration change is required. Test this beta on a non-critical server before upgrading production.

Full release notes: [`docs/releases/v1.0.8-beta10.md`](../docs/releases/v1.0.8-beta10.md).
