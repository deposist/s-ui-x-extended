package core

import "testing"

// TestStartReturnsErrorOnMalformedConfig pins the M7 fix: Core.Start must
// surface a config-unmarshal failure instead of swallowing it. Before the fix,
// Start only logged the error and fell through to NewBox+Start on an empty
// option set (which succeeds because nothing listens), returning nil — so the
// caller marked the core "running" while no inbound was actually serving.
func TestStartReturnsErrorOnMalformedConfig(t *testing.T) {
	c := NewCore()

	err := c.Start([]byte("{ this is not valid sing-box json"))
	if err == nil {
		t.Fatal("Start must return an error when the config cannot be unmarshaled")
	}
	if c.IsRunning() {
		t.Fatal("core must not report running after a failed config parse")
	}
	if c.GetInstance() != nil {
		t.Fatal("core must not retain a box instance after a failed config parse")
	}
}
