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

echo "s-ui status tests passed"
