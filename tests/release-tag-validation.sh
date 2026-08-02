#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
validator="$root_dir/scripts/validate-release-tag.sh"
checkout_verifier="$root_dir/scripts/verify-release-tag-checkout.sh"
asset_guard="$root_dir/scripts/check-release-assets.sh"
release_state_guard="$root_dir/scripts/check-release-state.sh"
if [[ ! -f $release_state_guard ]]; then
    echo "release state guard is missing: $release_state_guard" >&2
    exit 1
fi
for tool in jq sha256sum; do
    if ! command -v "$tool" >/dev/null; then
        echo "release tag validation smoke test requires $tool" >&2
        exit 1
    fi
done

for tag in v1.2.3 v2.0.0-rc.1 v1.5.2-beta-hotfix2 v3.4.5-hotfix1 v3.4.5-preview.7; do
    actual=$(bash "$validator" "$tag")
    test "$actual" = "$tag"
done
test "$(bash "$validator" --prerelease v1.2.3)" = false
for tag in v2.0.0-rc.1 v1.5.2-beta-hotfix2 v3.4.5-hotfix1 v3.4.5-preview.7; do
    test "$(bash "$validator" --prerelease "$tag")" = true
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
    if bash "$validator" "$tag" >/dev/null 2>&1; then
        printf 'accepted invalid release tag: %q\n' "$tag" >&2
        exit 1
    fi
done

for workflow in release.yml windows.yml docker.yml; do
    path="$root_dir/.github/workflows/$workflow"
    grep -q 'uses: ./.github/actions/checkout-release-tag' "$path"
    grep -q "event_commit: \${{ github.event_name == 'push' && github.sha || '' }}" "$path"
done
checkout_action="$root_dir/.github/actions/checkout-release-tag/action.yml"
grep -q "ref: refs/tags/\${{ steps.input.outputs.tag }}" "$checkout_action"
grep -q 'fetch-depth: 0' "$checkout_action"
# The composite action must execute the tested verifier, including event-SHA binding.
# shellcheck disable=SC2016
grep -q 's-ui-release-tooling/verify-release-tag-checkout.sh" "$TAG" "$EXPECTED_COMMIT" "$EVENT_COMMIT"' "$checkout_action"
grep -q 's-ui-release-tooling/.*check-release-state.sh' "$checkout_action"
stage_guard_line=$(grep -n 'cp .*check-release-state.sh' "$checkout_action" | cut -d: -f1)
checkout_tag_line=$(grep -n 'name: Check out release tag' "$checkout_action" | cut -d: -f1)
test -n "$stage_guard_line" && test -n "$checkout_tag_line" && test "$stage_guard_line" -lt "$checkout_tag_line"
# The grep needle intentionally contains shell variables.
# shellcheck disable=SC2016
grep -q 'version=$(git show "$tag_commit:config/version"' "$checkout_verifier"
release_workflow="$root_dir/.github/workflows/release.yml"
windows_workflow="$root_dir/.github/workflows/windows.yml"
docker_workflow="$root_dir/.github/workflows/docker.yml"
ci_workflow="$root_dir/.github/workflows/ci.yml"
# The grep needle intentionally contains a GitHub expression.
# shellcheck disable=SC2016
grep -q 'value: ${{ steps.verify.outputs.prerelease }}' "$checkout_action"
# The grep needle intentionally contains shell variables.
# shellcheck disable=SC2016
grep -Fq 'run: bash "$RUNNER_TEMP/s-ui-release-tooling/check-release-state.sh" "$GITHUB_REPOSITORY" "$TAG"' "$release_workflow"
grep -q 'scripts/check-release-assets.sh --verify' "$release_workflow"
grep -q 'uses: ./.github/workflows/windows.yml' "$release_workflow"
grep -q 'npm ci' "$windows_workflow"
grep -q 'npm run lint -- --max-warnings=0' "$windows_workflow"
grep -q 'npm run test' "$windows_workflow"
grep -q 'npm run verify:dist' "$windows_workflow"
grep -q 'npm run verify:dist' "$release_workflow"
grep -q 'npm run verify:dist' "$ci_workflow"
grep -q 'run: bash tests/release-tag-validation.sh' "$ci_workflow"
grep -q 'push-by-digest=true,name-canonical=true' "$docker_workflow"
grep -q 'Runtime smoke immutable platform image' "$docker_workflow"
grep -q 'Final runtime smoke for every candidate platform' "$docker_workflow"
# The grep needle intentionally contains shell variables.
# shellcheck disable=SC2016
grep -Fq '"$IMAGE_NAME@$platform_digest"' "$docker_workflow"
grep -q 'Promote final Docker tags' "$docker_workflow"
grep -q 'BOOTLIN_ARMV5_SHA256: 8cdb4ad70c6b5a66427fa3315fe3ddde1c19c90232674dec59045e04d2a36cf1' "$release_workflow"
grep -q 'BOOTLIN_S390X_SHA256: 23f536ff2bf1a9d3b93210465471996bc7c918fbb5702a277d8e1e42ffab8559' "$release_workflow"

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
printf '2.0.0-rc.1\n' > "$tmp_dir/config/version"
git -C "$tmp_dir" commit --quiet -am v2-rc
git -C "$tmp_dir" tag -a v2.0.0-rc.1 -m annotated
annotated_commit=$(git -C "$tmp_dir" rev-parse HEAD)
git -C "$tmp_dir" checkout --quiet v1.0.0
(
    cd "$tmp_dir"
    output=$(bash "$checkout_verifier" v1.0.0 "$old_commit")
    grep -q "commit=$old_commit" <<<"$output"
    bash "$checkout_verifier" v1.0.0 "$old_commit" "$old_commit" >/dev/null
    if bash "$checkout_verifier" v9.9.9 >/dev/null 2>&1; then
        echo 'nonexistent release tag was accepted' >&2
        exit 1
    fi
    if bash "$checkout_verifier" v1.0.0 "$new_commit" >/dev/null 2>&1; then
        echo 'moved release tag was accepted' >&2
        exit 1
    fi
    if bash "$checkout_verifier" v1.0.0 "$old_commit" "$new_commit" >/dev/null 2>&1; then
        echo 'tag push event commit mismatch was accepted' >&2
        exit 1
    fi
    if bash "$checkout_verifier" v9.9.8 >/dev/null 2>&1; then
        echo 'tag/version mismatch was accepted' >&2
        exit 1
    fi
    git checkout --quiet v2.0.0-rc.1
    bash "$checkout_verifier" v2.0.0-rc.1 "$annotated_commit" "$annotated_commit" >/dev/null
)

cat > "$tmp_dir/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ $1 == api ]]; then
    api_path=
    for arg in "$@"; do
        [[ $arg == repos/* ]] && api_path=$arg
    done
    case $api_path in
        *releases?per_page=100)
            if [[ -n ${FAKE_GH_RELEASES+x} ]]; then
                printf '%s\n' "$FAKE_GH_RELEASES"
            else
                printf '[[{"id":7,"tag_name":"v1.2.3","draft":true}]]\n'
            fi
            ;;
        *releases/7/assets?per_page=100)
            printf '[[%s]]\n' "${FAKE_GH_ASSETS}"
            ;;
        *)
            echo "unexpected fake gh API path: $api_path" >&2
            exit 1
            ;;
    esac
elif [[ $1 == release && $2 == upload ]]; then
    exit 0
else
    echo "unexpected fake gh invocation: $*" >&2
    exit 1
fi
EOF
chmod +x "$tmp_dir/bin/gh"

PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_RELEASES='[[]]' \
    bash "$release_state_guard" owner/repository v1.2.3
PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_RELEASES='[[{"id":7,"tag_name":"v1.2.3","draft":true}]]' \
    bash "$release_state_guard" owner/repository v1.2.3
if PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_RELEASES='[[{"id":7,"tag_name":"v1.2.3","draft":false}]]' \
    bash "$release_state_guard" owner/repository v1.2.3 >/dev/null 2>&1; then
    echo 'published release tag reuse was accepted' >&2
    exit 1
fi
if PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_RELEASES='[[{"id":7,"tag_name":"v1.2.3","draft":true},{"id":8,"tag_name":"v1.2.3","draft":true}]]' \
    bash "$release_state_guard" owner/repository v1.2.3 >/dev/null 2>&1; then
    echo 'duplicate matching releases were accepted' >&2
    exit 1
fi
if PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_RELEASES='[[{"id":7,"tag_name":"v1.2.3","draft":"yes"}]]' \
    bash "$release_state_guard" owner/repository v1.2.3 >/dev/null 2>&1; then
    echo 'malformed release state was accepted' >&2
    exit 1
fi
printf 'new asset\n' > "$tmp_dir/assets/new.tar.gz"
asset_digest=$(sha256sum "$tmp_dir/assets/new.tar.gz" | awk '{print $1}')
matching_asset=$(printf '{"name":"new.tar.gz","state":"uploaded","digest":"sha256:%s"}' "$asset_digest")
PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_ASSETS="$matching_asset" \
    bash "$asset_guard" owner/repository v1.2.3 "$tmp_dir/assets"

bad_asset='{"name":"new.tar.gz","state":"uploaded","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
if PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_ASSETS="$bad_asset" \
    bash "$asset_guard" owner/repository v1.2.3 "$tmp_dir/assets" >/dev/null 2>&1; then
    echo 'divergent existing release asset was accepted' >&2
    exit 1
fi

unexpected_asset='{"name":"old.tar.gz","state":"uploaded","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
if PATH="$tmp_dir/bin:$PATH" GH_TOKEN=test FAKE_GH_ASSETS="$unexpected_asset" \
    bash "$asset_guard" owner/repository v1.2.3 "$tmp_dir/assets" >/dev/null 2>&1; then
    echo 'unexpected release asset was accepted' >&2
    exit 1
fi

echo 'release tag validation smoke test passed'
