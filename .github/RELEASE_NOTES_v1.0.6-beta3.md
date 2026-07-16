# S-UI-X Extended v1.0.6-beta3

This prerelease adds an optional Sudoku Split Private Key for each panel client.

- The key is used only in that client's `sudoku://` link, QR code, and Sudoku JSON-subscription outbound. Leaving the field empty keeps the inbound-key fallback.
- Backend validation requires 128 hexadecimal characters and canonical Ed25519 scalar halves. Uppercase is normalized to lowercase. Invalid input is rejected without putting the key in errors or change history.
- JSON subscription merging can replace only `outbound.key`. Client config cannot override the server, port, AEAD, table, HTTP mask, tag, routing, or dialer fields.
- Changing a client key rebuilds that client's local links, keeps external links, clears the affected subscription cache, and does not reload the Sudoku inbound or restart the core.
- Existing backups preserve the field through `Client.Config`. No database migration is needed.

A Split Private Key is not independently revocable. Removing or disabling a client does not invalidate a key already imported into an application. Full revocation requires rotating the master pair and issuing new keys to every remaining client.

The `sing-box-extended` dependency and Sudoku server adapter are unchanged. Sudoku still has no server-side `users` array.

This is a prerelease. Install it explicitly from the tag.

Full release notes: [`docs/releases/v1.0.6-beta3.md`](../docs/releases/v1.0.6-beta3.md).
