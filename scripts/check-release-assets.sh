#!/usr/bin/env bash
set -euo pipefail

mode=reconcile
if [[ ${1-} == --verify ]]; then
    mode=verify
    shift
fi

repository=${1-}
tag=${2-}
asset_dir=${3-}

if [[ ! $repository =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
    printf 'invalid GitHub repository: %q\n' "$repository" >&2
    exit 1
fi
tag=$(bash "$(dirname "${BASH_SOURCE[0]}")/validate-release-tag.sh" "$tag")
if [[ ! -d $asset_dir ]]; then
    printf 'release asset directory does not exist: %q\n' "$asset_dir" >&2
    exit 1
fi
: "${GH_TOKEN:?GH_TOKEN is required}"
command -v gh >/dev/null
command -v jq >/dev/null
command -v sha256sum >/dev/null

declare -A local_path=()
declare -A local_digest=()
while IFS= read -r -d '' asset; do
    if [[ ! -f $asset || -L $asset ]]; then
        printf 'release asset is not a regular file: %q\n' "$asset" >&2
        exit 1
    fi
    asset_name=${asset##*/}
    if [[ -z $asset_name || $asset_name == *$'\n'* || $asset_name == *$'\t'* ]]; then
        printf 'invalid release asset name: %q\n' "$asset_name" >&2
        exit 1
    fi
    if [[ -n ${local_path[$asset_name]+x} ]]; then
        printf 'duplicate local release asset name: %s\n' "$asset_name" >&2
        exit 1
    fi
    digest=$(sha256sum "$asset" | awk '{print $1}')
    [[ $digest =~ ^[0-9a-f]{64}$ ]] || { echo "failed to hash $asset_name" >&2; exit 1; }
    local_path[$asset_name]=$asset
    local_digest[$asset_name]="sha256:$digest"
done < <(find "$asset_dir" -maxdepth 1 -mindepth 1 -print0 | sort -z)

if ((${#local_path[@]} == 0)); then
    printf 'release asset directory is empty: %q\n' "$asset_dir" >&2
    exit 1
fi

release_id=$(gh api --paginate --slurp "repos/$repository/releases?per_page=100" |
    jq -cer --arg tag "$tag" 'add | map(select(.tag_name == $tag)) | if length == 1 then .[0].id else error("expected exactly one release for " + $tag) end')
[[ $release_id =~ ^[0-9]+$ ]] || { echo "release for $tag has no valid id" >&2; exit 1; }

fetch_remote_assets() {
    gh api --paginate --slurp "repos/$repository/releases/$release_id/assets?per_page=100" | jq -ce 'add'
}

wait_for_remote_assets() {
    local remote_json attempt
    for attempt in {1..10}; do
        remote_json=$(fetch_remote_assets)
        if jq -e 'all(.[]; .state == "uploaded" and (.digest // "" | test("^sha256:[0-9a-f]{64}$")))' <<<"$remote_json" >/dev/null; then
            printf '%s\n' "$remote_json"
            return 0
        fi
        sleep 2
    done
    echo 'GitHub did not publish authoritative SHA-256 values for all release assets' >&2
    return 1
}

check_remote_assets() {
    local require_exact=$1 remote_json=$2 row name digest state
    declare -A seen=()
    while IFS= read -r row; do
        name=$(jq -r '.name' <<<"$row")
        digest=$(jq -r '.digest // ""' <<<"$row")
        state=$(jq -r '.state' <<<"$row")
        if [[ -z $name || $name == *$'\n'* || $name == *$'\t'* || -n ${seen[$name]+x} ]]; then
            printf 'invalid or duplicate remote release asset name: %q\n' "$name" >&2
            return 1
        fi
        seen[$name]=1
        if [[ -z ${local_path[$name]+x} ]]; then
            printf 'unexpected remote release asset: %s\n' "$name" >&2
            return 1
        fi
        if [[ $state != uploaded || ! $digest =~ ^sha256:[0-9a-f]{64}$ ]]; then
            printf 'remote release asset lacks an authoritative uploaded SHA-256 digest: %s\n' "$name" >&2
            return 1
        fi
        if [[ $digest != "${local_digest[$name]}" ]]; then
            printf 'release asset digest mismatch for %s: local=%s remote=%s\n' "$name" "${local_digest[$name]}" "$digest" >&2
            return 1
        fi
    done < <(jq -c '.[]' <<<"$remote_json")

    if [[ $require_exact == true ]]; then
        for name in "${!local_path[@]}"; do
            if [[ -z ${seen[$name]+x} ]]; then
                printf 'remote release asset is missing: %s\n' "$name" >&2
                return 1
            fi
        done
    fi
}

remote_assets=$(wait_for_remote_assets)
check_remote_assets false "$remote_assets"

if [[ $mode == reconcile ]]; then
    declare -A existing=()
    while IFS= read -r name; do existing[$name]=1; done < <(jq -r '.[].name' <<<"$remote_assets")
    while IFS= read -r name; do
        if [[ -z ${existing[$name]+x} ]]; then
            echo "Uploading missing release asset: $name"
            gh release upload "$tag" "${local_path[$name]}" --repo "$repository"
        else
            echo "Keeping matching release asset: $name"
        fi
    done < <(printf '%s\n' "${!local_path[@]}" | sort)
fi

# Refetch after upload (and in verify mode) so success always means the remote
# release contains exactly the expected names and authoritative SHA-256 values.
remote_assets=$(wait_for_remote_assets)
check_remote_assets true "$remote_assets"
printf 'verified %d remote release assets for %s\n' "${#local_path[@]}" "$tag"
