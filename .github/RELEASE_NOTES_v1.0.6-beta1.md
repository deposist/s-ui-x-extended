# S-UI-X Extended v1.0.6-beta1

Adds sudoku and mieru share links, makes the sudoku http-mask mode visible in the inbound editor, and ships a console menu for the AmneziaWG device key-encryption key.

- Sudoku and mieru now produce a share link and QR. Sudoku emits `sudoku://` (base64url of the short-link JSON, matching `SUDOKU-ASCII/sudoku`); mieru emits `mierus://` (the `enfein/mieru` simple URL). Both stay JSON-delivered, so the subscription path is unchanged and the link is an addition.
- Sudoku http-mask mode is now easy to find: the inbound fields sit in a titled card, the selector has a placeholder and a hint (empty means the core default, legacy), and the recommended preset fills it in. A regression test locks legacy/stream/poll/auto/ws into the client config.
- The raw-links tab no longer reads like a failure for JSON-delivered protocols; it explains that they are delivered through the subscription and points to the Sing-box tab.
- Console menu item 23 in `s-ui` ("Generate env keys") generates the AmneziaWG device key (`AWG_KEY_ENC`) when it is missing from `/etc/s-ui/secretbox.env`, wires up the systemd drop-in, and offers a restart. Also fixed `write_env_value` appending a duplicate env line instead of replacing it, and added `scripts/setup-awg-key.sh`.

This is a prerelease. A prerelease suffix sorts below the final release, so publish or install it explicitly from the tag.

Full release notes: [`docs/releases/v1.0.6-beta1.md`](../docs/releases/v1.0.6-beta1.md).
