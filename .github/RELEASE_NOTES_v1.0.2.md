# S-UI-X Extended v1.0.2

Stable v1.0.2 promotes the v1.0.2-beta1 line and reworks how managed AmneziaWG 2.0 servers are configured.

- A WireGuard endpoint can be marked as a managed AmneziaWG 2.0 server in the endpoint form: public address, DNS, and a default device limit. The AWG 2.0 profile is filled in automatically.
- Managed servers are assigned to clients next to inbounds. Removing the assignment revokes the client's devices; an endpoint with assignments cannot be deleted.
- Devices belong to a client and server pair, so several managed servers can run at once with isolated peers, IP allocation, reconciliation, and statistics.
- The client form has a devices tab: create a device, show the QR code, download the `.conf` file, rotate keys, revoke. Actions are audited and config downloads are not cached.
- The Telegram bot lists assigned servers and creates devices on the server the user picks. The old global configuration keeps working.

Database migrations run automatically. Device key encryption still requires `AWG_KEY_ENC`. Full notes: `docs/releases/v1.0.2.md`.
