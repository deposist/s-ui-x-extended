package cronjob

import (
	"context"
	"sync"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"
	"github.com/deposist/s-ui-x-extended/service"
)

// failoverProbeConcurrency bounds how many member probes run at once per cycle,
// matching DoctorService.outboundChecks so a many-member panel can't exhaust
// dialers.
const failoverProbeConcurrency = 4

func (j *FailoverJob) probeMembers(group service.FailoverGroupConfig) map[string]bool {
	results := make(map[string]bool, len(group.Members))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, failoverProbeConcurrency)
	// TODO: thread a job-level context from Run() through runGroup() so probes
	// cancel promptly on panel shutdown instead of running to group.Interval.
	ctx, cancel := context.WithTimeout(context.Background(), group.Interval)
	defer cancel()
	for _, member := range group.Members {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var ok bool
			if j.probe != nil {
				ok = j.probe(ctx, tag, group.ProbeTarget)
			} else {
				ok = j.ConfigService.CheckOutboundWithContext(ctx, tag, group.ProbeTarget).OK
			}
			mu.Lock()
			results[tag] = ok
			mu.Unlock()
		}(member)
	}
	wg.Wait()
	return results
}

// persistState records the per-member health to the observability table. A nil
// database (unit tests) makes this a no-op.
func (j *FailoverJob) persistState(group service.FailoverGroupConfig, snapshot map[string]service.MemberHealth) {
	nowUnix := j.now().Unix()
	states := make([]service.FailoverMemberState, 0, len(group.Members))
	for _, m := range group.Members {
		h := snapshot[m]
		states = append(states, service.FailoverMemberState{
			GroupTag:    group.Tag,
			MemberTag:   m,
			Healthy:     h.ConsecutiveUp >= 1,
			ConsecUp:    h.ConsecutiveUp,
			ConsecDown:  h.ConsecutiveDown,
			LastProbeAt: nowUnix,
		})
	}
	if err := service.WriteFailoverMemberStates(database.GetDB(), states); err != nil {
		logger.Warning("failover: persist state for ", group.Tag, ": ", err)
	}
}

// publishLiveStatus records the group's current live status (active member +
// per-member health) into the in-memory snapshot stats.go merges into the
// onlines push - the live channel that replaces the UI's status poll.
func (j *FailoverJob) publishLiveStatus(group service.FailoverGroupConfig, snapshot map[string]service.MemberHealth, active string, allDown bool) {
	members := make([]service.FailoverMemberStatus, 0, len(group.Members))
	for i, member := range group.Members {
		members = append(members, service.FailoverMemberStatus{
			Tag:      member,
			Healthy:  snapshot[member].ConsecutiveUp >= 1,
			Priority: i,
		})
	}
	service.SetFailoverLiveStatus(service.FailoverStatusEntry{
		Tag:     group.Tag,
		Active:  active,
		AllDown: allDown,
		Members: members,
	})
}

// alertAllDown surfaces the down->all-down edge through the durable audit log,
// a realtime warning, and a panel log line (which the UI also toasts) - reusing
// the existing audit + realtime warning machinery.
func (j *FailoverJob) alertAllDown(group service.FailoverGroupConfig) {
	logger.Warning("failover: all members down for group ", group.Tag)
	_ = (&service.AuditService{}).Record(service.AuditEvent{
		Actor:    "system",
		Event:    "failover_all_down",
		Resource: "outbound",
		Severity: service.AuditSeverityWarn,
		Details:  map[string]any{"group": group.Tag, "members": group.Members},
	})
	realtime.Publish(realtime.TopicCoreState, map[string]any{
		"warning": "failover_all_down",
		"group":   group.Tag,
	})
}

