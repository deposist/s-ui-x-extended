# Upstream sync workflow

This document describes how to bring changes from `deposist/s-ui-x` into `deposist/s-ui-x-extended` without losing the extended core, branding, or UI coverage.

The intended model is:

```text
Git brings the upstream diff.
A maintainer or coding agent adapts conflicts.
CI proves that the extended fork still works.
```

Do not copy the whole upstream tree over this repository.

## Repository roles

- `origin`: `https://github.com/deposist/s-ui-x-extended.git`
- `upstream-s-ui-x`: `https://github.com/deposist/s-ui-x.git`
- main working branch: `s-ui-x-extended`

The local directory `C:/s-ui-x` may be a worktree and must not be used as the source of truth for sync. Use the `upstream-s-ui-x` remote instead.

## One-time setup

Run from `C:/s-ui-x-ext`.

```bash
git checkout s-ui-x-extended
git status
```

The working tree must be clean before starting.

Add the upstream remote:

```bash
git remote add upstream-s-ui-x https://github.com/deposist/s-ui-x.git
git fetch upstream-s-ui-x --tags
```

If the remote already exists, update it:

```bash
git remote set-url upstream-s-ui-x https://github.com/deposist/s-ui-x.git
git fetch upstream-s-ui-x --tags
```

Verify that the upstream base tag is available locally:

```bash
git rev-parse --verify v1.5.10-beta3^{commit}
```

Create a backup branch before changing topology:

```bash
git branch backup/before-upstream-sync-foundation
```

Record the current upstream base without changing the working tree:

```bash
git merge -s ours --allow-unrelated-histories v1.5.10-beta3 \
  -m "chore: record upstream s-ui-x v1.5.10-beta3 base"
```

This merge commit is intentional. It tells Git that `s-ui-x-extended` already contains the upstream `v1.5.10-beta3` tree. The merge must not change files.

Verify it:

```bash
git diff HEAD^1 HEAD --stat
git merge-base --is-ancestor v1.5.10-beta3 HEAD
echo $?
```

Expected results:

- `git diff HEAD^1 HEAD --stat` prints nothing.
- `echo $?` prints `0` after the ancestor check.

Enable conflict-resolution reuse:

```bash
git config rerere.enabled true
git config rerere.autoupdate true
```

Verify:

```bash
git config --get rerere.enabled
git config --get rerere.autoupdate
```

Both should print `true`.

Push the foundation commit:

```bash
git push origin s-ui-x-extended
```

## Sync a new upstream release

Use this when `deposist/s-ui-x` publishes a new tag, for example `v1.5.11`.

```bash
cd C:/s-ui-x-ext
git fetch upstream-s-ui-x --tags
git checkout s-ui-x-extended
git pull origin s-ui-x-extended
git checkout -b sync/upstream-v1.5.11
git merge --no-ff v1.5.11
```

If there are conflicts, resolve them manually or with a coding agent. Do not use blind `ours` or `theirs` resolution for large files.

After conflicts are resolved:

```bash
git status
git add <resolved-files>
git commit
```

## Port one upstream commit

Use cherry-pick only for small, isolated upstream fixes.

```bash
cd C:/s-ui-x-ext
git fetch upstream-s-ui-x
git checkout s-ui-x-extended
git pull origin s-ui-x-extended
git checkout -b port/upstream-<short-sha>
git cherry-pick <upstream-commit-sha>
```

If there are conflicts:

```bash
git status
# resolve conflicts
git add <resolved-files>
git cherry-pick --continue
```

Do not use cherry-pick as the normal release-sync mechanism. A later full upstream merge may see similar changes again because cherry-picked commits have different SHAs.

## Conflict rules

Always preserve these fork-specific decisions:

- Go module path: `github.com/deposist/s-ui-x-extended`.
- `go.mod` replace directives for `shtorm-7` and `sing-box-extended` dependencies.
- `sing-tun` pin required by the extended core.
- Branding: `S-UI-X Extended`.
- Repository URLs: `deposist/s-ui-x-extended`.
- Container image: `ghcr.io/deposist/s-ui-x-extended`.
- Update API URL: `https://api.github.com/repos/deposist/s-ui-x-extended`.
- Download URL: `https://github.com/deposist/s-ui-x-extended/releases/download`.
- Version line: `v1.0.0-betaN` until the extended fork publishes a stable release.
- Classic and Nexus UI support for extended protocols.

Do not accept upstream blindly in these files:

```text
go.mod
go.sum
core/register.go
core/box.go
core/main.go
core/log.go
core/validate.go
service/config.go
api/apiHandler.go
api/apiService.go
api/apiV2Handler.go
database/db.go
database/backup.go
frontend/src/types/*
frontend/src/layouts/modals/*
frontend/src/components/nexus/drawers/*
frontend/src/components/protocols/*
README.md
.github/workflows/*
Dockerfile
build.sh
install.sh
s-ui.sh
```

These files can be changed by upstream, but every conflict must be reviewed with the extended behavior in mind.

## Required checks after sync

Run frontend checks:

```bash
cd C:/s-ui-x-ext/frontend
npm run lint
npm run build
npx vitest run
```

Refresh embedded frontend assets for local Go tests:

```bash
cd C:/s-ui-x-ext
rm -rf web/html
cp -r frontend/dist web/html
```

Run backend checks:

```bash
cd C:/s-ui-x-ext
go build ./...
go vet ./...
go test ./...
go test -race -timeout=10m ./...
```

`web/html` is ignored by Git. It is only needed locally so the embedded-asset tests can run after a frontend build.

## URL and identity checks

Verify the module path:

```bash
grep -n "^module " go.mod
```

Expected:

```text
module github.com/deposist/s-ui-x-extended
```

Verify the update repository:

```bash
grep -R "api.github.com/repos/deposist/s-ui" -n service api config frontend/src
```

Expected result should only reference:

```text
https://api.github.com/repos/deposist/s-ui-x-extended
```

Verify old install/download/container URLs did not return:

```bash
grep -R "deposist/s-ui" -n README.md install.sh s-ui.sh Dockerfile .github/workflows service frontend/src \
  | grep -v "s-ui-x-extended" \
  | grep -v "upstream s-ui-x" || true
```

The command should not print active install, update, or container coordinates for `deposist/s-ui-x`.

## Extended protocol coverage checklist

After a large upstream sync, check that these features are still available in backend and UI:

```text
Mieru
Sudoku
TrustTunnel
SSH inbound
MTProxy
MASQUE
OpenVPN
Bond
Failover
Fallback
Bandwidth limiter
Connection limiter
Traffic limiter
Rate limiter
VPN server endpoint
VPN client endpoint
Providers
Profiler
SDNS
DNS Fallback
DNS Resolved
VLESS encryption
mKCP
XHTTP
unified_delay
```

Backend files to inspect when conflicts touch protocols or config generation:

```text
core/register.go
core/capabilities/protocols.json
service/config.go
util/genLink.go
util/outJson.go
```

Classic UI files to inspect:

```text
frontend/src/layouts/modals/Inbound.vue
frontend/src/layouts/modals/Outbound.vue
frontend/src/layouts/modals/Endpoint.vue
frontend/src/layouts/modals/Dns.vue
frontend/src/layouts/modals/Service.vue
frontend/src/layouts/modals/Client.vue
frontend/src/layouts/modals/ClientAddBulk.vue
frontend/src/layouts/modals/ClientEditBulk.vue
```

Nexus UI files to inspect:

```text
frontend/src/components/nexus/drawers/InboundDrawer.vue
frontend/src/components/nexus/drawers/OutboundDrawer.vue
frontend/src/components/nexus/drawers/ServiceDrawer.vue
```

Shared endpoint and DNS dialogs may be used by both UI modes. Check the actual call path before assuming a Nexus-specific drawer exists.

## Versioning after sync

Do not copy the upstream `s-ui-x` version into `config/version`.

The extended fork uses its own release line. For example, if upstream publishes `s-ui-x v1.5.11`, the extended release can be:

```text
s-ui-x-extended v1.0.0-beta8
```

Update these files for an extended release:

```text
config/version
README.md
CHANGELOG-EN.md
CHANGELOG-RU.md
CHANGELOG-ZH.md
docs/releases/v1.0.0-betaN.md
.github/RELEASE_NOTES_v1.0.0-betaN.md
```

Release notes should state both facts:

```text
Includes upstream s-ui-x vX.Y.Z changes.
Runs on the sing-box-extended core.
```

## Recommended coding-agent prompt

When a merge or cherry-pick conflicts, give the agent a bounded task:

```text
Repository: C:/s-ui-x-ext
Branch: <sync branch>
Source: upstream-s-ui-x/<tag-or-commit>

Resolve the current merge/cherry-pick conflicts.

Must preserve:
- module github.com/deposist/s-ui-x-extended
- shtorm-7/sing-box-extended replace directives
- all extended protocols and providers
- Classic and Nexus protocol editors
- S-UI-X Extended branding
- update/install/release URLs for deposist/s-ui-x-extended
- extended version line v1.0.0-betaN

Do not modify C:/s-ui-x.
Run relevant tests after each code change and the full required checks before finishing.
```

## What not to do

- Do not copy `C:/s-ui-x` over `C:/s-ui-x-ext`.
- Do not use blind `git checkout --theirs` or `git checkout --ours` on large conflict files.
- Do not merge upstream and extended feature work in the same branch.
- Do not publish a release until frontend, backend, race, and extended protocol checks pass.
- Do not restore old `deposist/s-ui-x` install, update, or container coordinates.

## Branch naming

Use separate branches for separate work:

```text
sync/upstream-vX.Y.Z
port/upstream-<short-sha>
feature/<extended-feature>
release/v1.0.0-betaN
```

Keeping upstream sync separate from feature work makes conflicts easier to review and rollback.
