# Release workflow runbook

Use this checklist before creating any release tag. A tag starts the publication workflow, so failures found after the push already count as release failures.

## Mandatory pre-tag gate

Run from the repository root:

```sh
bash tests/release-tag-validation.sh
actionlint
shellcheck -e SC1003,SC1007 \
  install.sh s-ui.sh tests/release-tag-validation.sh \
  scripts/check-release-assets.sh scripts/check-release-state.sh \
  scripts/validate-release-tag.sh scripts/verify-release-tag-checkout.sh
go test ./... -count=1
go vet ./...
golangci-lint run
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/gosec-packages.ps1
govulncheck ./...
```

Run the frontend gate from `frontend`:

```sh
npm ci
npm run lint -- --max-warnings=0
npm run test
npm run build
npm run verify:dist
```

Before pushing the tag, confirm:

- `config/version`, frontend package metadata, Compose image, README, changelogs, and release notes contain the same version.
- `.github/RELEASE_NOTES_<tag>.md` exists.
- The local tag does not exist and GitHub has neither a tag nor a published release with that name.
- `git diff --cached --check` passes and the staged file list contains no unrelated files.
- Git author and committer are `deposist <235344042+deposist@users.noreply.github.com>`.

## Reliability rules

- Invoke tracked shell scripts through `bash`. Git stores these scripts without the executable bit.
- Stage trusted tag and release-state validators before checking out the requested tag. Code from the tag must not replace the guard that validates it.
- Treat published tags and assets as immutable. A different build needs a new SemVer tag.
- Permit a missing release or one draft release. Reject duplicate releases and published-tag reuse.
- Retry GitHub draft lookup for bounded API visibility delays before reconciling assets.
- Resolve each Docker platform from the OCI index and smoke-test its immutable platform digest. Do not load two platforms through one local index digest.
- Keep setup-go's exact executable when privileged commands are required. Do not let `sudo` select a different system Go through `secure_path`.
- Pin external toolchains and key material by URL and SHA-256. Never obtain release signing keys from a public keyserver during a release job.
- Run source security gates before building release artifacts. Gosec suppressions require a narrow rule number and a path-trust justification.
- Check Windows archives with native process exit handling and verify both runtime architecture and reported application version.

## Incident log

### 2026-08-02: incomplete Debian keyring in cronet builds

The first `v1.0.8` workflow attempt failed for `linux/arm64` and `linux/386`. Each matrix job deleted Chromium's vendored `keyring.gpg` and ran `generate_keyring.sh`, which downloaded Debian keys from `keyserver.ubuntu.com`. Some jobs received an incomplete key set. `gpgv` then failed with `Can't check signature: No public key`. A retry succeeded, which confirmed that the release source was unchanged and the key lookup was nondeterministic.

The workflow now downloads the official pinned `debian-archive-keyring` package from `deb.debian.org`, verifies its SHA-256 digest, extracts it with `dpkg-deb`, and installs the packaged keyring into the cronet sysroot tooling. The regression test checks the exact URL, digest, extraction command, and absence of `generate_keyring.sh`.

### Earlier failures covered by the gate

- Direct execution of tracked scripts failed with `Permission denied`. The workflows and regression tests require explicit Bash invocation.
- Validators read from the checked-out release tag could be stale or replaced. The checkout action stages trusted validators before changing refs.
- A newly created draft was temporarily absent from GitHub API responses. Release reconciliation now uses a bounded visibility retry.
- Docker tried to load two platform variants through the same multi-platform digest. Smoke tests now use platform manifest digests.
- Source checks passed locally but Gosec found an unsuppressed fixed-path file read in CI. The local gate now runs the same package scanner script as CI.

When a new release failure occurs, record the exact failing step, root cause, permanent invariant, and regression check here before the next tag.