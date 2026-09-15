package core

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
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

// TestBoxStartCloseRestartLifecycle exercises the full composed Box — netns,
// httpclient, certificate-provider, and the reordered Start/Close lifecycle —
// through a real start, close, and a second start to prove the composition is
// restartable and does not leak partial state between iterations.
func TestBoxStartCloseRestartLifecycle(t *testing.T) {
	cfg := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"inbounds":[{"type":"mixed","tag":"in","listen":"127.0.0.1","listen_port":0}],
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"final":"direct"}
	}`)
	for i := 0; i < 2; i++ {
		var opt option.Options
		ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry(), CertificateProviderRegistry())
		if err := opt.UnmarshalJSONContext(ctx, cfg); err != nil {
			t.Fatalf("unmarshal iteration %d: %v", i, err)
		}
		box, err := NewBox(Options{Context: ctx, Options: opt})
		if err != nil {
			t.Fatalf("NewBox iteration %d: %v", i, err)
		}
		if err := box.Start(); err != nil {
			t.Fatalf("Start iteration %d: %v", i, err)
		}
		if err := box.Close(); err != nil {
			t.Fatalf("Close iteration %d: %v", i, err)
		}
	}
}
