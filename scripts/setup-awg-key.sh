#!/usr/bin/env bash
# Ensures AWG_KEY_ENC exists in the s-ui panel environment file.
#
# AWG_KEY_ENC seals the per-device WireGuard private and preshared keys at rest.
# Without it every managed AmneziaWG device operation fails with
# "awg: AWG encryption key is unavailable".
#
# This mirrors the "Generate env keys -> AmneziaWG device key" console menu
# item. It is idempotent: an existing key is left untouched, because rotating
# it after devices exist makes their stored key material undecryptable.
#
# Usage: sudo bash setup-awg-key.sh
set -euo pipefail

ENV_FILE="/etc/s-ui/secretbox.env"
DROPIN_DIR="/etc/systemd/system/s-ui.service.d"
DROPIN_FILE="${DROPIN_DIR}/10-secretbox-env.conf"

if [[ "$(id -u)" -ne 0 ]]; then
    echo "Run as root (sudo)." >&2
    exit 1
fi

# Already present? Do nothing. Never rotate an existing key automatically.
if [[ -f "${ENV_FILE}" ]] && grep -q '^AWG_KEY_ENC=..*' "${ENV_FILE}"; then
    echo "AWG_KEY_ENC already set in ${ENV_FILE}; leaving it unchanged."
    exit 0
fi

key="$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')"

mkdir -p "$(dirname "${ENV_FILE}")"
if [[ -f "${ENV_FILE}" ]]; then
    printf '\nAWG_KEY_ENC=%s\n' "${key}" >>"${ENV_FILE}"
else
    (umask 077 && printf 'AWG_KEY_ENC=%s\n' "${key}" >"${ENV_FILE}")
fi
chmod 600 "${ENV_FILE}"

# Make sure systemd loads the env file for the service.
mkdir -p "${DROPIN_DIR}"
if [[ ! -f "${DROPIN_FILE}" ]]; then
    printf '[Service]\nEnvironmentFile=-%s\n' "${ENV_FILE}" >"${DROPIN_FILE}"
    chmod 644 "${DROPIN_FILE}"
    systemctl daemon-reload
fi

echo "###############################################"
echo "Generated AWG_KEY_ENC (shown once):"
echo "AWG_KEY_ENC: ${key}"
echo "Key file: ${ENV_FILE}"
echo "Keep this key. If it is lost, the configs of devices already created cannot be decrypted."
echo "###############################################"

systemctl restart s-ui
echo "s-ui restarted. The key is now active."
