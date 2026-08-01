package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/redact"
)

// UpdateStage is the current step of an in-flight panel self-update.
type UpdateStage string

const (
	UpdateStageIdle        UpdateStage = "idle"
	UpdateStageDownloading UpdateStage = "downloading"
	UpdateStageVerifying   UpdateStage = "verifying"
	UpdateStageApplying    UpdateStage = "applying"
	UpdateStageRestarting  UpdateStage = "restarting"
	UpdateStageFailed      UpdateStage = "failed"
)

// UpdateJob is the ephemeral, in-memory state of the single allowed update.
type UpdateJob struct {
	ID          string      `json:"id"`
	Channel     string      `json:"channel"`
	FromVersion string      `json:"fromVersion"`
	ToVersion   string      `json:"toVersion"`
	Stage       UpdateStage `json:"stage"`
	Error       string      `json:"error,omitempty"`
	StartedAt   int64       `json:"startedAt"`
	Initiator   string      `json:"initiator,omitempty"`
}

type PanelUpdateService struct {
	Runtime *Runtime
}

var errUpdateInProgress = errors.New("an update is already in progress")
var errUpdateRecoveryRequired = errors.New("a previous update requires recovery")

// ErrPanelUpdateRolledBack tells the process entrypoint to exit non-zero so the
// service manager restarts into the restored known-good executable.
var ErrPanelUpdateRolledBack = errors.New("self-update rolled back; restart required")

type panelUpdateProcessLock interface {
	release() error
}

// panelUpdateState holds the single allowed update job (FR-012, SR-009).
var panelUpdateState = struct {
	sync.Mutex
	job    *UpdateJob
	active bool
	cancel context.CancelFunc
	done   chan struct{}
}{}

// panelUpdateExit terminates the process with a non-zero status so systemd
// (Restart=on-failure) brings the service back up on the freshly-replaced
// binary (research R1). Injectable for tests.
var panelUpdateExit = func() { os.Exit(1) }

// newPanelUpdateDeps / panelUpdateAuditSink are injection seams for tests.
var (
	newPanelUpdateDeps      = defaultPanelUpdateDeps
	panelUpdateAuditSink    = writePanelUpdateAudit
	panelUpdateMarkerWriter = writePendingMarkerForCandidate
)

// Status returns a snapshot of the current/last update job (idle if none).
func (s *PanelUpdateService) Status() UpdateJob {
	panelUpdateState.Lock()
	defer panelUpdateState.Unlock()
	if panelUpdateState.job == nil {
		return UpdateJob{Stage: UpdateStageIdle}
	}
	return *panelUpdateState.job
}

// InProgress reports whether an update is currently running.
func (s *PanelUpdateService) InProgress() bool {
	panelUpdateState.Lock()
	defer panelUpdateState.Unlock()
	return panelUpdateState.active
}

// Apply starts an update to target. Only one update may run at a time (FR-012).
// It returns immediately; progress is observed via Status() (the client polls
// GET /api/update/status). The caller MUST have already validated step-up auth
// (SR-010), the target version (FR-016), and that target is newer (FR-013).
func (s *PanelUpdateService) Apply(target ReleaseTarget, initiator string) error {
	panelUpdateState.Lock()
	if panelUpdateState.active {
		panelUpdateState.Unlock()
		return errUpdateInProgress
	}
	deps := newPanelUpdateDeps()
	if deps.execPath != "" {
		if _, err := os.Stat(deps.execPath + pendingSuffix); err == nil {
			panelUpdateState.Unlock()
			return errUpdateRecoveryRequired
		} else if !errors.Is(err, os.ErrNotExist) {
			panelUpdateState.Unlock()
			return fmt.Errorf("inspect previous update state: %w", err)
		}
	}
	processLock, err := acquirePanelUpdateProcessLock(deps.execPath)
	if err != nil {
		panelUpdateState.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	startedAt := time.Now().Unix()
	panelUpdateState.active = true
	panelUpdateState.job = &UpdateJob{
		ID:          fmt.Sprintf("upd-%d", startedAt),
		Channel:     target.Channel,
		FromVersion: config.GetVersion(),
		ToVersion:   target.Version,
		Stage:       UpdateStageDownloading,
		StartedAt:   startedAt,
		Initiator:   initiator,
	}
	panelUpdateState.cancel = cancel
	panelUpdateState.done = done
	panelUpdateState.Unlock()

	go s.run(ctx, target, deps, processLock, done)
	return nil
}

func (s *PanelUpdateService) run(ctx context.Context, target ReleaseTarget, deps panelUpdateDeps, processLock panelUpdateProcessLock, done chan struct{}) {
	defer close(done)
	defer func() {
		if processLock != nil {
			if err := processLock.release(); err != nil {
				logger.Warning("panel update: release process lock: ", err)
			}
		}
	}()
	swapped, err := applyPipeline(ctx, target, deps, s.setStage)
	if err != nil {
		s.fail(err, deps.execPath, swapped)
		return
	}
	s.setStage(UpdateStageRestarting)
	// Durably record the successful outcome (SR-006/SC-008) BEFORE exiting, since
	// os.Exit bypasses the async audit writer's flush.
	panelUpdateAuditSink(s.Status(), "applied", "")
	logger.Info("panel update: applied", target.Version, "- restarting into new binary")
	panelUpdateExit()
}

// StopPanelUpdate cancels an active pre-swap operation and waits for its
// goroutine to finish. Cancellation cannot interrupt the atomic binary swap;
// once the applying stage begins the process proceeds to restart or rollback.
func StopPanelUpdate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	panelUpdateState.Lock()
	cancel := panelUpdateState.cancel
	done := panelUpdateState.done
	panelUpdateState.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *PanelUpdateService) setStage(stage UpdateStage) {
	panelUpdateState.Lock()
	defer panelUpdateState.Unlock()
	if panelUpdateState.job != nil {
		panelUpdateState.job.Stage = stage
	}
}

// fail marks the job failed, releases the guard, and restores only a binary
func (s *PanelUpdateService) fail(err error, execPath string, swapped bool) {
	logger.Warning("panel update failed:", err)
	if swapped && execPath != "" {
		if restoreErr := rollbackCurrentUpdate(execPath); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("automatic rollback failed: %w", restoreErr))
			logger.Error("panel update: automatic rollback remains unresolved: ", restoreErr)
		}
	}
	panelUpdateState.Lock()
	var job UpdateJob
	if panelUpdateState.job != nil {
		panelUpdateState.job.Stage = UpdateStageFailed
		panelUpdateState.job.Error = redact.String(err.Error())
		job = *panelUpdateState.job
	}
	panelUpdateState.cancel = nil
	panelUpdateState.done = nil
	panelUpdateState.active = false
	panelUpdateState.Unlock()
	// Record the failed outcome durably (SR-006/SC-008).
	panelUpdateAuditSink(job, "failed", err.Error())
}

// writePanelUpdateAudit durably records the terminal outcome of an apply
// (SR-006, SC-008). It writes SYNCHRONOUSLY because the success path exits the
// process (os.Exit) before the async audit writer would flush. The password is
// never part of the job, so it cannot leak here (CHK025).
func writePanelUpdateAudit(job UpdateJob, result string, errMsg string) {
	if job.ID == "" {
		return
	}
	details := map[string]any{
		"channel": job.Channel,
		"from":    job.FromVersion,
		"to":      job.ToVersion,
		"result":  result,
	}
	if errMsg != "" {
		details["error"] = redact.String(errMsg)
	}
	severity := AuditSeverityInfo
	if result != "applied" {
		severity = AuditSeverityWarn
	}
	record, err := buildAuditRecord(AuditEvent{
		Actor:    job.Initiator,
		Event:    "panel_update_apply",
		Resource: "update",
		Severity: severity,
		Details:  details,
	})
	if err != nil {
		logger.Warning("panel update: audit build failed:", err)
		return
	}
	if err := writeAuditEvents([]model.AuditEvent{record}); err != nil {
		logger.Warning("panel update: audit write failed:", err)
	}
}

// resetPanelUpdateStateForTest clears the singleton between tests.
func resetPanelUpdateStateForTest() {
	panelUpdateState.Lock()
	if panelUpdateState.cancel != nil {
		panelUpdateState.cancel()
	}
	panelUpdateState.job = nil
	panelUpdateState.active = false
	panelUpdateState.cancel = nil
	panelUpdateState.done = nil
	panelUpdateState.Unlock()
}
