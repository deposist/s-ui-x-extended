# S-UI-X Extended v1.0.8-beta4

This beta concentrates on the failure paths that matter when a database write, restore, or self-update goes wrong.

Self-update now requires a signed manifest. It checks the archive, version, channel, platform, and checksum before replacing the running binary. Database restore waits for active panel, subscription, and cron database work; the x-ui rollback route no longer waits on its own request lease. Backups read one source snapshot and check SQLite integrity before export.

Traffic, IP-monitor, token-use, and audit batches stay pending until their database commit succeeds. The initial admin password file is consumed only after the bootstrap password was changed successfully. The panel also clears stuck loading indicators, stops stale stats polling, and keeps valid API values such as `false`, `0`, and an empty string intact.

The default Compose image now uses `v1.0.8-beta4`. nftables permissions remain opt-in. No database migration is required.

This is a beta release. Test restore and self-update on a non-critical server first.

Full release notes: [`docs/releases/v1.0.8-beta4.md`](../docs/releases/v1.0.8-beta4.md).
