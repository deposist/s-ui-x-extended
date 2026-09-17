# Dependency patches

## WireGuard IPC access

`sing-box-extended-v1.13.14-wireguard-ipc.patch` is the source diff for commit
`6f937fbef74d4f26182ab8a95af270008743cdc3` in
`deposist/sing-box-extended`, tagged `v1.13.14-extended-2.5.1`.

Apply it to `shtorm-7/sing-box-extended@v1.13.14-extended-2.5.0`:

```bash
git apply --unidiff-zero sing-box-extended-v1.13.14-wireguard-ipc.patch
```

The zero-context format keeps the containing repository's whitespace checks
meaningful. Always apply it to the exact base tag above. The resulting diff must
match the tagged fork commit before building or publishing another dependency
version.

## AmneziaWG 3.1 switches

Merged upstream: the AmneziaWG 3.1 switches (`random_trailers`,
`disable_cookies`) are part of `deposist/sing-box-extended`
`v1.14.0-extended-2.7.5` (commit `cc14a04`), which the panel pins via the
`replace` directive in `go.mod`. The engine (`shtorm-7/wireguard-go
v0.0.5-extended-1.6.1`) already implements both switches; the tag only exposes
them in sing-box options and writes them into the wireguard-go UAPI setup.

Both flags default to off; the panel writes them only when enabled, so configs
without the keys stay valid for older cores and clients.
