# S-UI-X Extended v1.0.8-beta1

This security prerelease updates vulnerable dependencies and fixes the stalled IP certificate flow.

- Updates Go, `quic-go`, Axios, and `brace-expansion` to patched versions.
- Rejects oversized updater artifacts and cleans up partial downloads.
- Limits Telegram API response bodies.
- Attempts to restore the previous panel state after IP certificate issuance and returns control to the management menu.
- Returns an error for malformed Clash proxy-group templates instead of panicking.
- Makes security checks blocking and adds artifact scans, an SPDX SBOM, and build provenance to the release pipeline.

No database migration is required. Test this beta on a non-critical server before production use.

Full release notes: [`docs/releases/v1.0.8-beta1.md`](../docs/releases/v1.0.8-beta1.md).
