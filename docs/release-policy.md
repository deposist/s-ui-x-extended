# Release Policy

Version source of truth:

- `config/version` contains the application release version.
- The value must be `MAJOR.MINOR.PATCH[-PRERELEASE]`.
- The value must not include a leading `v`.
- The value must not include build metadata.
- Prerelease identifiers must be lowercase SemVer identifiers.

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
