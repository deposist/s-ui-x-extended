#!/usr/bin/env bash
set -euo pipefail

tag=$(bash "$(dirname "${BASH_SOURCE[0]}")/validate-release-tag.sh" "${1-}")
expected_commit=${2-}
event_commit=${3-}

if ! git show-ref --verify --quiet "refs/tags/$tag"; then
    printf 'release tag does not exist: %s\n' "$tag" >&2
    exit 1
fi
tag_commit=$(git rev-list -n 1 "refs/tags/$tag")
head_commit=$(git rev-parse HEAD)
if [[ $head_commit != "$tag_commit" ]]; then
    printf 'HEAD %s does not match release tag %s commit %s\n' "$head_commit" "$tag" "$tag_commit" >&2
    exit 1
fi
if [[ -n $expected_commit && $tag_commit != "$expected_commit" ]]; then
    printf 'release tag %s moved from %s to %s\n' "$tag" "$expected_commit" "$tag_commit" >&2
    exit 1
fi
if [[ -n $event_commit && $tag_commit != "$event_commit" ]]; then
    printf 'release tag %s commit %s does not match event commit %s\n' "$tag" "$tag_commit" "$event_commit" >&2
    exit 1
fi
version=$(git show "$tag_commit:config/version" | tr -d '\r\n')
if [[ $version != "${tag#v}" ]]; then
    printf 'release tag %s does not match config/version %s\n' "$tag" "$version" >&2
    exit 1
fi

printf 'tag=%s\ncommit=%s\nversion=%s\n' "$tag" "$tag_commit" "$version"
