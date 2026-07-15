# S-UI-X Extended v1.0.3

Bugfix release. No new features, no configuration changes. Everyone on v1.0.2 should upgrade.

- Creating an AmneziaWG 2.0 device from the client form failed with `awg: invalid request`: the API only accepted JSON while the panel sends form-encoded requests. The handler now decodes both.
- Upgrading from v1.0.2-beta1 crashed the panel on every start with `no such column: endpoint_id`. Startup no longer rebuilds the `awg_devices` table, and indexes are created after the column migrations. Affected installations recover on their own after this upgrade.
- The self-update rollback never worked on Linux (`text file busy`). The backup is now renamed into place instead of written over the running binary.
- A sudoku inbound could not be assigned to clients (#4): the panel only offered inbounds with per-user credentials, and sudoku uses a single shared key. It can now be assigned and reaches clients through the JSON subscription.
- Assignments to endpoints that are not managed AWG servers are rejected, `client_endpoint_access` is included in backups, and endpoints managed through the legacy global settings are marked and protected again.

Full notes: `docs/releases/v1.0.3.md`.
