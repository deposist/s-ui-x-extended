# S-UI-X Extended v1.0.4

Bugfix release for managed AmneziaWG endpoints.

- Fixed: saving any endpoint could stop sing-box from starting with `endpoints[N].awgManaged: json: unknown field "awgManaged"`. The panel-only marker leaked into the stored endpoint options and from there into the core config. It is now stripped on save and at config render, so already-affected installations recover without a migration.
- Fixed: a managed AWG endpoint could be saved with a `/32` host address, after which every reconcile failed with `managed AWG endpoint address does not match awgSubnet`. Saving now requires an IPv4 subnet of `/30` or larger, a host address that is not the network or broadcast address, and a set `listen_port`.
- The endpoint form widens the generated default `/32` to `/24` when managed mode is turned on, and validates public endpoint, DNS, and address before sending the save request.

If you already have a managed endpoint with a `/32` address, edit it once and widen the subnet (for example `10.0.0.20/32` to `10.0.0.20/24`).

Full release notes: [`docs/releases/v1.0.4.md`](../docs/releases/v1.0.4.md).
