package service

// MemberHealth is the rolling probe outcome for a single failover member.
type MemberHealth struct {
	ConsecutiveUp   int
	ConsecutiveDown int
}

// FailoverDecisionInput is the pure input to the failover selection decision.
type FailoverDecisionInput struct {
	Members        []string                // ordered priority; index 0 = primary
	Health         map[string]MemberHealth // keyed by member tag (priority members only)
	Current        string                  // current active member (= GroupNow()); "" if unknown
	Hysteresis     int                     // >= 1; consecutive-up samples required for failback
	DirectFallback string                  // direct outbound tag used when all members are down; "" if none
}

// FailoverDecision is the pure output: which member should be active and why.
type FailoverDecision struct {
	Target       string
	ShouldSwitch bool
	AllDown      bool
	Reason       string // priority | failover | failback | sticky | all_down_direct | all_down_hold
}

// SelectFailoverMember decides the active member for a failover group from a
// health snapshot. It is a pure function (no I/O, no clock, no core) so it is
// exhaustively table-testable. Policy: strict priority; failover-down is
// immediate (one failed probe = a 15s timeout); failback-up is gated by
// Hysteresis consecutive healthy samples; sticky otherwise; all-down routes to
// DirectFallback when set, else holds the senior member.
func SelectFailoverMember(in FailoverDecisionInput) FailoverDecision {
	hysteresis := in.Hysteresis
	if hysteresis < 1 {
		hysteresis = DefaultHysteresis
	}

	indexOf := func(tag string) int {
		for i, m := range in.Members {
			if m == tag {
				return i
			}
		}
		return -1
	}
	isUp := func(tag string) bool { return in.Health[tag].ConsecutiveUp >= 1 }
	isConfirmed := func(tag string) bool { return in.Health[tag].ConsecutiveUp >= hysteresis }

	bestUp, bestConfirmed := "", ""
	for _, m := range in.Members {
		if bestUp == "" && isUp(m) {
			bestUp = m
		}
		if bestConfirmed == "" && isConfirmed(m) {
			bestConfirmed = m
		}
	}

	decide := func(target, reason string, allDown bool) FailoverDecision {
		return FailoverDecision{
			Target:       target,
			ShouldSwitch: target != in.Current,
			AllDown:      allDown,
			Reason:       reason,
		}
	}

	// All members down → direct fallback if available, else hold the senior.
	if bestUp == "" {
		if in.DirectFallback != "" {
			return decide(in.DirectFallback, "all_down_direct", true)
		}
		return decide(in.Members[0], "all_down_hold", true)
	}

	// Cold start: nothing selected yet → highest-priority up member.
	if in.Current == "" {
		return decide(bestUp, "priority", false)
	}

	// Current is not a priority member (e.g. the direct fallback after an
	// all-down period) → restore service immediately to the best up member.
	currentIdx := indexOf(in.Current)
	if currentIdx < 0 {
		return decide(bestUp, "failback", false)
	}

	// Current down → fast failover to the best available member.
	if !isUp(in.Current) {
		return decide(bestUp, "failover", false)
	}

	// Current up: fail back only to a CONFIRMED strictly-higher-priority member.
	if bestConfirmed != "" && indexOf(bestConfirmed) < currentIdx {
		return decide(bestConfirmed, "failback", false)
	}

	// Otherwise stay put.
	return decide(in.Current, "sticky", false)
}
