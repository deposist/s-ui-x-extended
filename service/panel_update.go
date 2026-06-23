package service

import (
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

// panelUpdateState holds the single allowed update job (FR-012, SR-009).
var panelUpdateState = struct {
	sync.Mutex
	job    *UpdateJob
	active bool
}{}

// panelUpdateExit terminates the process with a non-zero status so systemd
// (Restart=on-failure) brings the service back up on the freshly-replaced
// binary (research R1). Injectable for tests.
var panelUpdateExit = func() { os.Exit(1) }

// newPanelUpdateDeps / panelUpdateAuditSink are injection seams for tests.
var (
	newPanelUpdateDeps   = defaultPanelUpdateDeps
	panelUpdateAuditSink = writePanelUpdateAudit
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
	panelUpdateState.Unlock()

	go s.run(target)
	return nil
}

func (s *PanelUpdateService) run(target ReleaseTarget) {
	deps := newPanelUpdateDeps()
	if err := applyPipeline(target, deps, s.setStage); err != nil {
		s.fail(err, deps.execPath)
		return
	}
	s.setStage(UpdateStageRestarting)
	// Mark the swap pending so the next boot can roll back a non-starting new
	// binary (SR-012). If the marker cannot be written, the boot-time rollback
	// safety net would be gone - restore the previous binary and abort the
	// restart rather than boot unprotected, instead of swallowing the error.
	if err := writePendingMarker(deps.execPath); err != nil {
		s.fail(fmt.Errorf("rollback marker could not be written after apply: %w", err), deps.execPath)
		return
	}
	// Durably record the successful outcome (SR-006/SC-008) BEFORE exiting, since
	// os.Exit bypasses the async audit writer's flush.
	panelUpdateAuditSink(s.Status(), "applied", "")
	logger.Info("panel update: applied", target.Version, "- restarting into new binary")
	panelUpdateExit()
}

func (s *PanelUpdateService) setStage(stage UpdateStage) {
	panelUpdateState.Lock()
	defer panelUpdateState.Unlock()
	if panelUpdateState.job != nil {
		panelUpdateState.job.Stage = stage
	}
}

// fail marks the job failed, releases the guard, and best-effort restores the
// previous binary from backup so a partial apply never leaves a broken panel
// (SR-007). The pipeline only touches the live binary at the final atomic
// rename, so in practice the backup restore is a belt-and-braces safety net.
func (s *PanelUpdateService) fail(err error, execPath string) {
	logger.Warning("panel update failed:", err)
	if execPath != "" {
		if restoreErr := RestoreBackup(execPath); restoreErr != nil && !os.IsNotExist(restoreErr) {
			logger.Warning("panel update: backup restore failed:", restoreErr)
		}
	}
	panelUpdateState.Lock()
	var job UpdateJob
	if panelUpdateState.job != nil {
		panelUpdateState.job.Stage = UpdateStageFailed
		panelUpdateState.job.Error = redact.String(err.Error())
		job = *panelUpdateState.job
	}
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
	panelUpdateState.job = nil
	panelUpdateState.active = false
	panelUpdateState.Unlock()
}
