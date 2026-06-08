package core

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/ipmonitor"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
)

type Counter struct {
	read  *atomic.Int64
	write *atomic.Int64
}

type StatsTracker struct {
	access    sync.Mutex
	inbounds  map[string]Counter
	outbounds map[string]Counter
	users     map[string]Counter
	inflight  *trackerWaitGroup
}

func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		inbounds:  make(map[string]Counter),
		outbounds: make(map[string]Counter),
		users:     make(map[string]Counter),
		inflight:  newTrackerWaitGroup(),
	}
}

func (c *StatsTracker) Reset() {
	c.access.Lock()
	resetCounters(c.inbounds)
	resetCounters(c.outbounds)
	resetCounters(c.users)
	waitGroup := c.inflight
	c.inflight = newTrackerWaitGroup()
	c.access.Unlock()
	waitForTrackerIdle("stats tracker", waitGroup, trackerResetWaitTimeout)
}

func resetCounters(counters map[string]Counter) {
	for _, counter := range counters {
		counter.read.Store(0)
		counter.write.Store(0)
	}
}

func (c *StatsTracker) getReadCounters(inbound string, outbound string, user string) ([]*atomic.Int64, []*atomic.Int64) {
	c.access.Lock()
	defer c.access.Unlock()
	return c.getReadCountersLocked(inbound, outbound, user)
}

func (c *StatsTracker) getTrackedReadCounters(inbound string, outbound string, user string) ([]*atomic.Int64, []*atomic.Int64, *trackerWaitGroup) {
	c.access.Lock()
	defer c.access.Unlock()
	readCounter, writeCounter := c.getReadCountersLocked(inbound, outbound, user)
	c.inflight.Add()
	return readCounter, writeCounter, c.inflight
}

func (c *StatsTracker) getReadCountersLocked(inbound string, outbound string, user string) ([]*atomic.Int64, []*atomic.Int64) {
	var readCounter []*atomic.Int64
	var writeCounter []*atomic.Int64

	if inbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.inbounds, inbound).read)
		writeCounter = append(writeCounter, c.inbounds[inbound].write)
	}
	if outbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.outbounds, outbound).read)
		writeCounter = append(writeCounter, c.outbounds[outbound].write)
	}
	if user != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.users, user).read)
		writeCounter = append(writeCounter, c.users[user].write)
	}
	return readCounter, writeCounter
}

func (c *StatsTracker) loadOrCreateCounter(obj *map[string]Counter, name string) Counter {
	counter, loaded := (*obj)[name]
	if loaded {
		return counter
	}
	counter = Counter{read: &atomic.Int64{}, write: &atomic.Int64{}}
	(*obj)[name] = counter
	return counter
}

func (c *StatsTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	sourceIP := sourceIPFromMetadata(metadata)
	if !ipmonitor.Allow(metadata.User, sourceIP) {
		_ = conn.Close()
		return conn
	}
	ipmonitor.Record(metadata.User, sourceIP)
	readCounter, writeCounter, waitGroup := c.getTrackedReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User)
	return newStatsTrackedConn(bufio.NewInt64CounterConn(conn, readCounter, writeCounter), waitGroup)
}

func (c *StatsTracker) RoutedPacketConnection(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) network.PacketConn {
	sourceIP := sourceIPFromMetadata(metadata)
	if !ipmonitor.Allow(metadata.User, sourceIP) {
		_ = conn.Close()
		return conn
	}
	ipmonitor.Record(metadata.User, sourceIP)
	readCounter, writeCounter, waitGroup := c.getTrackedReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User)
	return newStatsTrackedPacketConn(bufio.NewInt64CounterPacketConn(conn, readCounter, nil, writeCounter, nil), waitGroup)
}

func newStatsTrackedConn(conn net.Conn, waitGroup *trackerWaitGroup) net.Conn {
	return &statsTrackedConn{
		Conn:      conn,
		waitGroup: waitGroup,
	}
}

type statsTrackedConn struct {
	net.Conn
	waitGroup *trackerWaitGroup
	doneOnce  sync.Once
}

func (w *statsTrackedConn) done() {
	w.doneOnce.Do(func() {
		w.waitGroup.Done()
	})
}

func (w *statsTrackedConn) Read(b []byte) (int, error) {
	n, err := w.Conn.Read(b)
	if shouldUntrackIOErr(err) {
		w.done()
	}
	return n, err
}

func (w *statsTrackedConn) Write(b []byte) (int, error) {
	n, err := w.Conn.Write(b)
	if err != nil && shouldUntrackIOErr(err) {
		w.done()
	}
	return n, err
}

func (w *statsTrackedConn) Close() error {
	w.done()
	return w.Conn.Close()
}

func (w *statsTrackedConn) Upstream() any {
	return w.Conn
}

func newStatsTrackedPacketConn(conn network.PacketConn, waitGroup *trackerWaitGroup) network.PacketConn {
	return &statsTrackedPacketConn{
		PacketConn: conn,
		waitGroup:  waitGroup,
	}
}

type statsTrackedPacketConn struct {
	network.PacketConn
	waitGroup *trackerWaitGroup
	doneOnce  sync.Once
}

func (w *statsTrackedPacketConn) done() {
	w.doneOnce.Do(func() {
		w.waitGroup.Done()
	})
}

func (w *statsTrackedPacketConn) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	dest, err := w.PacketConn.ReadPacket(buffer)
	if shouldUntrackIOErr(err) {
		w.done()
	}
	return dest, err
}

func (w *statsTrackedPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	err := w.PacketConn.WritePacket(buffer, destination)
	if err != nil && shouldUntrackIOErr(err) {
		w.done()
	}
	return err
}

func (w *statsTrackedPacketConn) Close() error {
	w.done()
	return w.PacketConn.Close()
}

func (w *statsTrackedPacketConn) Upstream() any {
	return w.PacketConn
}

func sourceIPFromMetadata(metadata adapter.InboundContext) string {
	if metadata.Source.Addr.IsValid() {
		return metadata.Source.Addr.String()
	}
	return ""
}

func (c *StatsTracker) GetStats() *[]model.Stats {
	c.access.Lock()
	defer c.access.Unlock()

	dt := time.Now().Unix()

	s := []model.Stats{}
	for inbound, counter := range c.inbounds {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for outbound, counter := range c.outbounds {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for user, counter := range c.users {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: true,
				Traffic:   up,
			})
		}
	}
	return &s
}
