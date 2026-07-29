#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
validator="$root_dir/scripts/validate-release-tag.sh"
checkout_verifier="$root_dir/scripts/verify-release-tag-checkout.sh"
asset_guard="$root_dir/scripts/check-release-assets.sh"

for tag in v1.2.3 v2.0.0-rc.1 v1.5.2-beta-hotfix2; do
    actual=$("$validator" "$tag")
    test "$actual" = "$tag"
done

invalid_tags=(
    '1.2.3'
    'v1.2'
    'v01.2.3'
    'v1.2.3-01'
    'v1.2.3-RC.1'
    'v1.2.3 quoted'
    'v1.2.3"'
    'v1.2.3;echo injected'
    "v1.2.3\$(echo injected)"
    $'v1.2.3\necho injected'
)
for tag in "${invalid_tags[@]}"; do
    if "$validator" "$tag" >/dev/null 2>&1; then
        printf 'accepted invalid release tag: %q\n' "$tag" >&2
        exit 1
    fi
done

for workflow in release.yml windows.yml docker.yml; do
    path="$root_dir/.github/workflows/$workflow"
    grep -q 'uses: ./.github/actions/checkout-release-tag' "$path"
done
checkout_action="$root_dir/.github/actions/checkout-release-tag/action.yml"
grep -q "ref: refs/tags/\${{ steps.input.outputs.tag }}" "$checkout_action"
grep -q 'fetch-depth: 0' "$checkout_action"
# The grep needle intentionally contains shell variables.
# shellcheck disable=SC2016
grep -q 'test "$TAG_COMMIT" = "$EXPECTED_COMMIT"' "$checkout_action"
# The grep needle intentionally contains a shell variable.
# shellcheck disable=SC2016
grep -q 'git show "$TAG_COMMIT:config/version"' "$checkout_action"
grep -q 'overwrite_files: false' "$root_dir/.github/workflows/release.yml"
grep -q 'overwrite_files: false' "$root_dir/.github/workflows/windows.yml"

while IFS= read -r -d '' workflow; do
    if ! awk '
        function indentation(line) {
            match(line, /^ */)
            return RLENGTH
        }
        /^[ ]*run:[ ]*\|[+-]?[ ]*$/ {
            in_run = 1
            run_indent = indentation($0)
            next
        }
        in_run && NF > 0 && indentation($0) <= run_indent {
            in_run = 0
        }
        in_run && index($0, "${{ inputs.") {
            print FNR ":" $0
            found = 1
        }
        END { exit found }
    ' "$workflow"; then
        echo "untrusted workflow input is interpolated in run block: $workflow" >&2
        exit 1
    fi
done < <(find "$root_dir/.github/workflows" -type f \( -name '*.yml' -o -name '*.yaml' \) -print0)

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
mkdir "$tmp_dir/bin" "$tmp_dir/assets"

git -C "$tmp_dir" init --quiet
git -C "$tmp_dir" config user.name release-smoke
git -C "$tmp_dir" config user.email release-smoke@example.invalid
mkdir "$tmp_dir/config"
printf '1.0.0\n' > "$tmp_dir/config/version"
git -C "$tmp_dir" add config/version
git -C "$tmp_dir" commit --quiet -m v1
git -C "$tmp_dir" tag v1.0.0
old_commit=$(git -C "$tmp_dir" rev-parse HEAD)
git -C "$tmp_dir" tag v9.9.8 "$old_commit"
printf '2.0.0\n' > "$tmp_dir/config/version"
git -C "$tmp_dir" commit --quiet -am v2
git -C "$tmp_dir" tag v2.0.0
new_commit=$(git -C "$tmp_dir" rev-parse HEAD)
git -C "$tmp_dir" checkout --quiet v1.0.0
(
    cd "$tmp_dir"
    output=$("$checkout_verifier" v1.0.0 "$old_commit")
    grep -q "commit=$old_commit" <<<"$output"
    if "$checkout_verifier" v9.9.9 >/dev/null 2>&1; then
        echo 'nonexistent release tag was accepted' >&2
        exit 1
    fi
    if "$checkout_verifier" v1.0.0 "$new_commit" >/dev/null 2>&1; then
        echo 'moved release tag was accepted' >&2
        exit 1
    fi
    if "$checkout_verifier" v9.9.8 >/dev/null 2>&1; then
        echo 'tag/version mismatch was accepted' >&2
        exit 1
    fi
)

cat > "$tmp_dir/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "${FAKE_GH_ASSETS-}"
EOF
chmod +x "$tmp_dir/bin/gh"
printf 'new asset\n' > "$tmp_dir/assets/new.tar.gz"
PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_ASSETS='old.tar.gz' \
    "$asset_guard" owner/repository v1.2.3 "$tmp_dir/assets"
if PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_ASSETS='new.tar.gz' \
    "$asset_guard" owner/repository v1.2.3 "$tmp_dir/assets" >/dev/null 2>&1; then
    echo 'existing release asset was accepted' >&2
    exit 1
fi

echo 'release tag validation smoke test passed'
