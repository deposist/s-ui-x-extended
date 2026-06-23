# S-UI v1.5.4-beta4

Prerelease installer hardening for encrypted settings on systemd installs.

## Changed

- `install.sh` now creates a stable `SUI_SECRETBOX_KEY` for systemd installs
  when no installer-managed key exists yet.
- A newly generated key is shown once, saved in root-only
  `/etc/s-ui/secretbox.env`, and loaded by the service through an
  installer-owned systemd drop-in.
- Upgrade runs keep the existing installer-managed key instead of rotating
  it, and uninstall removes the drop-in with the rest of the systemd install
  state.
- README notes the key path and the requirement to preserve the same value
  across updates and restores.

## Validation

- `bash -n install.sh s-ui.sh` - PASS
- `git diff --check` - PASS
- `go test ./config ./database ./service` - PASS
- `go test ./middleware/... -run TestAdminSecurityHeaders` - PASS
- `cd frontend && npm run build` - PASS

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4-beta4
```

## Русский

Prerelease hardening для зашифрованных настроек в systemd installer.

- `install.sh` создаёт стабильный `SUI_SECRETBOX_KEY`, если
  installer-managed key ещё не существует.
- Новый ключ показывается один раз, сохраняется в root-only
  `/etc/s-ui/secretbox.env` и подключается к service через
  installer-owned systemd drop-in.
- Обновление не ротирует уже созданный installer-managed key, а uninstall
  убирает drop-in вместе с systemd install state.
- README фиксирует путь к ключу и требование сохранять то же значение при
  обновлениях и восстановлении.
