package core

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
)

type ConnectionInfo struct {
	ID         string
	Conn       net.Conn
	PacketConn network.PacketConn
	Flow       *trackedFlow
	Inbound    string
	Type       string // "tcp", "udp" or "flow"
}

type ConnTracker struct {
	access      sync.Mutex
	connections map[string]*ConnectionInfo
	inflight    *trackerWaitGroup
	epoch       uint64
}

func NewConnTracker() *ConnTracker {
	return &ConnTracker{
		connections: make(map[string]*ConnectionInfo),
		inflight:    newTrackerWaitGroup(),
	}
}

func (c *ConnTracker) Reset() {
	c.access.Lock()
	var flows []*trackedFlow
	for _, connInfo := range c.connections {
		if connInfo.Conn != nil {
			_ = connInfo.Conn.Close()
		}
		if connInfo.PacketConn != nil {
			_ = connInfo.PacketConn.Close()
		}
		if connInfo.Flow != nil {
			flows = append(flows, connInfo.Flow)
		}
	}
	c.connections = make(map[string]*ConnectionInfo)
	c.epoch++
	waitGroup := c.inflight
	c.inflight = newTrackerWaitGroup()
	c.access.Unlock()
	// Flow handles re-enter untrackConnection (which takes c.access) via the
	// dispatcher's synchronous CloseFlow callback, so close them after unlocking.
	for _, flow := range flows {
		_ = flow.Close()
	}
	waitForTrackerIdle("connection tracker", waitGroup, trackerResetWaitTimeout)
}

func (c *ConnTracker) generateConnectionID() string {
	return uuid.Must(uuid.NewV4()).String()
}

func (c *ConnTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	connID := c.generateConnectionID()
	connInfo := &ConnectionInfo{
		ID:      connID,
		Conn:    conn,
		Inbound: metadata.Inbound,
		Type:    "tcp",
	}

	epoch, waitGroup := c.trackConnection(connID, connInfo)

	return c.createWrappedConn(conn, connID, epoch, waitGroup)
}

func (c *ConnTracker) RoutedPacketConnection(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) network.PacketConn {
	connID := c.generateConnectionID()
	connInfo := &ConnectionInfo{
		ID:         connID,
		PacketConn: conn,
		Inbound:    metadata.Inbound,
		Type:       "udp",
	}

	epoch, waitGroup := c.trackConnection(connID, connInfo)

	return c.createWrappedPacketConn(conn, connID, epoch, waitGroup)
}

// RoutedFlow tracks TUN-level flows (sing-tun FlowTracker) so that
// CloseConnByInbound can also terminate flow-based connections via their
// FlowHandle. Flows carry no net.Conn/PacketConn, so the tracked
// ConnectionInfo stores the handle for forced close.
func (c *ConnTracker) RoutedFlow(ctx context.Context, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) tun.FlowTracker {
	connID := c.generateConnectionID()
	connInfo := &ConnectionInfo{
		ID:      connID,
		Inbound: metadata.Inbound,
		Type:    "flow",
	}
	epoch, waitGroup := c.trackConnection(connID, connInfo)
	flow := &trackedFlow{
		tracker:   c,
		connID:    connID,
		epoch:     epoch,
		waitGroup: waitGroup,
	}
	connInfo.Flow = flow
	return flow
}

type trackedFlow struct {
	tracker     *ConnTracker
	connID      string
	epoch       uint64
	waitGroup   *trackerWaitGroup
	handle      tun.FlowHandle
	untrackOnce sync.Once
}

func (t *trackedFlow) AttachFlow(handle tun.FlowHandle) {
	t.handle = handle
}

func (t *trackedFlow) CountForward(n int) {}
func (t *trackedFlow) CountReverse(n int) {}
func (t *trackedFlow) FlowEstablished()   {}

func (t *trackedFlow) CloseFlow(reason tun.FlowCloseReason) {
	t.doUntrack()
}

func (t *trackedFlow) doUntrack() {
	t.untrackOnce.Do(func() {
		t.tracker.untrackConnection(t.connID, t.epoch)
		t.waitGroup.Done()
	})
}

// Close terminates the underlying flow, used by CloseConnByInbound/Reset.
func (t *trackedFlow) Close() error {
	if t.handle != nil {
		t.handle.CloseFlow()
	}
	t.doUntrack()
	return nil
}

func (c *ConnTracker) CloseConnByInbound(inbound string) int {
	c.access.Lock()
	var flows []*trackedFlow
	closedCount := 0
	for connID, connInfo := range c.connections {
		if connInfo.Inbound == inbound {
			if connInfo.Conn != nil {
				_ = connInfo.Conn.Close()
			}
			if connInfo.PacketConn != nil {
				_ = connInfo.PacketConn.Close()
			}
			if connInfo.Flow != nil {
				flows = append(flows, connInfo.Flow)
			}
			delete(c.connections, connID)
			closedCount++
		}
	}
	c.access.Unlock()
	// Flow handles re-enter untrackConnection (which takes c.access) via the
	// dispatcher's synchronous CloseFlow callback, so close them after unlocking.
	for _, flow := range flows {
		_ = flow.Close()
	}
	return closedCount
}

func (c *ConnTracker) trackConnection(connID string, connInfo *ConnectionInfo) (uint64, *trackerWaitGroup) {
	c.access.Lock()
	defer c.access.Unlock()
	c.inflight.Add()
	c.connections[connID] = connInfo
	return c.epoch, c.inflight
}

func (c *ConnTracker) untrackConnection(connID string, epoch uint64) {
	c.access.Lock()
	defer c.access.Unlock()
	if epoch != c.epoch {
		return
	}
	delete(c.connections, connID)
}

// shouldUntrackIOErr reports whether err indicates the connection is done (peer closed, reset, etc.).
func shouldUntrackIOErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		// Temporary() is deprecated; a non-timeout net error indicates the connection is done.
		return !ne.Timeout()
	}
	return true
}

func (c *ConnTracker) createWrappedConn(conn net.Conn, connID string, epoch uint64, waitGroup *trackerWaitGroup) *wrappedConn {
	return &wrappedConn{
		Conn:      conn,
		tracker:   c,
		connID:    connID,
		epoch:     epoch,
		waitGroup: waitGroup,
	}
}

func (c *ConnTracker) createWrappedPacketConn(conn network.PacketConn, connID string, epoch uint64, waitGroup *trackerWaitGroup) *wrappedPacketConn {
	return &wrappedPacketConn{
		PacketConn: conn,
		tracker:    c,
		connID:     connID,
		epoch:      epoch,
		waitGroup:  waitGroup,
	}
}

type wrappedConn struct {
	net.Conn
	tracker     *ConnTracker
	connID      string
	epoch       uint64
	waitGroup   *trackerWaitGroup
	untrackOnce sync.Once
}

func (w *wrappedConn) doUntrack() {
	w.untrackOnce.Do(func() {
		w.tracker.untrackConnection(w.connID, w.epoch)
		w.waitGroup.Done()
	})
}

func (w *wrappedConn) Read(b []byte) (int, error) {
	n, err := w.Conn.Read(b)
	if shouldUntrackIOErr(err) {
		w.doUntrack()
	}
	return n, err
}

func (w *wrappedConn) Write(b []byte) (int, error) {
	n, err := w.Conn.Write(b)
	if err != nil && shouldUntrackIOErr(err) {
		w.doUntrack()
	}
	return n, err
}

func (w *wrappedConn) Close() error {
	w.doUntrack()
	return w.Conn.Close()
}

func (w *wrappedConn) Upstream() any {
	return w.Conn
}

type wrappedPacketConn struct {
	network.PacketConn
	tracker     *ConnTracker
	connID      string
	epoch       uint64
	waitGroup   *trackerWaitGroup
	untrackOnce sync.Once
}

func (w *wrappedPacketConn) doUntrack() {
	w.untrackOnce.Do(func() {
		w.tracker.untrackConnection(w.connID, w.epoch)
		w.waitGroup.Done()
	})
}

func (w *wrappedPacketConn) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	dest, err := w.PacketConn.ReadPacket(buffer)
	if shouldUntrackIOErr(err) {
		w.doUntrack()
	}
	return dest, err
}

func (w *wrappedPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	err := w.PacketConn.WritePacket(buffer, destination)
	if err != nil && shouldUntrackIOErr(err) {
		w.doUntrack()
	}
	return err
}

func (w *wrappedPacketConn) Close() error {
	w.doUntrack()
	return w.PacketConn.Close()
}

func (w *wrappedPacketConn) Upstream() any {
	return w.PacketConn
}
