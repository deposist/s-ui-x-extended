#!/usr/bin/env bash
set -euo pipefail

tag=${1-}
core='(0|[1-9][0-9]*)'
prerelease_id='((0|[1-9][0-9]*)|[0-9]*[a-z-][0-9a-z-]*)'
release_tag_regex="^v${core}\\.${core}\\.${core}(-${prerelease_id}(\\.${prerelease_id})*)?$"

if [[ ! $tag =~ $release_tag_regex ]]; then
    printf 'invalid release tag: %q\n' "$tag" >&2
    exit 1
fi

printf '%s\n' "$tag"
