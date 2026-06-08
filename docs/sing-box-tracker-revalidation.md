# Sing-box Tracker Revalidation Policy

Validated dependency:

- `github.com/sagernet/sing-box v1.13.12`

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
