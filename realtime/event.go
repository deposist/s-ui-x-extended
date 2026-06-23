package realtime

type Topic string

const (
	TopicOnlines           Topic = "onlines"
	TopicTrafficDelta      Topic = "traffic_delta"
	TopicCoreState         Topic = "core_state"
	TopicConfigInvalidated Topic = "config_invalidated"
	TopicRestartStatus     Topic = "restart_status"
	TopicNotification      Topic = "notification"
	TopicSecurityEvent     Topic = "security_event"
	TopicXUIImportProgress Topic = "xui_import_progress"
)

type Scope string

const (
	ScopeAdmin         Scope = "admin"
	ScopeRead          Scope = "read"
	ScopeWrite         Scope = "write"
	ScopeObservability Scope = "observability"
)

type Event struct {
	Type    Topic       `json:"type"`
	Ts      int64       `json:"ts"`
	Payload interface{} `json:"payload,omitempty"`
	// frame is the pre-marshalled JSON computed once per Publish and shared
	// across every subscriber, so a broadcast does not re-serialize the identical
	// event N times. Unexported: it is never part of the JSON output.
	frame []byte
}

// Frame returns the JSON frame the publisher pre-marshalled for broadcast, or
// nil when the consumer should marshal the event itself (e.g. per-connection
// events that were never broadcast).
func (e Event) Frame() []byte { return e.frame }

func topicAllowed(topic Topic, scope Scope) bool {
	if topic == TopicSecurityEvent || topic == TopicXUIImportProgress {
		return scope == ScopeAdmin
	}
	return true
}
