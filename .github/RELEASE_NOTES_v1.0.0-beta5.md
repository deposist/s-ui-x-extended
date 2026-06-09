# s-ui-x-extended v1.0.0-beta5

Pre-release. Unifies the navigation menu across the two UI environments. The
**Classic** and **Nexus** shells previously kept independent, hand-maintained
menu lists that had drifted apart — *Providers* showed only in Classic, *Paid
Subscriptions* only in Nexus. Both shells now read a single shared source, so the
tabs stay in sync and neither environment can silently lose one again.

## Highlights

- **One menu, both environments.** New `frontend/src/layouts/menu.ts` is the
  single source of truth; the Classic drawer and the Nexus sidebar both import it.
  `nexusMenu.ts` becomes a thin backward-compatible re-export, and the nexus-only
  `singBoxSettings` metadata now lives in the shared list.
- **Tabs reconciled.** Classic and Nexus now expose the same 16 tabs in the same
  order — Classic gains *Paid Subscriptions*, Nexus gains *Providers*.
  `/migrate-xui` stays a contextual page reached from the Backup dialog, not a
  top-level tab.
- **Localization parity.** `Providers` / `Paid Subscriptions` are now translated
  in zh-Hans, zh-Hant, Persian and Vietnamese instead of falling back to English.
- **Drift-guard test** pins the menu's paths, order, uniqueness and the nexus
  sing-box surfaces so the two environments can't diverge again unnoticed.

See [`CHANGELOG.md`](CHANGELOG.md) for the full bilingual list and
[`SECURITY.md`](SECURITY.md) for supply-chain and hardening notes. This is a
pre-release — review `SECURITY.md` before exposing the panel.
