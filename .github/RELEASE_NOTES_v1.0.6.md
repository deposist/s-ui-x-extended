# S-UI-X Extended v1.0.6

Adds a console menu entry for the AmneziaWG device key-encryption key.

- Web-panel self-update replaces the binary and restarts the service but does not run `install.sh`, so it never provisions `AWG_KEY_ENC`. A panel updated only through the web UI could still fail managed AmneziaWG device operations with `awg: AWG encryption key is unavailable`.
- Menu item 23 in `s-ui` is now "Generate env keys" with a submenu: session cookie key (`SUI_COOKIE_KEY`) and AmneziaWG device key (`AWG_KEY_ENC`). The AWG option generates the key when it is missing from `/etc/s-ui/secretbox.env`, wires up the systemd drop-in, and offers a restart. An existing key is kept unless a rotation is confirmed.
- Fixed `write_env_value` in the console script duplicating a line instead of replacing the value when an env key already existed (an `exit 0` inside the awk detector was overwritten by the trailing `END { exit 1 }`). This also affected rotating `SUI_COOKIE_KEY` from the menu.
- Added `scripts/setup-awg-key.sh` for the same check-and-generate as a standalone command.

Full release notes: [`docs/releases/v1.0.6.md`](../docs/releases/v1.0.6.md).
