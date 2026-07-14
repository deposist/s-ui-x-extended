#!/bin/bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="${repo_root}/s-ui.sh"

load_language=$(sed -n '/^load_language() {/,/^}/p' "${script}")

assert_language() {
    local expected="$1"
    shift
    local actual

    actual=$(LANG_FILE="$1" SUI_LANG="${2-}" bash -c "${load_language}"$'\n''load_language'$'\n''printf "%s" "${lang}"')
    if [[ "${actual}" != "${expected}" ]]; then
        echo "expected language ${expected}, got ${actual}" >&2
        exit 1
    fi
}

tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT

printf 'ru\n' > "${tmp_dir}/lang"
assert_language ru "${tmp_dir}/lang"

assert_language en "${tmp_dir}/missing"
assert_language zh "${tmp_dir}/lang" zh

echo "s-ui language tests passed"
