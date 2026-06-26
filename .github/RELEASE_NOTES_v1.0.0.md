# s-ui-x-extended v1.0.0

First stable release of s-ui-x-extended. This release follows the `1.0.0-beta` line and ships the s-ui-x web panel on the `shtorm-7/sing-box-extended` core.

## What is included

- Panel support for the extended protocol set used by this fork, including Sudoku, TrustTunnel, MASQUE, OpenVPN, MTProxy, WireGuard/WARP endpoints, DHCP DNS transport, and the native `bond`, `block`, and core failover types.
- Panel-managed outbound groups and provider-backed group membership. Selector, URLTest, fallback, and panel failover include tag validation, previews, capability metadata, and provider health data.
- Failover runtime state and operational health reporting. The panel records outbound and provider probe results, publishes bounded realtime health payloads, and shows operational health on the Nexus overview page.
- Explicit failover all-down policies. The default keeps the current member. Direct fallback is available only when selected by an administrator.
- Type-level and option-level coverage for sing-box-extended options. The generated option coverage matrix has zero unexplained missing fields.
- Shared inbound advanced options, outbound `domain_strategy`, endpoint advanced fields, and protocol-specific fields exposed in TypeScript and UI where they are safe to edit.
- `/api/capabilities` reports build tags, inbound/outbound availability, groups, and providers. The frontend uses this endpoint to disable protocol types that are not compiled into the running binary.
- Release artifacts are built with the full supported protocol tag set. Linux and Windows artifacts include Sudoku, TrustTunnel, MASQUE, and OpenVPN, with Naive kept conditional on platforms that prepare cronet support.
- English and Russian configuration guides were added under `docs/`.

## Upgrade notes

No manual database migration is needed. The binary runs the normal migration chain on startup and updates `settings.version` to `1.0.0`.

Panel-managed failover switches new connections. Existing sessions can still break when the active member changes.

Native core failover is exposed as `core-failover` in the panel to keep it separate from the panel-managed `failover` group. During config assembly, the panel serializes it to the native core `type: failover` form.

Direct fallback remains opt-in. Existing failover groups keep the conservative all-down behavior unless an administrator changes the policy.

## Verification

The release branch was checked with:

- `go test ./...`
- `go vet ./...`
- `go test ./core -run 'OptionCoverage|Validate' -count=1`
- `go test ./core/capabilities -run ReleaseBuildTagsCoverManifestProtocolCapabilities -count=1`
- `node frontend/scripts/gen-capabilities.cjs --check`
- `cd frontend && npm run build`
- `cd frontend && npm run lint`
- `cd frontend && npm run test`
- `git diff --check`

The beta9 release candidate was also checked against real GitHub release artifacts:

- Linux amd64 runtime smoke on an Ubuntu GitHub Actions runner.
- Windows amd64 runtime smoke in a local isolated database folder.
- All Linux release checksums.
- Embedded build-tag metadata across Linux and Windows artifacts.
- `/api/capabilities` confirmed Sudoku, TrustTunnel, MASQUE, and OpenVPN as available in runtime smoke tests.

The `shtorm-7/sing-box-extended` dependency was not modified in this release.
