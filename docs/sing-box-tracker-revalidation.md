# Sing-box Tracker Revalidation Policy

Validated dependency:

- `github.com/sagernet/sing-box v1.13.14`

> Note: `go.mod` keeps `require v1.13.14` but `replace`s it with the
> shtorm-7/sing-box-extended fork (`v1.13.14-extended-2.5.0`). The fork
> preserves the upstream module path so all imports remain
> `github.com/sagernet/sing-box/...`.

The local `ConnTracker` and `StatsTracker` wrap sing-box routed TCP and packet
connections. Any bump of `github.com/sagernet/sing-box` must revalidate this
contract before merge. `go test ./core` enforces that this document and
`core/tracker_policy.go` are updated when the sing-box version changes.

Required checks:

- RoutedConnection signature still matches sing-box adapter.RouterConnectionTracker
- RoutedPacketConnection signature still matches sing-box adapter.RouterConnectionTracker
- wrapped TCP connections always call Done exactly once on Close or terminal I/O error
- wrapped packet connections always call Done exactly once on Close or terminal I/O error
- Reset closes tracked connections and waits for active wrappers before replacing tracker state
- StatsTracker keeps counter pointers stable across Reset for already wrapped connections
- source IP extraction from adapter.InboundContext still uses metadata.Source.Addr

Validation gate for a sing-box bump:

- `go test ./core`
- `go test -race ./core`
- `go test -race ./service ./api`
- Manual smoke check: start core, create one TCP inbound and one UDP-capable
  inbound, confirm stats are collected, then restart core and confirm old
  wrapped connections do not keep changing new counters.

Revalidation log:

- 2026-06-14, v1.13.12 -> v1.13.13: revalidated against the fixed release
  commit (`78b2e12`). `adapter.ConnectionTracker` is unchanged -
  `RoutedConnection(ctx, net.Conn, InboundContext, Rule, Outbound) net.Conn` and
  `RoutedPacketConnection(ctx, N.PacketConn, InboundContext, Rule, Outbound) N.PacketConn`
  still match `ConnTracker`/`StatsTracker` (proven by `core` compiling the
  `router.AppendTracker` calls). `adapter.InboundContext.Source` is still an
  `M.Socksaddr`, so `metadata.Source.Addr` extraction is intact. Done-once,
  Reset-drain, and stable-counter-pointer invariants are local code and
  unchanged. `go build ./...`, `go vet ./...`, `go test ./core`, and the full
  non-race `go test ./...` (Windows TempDir-cleanup flakes excepted) pass.
- 2026-06-27, v1.13.13 -> v1.13.14: revalidated against fork
  `shtorm-7/sing-box-extended v1.13.14-extended-2.5.0` (upstream `v1.13.14`).
  `adapter.ConnectionTracker` signature is unchanged. The project compiles
  against the new version (`go build ./...`, `go vet ./...` pass). Four new
  option fields surfaced in the coverage test (`MASQUEOutboundOptions.congestion_controller`,
  `MASQUEOutboundOptions.cwnd`, `OpenVPNOutboundOptions.ping_restart`,
  `FallbackOutboundOptions.blacklist_timeout`); all four were added to the
  corresponding TS interfaces in `frontend/src/types/outbounds.ts`.
  `go test ./core` passes. The sing-tun v0.8.9 pin was removed because the
  fork no longer uses the removed `MyInterface()` method.
