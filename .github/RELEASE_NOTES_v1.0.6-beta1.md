# S-UI-X Extended v1.0.6-beta1

Adds sudoku and mieru share links, makes the sudoku http-mask mode visible in the inbound editor, and ships a console menu for the AmneziaWG device key-encryption key.

- Sudoku and mieru now produce a share link and QR. Sudoku emits `sudoku://` (base64url of the short-link JSON, matching `SUDOKU-ASCII/sudoku`); mieru emits `mierus://` (the `enfein/mieru` simple URL). Both stay JSON-delivered, so the subscription path is unchanged and the link is an addition.
- Sudoku http-mask mode is now easy to find: the inbound fields sit in a titled card, the selector has a placeholder and a hint (empty means the core default, legacy), and the recommended preset fills it in. A regression test locks legacy/stream/poll/auto/ws into the client config.
- The raw-links tab no longer reads like a failure for JSON-delivered protocols; it explains that they are delivered through the subscription and points to the Sing-box tab.
- Web-panel self-update replaces the binary and restarts the service but does not run `install.sh`, so it never provisions `AWG_KEY_ENC`. A panel updated only through the web UI could still fail managed AmneziaWG device operations with `awg: AWG encryption key is unavailable`.
- Menu item 23 in `s-ui` is now "Generate env keys" with a submenu: session cookie key (`SUI_COOKIE_KEY`) and AmneziaWG device key (`AWG_KEY_ENC`). The AWG option generates the key when it is missing from `/etc/s-ui/secretbox.env`, wires up the systemd drop-in, and offers a restart. An existing key is kept unless a rotation is confirmed.
- Fixed `write_env_value` in the console script duplicating a line instead of replacing the value when an env key already existed (an `exit 0` inside the awk detector was overwritten by the trailing `END { exit 1 }`). This also affected rotating `SUI_COOKIE_KEY` from the menu.
- Added `scripts/setup-awg-key.sh` for the same check-and-generate as a standalone command.

This is a prerelease. A prerelease suffix sorts below the final release, so publish or install it explicitly from the tag.

Full release notes: [`docs/releases/v1.0.6-beta1.md`](../docs/releases/v1.0.6-beta1.md).
