#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="${repo_root}/s-ui.sh"
tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT

helpers="${tmp_dir}/helpers.sh"
for name in install update generate_self_signed_cert; do
    sed -n "/^${name}() {/,/^}/p" "${script}" >>"${helpers}"
done

# Source only the selected production helpers; never run the privileged menu.
# shellcheck disable=SC1090
source "${helpers}"
t() { :; }
LOGI() { :; }
LOGE() { :; }
before_show_menu() { :; }
confirm() { return 0; }
# Invoked by the sourced install/update helpers.
# shellcheck disable=SC2329
run_remote_installer() { return 37; }
start() { printf '%s\n' "$*" >"${tmp_dir}/start-args"; }
failed=0

for command in install update; do
    status=0
    ("${command}" 0) || status=$?
    if [[ ${status} != 37 ]]; then
        echo "${command} hid installer failure: exit ${status}" >&2
        failed=1
    fi
done

run_remote_installer() { return 0; }
install 0
[[ $(<"${tmp_dir}/start-args") == 's-ui 0' ]] || {
    echo 'install started the wrong service' >&2
    failed=1
}

# Redirect the certificate output to a temporary path and use real OpenSSL.
sed -n '/^generate_self_signed_cert() {/,/^}/p' "${script}" |
    sed "s|cert_dir=\"/etc/sing-box\"|cert_dir=\"${tmp_dir}/cert\"|" >"${tmp_dir}/cert-helper.sh"
# shellcheck source=/dev/null
source "${tmp_dir}/cert-helper.sh"
export green='' plain=''
export MSYS2_ARG_CONV_EXCL='/CN='
for certificate_type in 1 2 3 4 5; do
    printf '%s\n' "${certificate_type}" | generate_self_signed_cert
    if ! openssl x509 -in "${tmp_dir}/cert/self.crt" -noout >/dev/null 2>&1 ||
        ! openssl pkey -in "${tmp_dir}/cert/self.key" -check -noout >/dev/null 2>&1; then
        echo "certificate type ${certificate_type} failed" >&2
        failed=1
    fi
done

[[ ${failed} == 0 ]] || exit 1

echo 's-ui maintenance tests: PASS'
