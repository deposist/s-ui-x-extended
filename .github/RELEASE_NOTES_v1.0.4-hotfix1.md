# S-UI-X Extended v1.0.4-hotfix1

Hotfix for the AWG device key-encryption key.

- The systemd installer now generates `AWG_KEY_ENC` on install and upgrade, alongside the existing `SUI_SECRETBOX_KEY` and `SUI_COOKIE_KEY`. Without it, creating a managed AmneziaWG device failed with `awg: AWG encryption key is unavailable`. The key is written to `/etc/s-ui/secretbox.env` and shown once when first generated.
- An existing key is never regenerated: rotating it after devices exist would make their stored key material undecryptable.

Note: this build is versioned `1.0.4-hotfix1`. A prerelease suffix sorts below `1.0.4`, so the in-panel auto-update on a `1.0.4` install will not offer it; install it explicitly by re-running the installer or updating from the tag.

If your panel already hit this error, add the key once and restart (only when no managed AWG devices exist yet):

```
umask 077
printf 'AWG_KEY_ENC=%s\n' "$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')" >> /etc/s-ui/secretbox.env
chmod 600 /etc/s-ui/secretbox.env
systemctl restart s-ui
```

Full release notes: [`docs/releases/v1.0.4-hotfix1.md`](../docs/releases/v1.0.4-hotfix1.md).
