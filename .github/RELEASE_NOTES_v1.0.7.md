# S-UI-X Extended v1.0.7

This release fixes the Ubuntu management menu and makes domain cleanup disable the TLS settings used by the panel and subscriptions.

- Keeps the management menu open while the service is starting, stopped, or missing.
- Clears panel and subscription certificate paths through menu item 11 without deleting certificate files or IP-certificate settings.
- Removes the reset control from automatically managed Sudoku keys, hides revealed keys when switching clients, and localizes the Split Private Key label.
- Removes the obsolete manual split-key guide.

No database migration is required.

Full release notes: [`docs/releases/v1.0.7.md`](../docs/releases/v1.0.7.md).
