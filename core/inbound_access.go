package core

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
	"golang.org/x/time/rate"
)

type inboundAccessRouter struct{ adapter.Router }

var activePeerConnections = struct {
	sync.Mutex
	byUser map[string]*peerConnectionPool
}{byUser: make(map[string]*peerConnectionPool)}

type peerConnectionPool struct {
	policy ipmonitor.ConnectionPolicy
	count  uint32
	up     *rate.Limiter
	down   *rate.Limiter
}

func registerAccessInbound[Options any](registry *inbound.Registry, protocol string, constructor inbound.ConstructorFunc[Options]) {
	inbound.Register(registry, protocol, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options Options) (adapter.Inbound, error) {
		return constructor(ctx, inboundAccessRouter{router}, logger, tag, options)
	})
}

func (router inboundAccessRouter) RouteConnectionEx(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, onClose network.CloseHandlerFunc) {
	if pool, done, allowed := admitInbound(ctx, conn, metadata, onClose, true); allowed {
		if pool.metersTraffic() {
			limitCtx, cancel := context.WithCancel(ctx)
			conn = peerRateConn{Conn: conn, ctx: limitCtx, cancel: cancel, pool: pool}
			done = network.AppendClose(done, func(error) { cancel() })
		}
		router.Router.RouteConnectionEx(ctx, conn, metadata, done)
	}
}

func (router inboundAccessRouter) RoutePacketConnectionEx(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, onClose network.CloseHandlerFunc) {
	if pool, done, allowed := admitInbound(ctx, conn, metadata, onClose, false); allowed {
		if pool.metersTraffic() {
			limitCtx, cancel := context.WithCancel(ctx)
			conn = peerRatePacketConn{PacketConn: conn, ctx: limitCtx, cancel: cancel, pool: pool}
			done = network.AppendClose(done, func(error) { cancel() })
		}
		router.Router.RoutePacketConnectionEx(ctx, conn, metadata, done)
	}
}

func admitInbound(ctx context.Context, conn io.Closer, metadata adapter.InboundContext, onClose network.CloseHandlerFunc, tcp bool) (*peerConnectionPool, network.CloseHandlerFunc, bool) {
	allowed, expiry, policy := ipmonitor.ObserveConnection(ctx, metadata.User, sourceIPFromMetadata(metadata))
	if allowed && policy.ProbeDestination != "" {
		destination, err := netip.ParseAddrPort(policy.ProbeDestination)
		allowed = tcp && err == nil && metadata.Destination.Addr.Unmap() == destination.Addr() && metadata.Destination.Port == destination.Port()
	}
	var deadline time.Time
	var awaitingTraffic bool
	if allowed && policy.NodeUUID != "" {
		deadline, awaitingTraffic = peerAccessDeadline(ctx, policy)
		allowed = time.Now().Before(deadline)
	}
	var pool *peerConnectionPool
	if allowed && policy != (ipmonitor.ConnectionPolicy{}) {
		activePeerConnections.Lock()
		pool = activePeerConnections.byUser[metadata.User]
		if pool == nil || pool.policy != policy {
			pool = &peerConnectionPool{policy: policy, up: peerLimiter(policy.Upload), down: peerLimiter(policy.Download)}
			if policy.Speed != 0 {
				pool.up = peerLimiter(policy.Speed)
				pool.down = pool.up
			}
			activePeerConnections.byUser[metadata.User] = pool
		}
		allowed = policy.Connections == 0 || pool.count < policy.Connections
		if allowed {
			pool.count++
		}
		activePeerConnections.Unlock()
	}
	if !allowed {
		_ = conn.Close()
		if onClose != nil {
			onClose(net.ErrClosed)
		}
		return nil, nil, false
	}
	if expiry == 0 && pool == nil {
		return nil, onClose, true
	}
	timer := closeAtExpiry(conn, expiry)
	var stopLease context.CancelFunc
	if !deadline.IsZero() {
		leaseContext, cancel := context.WithCancel(ctx)
		stopLease = cancel
		go watchPeerAccess(leaseContext, conn, policy, deadline, awaitingTraffic)
	}
	return pool, network.OnceClose(func(err error) {
		if stopLease != nil {
			stopLease()
		}
		if timer != nil {
			timer.Stop()
		}
		if pool != nil {
			activePeerConnections.Lock()
			pool.count--
			if pool.count == 0 {
				time.AfterFunc(time.Second, func() {
					activePeerConnections.Lock()
					defer activePeerConnections.Unlock()
					if pool.count == 0 && activePeerConnections.byUser[metadata.User] == pool {
						delete(activePeerConnections.byUser, metadata.User)
					}
				})
			}
			activePeerConnections.Unlock()
		}
		if onClose != nil {
			onClose(err)
		}
	}), true
}

func peerAccessDeadline(ctx context.Context, policy ipmonitor.ConnectionPolicy) (time.Time, bool) {
	leave, ok := database.TryEnterDBOperation()
	if !ok {
		return time.Time{}, false
	}
	defer leave()
	db := database.GetDB()
	if db == nil {
		return time.Time{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var clocks struct{ LastSync, PeerAccessUntil, StartedAt int64 }
	err := db.WithContext(ctx).Raw(`SELECT panel.last_sync, COALESCE(panel.peer_access_until, 0) AS peer_access_until, COALESCE(traffic.started_at, 0) AS started_at
		FROM node_panels AS panel LEFT JOIN peer_traffics AS traffic ON traffic.node_uuid = panel.uuid AND traffic.client_id = ?
		WHERE panel.id = 1 AND panel.role = 'slave' AND panel.uuid = ?`, policy.ClientID, policy.NodeUUID).Scan(&clocks).Error
	if err != nil || clocks.LastSync <= 0 {
		return time.Time{}, false
	}
	deadline := time.Unix(clocks.LastSync, 0).Add(model.PeerAccessLease)
	if clocks.PeerAccessUntil > 0 {
		deadline = time.Unix(min(deadline.Unix(), clocks.PeerAccessUntil), 0)
	}
	if !policy.DelayStart || policy.AutoReset {
		return deadline, false
	}
	if clocks.StartedAt == 0 {
		return deadline, true
	}
	return time.Unix(min(deadline.Unix(), clocks.StartedAt+policy.Period), 0), false
}

func watchPeerAccess(ctx context.Context, conn io.Closer, policy ipmonitor.ConnectionPolicy, deadline time.Time, awaitingTraffic bool) {
	timer := time.NewTimer(peerAccessCheckAfter(deadline, awaitingTraffic))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			deadline, awaitingTraffic = peerAccessDeadline(ctx, policy)
			if !time.Now().Before(deadline) {
				_ = conn.Close()
				return
			}
			timer.Reset(peerAccessCheckAfter(deadline, awaitingTraffic))
		}
	}
}

func peerAccessCheckAfter(deadline time.Time, awaitingTraffic bool) time.Duration {
	until := time.Until(deadline)
	if awaitingTraffic {
		return min(until, time.Second)
	}
	return until
}

func (pool *peerConnectionPool) metersTraffic() bool {
	return pool != nil && (pool.up != nil || pool.down != nil || pool.policy.Volume > 0 || pool.policy.DelayStart)
}

func (pool *peerConnectionPool) charge(ctx context.Context, limiter *rate.Limiter, length int) error {
	if err := waitPeerRate(ctx, limiter, length); err != nil {
		return err
	}
	policy := pool.policy
	if length == 0 || (policy.Volume == 0 && !policy.DelayStart) {
		return nil
	}
	if (policy.Volume > 0 && int64(length) > policy.Volume) || policy.ClientID == 0 || policy.NodeUUID == "" {
		return errors.New("node traffic quota exhausted")
	}
	leave, ok := database.TryEnterDBOperation()
	if !ok {
		return errors.New("node traffic accounting unavailable")
	}
	defer leave()
	db := database.GetDB()
	if db == nil {
		return errors.New("node traffic accounting unavailable")
	}
	now := time.Now().Unix()
	var startedAt, delay, reset, used int64
	if policy.DelayStart {
		startedAt = now
		if !policy.AutoReset {
			delay = policy.Period
		}
	}
	if policy.AutoReset {
		reset = policy.Period
	}
	if policy.Volume > 0 {
		used = int64(length)
	}
	result := db.WithContext(ctx).Exec(`INSERT INTO peer_traffics(node_uuid, client_id, used, started_at, reset_at)
		VALUES (@node, @client, @used, @started, CASE WHEN @reset > 0 THEN @now + @reset ELSE 0 END)
		ON CONFLICT(node_uuid, client_id) DO UPDATE SET
			used = CASE WHEN @reset > 0 AND reset_at BETWEEN 1 AND @now THEN excluded.used ELSE used + excluded.used END,
			started_at = CASE WHEN started_at = 0 THEN excluded.started_at ELSE started_at END,
			reset_at = CASE
				WHEN @reset > 0 AND reset_at BETWEEN 1 AND @now THEN reset_at + ((@now - reset_at) / @reset + 1) * @reset
				WHEN @reset > 0 AND reset_at = 0 THEN excluded.reset_at
				ELSE reset_at END
		WHERE (@delay = 0 OR started_at = 0 OR started_at + @delay > @now)
			AND (@unlimited OR used <= @remaining OR (@reset > 0 AND reset_at BETWEEN 1 AND @now))`,
		sql.Named("node", policy.NodeUUID), sql.Named("client", policy.ClientID), sql.Named("used", used),
		sql.Named("started", startedAt), sql.Named("now", now), sql.Named("reset", reset), sql.Named("delay", delay),
		sql.Named("unlimited", policy.Volume == 0), sql.Named("remaining", policy.Volume-used))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("node traffic quota exhausted")
	}
	return nil
}

func peerLimiter(speed uint64) *rate.Limiter {
	if speed == 0 {
		return nil
	}
	burst := 64 << 10
	if speed < 64<<10 {
		burst = int(speed)
	}
	return rate.NewLimiter(rate.Limit(speed), burst)
}

func waitPeerRate(ctx context.Context, limiter *rate.Limiter, length int) error {
	if limiter == nil {
		return nil
	}
	for length > 0 {
		chunk := min(length, limiter.Burst())
		if err := limiter.WaitN(ctx, chunk); err != nil {
			return err
		}
		length -= chunk
	}
	return nil
}

type peerRateConn struct {
	net.Conn
	ctx    context.Context
	cancel context.CancelFunc
	pool   *peerConnectionPool
}

func (conn peerRateConn) Upstream() any { return conn.Conn }

func (conn peerRateConn) Close() error {
	conn.cancel()
	return conn.Conn.Close()
}

func (conn peerRateConn) Read(data []byte) (int, error) {
	length, err := conn.Conn.Read(data)
	if limitErr := conn.pool.charge(conn.ctx, conn.pool.up, length); limitErr != nil {
		return 0, limitErr
	}
	return length, err
}

func (conn peerRateConn) Write(data []byte) (int, error) {
	if err := conn.pool.charge(conn.ctx, conn.pool.down, len(data)); err != nil {
		return 0, err
	}
	return conn.Conn.Write(data)
}

type peerRatePacketConn struct {
	network.PacketConn
	ctx    context.Context
	cancel context.CancelFunc
	pool   *peerConnectionPool
}

func (conn peerRatePacketConn) Upstream() any { return conn.PacketConn }

func (conn peerRatePacketConn) Close() error {
	conn.cancel()
	return conn.PacketConn.Close()
}

func (conn peerRatePacketConn) ReadPacket(buffer *buf.Buffer) (metadata.Socksaddr, error) {
	destination, err := conn.PacketConn.ReadPacket(buffer)
	if err == nil {
		err = conn.pool.charge(conn.ctx, conn.pool.up, buffer.Len())
	}
	return destination, err
}

func (conn peerRatePacketConn) WritePacket(buffer *buf.Buffer, destination metadata.Socksaddr) error {
	if err := conn.pool.charge(conn.ctx, conn.pool.down, buffer.Len()); err != nil {
		buffer.Release()
		return err
	}
	return conn.PacketConn.WritePacket(buffer, destination)
}
