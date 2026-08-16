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
