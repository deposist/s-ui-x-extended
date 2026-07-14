# S-UI-X Extended v1.0.2-beta1

- Added managed personal AmneziaWG 2.0 devices with Telegram configuration and QR delivery, encrypted keys, device limits, traffic statistics, key rotation, revocation, and crash recovery.
- Protected managed endpoint peers from manual edits and kept preshared keys out of endpoint API responses and stored endpoint options.
- Included paid-subscription tables in database backup and restore.
- Simplified RU and ZH routing and DNS presets. They now use one direct outbound and preserve custom rules.
- Fixed Edit Bulk so an inbound can be added to all clients.
- Fixed `s-ui` waiting for terminal input when a saved language exists.
- Added HTTPS-only release downloads with retries, timeouts, and SHA-256 verification.

Database migrations run automatically. Managed AmneziaWG devices require `SUI_SECRETBOX_KEY` and a configured AWG 2.0 endpoint.
