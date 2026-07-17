# S-UI-X Extended v1.0.6

Version 1.0.6 moves the Sudoku and mieru delivery changes from the beta series into the stable channel and fixes QR delivery for full AmneziaWG 2.0 configs.

- Creating a Sudoku inbound without a key now generates a canonical master key on the backend. The panel keeps the private scalar out of sing-box configs, links, QR codes, and subscriptions.
- Each client/inbound pair gets its own Sudoku Split Private Key. Assignment creates the key; master-key rotation replaces it; removing an inbound cleans it up. Delivery selects the correct key by inbound ID.
- Legacy Sudoku PSK/public-key inbounds keep their shared-key fallback. Client keys saved by beta3 migrate on the next save.
- The panel emits `sudoku://` and `mierus://` links, repairs missing links from older assignments, and provides a manual rebuild action.
- Full AmneziaWG 2.0 configs now use the QR renderer's actual capacity. J1-J3 and Itime are included, and failures direct the operator to the `.conf` download instead of leaving an empty image.
- The console menu can provision `AWG_KEY_ENC` after a web update without overwriting an existing key.

No database migration is required.

Full release notes: [`docs/releases/v1.0.6.md`](../docs/releases/v1.0.6.md).
