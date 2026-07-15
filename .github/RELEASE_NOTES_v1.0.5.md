# S-UI-X Extended v1.0.5

Rolls up the AWG device key-encryption key hotfix (first tagged `v1.0.4-hotfix1`) and adds a version-precedence rule for future hotfixes.

- The systemd installer now generates `AWG_KEY_ENC` on install and upgrade, alongside the existing `SUI_SECRETBOX_KEY` and `SUI_COOKIE_KEY`. Without it, creating a managed AmneziaWG device failed with `awg: AWG encryption key is unavailable`. The key is written to `/etc/s-ui/secretbox.env` and shown once when first generated. An existing key is never regenerated.
- A `hotfixN` prerelease now ranks above its base release (`1.0.4-hotfix1` is newer than `1.0.4`), so panel auto-update offers a hotfix to installs on the base version. A later patch such as `1.0.5` still outranks it, and beta/rc lines keep normal SemVer precedence.

If your panel already hit the AWG error, add the key once and restart (only when no managed AWG devices exist yet):

```
umask 077
printf 'AWG_KEY_ENC=%s\n' "$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')" >> /etc/s-ui/secretbox.env
chmod 600 /etc/s-ui/secretbox.env
systemctl restart s-ui
```

Full release notes: [`docs/releases/v1.0.5.md`](../docs/releases/v1.0.5.md).
