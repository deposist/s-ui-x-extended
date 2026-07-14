#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
# The test intentionally extracts the current production function.
# shellcheck disable=SC1090
source <(sed -n '/^download_file() {/,/^}/p' "${repo_root}/install.sh")

tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT

cat >"${tmp_dir}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${CURL_LOG:?}"
: "${CURL_ATTEMPTS:?}"

printf '%s\n' "$*" >>"${CURL_LOG}"
attempt=0
[[ ! -f "${CURL_ATTEMPTS}" ]] || attempt=$(cat "${CURL_ATTEMPTS}")
attempt=$((attempt + 1))
printf '%s\n' "${attempt}" >"${CURL_ATTEMPTS}"

destination=""
while (($#)); do
    if [[ "$1" == "--output" ]]; then
        destination="$2"
        break
    fi
    shift
done
: "${destination:?}"

if ((attempt < CURL_SUCCEED_ON)); then
    printf 'partial' >"${destination}"
    exit 35
fi
printf 'complete' >"${destination}"
EOF
chmod +x "${tmp_dir}/curl"

cat >"${tmp_dir}/sleep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "${tmp_dir}/sleep"

export PATH="${tmp_dir}:${PATH}"
export CURL_LOG="${tmp_dir}/curl.log"
export CURL_ATTEMPTS="${tmp_dir}/curl.attempts"
export CURL_SUCCEED_ON=3

destination="${tmp_dir}/artifact.tar.gz"
download_file "https://example.test/artifact.tar.gz" "${destination}"

[[ $(<"${destination}") == "complete" ]]
[[ $(<"${CURL_ATTEMPTS}") == "3" ]]
[[ ! -e "${destination}.part" ]]
grep -q -- "--proto =https" "${CURL_LOG}"
grep -q -- "--proto-redir =https" "${CURL_LOG}"
grep -q -- "--tlsv1.2" "${CURL_LOG}"
grep -q -- "--connect-timeout 20" "${CURL_LOG}"
grep -q -- "--speed-limit 1024 --speed-time 60" "${CURL_LOG}"

printf '%s  %s\n' \
    "$(printf 'complete' | sha256sum | cut -d ' ' -f 1)" \
    "$(basename "${destination}")" >"${destination}.sha256"
(cd "${tmp_dir}" && sha256sum -c "$(basename "${destination}.sha256")")

rm -f "${CURL_ATTEMPTS}" "${destination}" "${destination}.sha256"
export CURL_SUCCEED_ON=99
if download_file "https://example.test/artifact.tar.gz" "${destination}"; then
    echo "download_file unexpectedly succeeded" >&2
    exit 1
fi
[[ $(<"${CURL_ATTEMPTS}") == "5" ]]
[[ ! -e "${destination}" ]]
[[ ! -e "${destination}.part" ]]

echo "install download tests: PASS"
