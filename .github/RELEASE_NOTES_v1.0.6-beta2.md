# S-UI-X Extended v1.0.6-beta2

Backfills sudoku/mieru client links at startup and fixes the empty QR code in the AmneziaWG 2.0 device manager.

- Sudoku and mieru links used to be generated only when a client or an inbound was saved. Clients assigned before that build had no local link and no QR until someone re-saved them by hand. Startup now regenerates the missing local links for every client on a sudoku/mieru inbound and persists only what changed. Non-local links are kept, missing mieru credentials are filled in, and clients on no such inbound are left alone. The pass is idempotent, so a second start does nothing.
- The QR button in the AmneziaWG 2.0 device manager showed an empty picture for full AWG 2.0 configs. The QR renderer capped the config at 1500 bytes, but a config with the junk and init-packet fields (I1-I5, J1-J3, Itime) runs longer than that, so the renderer returned a JSON error under HTTP 200 that the `<img>` tag then tried to draw. The size cap is gone; the QR codec now decides the limit at Low error-correction (about 2953 bytes), and J1-J3 and Itime are written to the config, which were parsed but dropped before. When a config still exceeds the codec limit the endpoint returns a real HTTP error and the panel shows a message instead of a blank image.

This is a prerelease. A prerelease suffix sorts below the final release, so publish or install it explicitly from the tag.

Full release notes: [`docs/releases/v1.0.6-beta2.md`](../docs/releases/v1.0.6-beta2.md).
