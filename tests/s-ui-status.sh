#!/bin/bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="${repo_root}/s-ui.sh"

tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT

service_dir="${tmp_dir}/etc/systemd/system"
mock_bin="${tmp_dir}/bin"
mkdir -p "${service_dir}" "${mock_bin}"
touch "${service_dir}/s-ui.service"

cat >"${mock_bin}/systemctl" <<'EOF'
#!/bin/bash
if [[ "$1" == "is-active" ]]; then
    printf '%s\n' "${MOCK_SYSTEMCTL_STATE}"
    exit "${MOCK_SYSTEMCTL_EXIT}"
fi
exit 1
EOF
chmod +x "${mock_bin}/systemctl"

check_status=$(sed -n '/^check_status() {/,/^}/p' "${script}" |
    sed "s|/etc/systemd/system|${service_dir}|g")
show_status=$(sed -n '/^show_status() {/,/^}/p' "${script}")
check_install=$(sed -n '/^check_install() {/,/^}/p' "${script}")
check_uninstall=$(sed -n '/^check_uninstall() {/,/^}/p' "${script}")
stop_service=$(sed -n '/^stop() {/,/^}/p' "${script}")
issue_ip_cert=$(sed -n '/^ssl_cert_issue_ip() {/,/^}/p' "${script}")

output=$(MOCK_SYSTEMCTL_STATE=activating MOCK_SYSTEMCTL_EXIT=3 PATH="${mock_bin}:${PATH}" \
    bash -c "set -euo pipefail
${check_status}
check_status s-ui
printf reached")
if [[ "${output}" != "reached" ]]; then
    echo "activating service did not remain usable: ${output}" >&2
    exit 1
fi

output=$(MOCK_SYSTEMCTL_STATE=inactive MOCK_SYSTEMCTL_EXIT=3 PATH="${mock_bin}:${PATH}" \
    bash -c "set -euo pipefail
green=
yellow=
red=
plain=
t() { printf '%s %s' \"\$1\" \"\${2-}\"; }
show_enable_status() { :; }
${check_status}
${show_status}
show_status s-ui
printf '\nreached'")
if [[ "${output}" != *"status_stopped s-ui"* || "${output}" != *"reached" ]]; then
    echo "inactive service caused the menu status path to exit: ${output}" >&2
    exit 1
fi

output=$(MOCK_SYSTEMCTL_STATE=inactive MOCK_SYSTEMCTL_EXIT=3 PATH="${mock_bin}:${PATH}" \
    bash -c "set -euo pipefail
t() { :; }
LOGE() { :; }
before_show_menu() { :; }
${check_status}
${check_install}
${check_uninstall}
check_install
if check_uninstall; then
    exit 1
fi
printf reached")
if [[ "${output}" != *"reached" ]]; then
    echo "install guards did not handle an inactive installed service: ${output}" >&2
    exit 1
fi

output=$(MOCK_SYSTEMCTL_STATE=inactive MOCK_SYSTEMCTL_EXIT=3 PATH="${mock_bin}:${PATH}" \
    bash -c "set -euo pipefail
t() { printf '%s' \"\$1\"; }
LOGI() { :; }
LOGE() { :; }
before_show_menu() { :; }
${check_status}
${stop_service}
stop s-ui 0
printf reached")
if [[ "${output}" != *"reached"* ]]; then
    echo "stopping an inactive service exited the menu: ${output}" >&2
    exit 1
fi

cert_tmp="${tmp_dir}/ip-cert"
mkdir -p "${cert_tmp}/bin" "${cert_tmp}/etc"
cat >"${cert_tmp}/bin/sui" <<'EOF'
#!/bin/bash
exit "${MOCK_CERT_EXIT}"
EOF
chmod +x "${cert_tmp}/bin/sui"

issue_ip_cert=$(printf '%s\n' "${issue_ip_cert}" |
    sed "s|local bin=\"/usr/local/s-ui/sui\"|local bin=\"${cert_tmp}/bin/sui\"|")
output=$(printf '93.184.216.34\nadmin@example.com\n80\n' | \
    MOCK_CERT_EXIT=1 MOCK_SYSTEMCTL_STATE=active MOCK_SYSTEMCTL_EXIT=0 PATH="${mock_bin}:${PATH}" \
    bash -c "set -euo pipefail
yellow=
plain=
SECRETBOX_ENV_FILE='${cert_tmp}/etc/secretbox.env'
t() { printf '%s' \"\$1\"; }
LOGI() { :; }
LOGE() { printf 'cert_failed'; }
before_show_menu() { printf 'menu_returned'; }
${check_status}
stop() { :; }
start() { :; }
${issue_ip_cert}
ssl_cert_issue_ip
printf reached")
if [[ "${output}" != *"cert_failed"* || "${output}" != *"menu_returned"* || "${output}" != *"reached"* ]]; then
    echo "failed IP-certificate issuance did not return to the menu: ${output}" >&2
    exit 1
fi

output=$(printf '93.184.216.34\nadmin@example.com\n80\n' | \
    MOCK_CERT_EXIT=1 MOCK_SYSTEMCTL_STATE=active MOCK_SYSTEMCTL_EXIT=0 PATH="${mock_bin}:${PATH}" \
    bash -c "set -euo pipefail
yellow=
plain=
SECRETBOX_ENV_FILE='${cert_tmp}/etc/secretbox.env'
t() { printf '%s' \"\$1\"; }
LOGI() { :; }
LOGE() { printf 'cert_failed'; }
before_show_menu() { printf 'menu_returned'; }
${check_status}
stop() { return 1; }
start() { return 1; }
${issue_ip_cert}
ssl_cert_issue_ip
printf reached")
if [[ "${output}" != *"cert_failed"* || "${output}" != *"menu_returned"* || "${output}" != *"reached"* ]]; then
    echo "service stop/start status failures exited IP-certificate flow: ${output}" >&2
    exit 1
fi

echo "s-ui status tests passed"
