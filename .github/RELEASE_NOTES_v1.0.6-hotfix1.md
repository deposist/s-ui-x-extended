# S-UI-X Extended v1.0.6-hotfix1

This hotfix repairs the Sudoku section in the client editor.

- Removes the stale beta3 instruction for manual CLI key generation.
- Shows the generated Split Private Key for each assigned Sudoku inbound.
- Labels keys by inbound, keeps them hidden by default, and allows reveal/copy when needed.
- Leaves the separate editable `mtproxy` Secret field unchanged.

No database migration is required. Existing Sudoku keys and delivery output are unchanged.

Full release notes: [`docs/releases/v1.0.6-hotfix1.md`](../docs/releases/v1.0.6-hotfix1.md).
