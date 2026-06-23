package service

import (
	"github.com/deposist/s-ui-x-extended/logger"
)

// postCommitCorePlan describes how to bring the running core in sync with the
// configuration that was just committed: either a full restart, or per-object
// tag removals plus reloads of the affected entities.
type postCommitCorePlan struct {
	needsCoreRestart bool
	restartReason    string

	inboundIds  []uint
	serviceIds  []uint
	outboundIds []uint
	endpointIds []uint

	removedInboundTags  []string
	removedServiceTags  []string
	removedOutboundTags []string
	removedEndpointTags []string
}

func (p postCommitCorePlan) hasObjectChanges() bool {
	return len(p.inboundIds) > 0 || len(p.serviceIds) > 0 ||
		len(p.outboundIds) > 0 || len(p.endpointIds) > 0 ||
		len(p.removedInboundTags) > 0 || len(p.removedServiceTags) > 0 ||
		len(p.removedOutboundTags) > 0 || len(p.removedEndpointTags) > 0
}

// entityCoreChange is returned by the per-entity Save methods to describe the
// core-level consequence of the committed change.
type entityCoreChange struct {
	needsRestart      bool
	restartReason     string
	reloadIds         []uint
	removeTags        []string
	cascadeServiceIds []uint
}

func (p *postCommitCorePlan) mergeRestart(change *entityCoreChange) {
	if !change.needsRestart {
		return
	}
	p.needsCoreRestart = true
	if change.restartReason != "" {
		p.restartReason = change.restartReason
	}
}

func (p *postCommitCorePlan) mergeServiceChange(change *entityCoreChange) {
	if change == nil {
		return
	}
	p.mergeRestart(change)
	p.serviceIds = append(p.serviceIds, change.reloadIds...)
	p.removedServiceTags = append(p.removedServiceTags, change.removeTags...)
}

func (p *postCommitCorePlan) mergeInboundChange(change *entityCoreChange) {
	if change == nil {
		return
	}
	p.mergeRestart(change)
	p.inboundIds = append(p.inboundIds, change.reloadIds...)
	p.removedInboundTags = append(p.removedInboundTags, change.removeTags...)
	p.serviceIds = append(p.serviceIds, change.cascadeServiceIds...)
}

func (p *postCommitCorePlan) mergeOutboundChange(change *entityCoreChange) {
	if change == nil {
		return
	}
	p.mergeRestart(change)
	p.outboundIds = append(p.outboundIds, change.reloadIds...)
	p.removedOutboundTags = append(p.removedOutboundTags, change.removeTags...)
}

func (p *postCommitCorePlan) mergeEndpointChange(change *entityCoreChange) {
	if change == nil {
		return
	}
	p.mergeRestart(change)
	p.endpointIds = append(p.endpointIds, change.reloadIds...)
	p.removedEndpointTags = append(p.removedEndpointTags, change.removeTags...)
}

// applyPostCommitCoreChanges brings the running core in sync with the
// just-committed configuration: a full restart for config-level changes, or a
// hot reload of only the affected objects. A failed hot reload falls back to a
// full restart so the core never keeps serving a partially updated state.
// The whole apply runs as one exclusive restart-manager section, so it cannot
// interleave with a user-initiated restart or the cron core starter, and it is
// never silently skipped.
func (s *ConfigService) applyPostCommitCoreChanges(plan postCommitCorePlan) {
	if s.coreInstance() == nil {
		return
	}
	manager := s.runtime().restart()
	if manager == nil {
		logger.Warning("sing-box post-save sync skipped: restart manager not initialized")
		return
	}
	_ = manager.runBlocking(func() error {
		s.applyPostCommitCoreChangesLocked(plan)
		return nil
	})
}

func (s *ConfigService) applyPostCommitCoreChangesLocked(plan postCommitCorePlan) {
	coreInstance := s.coreInstance()
	if coreInstance == nil {
		return
	}
	if plan.needsCoreRestart {
		if plan.restartReason != "" {
			logger.Info("sing-box full restart after save: ", plan.restartReason)
		}
		if coreInstance.IsRunning() {
			if restartErr := s.restartCoreLocked(); restartErr != nil {
				logger.Warning("sing-box restart after save failed: ", restartErr)
			}
		} else if startErr := s.startCoreLocked(true); startErr != nil {
			logger.Warning("sing-box start after save failed: ", startErr)
		}
		return
	}
	if !coreInstance.IsRunning() {
		if startErr := s.startCoreLocked(true); startErr != nil {
			logger.Warning("sing-box start after save failed: ", startErr)
		}
		return
	}
	if !plan.hasObjectChanges() {
		return
	}
	reloadErr := s.applyObjectChangesLocked(plan)
	if reloadErr == nil {
		return
	}
	logger.Warning("sing-box partial reload after save failed: ", reloadErr)
	if restartErr := s.restartCoreLocked(); restartErr != nil {
		logger.Error("sing-box restart after failed partial reload also failed; core may be out of sync: ", restartErr)
	}
}

// applyObjectChangesLocked performs tag removals first, then reloads in
// dependency-safe order (outbounds → endpoints → inbounds → services) so that
// recreated adapters are registered before their dependents are recreated.
func (s *ConfigService) applyObjectChangesLocked(plan postCommitCorePlan) error {
	if err := s.applyObjectRemovalsLocked(plan); err != nil {
		return err
	}
	if len(plan.outboundIds) > 0 {
		if err := restartOutboundsAfterSave(s, plan.outboundIds); err != nil {
			return err
		}
	}
	if len(plan.endpointIds) > 0 {
		if err := restartEndpointsAfterSave(s, plan.endpointIds); err != nil {
			return err
		}
	}
	if len(plan.inboundIds) > 0 {
		if err := restartInboundsAfterSave(s, plan.inboundIds); err != nil {
			return err
		}
	}
	if len(plan.serviceIds) > 0 {
		if err := restartServicesAfterSave(s, plan.serviceIds); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConfigService) applyObjectRemovalsLocked(plan postCommitCorePlan) error {
	if len(plan.removedOutboundTags) > 0 {
		if err := removeOutboundsFromCoreAfterSave(s, plan.removedOutboundTags); err != nil {
			return err
		}
	}
	if len(plan.removedEndpointTags) > 0 {
		if err := removeEndpointsFromCoreAfterSave(s, plan.removedEndpointTags); err != nil {
			return err
		}
	}
	if len(plan.removedInboundTags) > 0 {
		if err := removeInboundsFromCoreAfterSave(s, plan.removedInboundTags); err != nil {
			return err
		}
	}
	if len(plan.removedServiceTags) > 0 {
		if err := removeServicesFromCoreAfterSave(s, plan.removedServiceTags); err != nil {
			return err
		}
	}
	return nil
}
