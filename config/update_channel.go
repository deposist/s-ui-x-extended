package config

// UpdateChannelMain and UpdateChannelBeta are the only valid panel self-update
// channels (spec FR-003): "main" tracks stable releases, "beta" tracks the
// newest release including pre-releases.
const (
	UpdateChannelMain = "main"
	UpdateChannelBeta = "beta"
)

// NormalizeUpdateChannel validates a channel value against the allowlist
// (SR-004) and falls back to "main" for anything unrecognized.
func NormalizeUpdateChannel(channel string) string {
	if channel == UpdateChannelBeta {
		return UpdateChannelBeta
	}
	return UpdateChannelMain
}
