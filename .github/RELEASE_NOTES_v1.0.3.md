# S-UI-X Extended v1.0.3

v1.0.3 adds AmneziaWG obfuscation tooling, client config overrides, per-device expiry, and a Russia-direct routing preset. It also carries the fixes for AWG device creation from the panel, the v1.0.2-beta1 upgrade path, the self-update rollback, and sudoku inbound assignment.

- The endpoint form validates Amnezia obfuscation parameters and blocks saving combinations that make the tunnel silently fail: reserved H1-H4 values 0-4 (the old panel default provided no header masking), overlapping H1-H4 ranges, and colliding padded packet sizes.
- A Randomize button fills H1-H4 with non-overlapping ranges generated server-side with crypto/rand. Presets cover wired networks (Balanced) and mobile carriers (Mobile); a preset sets only the junk parameters, headers stay unique per endpoint.
- A managed AWG endpoint can override `AllowedIPs` and `PersistentKeepalive` in rendered device configs (client-side split tunneling). Entries are validated as CIDR prefixes and checked against newline injection; empty values keep the old defaults byte-for-byte.
- Devices can be created with an expiry date. Expired devices are deprovisioned by the reconciler but keep their device-limit slot until deleted; extending the date restores the peer. The panel and the Telegram bot show the term.
- The regional presets drawer can create a Russia-direct rule for an AWG endpoint: device traffic to Russian IPs leaves the server directly, the rest follows the existing rules.
- Fixed: device creation from the client form (`awg: invalid request`), the v1.0.2-beta1 upgrade boot-loop (`no such column: endpoint_id`), the self-update rollback on Linux (`text file busy`), and sudoku inbound assignment (#4).
- Also: assignments to non-managed endpoints are rejected, `client_endpoint_access` is included in backups, and endpoints managed through the legacy global settings are marked and protected again.

The `expires_at` column is added automatically. If an existing AWG endpoint still uses the old default headers (H1=1 to H4=4), press Randomize, save, and have devices re-download their configs. Full notes: `docs/releases/v1.0.3.md`.
