package core

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sagernet/sing-box/log"
)

type lifecycleTestLogFactory struct {
	startErr   error
	closeCalls atomic.Int32
}

func (f *lifecycleTestLogFactory) Start() error              { return f.startErr }
func (f *lifecycleTestLogFactory) Close() error              { f.closeCalls.Add(1); return nil }
func (f *lifecycleTestLogFactory) Level() log.Level          { return log.LevelTrace }
func (f *lifecycleTestLogFactory) SetLevel(log.Level)        {}
func (f *lifecycleTestLogFactory) Logger() log.ContextLogger { return log.NewNOPFactory().Logger() }
func (f *lifecycleTestLogFactory) NewLogger(string) log.ContextLogger {
	return log.NewNOPFactory().Logger()
}

func TestBoxStartRollsBackWhenLoggerStartFails(t *testing.T) {
	startErr := errors.New("start failed")
	factory := &lifecycleTestLogFactory{startErr: startErr}
	box := &Box{
		logFactory: factory,
		logger:     factory.Logger(),
		done:       make(chan struct{}),
	}

	if err := box.Start(); err == nil || !strings.Contains(err.Error(), startErr.Error()) {
		t.Fatalf("Start() error = %v, want message containing %q", err, startErr)
	}
	if calls := factory.closeCalls.Load(); calls != 1 {
		t.Fatalf("log factory Close() calls = %d, want 1", calls)
	}
	select {
	case <-box.done:
	default:
		t.Fatal("Box.Start failure did not close done")
	}
}

func TestBoxCloseIsConcurrentSafe(t *testing.T) {
	factory := &lifecycleTestLogFactory{}
	box := &Box{
		logFactory: factory,
		logger:     factory.Logger(),
		done:       make(chan struct{}),
	}
	const callers = 16
	var wg sync.WaitGroup
	wg.Add(callers)

	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			if err := box.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if calls := factory.closeCalls.Load(); calls != 1 {
		t.Fatalf("log factory Close() calls = %d, want 1", calls)
	}
}
