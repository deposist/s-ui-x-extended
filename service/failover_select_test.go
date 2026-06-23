package service

import "testing"

func mh(up, down int) MemberHealth { return MemberHealth{ConsecutiveUp: up, ConsecutiveDown: down} }

func TestSelectFailoverMember(t *testing.T) {
	cases := []struct {
		name        string
		members     []string
		health      map[string]MemberHealth
		current     string
		hysteresis  int
		direct      string
		wantTarget  string
		wantSwitch  bool
		wantAllDown bool
		wantReason  string
	}{
		{"sticky primary up", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(3, 0)}, "a", 2, "direct", "a", false, false, "sticky"},
		{"failover primary down", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(0, 1), "b": mh(2, 0)}, "a", 2, "direct", "b", true, false, "failover"},
		{"cascade down", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(0, 2), "b": mh(0, 1), "c": mh(2, 0)}, "b", 2, "direct", "c", true, false, "failover"},
		{"sticky while primary recovering", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(1, 0), "b": mh(3, 0)}, "b", 2, "direct", "b", false, false, "sticky"},
		{"failback once confirmed", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(2, 0), "b": mh(3, 0)}, "b", 2, "direct", "a", true, false, "failback"},
		{"all down -> direct", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(0, 3), "b": mh(0, 2), "c": mh(0, 1)}, "b", 2, "direct", "direct", true, true, "all_down_direct"},
		{"all down -> hold senior", []string{"a", "b", "c"}, map[string]MemberHealth{"a": mh(0, 3), "b": mh(0, 2), "c": mh(0, 1)}, "b", 2, "", "a", true, true, "all_down_hold"},
		{"single member down -> direct", []string{"a"}, map[string]MemberHealth{"a": mh(0, 1)}, "a", 2, "direct", "direct", true, true, "all_down_direct"},
		{"single member down -> hold", []string{"a"}, map[string]MemberHealth{"a": mh(0, 1)}, "a", 2, "", "a", false, true, "all_down_hold"},
		{"cold start", []string{"a", "b"}, map[string]MemberHealth{"a": mh(1, 0), "b": mh(1, 0)}, "", 2, "direct", "a", true, false, "priority"},
		{"recover from all-down", []string{"a", "b"}, map[string]MemberHealth{"a": mh(2, 0), "b": mh(3, 0)}, "direct", 2, "direct", "a", true, false, "failback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SelectFailoverMember(FailoverDecisionInput{
				Members:        tc.members,
				Health:         tc.health,
				Current:        tc.current,
				Hysteresis:     tc.hysteresis,
				DirectFallback: tc.direct,
			})
			if got.Target != tc.wantTarget || got.ShouldSwitch != tc.wantSwitch ||
				got.AllDown != tc.wantAllDown || got.Reason != tc.wantReason {
				t.Fatalf("got %+v; want target=%q switch=%v allDown=%v reason=%q",
					got, tc.wantTarget, tc.wantSwitch, tc.wantAllDown, tc.wantReason)
			}
		})
	}
}
