package cronjob

import (
	"context"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
)

type failoverGroupState struct {
	lastProbe   time.Time
	prevAllDown bool
	health      map[string]service.MemberHealth
}

// FailoverJob is the in-process auto-failover manager. Every base tick it probes
// the due groups' members (reusing CheckOutbound), runs the pure selection
// decision, and switches the active member via the sing-box selector. It holds
// NO core/selector pointers between ticks (re-resolves each cycle) and only
// per-group rolling health under its mutex — so it survives core restarts.
type FailoverJob struct {
	service.ConfigService

	mu     sync.Mutex
	states map[string]*failoverGroupState

	// now is injectable for deterministic tests.
	now func() time.Time
	// probe is injectable for tests; defaults to the live CheckOutbound.
	probe func(ctx context.Context, tag, target string) bool
	// switchMember is injectable for tests; defaults to the live core.
	switchMember func(groupTag, memberTag string) error
	// activeMember is injectable for tests; defaults to the live core.
	activeMember func(groupTag string) (string, bool)
	// alert fires once on the down->all-down edge; defaults to alertAllDown.
	alert func(group service.FailoverGroupConfig)
}

func NewFailoverJob() *FailoverJob {
	j := &FailoverJob{
		states: make(map[string]*failoverGroupState),
		now:    time.Now,
	}
	j.alert = j.alertAllDown
	return j
}

func (j *FailoverJob) Run() {
	db := database.GetDB()
	groups, err := service.LoadFailoverGroups(db)
	if err != nil {
		logger.Warning("failover: load groups: ", err)
		return
	}
	tags := make([]string, 0, len(groups))
	for _, group := range groups {
		tags = append(tags, group.Tag)
	}
	service.PruneFailoverLiveStatus(tags)
	if len(groups) == 0 {
		return
	}
	coreInst := service.DefaultRuntime().Core()
	if coreInst == nil || !coreInst.IsRunning() {
		return
	}
	directTag := service.DirectFallbackTag(db)
	for _, group := range groups {
		j.runGroup(coreInst, group, directTag)
	}
}

func (j *FailoverJob) runGroup(coreInst *core.Core, group service.FailoverGroupConfig, directTag string) {
	if !group.Enabled || len(group.Members) == 0 {
		return
	}

	j.mu.Lock()
	st := j.states[group.Tag]
	if st == nil {
		st = &failoverGroupState{health: map[string]service.MemberHealth{}}
		j.states[group.Tag] = st
	}
	due := st.lastProbe.IsZero() || j.now().Sub(st.lastProbe) >= group.Interval
	j.mu.Unlock()
	if !due {
		return
	}

	results := j.probeMembers(group)

	j.mu.Lock()
	for _, member := range group.Members {
		h := st.health[member]
		if results[member] {
			h.ConsecutiveUp++
			h.ConsecutiveDown = 0
		} else {
			h.ConsecutiveDown++
			h.ConsecutiveUp = 0
		}
		st.health[member] = h
	}
	st.lastProbe = j.now()
	snapshot := make(map[string]service.MemberHealth, len(st.health))
	for k, v := range st.health {
		snapshot[k] = v
	}
	j.mu.Unlock()

	j.persistState(group, snapshot)

	current, _ := j.now0(coreInst, group.Tag)

	fallback := ""
	if directTag != "" && !memberListContains(group.Members, directTag) {
		fallback = directTag
	}

	decision := service.SelectFailoverMember(service.FailoverDecisionInput{
		Members:        group.Members,
		Health:         snapshot,
		Current:        current,
		Hysteresis:     group.Hysteresis,
		DirectFallback: fallback,
	})

	active := current
	if decision.ShouldSwitch {
		if err := j.switch0(coreInst, group.Tag, decision.Target); err != nil {
			logger.Warning("failover: switch ", group.Tag, " -> ", decision.Target, ": ", err)
		} else {
			active = decision.Target
			logger.Info("failover: group ", group.Tag, " -> ", decision.Target, " (", decision.Reason, ")")
		}
	}

	// Publish the live status every due cycle (not only on a switch) so the UI
	// reflects per-member health without a dedicated poll.
	j.publishLiveStatus(group, snapshot, active, decision.AllDown)

	// Edge-triggered all-down alert: fire once on the down->all-down transition.
	j.mu.Lock()
	edge := decision.AllDown && !st.prevAllDown
	st.prevAllDown = decision.AllDown
	j.mu.Unlock()
	if edge && j.alert != nil {
		j.alert(group)
	}
}

// now0 reads the group's active member, preferring the test hook.
func (j *FailoverJob) now0(coreInst *core.Core, groupTag string) (string, bool) {
	if j.activeMember != nil {
		return j.activeMember(groupTag)
	}
	return coreInst.GroupNow(groupTag)
}

// switch0 applies a member switch, preferring the test hook.
func (j *FailoverJob) switch0(coreInst *core.Core, groupTag, memberTag string) error {
	if j.switchMember != nil {
		return j.switchMember(groupTag, memberTag)
	}
	return coreInst.SelectGroupMember(groupTag, memberTag)
}

func memberListContains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
