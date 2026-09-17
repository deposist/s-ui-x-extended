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
`replace` directive in `go.mod`.

The engine is pinned to `deposist/wireguard-go v0.0.5-extended-1.6.2`
(commit `8f4b19e`, fork of `shtorm-7/wireguard-go v0.0.5-extended-1.6.1`).
It fixes the random trailers wire format: handshake messages are now
marshaled into the exact-size sub-slice instead of the trailer-extended
buffer, so the message body is actually written and peers can classify and
authenticate handshake packets. Without this fix, enabling random trailers
made handshakes fail silently in both directions.

Both flags default to off; the panel writes them only when enabled, so configs
without the keys stay valid for older cores and clients.
