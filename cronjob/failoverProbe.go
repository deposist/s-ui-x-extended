package cronjob

import (
	"context"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"
	"github.com/deposist/s-ui-x-extended/service"
)

// failoverProbeConcurrency bounds how many member probes run at once per cycle,
// matching DoctorService.outboundChecks so a many-member panel can't exhaust
// dialers.
const failoverProbeConcurrency = 4

func (j *FailoverJob) probeMembers(group service.FailoverGroupConfig) map[string]service.OutboundHealthSnapshot {
	results := make(map[string]service.OutboundHealthSnapshot, len(group.Members))
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
			var delay uint16
			var errMsg string
			if j.probe != nil {
				ok = j.probe(ctx, tag, group.ProbeTarget)
			} else {
				check := j.ConfigService.CheckOutboundWithContext(ctx, tag, group.ProbeTarget)
				ok = check.OK
				delay = check.Delay
				errMsg = check.Error
			}
			snapshot := service.RecordOutboundHealth(tag, ok, delay, errMsg, time.Now())
			mu.Lock()
			results[tag] = snapshot
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
			LastDelayMs: h.LastDelayMs,
			LastError:   h.LastError,
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
		h := snapshot[member]
		members = append(members, service.FailoverMemberStatus{
			Tag:      member,
			Healthy:  h.ConsecutiveUp >= 1,
			Priority: i,
			DelayMs:  h.LastDelayMs,
			Error:    h.LastError,
		})
	}
	service.SetFailoverLiveStatus(service.FailoverStatusEntry{
		Tag:           group.Tag,
		Active:        active,
		AllDown:       allDown,
		AllDownPolicy: group.AllDownPolicy,
		Members:       members,
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
		"warning":       "failover_all_down",
		"group":         group.Tag,
		"allDownPolicy": group.AllDownPolicy,
		"members":       group.Members,
	})
}
