package paidsub

import (
	"encoding/json"
	"strings"
	"testing"
)

// The frontend's isMsg() requires the keys success, msg AND obj to all be
// present; omitempty on any of them makes the client report "unknown data".
func TestApiMsgAlwaysIncludesAllKeys(t *testing.T) {
	for _, m := range []apiMsg{
		{Success: true},               // respOK(c, nil)
		{Success: true, Obj: []int{}}, // respOK(c, list)
		{Success: false, Msg: "x"},    // respFail
	} {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, key := range []string{`"success"`, `"msg"`, `"obj"`} {
			if !strings.Contains(s, key) {
				t.Errorf("apiMsg JSON %s missing required key %s", s, key)
			}
		}
	}
}
