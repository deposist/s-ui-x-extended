package core

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
)

func TestConnTrackerResetWaitsForBlockedRead(t *testing.T) {
	tracker := NewConnTracker()
	raw := newBlockingTestConn()
	wrapped := tracker.RoutedConnection(context.Background(), raw, adapter.InboundContext{Inbound: "in"}, nil, nil)

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

	waitForTestChannel(t, resetDone, "Reset did not wait for blocked read to finish")
	if err := <-readDone; !errors.Is(err, net.ErrClosed) {
		t.Fatalf("expected closed read error, got %v", err)
	}

	tracker.access.Lock()
	connectionCount := len(tracker.connections)
	active := tracker.inflight.Active()
	tracker.access.Unlock()
	if connectionCount != 0 || active != 0 {
		t.Fatalf("tracker not idle after reset: connections=%d active=%d", connectionCount, active)
	}
}

type blockingTestConn struct {
	readStarted chan struct{}
	closed      chan struct{}
	startOnce   sync.Once
	closeOnce   sync.Once
}

func newBlockingTestConn() *blockingTestConn {
	return &blockingTestConn{
		readStarted: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (c *blockingTestConn) Read([]byte) (int, error) {
	c.startOnce.Do(func() {
		close(c.readStarted)
	})
	<-c.closed
	return 0, net.ErrClosed
}

func (c *blockingTestConn) Write(b []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
		return len(b), nil
	}
}

func (c *blockingTestConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *blockingTestConn) LocalAddr() net.Addr {
	return testAddr("local")
}

func (c *blockingTestConn) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (c *blockingTestConn) SetDeadline(time.Time) error {
	return nil
}

func (c *blockingTestConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *blockingTestConn) SetWriteDeadline(time.Time) error {
	return nil
}

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func waitForTestChannel(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}
