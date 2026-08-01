#!/usr/bin/env bash
set -euo pipefail

mode=tag
if [[ ${1-} == --prerelease ]]; then
    mode=prerelease
    shift
fi
tag=${1-}
core='(0|[1-9][0-9]*)'
prerelease_id='((0|[1-9][0-9]*)|[0-9]*[a-z-][0-9a-z-]*)'
release_tag_regex="^v${core}\\.${core}\\.${core}(-${prerelease_id}(\\.${prerelease_id})*)?$"

if [[ ! $tag =~ $release_tag_regex ]]; then
    printf 'invalid release tag: %q\n' "$tag" >&2
    exit 1
fi

if [[ $mode == prerelease ]]; then
    if [[ $tag == *-* ]]; then
        printf 'true\n'
    else
        printf 'false\n'
    fi
    exit 0
fi

printf '%s\n' "$tag"
