package common

import "testing"

func TestNewErrorPreservesArgumentSpacingWithoutTrailingNewline(t *testing.T) {
	err := NewError("unknown action: ", "set")
	if err.Error() != "unknown action:  set" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}
