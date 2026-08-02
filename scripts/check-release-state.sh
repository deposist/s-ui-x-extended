#!/usr/bin/env bash
set -euo pipefail

repository=${1-}
tag=${2-}

if [[ ! $repository =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
    printf 'invalid GitHub repository: %q\n' "$repository" >&2
    exit 1
fi
tag=$(bash "$(dirname "${BASH_SOURCE[0]}")/validate-release-tag.sh" "$tag")
: "${GH_TOKEN:?GH_TOKEN is required}"
command -v gh >/dev/null
command -v jq >/dev/null

release=$(gh api --paginate --slurp "repos/$repository/releases?per_page=100" |
    jq -cr --arg tag "$tag" '
        add
        | map(select(.tag_name == $tag))
        | if length == 0 then null
          elif length == 1 then .[0]
          else error("expected at most one release for " + $tag)
          end
    ')
if [[ $release == null ]]; then
    printf 'release state is clear for %s\n' "$tag"
    exit 0
fi

if [[ $(jq -er '.draft | type == "boolean" and .' <<<"$release") == true ]]; then
    printf 'draft release is recoverable for %s\n' "$tag"
    exit 0
fi

printf 'release %s is already published; use a new SemVer tag for a different build\n' "$tag" >&2
exit 1
