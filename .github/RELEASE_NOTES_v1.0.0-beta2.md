# s-ui-x-extended v1.0.0-beta2

Second public beta — the s-ui-x web panel on the `sing-box-extended` core
(`shtorm-7/sing-box-extended`, a fork of `SagerNet/sing-box`).

## Highlights

- **Full protocol tag set in every build path.** Prebuilt Linux tarballs and the
  Windows packages now include the same protocols as the Docker image / `build.sh`:
  WireGuard/WARP, MASQUE, OpenVPN, MTProxy, Sudoku, TrustTunnel, DHCP-DNS and
  CCM/OCM/OOMKiller. Previously these were compiled out of the prebuilt binaries and
  returned a `not included in this build, rebuild with -tags ...` error at runtime.
  Only **Naive** (cronet/CGO) still varies by platform — present on Linux
  amd64/arm64/armv7/armv6/386, Docker, and Windows amd64; absent on armv5/s390x and
  Windows arm64.
- **Fix — `database` token scope.** API tokens scoped to `database` now grant database
  export/import (`getdb`/`importdb`) and x-ui / 3x-ui migration (`import-xui`). These
  were unintentionally admin-only before.
- **Docs.** `docs/scope-matrix.md` corrected to all six token scopes; the README was
  restructured and fact-checked (HTTP API, migration, backup, Telegram, paid
  subscriptions, security & hardening, monitoring, transports/TLS, build matrix).

See [`CHANGELOG.md`](CHANGELOG.md) for the full list and [`SECURITY.md`](SECURITY.md)
for supply-chain and hardening notes.

This is a beta build — review `SECURITY.md` before exposing the panel.
