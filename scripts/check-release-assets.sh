#!/usr/bin/env bash
set -euo pipefail

repository=${1-}
tag=${2-}
asset_dir=${3-}

if [[ ! $repository =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
    printf 'invalid GitHub repository: %q\n' "$repository" >&2
    exit 1
fi
tag=$("$(dirname "${BASH_SOURCE[0]}")/validate-release-tag.sh" "$tag")
if [[ ! -d $asset_dir ]]; then
    printf 'release asset directory does not exist: %q\n' "$asset_dir" >&2
    exit 1
fi
: "${GH_TOKEN:?GH_TOKEN is required}"

owner=${repository%%/*}
name=${repository#*/}
# GraphQL variables are intentionally literal.
# shellcheck disable=SC2016
query='query($owner:String!, $name:String!, $tag:String!) {
  repository(owner:$owner, name:$name) {
    release(tagName:$tag) {
      releaseAssets(first:100) {
        nodes { name }
        pageInfo { hasNextPage }
      }
    }
  }
}'
existing_assets=$(gh api graphql \
    -f query="$query" \
    -F owner="$owner" \
    -F name="$name" \
    -F tag="$tag" \
    --jq 'if (.data.repository.release.releaseAssets.pageInfo.hasNextPage // false) then error("release has more than 100 assets; refusing incomplete collision check") else .data.repository.release.releaseAssets.nodes[]?.name end')

shopt -s nullglob
assets=("$asset_dir"/*)
if ((${#assets[@]} == 0)); then
    printf 'release asset directory is empty: %q\n' "$asset_dir" >&2
    exit 1
fi
for asset in "${assets[@]}"; do
    asset_name=${asset##*/}
    if grep -Fqx -- "$asset_name" <<<"$existing_assets"; then
        printf 'release asset already exists and cannot be overwritten: %s\n' "$asset_name" >&2
        exit 1
    fi
done
