# s-ui-x-extended v1.0.0-beta9

Pre-release. This release follows v1.0.0-beta8 and fixes the official release builds. The beta8 source already had panel support for the extended protocols, but some GitHub release artifacts were built without the optional protocol tags, so the UI correctly marked those protocols as unavailable.

## Changes since beta8

- Updated Linux release builds to include the full supported protocol tag set, including `with_sudoku`, `with_trusttunnel`, `with_masque`, `with_openvpn`, `with_mtproxy`, `with_wireguard`, and related extended tags.
- Updated Windows release builds and local Windows build scripts to use the same supported protocol tag set.
- Added a regression test that checks release workflows, local build scripts, `build.sh`, and the Dockerfile against the protocol tags declared in `core/capabilities/protocols.json`.
- Kept `with_naive_outbound` conditional on Linux release platforms that prepare the cronet/naive toolchain.
- Kept the core dependency unchanged. No files from `shtorm-7/sing-box-extended` were modified.

## Operator notes

No manual database migration is needed. The normal startup migration chain updates `settings.version` to `1.0.0-beta9`.

The main reason to upgrade from beta8 is the official binary build. In beta9, release artifacts should report Sudoku, TrustTunnel, MASQUE, and OpenVPN as available when `/api/capabilities` is queried.

See `CHANGELOG-EN.md`, `CHANGELOG-RU.md`, and `CHANGELOG-ZH.md` for the full history. This is a pre-release. Review `SECURITY.md` before exposing the panel.
