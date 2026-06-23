package service

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
)

const restartSignalDelay = 3 * time.Second

type restartManager struct {
	mu           sync.Mutex
	cond         *sync.Cond
	inFlight     bool
	pendingTimer *time.Timer
	signalDelay  time.Duration
	signal       func() error
}

type RestartScheduler interface {
	ScheduleRestart(delay time.Duration) error
}

func init() {
	database.SetSendSighupHook(func() error {
		manager := DefaultRuntime().restart()
		if manager == nil {
			return nil
		}
		return manager.sendSighup()
	})
}

func newRestartManager(signalDelay time.Duration, signal func() error) *restartManager {
	m := &restartManager{
		signalDelay: signalDelay,
		signal:      signal,
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func StopRestartManager() {
	manager := DefaultRuntime().restart()
	if manager != nil {
		manager.cancelPending()
	}
}

func (m *restartManager) run(operation func() error) error {
	if !m.begin() {
		return nil
	}
	defer m.end()
	return operation()
}

// runBlocking waits for any in-flight operation (including a pending restart
// signal) to finish and then runs the operation exclusively. Unlike run, it
// never skips: callers use it when the operation must not be silently dropped,
// e.g. when syncing the core with just-committed configuration.
func (m *restartManager) runBlocking(operation func() error) error {
	m.beginBlocking()
	defer m.end()
	return operation()
}

func (m *restartManager) sendSighup() error {
	return m.ScheduleRestart(m.signalDelay)
}

func (m *restartManager) ScheduleRestart(delay time.Duration) error {
	if delay <= 0 {
		delay = m.signalDelay
	}
	if !m.begin() {
		return nil
	}
	m.armRestartTimer(delay)
	return nil
}

// scheduleRestartBlocking schedules a restart that must NOT be silently dropped
// when it matters. Used by the IP-certificate renewal cron after the new cert
// settings are written: a full restart is what reloads them.
//
//   - A SIGHUP restart is already pending (pendingTimer != nil): it will fire
//     after our settings write and reload the cert, so this call is a safe no-op
//     (matches the self-healing case).
//   - Another op is in flight but no restart is pending (e.g. a core runBlocking
//     sync): WAIT for it to finish, then arm the restart, so the panel restart is
//     not silently dropped (which would keep serving the old certificate).
//   - Nothing in flight: arm the restart immediately.
func (m *restartManager) scheduleRestartBlocking(delay time.Duration) error {
	if delay <= 0 {
		delay = m.signalDelay
	}
	m.mu.Lock()
	for m.inFlight {
		if m.pendingTimer != nil {
			m.mu.Unlock()
			return nil
		}
		m.cond.Wait()
	}
	m.inFlight = true
	m.mu.Unlock()
	m.armRestartTimer(delay)
	return nil
}

// armRestartTimer publishes the pending SIGHUP timer under the mutex. The caller
// must already hold the in-flight slot (begin/beginBlocking). The timer callback
// may fire concurrently, so the captured timer variable is read under the same
// mutex it is published with.
func (m *restartManager) armRestartTimer(delay time.Duration) {
	m.mu.Lock()
	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		m.mu.Lock()
		self := timer
		m.mu.Unlock()
		defer m.endPending(self)
		if err := m.signal(); err != nil {
			logger.Error("send signal SIGHUP failed:", err)
		}
	})
	m.pendingTimer = timer
	m.mu.Unlock()
}

func (m *restartManager) cancelPending() {
	m.mu.Lock()
	timer := m.pendingTimer
	if timer == nil {
		m.mu.Unlock()
		return
	}
	m.pendingTimer = nil
	if timer.Stop() {
		m.inFlight = false
		m.cond.Broadcast()
	}
	m.mu.Unlock()
}

func (m *restartManager) begin() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inFlight {
		return false
	}
	m.inFlight = true
	return true
}

func (m *restartManager) beginBlocking() {
	m.mu.Lock()
	for m.inFlight {
		m.cond.Wait()
	}
	m.inFlight = true
	m.mu.Unlock()
}

func (m *restartManager) end() {
	m.mu.Lock()
	m.inFlight = false
	m.cond.Broadcast()
	m.mu.Unlock()
}

func (m *restartManager) endPending(timer *time.Timer) {
	m.mu.Lock()
	if m.pendingTimer == timer {
		m.pendingTimer = nil
	}
	m.inFlight = false
	m.cond.Broadcast()
	m.mu.Unlock()
}

func signalCurrentProcess() error {
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return process.Kill()
	}
	return process.Signal(syscall.SIGHUP)
}
