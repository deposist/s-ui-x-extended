#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT
export cert_test_root="${tmp_dir}"
mkdir -p "${tmp_dir}/account/.acme.sh/old.test" "${tmp_dir}/cert/old.test" "${tmp_dir}/cert-CF/old.test"
printf old >"${tmp_dir}/account/.acme.sh/old.test/keep"
printf old >"${tmp_dir}/cert/old.test/keep"
printf old >"${tmp_dir}/cert-CF/old.test/keep"
cat >"${tmp_dir}/account/.acme.sh/acme.sh" <<'ACME'
#!/usr/bin/env bash
[[ $1 != --issue || ${CERT_ISSUE_FAIL:-0} != 1 ]]
ACME
chmod +x "${tmp_dir}/account/.acme.sh/acme.sh"

for name in ssl_cert_issue ssl_cert_issue_CF; do
    # Keep the variables literal in the extracted shell functions.
    # shellcheck disable=SC2016
    sed -n "/^${name}() {/,/^}/p" "${repo_root}/s-ui.sh" |
        sed 's|${HOME}|${cert_test_root}/account|g; s|/root/cert|${cert_test_root}/cert|g' >>"${tmp_dir}/helpers.sh"
done
# shellcheck source=/dev/null
source "${tmp_dir}/helpers.sh"
LOGD() { :; }
LOGI() { :; }
LOGE() { :; }
t() { :; }
show_menu() { :; }
apt() { :; }
export release=ubuntu red='' yellow='' plain=''

export CERT_ISSUE_FAIL=1
status=0
(printf 'old.test\n80\n' | ssl_cert_issue) || status=$?
[[ ${status} != 0 && -f "${tmp_dir}/cert/old.test/keep" && -f "${tmp_dir}/account/.acme.sh/old.test/keep" ]] || {
    echo 'failed issuance deleted existing certificate data' >&2
    exit 1
}

export CERT_ISSUE_FAIL=0
printf '1\nnew.test\ntest-key\nuser@example.test\n' | ssl_cert_issue_CF
[[ -f "${tmp_dir}/cert-CF/old.test/keep" ]] || {
    echo 'Cloudflare issuance deleted other domains' >&2
    exit 1
}

echo 's-ui certificate preservation tests: PASS'
