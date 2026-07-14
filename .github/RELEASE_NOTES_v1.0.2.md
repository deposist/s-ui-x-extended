# S-UI-X-Extended v1.0.2

- Added managed personal AmneziaWG 2.0 devices with Telegram config and QR delivery, traffic accounting, rotation, revocation, and crash recovery.
- Encrypted device keys at rest and kept preshared keys out of endpoint APIs and stored endpoint options.
- Added global and per-tariff device limits plus admin status and managed-peer edit protection.
- Fixed `s-ui` waiting for terminal input before showing the menu when a language is already saved.
- Hardened release downloads with curl retries, timeouts, HTTPS-only redirects, and SHA-256 verification.

Database migrations run automatically. Managed AmneziaWG devices require `SUI_SECRETBOX_KEY` and a configured AWG 2.0 endpoint.
