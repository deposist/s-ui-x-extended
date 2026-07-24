# Release Policy

Version source of truth:

- `config/version` contains the application release version.
- The value must be `MAJOR.MINOR.PATCH[-PRERELEASE]`.
- The value must not include a leading `v`.
- The value must not include build metadata.
- Prerelease identifiers must be lowercase SemVer identifiers.

Hotfix precedence:

- A release patched after the fact uses a single `hotfixN` prerelease, for
  example `1.0.4-hotfix1`. Unlike plain SemVer, a `hotfixN` prerelease ranks
  ABOVE its base release (`1.0.4-hotfix1` is newer than `1.0.4`), so
  auto-update offers it to installs on the base release. Higher `N` is newer.
- A later patch still wins: `1.0.5` is newer than `1.0.4-hotfix1`.
- This rule is limited to the lone `hotfixN` form. Beta and rc lines keep
  normal SemVer precedence and stay below the release; a `beta-hotfixN` line is
  a beta prerelease and is never published as a release.

Git release tags:

- Git tag names use `v` plus the exact `config/version` value.
- Example: `config/version` = `1.5.2-beta-hotfix2`, Git tag =
  `v1.5.2-beta-hotfix2`.

Database version policy:

- `settings.version` records the newest application version that successfully
  migrated or adapted the database.
- `settings.version` must never be downgraded by an older binary.
- Legacy database values with only `MAJOR.MINOR` are accepted for comparison
  and treated as `MAJOR.MINOR.0`.

Release checklist:

- Update `config/version`.
- Add `Unreleased` changelog entries before cutting the tag, then move them
  under the release heading.
- Add or update migrations when the schema changes.
- Run `go test ./config ./database ./service`.
- Run the full validation gate before publishing artifacts.

Signed self-update manifest (AUD-02):

- Every `s-ui-linux-<platform>.tar.gz` release asset must include its adjacent
  `.sha256`, `.manifest.json`, and `.manifest.json.sig` assets. Unsigned
  releases are intentionally not offered for self-update.
- The manifest is compact UTF-8 JSON with exactly the update binding fields:
  `version`, `channel` (`main` or `beta`), `platform`, `filename`, and the
  archive `sha256`. The `.sig` file is a raw Ed25519 signature over the exact
  manifest bytes, not a signature over re-serialized JSON.
- Before publishing, provision the repository secret
  `RELEASE_MANIFEST_ED25519_PRIVATE_KEY` with the base64-encoded PEM private
  key matching the Ed25519 public key pinned in
  `service/panel_update_apply.go`. Never put that private key in a repository,
  release asset, workflow log, or release notes. Public-key rotation requires a
  separately trusted application release that changes the pinned key first.
