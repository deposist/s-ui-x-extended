#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
installer="${repo_root}/install.sh"
menu_script="${repo_root}/s-ui.sh"

tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT

extract_function() {
    local function_name="$1"
    awk -v name="${function_name}" '
        $0 == name "() {" { printing=1; depth=0 }
        printing {
            print
            opens=$0; closes=$0
            depth += gsub(/\{/, "", opens)
            depth -= gsub(/\}/, "", closes)
            if (depth == 0) exit
        }
    ' "${installer}"
}

# Extract production helpers into a normal file: the production menu is being
# checked for process substitution, so its test should not require it either.
helper_file="${tmp_dir}/helpers.sh"
for helper in download_file verify_download_checksum validate_archive_paths \
    path_exists restore_promoted_path promote_path; do
    extract_function "${helper}" >>"${helper_file}"
done
# shellcheck disable=SC1090
source "${helper_file}"

cat >"${tmp_dir}/curl" <<'MOCK_CURL'
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
    if [[ "$1" == "--output" ]]; then destination="$2"; break; fi
    shift
done
: "${destination:?}"
if ((attempt < CURL_SUCCEED_ON)); then
    printf partial >"${destination}"
    exit 35
fi
printf complete >"${destination}"
MOCK_CURL
chmod +x "${tmp_dir}/curl"
printf '#!/usr/bin/env bash\nexit 0\n' >"${tmp_dir}/sleep"
chmod +x "${tmp_dir}/sleep"

export PATH="${tmp_dir}:${PATH}"
export CURL_LOG="${tmp_dir}/curl.log"
export CURL_ATTEMPTS="${tmp_dir}/curl.attempts"
export CURL_SUCCEED_ON=3

destination="${tmp_dir}/artifact.tar.gz"
original_cwd=$(pwd)
mkdir "${tmp_dir}/caller-cwd"
cd "${tmp_dir}/caller-cwd"
download_file "https://example.test/artifact.tar.gz" "${destination}"
[[ $(<"${destination}") == complete ]]
[[ $(<"${CURL_ATTEMPTS}") == 3 ]]
[[ ! -e "${destination}.part" ]]
grep -q -- "--proto =https" "${CURL_LOG}"
grep -q -- "--proto-redir =https" "${CURL_LOG}"
grep -q -- "--tlsv1.2" "${CURL_LOG}"
grep -q -- "--connect-timeout 20" "${CURL_LOG}"
grep -q -- "--speed-limit 1024 --speed-time 60" "${CURL_LOG}"
if download_file "http://example.test/insecure" "${destination}"; then
    echo "download_file accepted plain HTTP" >&2
    exit 1
fi
cd "${original_cwd}"

artifact_name=$(basename "${destination}")
digest=$(sha256sum "${destination}")
digest=${digest%% *}
checksum="${destination}.sha256"
printf '%s  %s\n' "${digest}" "${artifact_name}" >"${checksum}"
verify_download_checksum "${destination}" "${checksum}" "${artifact_name}"
printf '%s  other.tar.gz\n' "${digest}" >"${checksum}"
if verify_download_checksum "${destination}" "${checksum}" "${artifact_name}"; then
    echo "checksum validator accepted the wrong filename" >&2
    exit 1
fi
printf '%s  %s\nextra\n' "${digest}" "${artifact_name}" >"${checksum}"
if verify_download_checksum "${destination}" "${checksum}" "${artifact_name}"; then
    echo "checksum validator accepted multiple records" >&2
    exit 1
fi

archive_root="${tmp_dir}/archive"
mkdir -p "${archive_root}/s-ui"
printf '#!/usr/bin/env bash\nexit 0\n' >"${archive_root}/s-ui/sui"
printf '#!/usr/bin/env bash\nexit 0\n' >"${archive_root}/s-ui/s-ui.sh"
printf '[Service]\n' >"${archive_root}/s-ui/s-ui.service"
tar -czf "${tmp_dir}/safe.tar.gz" -C "${archive_root}" s-ui
validate_archive_paths "${tmp_dir}/safe.tar.gz" "${tmp_dir}/safe.list" "${tmp_dir}/safe.verbose"
tar -czf "${tmp_dir}/bad.tar.gz" --transform='s#s-ui/sui#../sui#' -C "${archive_root}" s-ui
if validate_archive_paths "${tmp_dir}/bad.tar.gz" "${tmp_dir}/bad.list" "${tmp_dir}/bad.verbose"; then
    echo "archive validator accepted a traversal path" >&2
    exit 1
fi

# Exercise the rename/restore primitives in an arbitrary temporary hierarchy.
live="${tmp_dir}/tx/live"
backup="${tmp_dir}/tx/backup"
failed="${tmp_dir}/tx/failed"
new="${tmp_dir}/tx/new"
mkdir -p "${live}" "${new}"
printf old >"${live}/value"
printf new >"${new}/value"
had_old=0
state=untouched
promote_path "${new}" "${live}" "${backup}" had_old state
[[ $(<"${live}/value") == new && ${had_old} == 1 && ${state} == promoted ]]
restore_promoted_path "${live}" "${backup}" "${failed}" "${new}" "${had_old}" "${state}"
[[ $(<"${live}/value") == old && ! -e "${backup}" && ! -e "${failed}" ]]

# Static ordering/trap checks cover the privileged systemd transaction without
# executing it: verification must occur before state capture/stop, and all
# failure signals must enter rollback.
verify_line=$(grep -n 'verify_staged_tree "${extracted_tree}"' "${installer}" | cut -d: -f1)
state_line=$(grep -n '^[[:space:]]*record_service_state$' "${installer}" | cut -d: -f1)
stop_line=$(grep -n '^[[:space:]]*systemctl stop s-ui$' "${installer}" | tail -n1 | cut -d: -f1)
((verify_line < state_line && state_line < stop_line))
grep -Fq "trap 'abort_install_transaction \$?' ERR" "${installer}"
grep -Fq "trap 'abort_install_transaction 130' INT" "${installer}"
grep -Fq "trap 'abort_install_transaction 143' TERM" "${installer}"
grep -Fq 'restore_service_state' "${installer}"
grep -Fq 'config_after_install' "${installer}"

# Every remote shell download is a successful HTTPS-to-file transfer before a
# separately named interpreter invocation.  Streaming/process substitution is
# forbidden in the management script.
grep -Fq "--proto-redir '=https'" "${menu_script}"
grep -Fq -- '--output "${temporary}"' "${menu_script}"
grep -Fq 'bash "${installer}"' "${menu_script}"
if grep -Eq '(bash|sh)[[:space:]]*<\(|<\(curl|curl[^|]*\|[[:space:]]*(bash|sh)|wget[^|]*\|[[:space:]]*(bash|sh)' "${menu_script}"; then
    echo "management script still streams remote content to a shell" >&2
    exit 1
fi

rm -f "${CURL_ATTEMPTS}" "${destination}" "${destination}.part"
export CURL_SUCCEED_ON=99
if download_file "https://example.test/artifact.tar.gz" "${destination}"; then
    echo "download_file unexpectedly succeeded" >&2
    exit 1
fi
[[ $(<"${CURL_ATTEMPTS}") == 5 ]]
[[ ! -e "${destination}" && ! -e "${destination}.part" ]]

echo "install download tests: PASS"
