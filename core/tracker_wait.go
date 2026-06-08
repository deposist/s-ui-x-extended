package core

import (
	"sync"
	stdatomic "sync/atomic"
	"time"

	"github.com/deposist/s-ui-x/logger"
)

const trackerResetWaitTimeout = 5 * time.Second

type trackerWaitGroup struct {
	wg     sync.WaitGroup
	active stdatomic.Int64
}

func newTrackerWaitGroup() *trackerWaitGroup {
	return &trackerWaitGroup{}
}

func (g *trackerWaitGroup) Add() {
	g.wg.Add(1)
	g.active.Add(1)
}

func (g *trackerWaitGroup) Done() {
	g.active.Add(-1)
	g.wg.Done()
}

func (g *trackerWaitGroup) Active() int64 {
	return g.active.Load()
}

func waitForTrackerIdle(name string, group *trackerWaitGroup, timeout time.Duration) {
	if group == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		group.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		logger.Warningf("%s reset timed out waiting for %d active wrapped connections", name, group.Active())
	}
}
