package core

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
)

func TestStatsTrackerResetPreservesConcurrentCounterPointers(t *testing.T) {
	tracker := NewStatsTracker()
	readCounters, writeCounters := tracker.getReadCounters("in", "out", "user")
	if len(readCounters) != 3 || len(writeCounters) != 3 {
		t.Fatalf("unexpected counter count: read=%d write=%d", len(readCounters), len(writeCounters))
	}

	const goroutines = 8
	const increments = 1000
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				for _, counter := range readCounters {
					counter.Add(1)
				}
				for _, counter := range writeCounters {
					counter.Add(1)
				}
				if j%17 == 0 {
					tracker.Reset()
				}
			}
		}()
	}
	wg.Wait()
	tracker.Reset()

	for i, counter := range readCounters {
		counter.Add(1)
		if got := counter.Load(); got != 1 {
			t.Fatalf("read counter %d detached or not reset safely, got %d", i, got)
		}
	}
	for i, counter := range writeCounters {
		counter.Add(1)
		if got := counter.Load(); got != 1 {
			t.Fatalf("write counter %d detached or not reset safely, got %d", i, got)
		}
	}
	stats := *tracker.GetStats()
	if len(stats) != 6 {
		t.Fatalf("expected live counters to remain registered after reset, got %d stats: %#v", len(stats), stats)
	}
}

func TestStatsTrackerResetWaitsForInflightRead(t *testing.T) {
	tracker := NewStatsTracker()
	raw := newBlockingTestConn()
	wrapped := tracker.RoutedConnection(context.Background(), raw, adapter.InboundContext{Inbound: "in"}, nil, fakeStatsOutbound{tag: "out"})

	readDone := make(chan error, 1)
	go func() {
		_, err := wrapped.Read(make([]byte, 1))
		readDone <- err
	}()
	waitForTestChannel(t, raw.readStarted, "read did not start")

	resetDone := make(chan struct{})
	go func() {
		tracker.Reset()
		close(resetDone)
	}()

	select {
	case <-resetDone:
		t.Fatal("Reset returned while a wrapped stats connection was still reading")
	case <-time.After(20 * time.Millisecond):
	}
	_ = raw.Close()
	waitForTestChannel(t, resetDone, "Reset did not finish after read was closed")
	if err := <-readDone; !errors.Is(err, net.ErrClosed) {
		t.Fatalf("expected closed read error, got %v", err)
	}

	tracker.access.Lock()
	active := tracker.inflight.Active()
	tracker.access.Unlock()
	if active != 0 {
		t.Fatalf("stats tracker has active writers after reset: %d", active)
	}
}

type fakeStatsOutbound struct {
	tag string
}

func (o fakeStatsOutbound) Type() string {
	return "test"
}

func (o fakeStatsOutbound) Tag() string {
	return o.tag
}

func (o fakeStatsOutbound) Network() []string {
	return nil
}

func (o fakeStatsOutbound) Dependencies() []string {
	return nil
}

func (o fakeStatsOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, net.ErrClosed
}

func (o fakeStatsOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, net.ErrClosed
}
